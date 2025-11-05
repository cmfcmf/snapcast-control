package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func snapServersHandler(w http.ResponseWriter, r *http.Request) {
	snapServersMu.RLock()
	defer snapServersMu.RUnlock()
	writeJSON(w, snapServers)
}

func mopidyServersHandler(w http.ResponseWriter, r *http.Request) {
	mopidyServersMu.RLock()
	defer mopidyServersMu.RUnlock()
	if mopidyServers == nil {
		writeJSON(w, []MopidyServer{})
	} else {
		writeJSON(w, mopidyServers)
	}
}

func clientSettingsHandler(w http.ResponseWriter, r *http.Request) {
	serverName := r.URL.Query().Get("server_name")
	clientID := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action")

	snapServersMu.RLock()
	server, exists := snapServers[serverName]
	snapServersMu.RUnlock()

	if !exists {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	var err error
	switch action {
	case "mute":
		err = server.setClientMuted(clientID, true)
	case "unmute":
		err = server.setClientMuted(clientID, false)
	case "delete":
		err = server.deleteClient(clientID)
	case "set_latency":
		latencyStr := r.URL.Query().Get("latency")
		latency, parseErr := strconv.Atoi(latencyStr)
		if parseErr != nil {
			http.Error(w, "Invalid latency", http.StatusBadRequest)
			return
		}
		err = server.setClientLatency(clientID, latency)
	case "set_stream":
		streamID := r.URL.Query().Get("stream")
		err = server.setClientStream(clientID, streamID)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error performing action %s: %v", action, err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{})
}

func browseHandler(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	name := r.URL.Query().Get("name")

	if name == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}

	mopidyServersMu.RLock()
	var mopidyServer *MopidyServer
	for _, srv := range mopidyServers {
		if srv.Name == name {
			mopidyServer = srv
			break
		}
	}
	mopidyServersMu.RUnlock()

	if mopidyServer == nil {
		http.Error(w, "Mopidy server not found", http.StatusNotFound)
		return
	}

	var params map[string]any
	if uri != "" {
		params = map[string]any{"uri": uri}
	} else {
		params = map[string]any{"uri": nil}
	}

	result, err := mopidyRPCRequest(mopidyServer, "core.library.browse", params)
	if err != nil {
		log.Printf("Error browsing: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

func playHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	uris := r.URL.Query()["uri"]

	if name == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}

	mopidyServersMu.RLock()
	var mopidyServer *MopidyServer
	for _, srv := range mopidyServers {
		if srv.Name == name {
			mopidyServer = srv
			break
		}
	}
	mopidyServersMu.RUnlock()

	if mopidyServer == nil {
		http.Error(w, "Mopidy server not found", http.StatusNotFound)
		return
	}

	// Clear tracklist
	_, err := mopidyRPCRequest(mopidyServer, "core.tracklist.clear", nil)
	if err != nil {
		log.Printf("Error clearing tracklist: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Add tracks
	tracks, err := mopidyRPCRequest(mopidyServer, "core.tracklist.add", map[string]any{
		"uris": uris,
	})
	if err != nil {
		log.Printf("Error adding tracks: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Play first track
	tracksArray, ok := tracks.([]any)
	if ok && len(tracksArray) > 0 {
		firstTrack, ok := tracksArray[0].(map[string]any)
		if ok {
			tlid := firstTrack["tlid"]
			_, err = mopidyRPCRequest(mopidyServer, "core.playback.play", map[string]any{
				"tlid": tlid,
			})
			if err != nil {
				log.Printf("Error playing track: %v", err)
				http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	writeJSON(w, map[string]any{})
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	if name == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}

	mopidyServersMu.RLock()
	var mopidyServer *MopidyServer
	for _, srv := range mopidyServers {
		if srv.Name == name {
			mopidyServer = srv
			break
		}
	}
	mopidyServersMu.RUnlock()

	if mopidyServer == nil {
		http.Error(w, "Mopidy server not found", http.StatusNotFound)
		return
	}

	_, err := mopidyRPCRequest(mopidyServer, "core.tracklist.clear", nil)
	if err != nil {
		log.Printf("Error clearing tracklist: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = mopidyRPCRequest(mopidyServer, "core.playback.stop", nil)
	if err != nil {
		log.Printf("Error stopping playback: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{})
}
