# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This is a simple Echo API server written in Go that serves as a debugging and investigation tool. The application dumps HTTP request information as JSON responses, making it useful for testing and debugging HTTP clients and network configurations.

### Core Components

- **`server.go`**: Main server entry point with CLI argument parsing and Echo server setup
- **`echo.go`**: Request handling logic with the `dumpRequest` function that extracts and returns request metadata
- **`version.go`**: Version information display functionality (version info injected via LDFLAGS during build)

The server accepts any GET request to any path (`/*` route) and returns detailed information about the request including headers, timestamp, hostname, and remote address.

## Development Commands

### Building and Running

```bash
# Build for development (creates binary in current directory)
go build -o echo-api .

# Run locally with default port (8080)
go run . 

# Run with custom port
go run . -p 8081

# Show version
go run . -version
```

### Release Building

```bash
# Build release artifacts using GoReleaser
goreleaser release --clean --snapshot

# This creates:
# - Cross-platform binaries in dist/
# - Docker images for linux/amd64 and linux/arm64
```

### Docker

```bash
# Build Docker image (handled by GoReleaser)
docker build -t littlef/echo-api .

# Run in Docker
docker run --rm -p 8081:8081 littlef/echo-api:latest -p 8081
```

## Configuration

- **Port**: Set via `-p` flag, `--port` flag, or `PORT` environment variable (default: 8080)
- **Application Name**: Set `X_APP_NAME` environment variable to add custom header to responses
- **GoReleaser**: Uses `.goreleaser.yml` for multi-platform builds and Docker image creation

## Dependencies

- **Echo v4**: Web framework for HTTP server
- **Aqua**: Tool version management (see `aqua.yaml`)
- **Renovate**: Automated dependency updates (see `renovate.json`)

## Project Structure

```
├── server.go          # Main server and CLI setup
├── echo.go           # HTTP request handler
├── version.go        # Version display
├── Dockerfile        # Distroless container image
├── .goreleaser.yml   # Release build configuration
└── aqua.yaml        # Tool version management
```

The codebase is intentionally minimal with no test files - this is a utility application focused on request debugging rather than complex business logic.