package main

// Client represents a Snapcast client
type Client struct {
	ID        string `json:"id"`
	Muted     bool   `json:"muted"`
	Volume    int    `json:"volume"`
	Name      string `json:"name"`
	Latency   int    `json:"latency"`
	Connected bool   `json:"connected"`
	Stream    string `json:"stream"`
}

// Stream represents a Snapcast audio stream
type Stream struct {
	ID     string         `json:"id"`
	Status string         `json:"status"`
	Meta   map[string]any `json:"meta"`
}

// SnapServer represents a Snapcast server instance
type SnapServer struct {
	Host         string
	Port         int
	Clients      []Client `json:"clients"`
	Streams      []Stream `json:"streams"`
	conn         *SnapcastConnection
	clientGroups map[string]string // maps client ID to group ID
}

// MopidyServer represents a Mopidy music server
type MopidyServer struct {
	Name string `json:"name"`
	Host string `json:"-"`
	Port int    `json:"-"`
}

// BibItem represents a browsable item from Mopidy
type BibItem struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
	Type string `json:"type"`
}
