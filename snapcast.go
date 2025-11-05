package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type SnapcastConnection struct {
	conn   net.Conn
	mu     sync.Mutex
	nextID int
}

type SnapcastRequest struct {
	ID      int            `json:"id"`
	JSONRpc string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type SnapcastResponse struct {
	ID      int            `json:"id"`
	JSONRpc string         `json:"jsonrpc"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *SnapcastError `json:"error,omitempty"`
}

type SnapcastError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *SnapServer) connect(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), 5*time.Second)
			if err != nil {
				log.Printf("Failed to connect to Snapcast server at %s:%d: %v", s.Host, s.Port, err)
				time.Sleep(10 * time.Second)
				continue
			}

			s.conn = &SnapcastConnection{
				conn:   conn,
				nextID: 1,
			}

			log.Printf("Connected to Snapcast server at %s:%d", s.Host, s.Port)

			// Initial status sync
			s.syncStatus()

			// Keep connection alive and handle incoming messages
			buf := make([]byte, 8192)
			for {
				if err := conn.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
					log.Printf("Failed to set read deadline: %v", err)
					break
				}
				n, err := conn.Read(buf)
				if err != nil {
					log.Printf("Connection to Snapcast server at %s:%d lost: %v", s.Host, s.Port, err)
					conn.Close()
					s.conn = nil
					break
				}
				// Log received data for debugging
				if n > 0 {
					log.Printf("Received %d bytes from Snapcast server", n)
				}
			}

			time.Sleep(5 * time.Second)
		}
	}
}

func (s *SnapServer) syncStatus() {
	if s.conn == nil {
		return
	}

	status, err := s.conn.request("Server.GetStatus", nil)
	if err != nil {
		log.Printf("Failed to get server status: %v", err)
		return
	}

	// Parse server status
	if serverData, ok := status["server"].(map[string]any); ok {
		// Parse streams
		if streamsData, ok := serverData["streams"].([]any); ok {
			streams := make([]Stream, 0, len(streamsData))
			for _, streamData := range streamsData {
				if stream, ok := streamData.(map[string]any); ok {
					streamObj := Stream{
						ID:     getString(stream, "id"),
						Status: getString(stream, "status"),
						Meta:   make(map[string]any),
					}
					if meta, ok := stream["meta"].(map[string]any); ok {
						streamObj.Meta = meta
					}
					streams = append(streams, streamObj)
				}
			}
			s.Streams = streams
		}

		// Parse groups and clients
		if groupsData, ok := serverData["groups"].([]any); ok {
			clients := []Client{}
			clientGroups := make(map[string]string)
			for _, groupData := range groupsData {
				if group, ok := groupData.(map[string]any); ok {
					streamID := getString(group, "stream_id")
					groupID := getString(group, "id")
					if clientsData, ok := group["clients"].([]any); ok {
						for _, clientData := range clientsData {
							if clientMap, ok := clientData.(map[string]any); ok {
								clientID := getString(clientMap, "id")
								client := Client{
									ID:        clientID,
									Connected: getBool(clientMap, "connected"),
									Stream:    streamID,
								}

								// Cache the client-to-group mapping
								clientGroups[clientID] = groupID

								// Parse config
								if config, ok := clientMap["config"].(map[string]any); ok {
									client.Name = getString(config, "name")
									if client.Name == "" {
										if host, ok := clientMap["host"].(map[string]any); ok {
											client.Name = getString(host, "name")
										}
									}
									client.Latency = getInt(config, "latency")
									if volume, ok := config["volume"].(map[string]any); ok {
										client.Volume = getInt(volume, "percent")
										client.Muted = getBool(volume, "muted")
									}
								}

								clients = append(clients, client)
							}
						}
					}
				}
			}
			s.Clients = clients
			s.clientGroups = clientGroups
		}
	}
}

func (s *SnapServer) setClientMuted(clientID string, muted bool) error {
	if s.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	_, err := s.conn.request("Client.SetVolume", map[string]any{
		"id": clientID,
		"volume": map[string]any{
			"muted": muted,
		},
	})
	if err != nil {
		return err
	}

	s.syncStatus()
	return nil
}

func (s *SnapServer) setClientLatency(clientID string, latency int) error {
	if s.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	_, err := s.conn.request("Client.SetLatency", map[string]any{
		"id":      clientID,
		"latency": latency,
	})
	if err != nil {
		return err
	}

	s.syncStatus()
	return nil
}

func (s *SnapServer) deleteClient(clientID string) error {
	if s.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	_, err := s.conn.request("Server.DeleteClient", map[string]any{
		"id": clientID,
	})
	if err != nil {
		return err
	}

	s.syncStatus()
	return nil
}

func (s *SnapServer) setClientStream(clientID string, streamID string) error {
	if s.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	// Get the group ID from the cache
	groupID, exists := s.clientGroups[clientID]
	if !exists {
		// If not in cache, refresh status and try again
		s.syncStatus()
		groupID, exists = s.clientGroups[clientID]
		if !exists {
			return fmt.Errorf("could not find group for client %s", clientID)
		}
	}

	_, err := s.conn.request("Group.SetStream", map[string]any{
		"id":     groupID,
		"stream": streamID,
	})
	if err != nil {
		return err
	}

	s.syncStatus()
	return nil
}

func (c *SnapcastConnection) request(method string, params map[string]any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID
	c.nextID++

	req := SnapcastRequest{
		ID:      id,
		JSONRpc: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	data = append(data, '\n')

	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}
	_, err = c.conn.Write(data)
	if err != nil {
		return nil, err
	}

	// Read response - accumulate data until we have a complete JSON message
	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}
	buf := make([]byte, 65536)
	totalRead := 0

	for {
		n, err := c.conn.Read(buf[totalRead:])
		if err != nil {
			return nil, err
		}
		totalRead += n

		// Try to parse JSON - if successful, we have a complete message
		var resp SnapcastResponse
		err = json.Unmarshal(buf[:totalRead], &resp)
		if err == nil {
			// Successfully parsed
			if resp.Error != nil {
				return nil, fmt.Errorf("snapcast error: %s", resp.Error.Message)
			}
			return resp.Result, nil
		}

		// If buffer is full and still can't parse, something is wrong
		if totalRead >= len(buf) {
			return nil, fmt.Errorf("response too large or invalid JSON")
		}
	}

}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
