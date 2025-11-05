package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected string
	}{
		{
			name:     "empty object",
			data:     map[string]any{},
			expected: "{}\n",
		},
		{
			name:     "simple map",
			data:     map[string]string{"key": "value"},
			expected: `{"key":"value"}` + "\n",
		},
		{
			name:     "empty array",
			data:     []MopidyServer{},
			expected: "[]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.data)

			if w.Body.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, w.Body.String())
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestSnapServersHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/snap_servers.json", nil)
	w := httptest.NewRecorder()

	snapServersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	// Should return an empty object when no servers are discovered
	if w.Body.String() != "{}\n" {
		t.Errorf("Expected empty object, got %s", w.Body.String())
	}
}

func TestMopidyServersHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/mopidy_servers.json", nil)
	w := httptest.NewRecorder()

	mopidyServersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	// Should return an empty array when no servers are discovered
	expected := "[]\n"
	if w.Body.String() != expected {
		t.Errorf("Expected %q, got %q", expected, w.Body.String())
	}
}



func TestClientSettingsHandlerMissingParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{
			name:       "missing all params",
			query:      "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing client id",
			query:      "server_name=test&action=mute",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid server",
			query:      "server_name=nonexistent&id=test&action=mute",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/client?"+tt.query, nil)
			w := httptest.NewRecorder()

			clientSettingsHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestBrowseHandlerMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/browse.json", nil)
	w := httptest.NewRecorder()

	browseHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStopHandlerMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stop", nil)
	w := httptest.NewRecorder()

	stopHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
