package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

//go:embed frontend-react/build
var frontendFS embed.FS

var (
	snapServers   = make(map[string]*SnapServer)
	snapServersMu sync.RWMutex

	mopidyServers   []*MopidyServer
	mopidyServersMu sync.RWMutex
)

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
		Handler: mux,
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
