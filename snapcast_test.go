package main

import (
	"testing"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{
			name:     "existing string",
			m:        map[string]any{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "missing key",
			m:        map[string]any{},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			m:        map[string]any{"key": 123},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected int
	}{
		{
			name:     "existing int as float64",
			m:        map[string]any{"key": float64(42)},
			key:      "key",
			expected: 42,
		},
		{
			name:     "missing key",
			m:        map[string]any{},
			key:      "key",
			expected: 0,
		},
		{
			name:     "non-numeric value",
			m:        map[string]any{"key": "not a number"},
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected bool
	}{
		{
			name:     "existing true bool",
			m:        map[string]any{"key": true},
			key:      "key",
			expected: true,
		},
		{
			name:     "existing false bool",
			m:        map[string]any{"key": false},
			key:      "key",
			expected: false,
		},
		{
			name:     "missing key",
			m:        map[string]any{},
			key:      "key",
			expected: false,
		},
		{
			name:     "non-bool value",
			m:        map[string]any{"key": "true"},
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBool(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSnapServer(t *testing.T) {
	server := &SnapServer{
		Host:         "localhost",
		Port:         1705,
		Clients:      []Client{},
		Streams:      []Stream{},
		clientGroups: make(map[string]string),
	}

	if server.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", server.Host)
	}

	if server.Port != 1705 {
		t.Errorf("Expected port 1705, got %d", server.Port)
	}
}
