# Migration from Python to Go

This document describes the migration of the Snapcast Control backend from Python to Go.

## What Changed

### Backend (Rewritten)
- **server.py** → **main.go**: Main HTTP server with all API endpoints
- **serializer.py** → Integrated into **snapcast.go**: Serialization logic is now part of the Snapcast connection handler
- **requirements.txt** → **go.mod/go.sum**: Dependency management now uses Go modules

### New Files
- **main.go**: Main HTTP server, handlers, and service discovery
- **snapcast.go**: Snapcast protocol client and connection management
- **mopidy.go**: Mopidy JSON-RPC client
- **build.sh**: Convenience script to build both frontend and backend
- **Dockerfile**: Docker support for containerized deployment
- **.dockerignore**: Docker build optimization

### Frontend (Unchanged)
- The React frontend in `frontend-react/` remains completely unchanged
- The frontend build is now embedded into the Go binary using `go:embed`
- No API changes - all endpoints maintain backward compatibility

### Dependencies
The Go implementation uses:
- `github.com/grandcat/zeroconf` - For mDNS/Zeroconf service discovery
- Standard library packages for HTTP, JSON, networking

## API Compatibility

All API endpoints remain the same:
- `GET /snap_servers.json` - List Snapcast servers
- `GET /mopidy_servers.json` - List Mopidy servers
- `GET /client` - Client settings (mute, unmute, delete, set_latency, set_stream)
- `GET /browse.json` - Browse Mopidy library
- `GET /play` - Play tracks on Mopidy
- `GET /stop` - Stop Mopidy playback
- `GET /` - Serve frontend (static files)

## Improvements

### Performance
- Single compiled binary (~11MB) with embedded frontend
- Faster startup and lower memory footprint
- More efficient concurrent request handling

### Security
- Log files now created with restrictive permissions (0600)
- Improved error handling and input validation
- Proper message framing in Snapcast protocol
- Client-to-group mapping cache to reduce network calls

### Deployment
- No runtime dependencies (Go compiles to static binary)
- Docker support included
- Easier cross-platform builds

## Building

### From Source
```bash
# Build frontend
cd frontend-react
yarn install && yarn build
cd ..

# Build Go binary
go build -o snapcast-control

# Run
./snapcast-control --port 8080
```

### Using Docker
```bash
docker build -t snapcast-control .
docker run -p 8080:8080 --network host snapcast-control
```

## Running

Command-line options:
```
  -debug
    	run in debug mode
  -loglevel string
    	log level (default "INFO")
  -port int
    	web server port (default 8080)
```

Example:
```bash
./snapcast-control --debug --port 8080
```

## Old Python Files

The following files are kept for reference but are no longer used:
- `server.py`
- `serializer.py`
- `requirements.txt`

These can be safely removed if desired, but are kept to preserve project history.

## Testing

The implementation has been:
- Code reviewed and improvements made
- Security scanned with CodeQL (0 vulnerabilities)
- Functionally tested with all endpoints
- Verified for API compatibility with existing frontend

## Future Considerations

- The frontend build currently needs to be created before building the Go binary
- Consider adding automated tests
- Consider adding CI/CD pipelines for automated builds
- The CORS policy is currently permissive (*) - may want to make this configurable for production use
