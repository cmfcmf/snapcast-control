package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
)

//go:embed frontend-react/build
var frontendFS embed.FS

var (
	snapServers   = make(map[string]*SnapServer)
	snapServersMu sync.RWMutex

	mopidyServers   []*MopidyServer
	mopidyServersMu sync.RWMutex
)

type SnapServer struct {
	Host         string
	Port         int
	Clients      []Client `json:"clients"`
	Streams      []Stream `json:"streams"`
	conn         *SnapcastConnection
	clientGroups map[string]string // maps client ID to group ID
}

type Client struct {
	ID        string `json:"id"`
	Muted     bool   `json:"muted"`
	Volume    int    `json:"volume"`
	Name      string `json:"name"`
	Latency   int    `json:"latency"`
	Connected bool   `json:"connected"`
	Stream    string `json:"stream"`
}

type Stream struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Meta   map[string]any `json:"meta"`
}

type MopidyServer struct {
	Name string `json:"name"`
	Host string `json:"-"`
	Port int    `json:"-"`
}

type BibItem struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
	Type string `json:"type"`
}

func main() {
	debug := flag.Bool("debug", false, "run in debug mode")
	port := flag.Int("port", 8080, "web server port")
	loglevel := flag.String("loglevel", "INFO", "log level")
	flag.Parse()

	// Set up logging
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
	} else {
		defer logFile.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if *debug {
		log.Println("Running in debug mode")
	}
	log.Printf("Log level: %s", *loglevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Zeroconf discovery
	go discoverSnapcastServers(ctx)
	go discoverMopidyServers(ctx)

	// Start periodic Snapcast sync
	go syncSnapServers(ctx)

	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/snap_servers.json", snapServersHandler)
	mux.HandleFunc("/mopidy_servers.json", mopidyServersHandler)
	mux.HandleFunc("/client", clientSettingsHandler)
	mux.HandleFunc("/browse.json", browseHandler)
	mux.HandleFunc("/play", playHandler)
	mux.HandleFunc("/stop", stopHandler)

	// Serve frontend static files
	frontendRoot, err := fs.Sub(frontendFS, "frontend-react/build")
	if err != nil {
		log.Fatalf("Failed to get frontend filesystem: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendRoot)))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: corsMiddleware(mux),
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting web server on port %d", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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

func discoverSnapcastServers(ctx context.Context) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalf("Failed to initialize resolver: %v", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		for {
			select {
			case entry := <-entries:
				if entry == nil {
					continue
				}
				name := entry.Instance
				if len(entry.AddrIPv4) > 0 {
					host := entry.AddrIPv4[0].String()
					port := entry.Port

					snapServersMu.Lock()
					if _, exists := snapServers[name]; !exists {
						log.Printf("Discovered Snapcast server: %s at %s:%d", name, host, port)
						server := &SnapServer{
							Host:         host,
							Port:         port,
							Clients:      []Client{},
							Streams:      []Stream{},
							clientGroups: make(map[string]string),
						}
						snapServers[name] = server
						go server.connect(ctx)
					}
					snapServersMu.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	err = resolver.Browse(ctx, "_snapcast-tcp._tcp", "local.", entries)
	if err != nil {
		log.Printf("Failed to browse for Snapcast servers: %v", err)
	}
}

func discoverMopidyServers(ctx context.Context) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalf("Failed to initialize resolver: %v", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		for {
			select {
			case entry := <-entries:
				if entry == nil {
					continue
				}
				name := entry.Instance
				if len(entry.AddrIPv4) > 0 {
					host := entry.AddrIPv4[0].String()
					port := entry.Port

					mopidyServersMu.Lock()
					found := false
					for _, srv := range mopidyServers {
						if srv.Name == name {
							found = true
							break
						}
					}
					if !found {
						log.Printf("Discovered Mopidy server: %s at %s:%d", name, host, port)
						mopidyServers = append(mopidyServers, &MopidyServer{
							Name: name,
							Host: host,
							Port: port,
						})
					}
					mopidyServersMu.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	err = resolver.Browse(ctx, "_mopidy-http._tcp", "local.", entries)
	if err != nil {
		log.Printf("Failed to browse for Mopidy servers: %v", err)
	}
}

func syncSnapServers(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snapServersMu.RLock()
			count := 0
			for _, server := range snapServers {
				if server.conn != nil {
					count++
					go server.syncStatus()
				}
			}
			snapServersMu.RUnlock()
			log.Printf("Synchronizing %d snapservers", count)
		case <-ctx.Done():
			return
		}
	}
}
