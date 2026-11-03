# echo-api

English | [日本語](README_ja.md)

[![CI](https://github.com/little-forest/echo-api/actions/workflows/ci.yaml/badge.svg)](https://github.com/little-forest/echo-api/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/v/release/little-forest/echo-api)](https://github.com/little-forest/echo-api/releases)
[![Docker Image](https://img.shields.io/docker/v/littlef/echo-api?label=docker&sort=semver)](https://hub.docker.com/r/littlef/echo-api)
[![Go Version](https://img.shields.io/github/go-mod/go-version/little-forest/echo-api)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A tiny HTTP server that echoes back what it received as JSON — the path, headers, remote address and more.
Use it to see exactly what reaches your backend through proxies, load balancers, ingress controllers and HTTP clients.

```console
$ curl -s 'localhost:8080/hello?name=world' | jq
{
  "path": "/hello?name=world",
  "time": "2026-11-03T11:58:22.994029+09:00",
  "remoteAddr": "[::1]:60811",
  "hostname": "my-host",
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl/8.7.1"
  }
}
```

## Features

- **Answers any path**: every `GET` request, regardless of path, returns a JSON dump of the request.
- **Sees the raw request**: paths are never cleaned or redirected, so `/foo//bar` or `/a/../b` is reported as sent.
- **No dependencies**: built only on the Go standard library (`net/http`, `log/slog`, `encoding/json`).
- **Small container image**: multi-arch (`linux/amd64`, `linux/arm64`) images based on distroless.
- **Access log**: one structured log line per request via `log/slog`.
- **Graceful shutdown** on `SIGINT` / `SIGTERM`.

## Quick start

```console
$ docker run --rm -p 8080:8080 littlef/echo-api:1.0.0
$ curl localhost:8080/
```

## Installation

### Docker

Images are published on [Docker Hub](https://hub.docker.com/r/littlef/echo-api) for `linux/amd64` and `linux/arm64`.
Tags are the release version without the leading `v` (e.g. `1.0.0`).

```console
$ docker pull littlef/echo-api:1.0.0
```

### Prebuilt binaries

Download an archive for Linux, macOS or Windows (amd64 / arm64) from the [Releases](https://github.com/little-forest/echo-api/releases) page.

### go install

```console
$ go install github.com/little-forest/echo-api@latest
```

## Usage

```console
$ echo-api [-p PORT] [-v]
```

| Option | Description | Default |
| --- | --- | --- |
| `-p`, `--port` | Port number to listen on | `8080` |
| `-v`, `--version` | Show version information and exit | |

| Environment variable | Description |
| --- | --- |
| `PORT` | Port number to listen on. **When set, it takes precedence over `-p` / `--port`.** |
| `X_APP_NAME` | When set, an `X-Application-Name` entry with this value is added to the `headers` field of the JSON response. It is not sent as an HTTP response header. Handy for telling multiple deployments apart. |

Examples:

```console
# Listen on port 8081
$ echo-api -p 8081

# Same thing with Docker
$ docker run --rm -p 8081:8081 littlef/echo-api:1.0.0 -p 8081

# Identify the instance in responses
$ docker run --rm -p 8080:8080 -e X_APP_NAME=frontend-a littlef/echo-api:1.0.0
```

## Response format

A `GET` request returns `200 OK` with `Content-Type: application/json`:

| Field | Description |
| --- | --- |
| `path` | The request URI as received, including the query string |
| `time` | Time the request was handled, in RFC 3339 with nanoseconds |
| `remoteAddr` | Address of the connecting client (`host:port`). Behind a proxy, this is the proxy's address. |
| `hostname` | Host name of the machine (or container) running echo-api |
| `headers` | Request headers as a name-to-value object |

Notes on `headers`:

- Header names are in Go's canonical form (e.g. `x-foo` becomes `X-Foo`).
- If a header appears multiple times, only the last value is shown.
- The `Host` header is not included, because Go's `net/http` moves it out of the header map.

## HTTP behavior

| Method | Status | Response |
| --- | --- | --- |
| `GET` | `200 OK` | JSON dump of the request (see above) |
| `OPTIONS` | `204 No Content` | `Allow: OPTIONS, GET`, no body |
| `HEAD` | `405 Method Not Allowed` | `Allow: OPTIONS, GET`, no body |
| Others (`POST`, `PUT`, ...) | `405 Method Not Allowed` | `Allow: OPTIONS, GET`, body `{"message":"Method Not Allowed"}` |

Every response also carries `X-Content-Type-Options: nosniff` and `X-Frame-Options: SAMEORIGIN`.

## Logging

Logs are written to standard error with `log/slog`. Each request produces one line:

```text
2026/11/03 11:58:22 INFO http server started address=[::]:8080
2026/11/03 11:58:22 INFO request remoteIP=::1 method=GET status=200 uri="/foo//bar?x=1" latency=28.459µs
```

## Use cases

- Check which headers (`X-Forwarded-For`, `X-Forwarded-Proto`, ...) your reverse proxy, load balancer or ingress adds or rewrites.
- Verify the exact path and query string that reach the backend after routing and rewrite rules.
- Debug HTTP clients by seeing what they actually send.
- Use it as a placeholder backend for connectivity tests on Kubernetes or other platforms, with `X_APP_NAME` and `hostname` showing which instance answered.

## Development

Requirements:

- [Go](https://go.dev/) (version in [`go.mod`](go.mod))
- [Task](https://taskfile.dev/)
- [aqua](https://aquaproj.github.io/) (installs `golangci-lint` and other tools)

```console
$ task setup    # Install tools with aqua
$ task run      # Run locally on port 8080
$ task test     # Run tests
$ task lint     # Run golangci-lint
$ task format   # Format code
$ task build    # Build release artifacts locally with GoReleaser (snapshot)
```

## Release

Pushing a tag like `v1.2.3` triggers GitHub Actions, which uses [GoReleaser](https://goreleaser.com/) to publish binaries to GitHub Releases and Docker images to Docker Hub.

## Contributing

Issues and pull requests are welcome. Before opening a pull request, please make sure `task lint` and `task test` pass.

echo-api exists to observe requests, so its responses are its public interface.
The HTTP contract (JSON shape, status codes, headers) is pinned by tests in [`server_test.go`](server_test.go), and any change to it is treated as a breaking change.

## License

[MIT](LICENSE) © KOMORI Yusuke
