# Snapcast Control Web Interface

![Overview](docs/overview.png)

A web interface for [Snapcast](https://github.com/badaix/snapcast).
It allows you select which stream is played on which client.
It also has support to select local radio and files from Mopidy instances.
Currently, the UI is in German. If you are interested in using an English version, please open an issue.

**Note:** The backend has been rewritten in Go while the React frontend remains unchanged. The frontend is embedded into the Go binary using go:embed.

## Installation

```bash
# Install Go (if not already installed)
# Download from https://golang.org/dl/ or use your package manager

# Build the frontend
cd frontend-react
pnpm install && pnpm run build
cd ..

# Build the Go server
go build -o snapcast-control

# Run the server
./snapcast-control --port 8080
```

Add an entry to crontab to start snapcast-control after booting:

```bash
sudo crontab -e # sudo is only needed for ports < 1000
```

```crontab
@reboot sleep 10 && /absolute/path/to/snapcast-control/snapcast-control --port 80
```

## Development

Server:

```bash
# Build and run
go build -o snapcast-control && ./snapcast-control --debug --port 8080

# Or run directly
go run . --debug --port 8080
```

Client:

```
cd frontend-react
pnpm install && pnpm run start
```

## Docker

You can also build and run using Docker:

```bash
docker build -t snapcast-control .
docker run -p 8080:8080 --network host snapcast-control
```

Note: `--network host` is required for Zeroconf/mDNS service discovery to work properly.

## Command-line Options

```
  -debug
    	run in debug mode
  -loglevel string
    	log level (default "INFO")
  -port int
    	web server port (default 8080)
```

## SnapCast Update

```
cd ansible
ansible-playbook playbook.yml
```