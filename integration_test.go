// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// These tests require a running snapcast server
// Run with: go test -tags=integration -v

func TestIntegrationSnapcastConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if SNAPCAST_HOST is set
	host := os.Getenv("SNAPCAST_HOST")
	if host == "" {
		t.Skip("SNAPCAST_HOST environment variable not set, skipping integration test")
	}

	port := 1705
	if portStr := os.Getenv("SNAPCAST_PORT"); portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test server
	server := &SnapServer{
		Host:         host,
		Port:         port,
		Clients:      []Client{},
		Streams:      []Stream{},
		clientGroups: make(map[string]string),
	}

	// Try to connect
	go server.connect(ctx)

	// Wait for connection
	time.Sleep(2 * time.Second)

	if server.conn == nil {
		t.Fatalf("Failed to connect to Snapcast server at %s:%d", host, port)
	}

	// Try to sync status
	server.syncStatus()

	t.Logf("Connected to Snapcast server")
	t.Logf("Found %d clients and %d streams", len(server.Clients), len(server.Streams))

	if len(server.Streams) > 0 {
		t.Logf("First stream: %+v", server.Streams[0])
	}
}

func TestIntegrationAPIEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start the server in a separate goroutine
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		// Start our own server for testing
		serverURL = "http://localhost:18080"

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a minimal server setup
		go func() {
			// Set up minimal test environment
			cmd := exec.CommandContext(ctx, "./snapcast-control", "--port", "18080")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil && ctx.Err() == nil {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(2 * time.Second)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name     string
		endpoint string
		wantCode int
	}{
		{
			name:     "snap servers endpoint",
			endpoint: "/snap_servers.json",
			wantCode: http.StatusOK,
		},
		{
			name:     "mopidy servers endpoint",
			endpoint: "/mopidy_servers.json",
			wantCode: http.StatusOK,
		},
		{
			name:     "root endpoint",
			endpoint: "/",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(serverURL + tt.endpoint)
			if err != nil {
				t.Skipf("Server not available: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, resp.StatusCode)
			}

			// For JSON endpoints, verify we can parse the response
			if tt.endpoint == "/snap_servers.json" || tt.endpoint == "/mopidy_servers.json" {
				var data any
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}
				t.Logf("Response: %+v", data)
			}
		})
	}
}

func TestIntegrationDockerSnapcast(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if docker is available
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker not available, skipping integration test")
	}

	// This is a placeholder for full Docker-based integration tests
	// In a real scenario, we would:
	// 1. Start a snapcast server container
	// 2. Start a snapcast client container
	// 3. Start our application
	// 4. Test the full integration

	t.Log("Docker integration test placeholder - implement full Docker-based tests as needed")
}
