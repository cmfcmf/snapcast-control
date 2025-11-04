# Testing Guide

This document describes how to run tests for snapcast-control.

## Unit Tests

Run unit tests with:

```bash
go test -v ./...
```

Run with coverage:

```bash
go test -v -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Linting

Run the linter:

```bash
golangci-lint run --timeout 5m
```

Or install and run:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run --timeout 5m
```

## Integration Tests

Integration tests require a running Snapcast server instance.

### Using Docker Compose

Start the test environment:

```bash
docker-compose -f docker-compose.test.yml up -d
```

Run integration tests:

```bash
SNAPCAST_HOST=localhost SNAPCAST_PORT=1705 go test -tags=integration -v ./...
```

Stop the test environment:

```bash
docker-compose -f docker-compose.test.yml down
```

### Using a Real Snapcast Server

If you have a Snapcast server running:

```bash
SNAPCAST_HOST=your-server-ip SNAPCAST_PORT=1705 go test -tags=integration -v ./...
```

### Manual Integration Testing

1. Start the application:
   ```bash
   go run . --debug --port 8080
   ```

2. In another terminal, test the endpoints:
   ```bash
   # Test snap servers
   curl http://localhost:8080/snap_servers.json
   
   # Test mopidy servers
   curl http://localhost:8080/mopidy_servers.json
   
   # Test frontend
   curl http://localhost:8080/
   ```

## CI/CD

Tests run automatically on GitHub Actions for:
- Every push to main and copilot/* branches
- Every pull request

The workflow:
1. Runs unit tests
2. Runs the linter
3. Builds the binary

See `.github/workflows/go.yml` for details.

## Cross-Platform Build Testing

Test building for different platforms:

```bash
# Linux AMD64 (default)
go build -o snapcast-control-amd64

# Linux ARM32 (armhf)
GOOS=linux GOARCH=arm GOARM=7 go build -o snapcast-control-armhf

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o snapcast-control-arm64

# Or use the script
./build-armhf.sh
```

## Test Coverage Goals

- Unit tests should cover:
  - All HTTP handlers
  - JSON serialization/deserialization
  - Error handling paths
  - Helper functions

- Integration tests should cover:
  - Snapcast server connection
  - Snapcast protocol communication
  - Mopidy RPC communication
  - End-to-end API workflows
