package main

import (
	"context"
	"log"
	"time"

	"github.com/grandcat/zeroconf"
)

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
