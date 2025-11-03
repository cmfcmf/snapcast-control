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
	ID      int                    `json:"id"`
	JSONRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type SnapcastResponse struct {
	ID      int                    `json:"id"`
	JSONRpc string                 `json:"jsonrpc"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *SnapcastError         `json:"error,omitempty"`
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

			// Keep connection alive and handle disconnections
			buf := make([]byte, 8192)
			for {
				conn.SetReadDeadline(time.Now().Add(90 * time.Second))
				_, err := conn.Read(buf)
				if err != nil {
					log.Printf("Connection to Snapcast server at %s:%d lost: %v", s.Host, s.Port, err)
					conn.Close()
					s.conn = nil
					break
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
	if serverData, ok := status["server"].(map[string]interface{}); ok {
		// Parse streams
		if streamsData, ok := serverData["streams"].([]interface{}); ok {
			streams := make([]Stream, 0, len(streamsData))
			for _, streamData := range streamsData {
				if stream, ok := streamData.(map[string]interface{}); ok {
					streamObj := Stream{
						ID:     getString(stream, "id"),
						Status: getString(stream, "status"),
						Meta:   make(map[string]interface{}),
					}
					if meta, ok := stream["meta"].(map[string]interface{}); ok {
						streamObj.Meta = meta
					}
					streams = append(streams, streamObj)
				}
			}
			s.Streams = streams
		}

		// Parse groups and clients
		if groupsData, ok := serverData["groups"].([]interface{}); ok {
			clients := []Client{}
			for _, groupData := range groupsData {
				if group, ok := groupData.(map[string]interface{}); ok {
					streamID := getString(group, "stream_id")
					if clientsData, ok := group["clients"].([]interface{}); ok {
						for _, clientData := range clientsData {
							if clientMap, ok := clientData.(map[string]interface{}); ok {
								client := Client{
									ID:        getString(clientMap, "id"),
									Connected: getBool(clientMap, "connected"),
									Stream:    streamID,
								}

								// Parse config
								if config, ok := clientMap["config"].(map[string]interface{}); ok {
									client.Name = getString(config, "name")
									if client.Name == "" {
										if host, ok := clientMap["host"].(map[string]interface{}); ok {
											client.Name = getString(host, "name")
										}
									}
									client.Latency = getInt(config, "latency")
									if volume, ok := config["volume"].(map[string]interface{}); ok {
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
		}
	}
}

func (s *SnapServer) setClientMuted(clientID string, muted bool) error {
	if s.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	_, err := s.conn.request("Client.SetVolume", map[string]interface{}{
		"id": clientID,
		"volume": map[string]interface{}{
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

	_, err := s.conn.request("Client.SetLatency", map[string]interface{}{
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

	_, err := s.conn.request("Server.DeleteClient", map[string]interface{}{
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

	// First, find the group ID for this client
	groupID := ""
	for _, client := range s.Clients {
		if client.ID == clientID {
			// We need to get the group from the full status
			status, err := s.conn.request("Server.GetStatus", nil)
			if err != nil {
				return err
			}

			if serverData, ok := status["server"].(map[string]interface{}); ok {
				if groupsData, ok := serverData["groups"].([]interface{}); ok {
					for _, groupData := range groupsData {
						if group, ok := groupData.(map[string]interface{}); ok {
							if clientsData, ok := group["clients"].([]interface{}); ok {
								for _, clientData := range clientsData {
									if clientMap, ok := clientData.(map[string]interface{}); ok {
										if getString(clientMap, "id") == clientID {
											groupID = getString(group, "id")
											break
										}
									}
								}
							}
							if groupID != "" {
								break
							}
						}
					}
				}
			}
			break
		}
	}

	if groupID == "" {
		return fmt.Errorf("could not find group for client %s", clientID)
	}

	_, err := s.conn.request("Group.SetStream", map[string]interface{}{
		"id":     groupID,
		"stream": streamID,
	})
	if err != nil {
		return err
	}

	s.syncStatus()
	return nil
}

func (c *SnapcastConnection) request(method string, params map[string]interface{}) (map[string]interface{}, error) {
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

	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = c.conn.Write(data)
	if err != nil {
		return nil, err
	}

	// Read response
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 65536)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	var resp SnapcastResponse
	err = json.Unmarshal(buf[:n], &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("snapcast error: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
