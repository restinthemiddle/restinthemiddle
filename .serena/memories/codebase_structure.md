# Codebase Structure

## Directory Layout

```
restinthemiddle/
├── cmd/restinthemiddle/          # Main application entry point
│   ├── main.go                    # CLI setup, flag parsing, app initialization
│   └── main_test.go               # Application-level tests
├── internal/                      # Private application code
│   ├── version/                   # Version info (injected at build time)
│   │   ├── version.go             # Version strings, build metadata
│   │   └── version_test.go        # Version package tests (100% coverage)
│   └── zapwriter/                 # Zap logger integration for HTTP logging
│       ├── zapwriter.go           # Structured logging with zap
│       └── zapwriter_test.go      # Zapwriter tests (80% coverage)
├── pkg/core/                      # Core proxy logic (public API)
│   ├── core.go                    # Main proxy server orchestration
│   ├── server.go                  # HTTP server interface
│   ├── writer.go                  # HTTP logging writer interface
│   ├── config/                    # Configuration management
│   │   ├── config.go              # Config parsing, validation, translation
│   │   └── config_test.go         # Config tests (100% coverage)
│   ├── proxy/                     # Reverse proxy implementation
│   │   ├── proxy.go               # HTTP reverse proxy with header handling
│   │   └── proxy_test.go          # Proxy tests (94% coverage)
│   └── transport/                 # HTTP transport layer
│       ├── transport.go           # Custom HTTP transport with timing
│       └── transport_test.go      # Transport tests
├── tests/integration/             # End-to-end integration tests
│   ├── integration_test.go        # Full HTTP proxy scenarios
│   └── mock-server.go             # Test HTTP server
├── Makefile                       # Build automation
├── Dockerfile                     # Multi-stage Docker build
├── go.mod / go.sum                # Go module dependencies
├── .golangci.yml                  # Linter configuration
└── .github/workflows/             # CI/CD pipelines
    └── github-pipeline.yml        # Main GitHub Actions workflow
```

## Key Files

### Root Files

- `Makefile`: Build automation with targets for fmt, build, test, lint, integration
- `Dockerfile`: Multi-stage build (golang:1.25.4-alpine → alpine:3.22.2)
- `go.mod`: Go 1.25 module with viper, zap, pflag, uuid dependencies
- `.golangci.yml`: Linter config (godox flags TODO/FIXME/HACK)
- `README.md`: User documentation with examples
- `CHANGELOG.md`: Release history (managed by release-please)
- `release-please-config.json`: Conventional commit configuration

### Configuration Files

- `.github/workflows/github-pipeline.yml`: CI/CD with unit tests, linting, integration tests, Docker builds
- `.golangci.yml`: Enabled linters include errcheck, govet, staticcheck, godox, gosec
- `.gitignore`: Ignores bin/, coverage files, todos.md, notes.md

## Architecture Flow

1. **Entry Point**: `cmd/restinthemiddle/main.go`
   - Sets up CLI flags via spf13/pflag
   - Loads config via viper (defaults → YAML → env → flags)
   - Creates zap logger
   - Calls `core.Run()`

2. **Core Orchestration**: `pkg/core/core.go`
   - `Run()` function initializes proxy server
   - Creates HTTP multiplexer
   - Sets up request handler

3. **Proxy Layer**: `pkg/core/proxy/proxy.go`
   - `NewServer()` creates reverse proxy
   - Custom Director function handles:
     - Request ID injection
     - Header manipulation
     - Basic auth merging
     - Path/query forwarding

4. **Transport Layer**: `pkg/core/transport/transport.go`
   - Custom RoundTripper for timing metrics
   - Captures connection and request timing

5. **Logging**: `internal/zapwriter/zapwriter.go`
   - Implements Writer interface
   - Logs structured JSON with request/response data
   - Respects exclude patterns and body logging flags

## Key Architectural Decisions

- **Standard Library First**: Uses net/http.ReverseProxy, minimal external deps
- **Dependency Injection**: `Run()` accepts interfaces (Writer, HTTPServer)
- **Testing**: Interfaces enable mocking in unit tests
- **Configuration**: Viper provides flexible config with clear precedence
- **Structured Logging**: Zap for performance, JSON for machine parsing
