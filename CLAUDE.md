# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This is a simple Echo API server written in Go that serves as a debugging and investigation tool. The application dumps HTTP request information as JSON responses, making it useful for testing and debugging HTTP clients and network configurations.

### Core Components

- **`server.go`**: Main server entry point with CLI argument parsing and Echo server setup. Handles port configuration via flags (`-p`, `--port`) or `PORT` environment variable. Sets up Logger and Secure middlewares.
- **`echo.go`**: Request handling logic with the `dumpRequest` function that extracts and returns request metadata as JSON. Includes `DumpResult` struct and `HttpHeader` helper type.
- **`version.go`**: Version information display functionality with build-time variables injected via LDFLAGS during GoReleaser builds.

The server accepts any GET request to any path (`/*` route) and returns detailed information about the request including headers, timestamp, hostname, and remote address.

## Development Commands

Use Task runner for development workflow:

```bash
# Setup development environment
task setup

# Run locally (default port 8080)
task run

# Lint code
task lint

# Format code
task format

# Run tests
task test

# Build release artifacts
task build
```

### Alternative Go Commands

```bash
# Build for development (creates binary in current directory)
go build -o echo-api .

# Run locally with default port (8080)
go run . 

# Run with custom port
go run . -p 8081

# Show version
go run . -version

# Run tests
go test ./...
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
# Run in Docker
docker run --rm -p 8081:8081 littlef/echo-api:0.0.1 -p 8081
```

## Configuration

- **Port**: Set via `-p` flag, `--port` flag, or `PORT` environment variable (default: 8080)
- **Application Name**: Set `X_APP_NAME` environment variable to add custom header to responses
- **GoReleaser**: Uses `.goreleaser.yml` for multi-platform builds and Docker image creation

## Tools and Dependencies

- **Echo v4**: Web framework for HTTP server
- **Task**: Task runner for development workflow (see `Taskfile.yaml`)
- **Aqua**: Tool version management - includes `golangci-lint` and `github-comment`
- **GoReleaser**: Multi-platform builds and Docker image creation
- **Renovate**: Automated dependency updates

## Build Process

Version information is injected at build time via LDFLAGS in GoReleaser configuration:
- `version`, `revision`, `date`, `osArch` variables in `version.go`
- Cross-platform builds for linux/darwin/windows on amd64/arm64
- Docker multi-arch manifests for linux/amd64 and linux/arm64

The codebase is intentionally minimal with no test files - this is a utility application focused on request debugging rather than complex business logic.