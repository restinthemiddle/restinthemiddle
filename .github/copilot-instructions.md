# Copilot Coding Agent Instructions for restinthemiddle

## **CRITICAL: Always Use Serena First (#serena MCP server)**

**For ALL analysis, investigation, and code understanding tasks, use Serena semantic tools:**

### **Standard Serena Workflow**
1. **Start with Serena memories**: Use Serena to list memories and read relevant ones for context #serena
   - Available memories: `project_overview`, `codebase_structure`, `suggested_commands`, `code_style_conventions`, `task_completion_checklist`, `ci_cd_pipeline`, `constraints_gotchas`
2. **Use semantic analysis**: Use Serena to find [symbols/functions/patterns] related to [issue] #serena
3. **Get symbol-level insights**: Use Serena to analyze [specific function] and show all referencing symbols #serena
4. **Create new memories**: Use Serena to write a memory about [findings] for future reference #serena

### **Serena-First Examples**

* Instead of: "Search the codebase for database queries"
* Use: "Use Serena to find all database query functions and analyze their performance patterns #serena"

* Instead of: "Find all admin functions"
* Use: "Use Serena to get symbols overview of admin files and find capability-checking functions #serena"

* Instead of: "What are the build commands?"
* Use: "Use Serena to read the suggested_commands memory for complete build workflow #serena"

* Instead of: "What are the coding standards?"
* Use: "Use Serena to read the code_style_conventions memory #serena"

## Project Overview

**restinthemiddle** is a lightweight HTTP logging proxy server written in Go, designed for development and staging environments. It acts as a transparent middleware between HTTP clients and servers, logging all requests and responses for debugging and monitoring purposes.

- **Language**: Go 1.25
- **Size**: Small (~7,500 lines of code)
- **Type**: CLI application with Docker support
- **Key Dependencies**: spf13/viper (config), go.uber.org/zap (logging), Google UUID
- **Main Entry Point**: `cmd/restinthemiddle/main.go`

## Build & Test Commands

### Prerequisites
- Go 1.25.4+ must be installed
- For linting: `golangci-lint` must be available in PATH

### Essential Commands (Always Run in Order)

**ALWAYS run these commands before creating a pull request:**

1. **Format Code** (REQUIRED before any commit):
   ```bash
   make fmt
   ```

2. **Build** (Validates compilation):
   ```bash
   make build
   ```
   - Output: `bin/restinthemiddle` binary
   - Build automatically runs `go mod download`
   - Duration: ~3-5 seconds

3. **Run Unit Tests** (REQUIRED):
   ```bash
   make test
   ```
   - Tests all packages under `./pkg/...`, `./internal/...`, `./cmd/...`
   - Duration: ~3-4 seconds
   - Must pass with 0 failures

4. **Run Linting** (REQUIRED):
   ```bash
   make lint
   ```
   - Uses golangci-lint with config from `.golangci.yml`
   - Timeout: 5 minutes
   - Must show "0 issues"

5. **Run Integration Tests** (REQUIRED):
   ```bash
   make test-integration
   ```
   - Tests end-to-end functionality with real HTTP server
   - Duration: ~5 seconds
   - Tagged with `-tags=integration`

### Coverage Commands (Optional but Recommended)

```bash
make test-coverage          # Shows coverage percentages
make test-coverage-html     # Generates coverage.html report
```

### Complete Pre-Commit Sequence

```bash
make fmt && make build && make test && make lint && make test-integration
```

**Expected Result**: All commands must succeed with no errors. Linter must report "0 issues".

## GitHub CI/CD Pipeline

The `.github/workflows/github-pipeline.yml` defines the full CI pipeline. **Your changes must pass all these checks:**

### Pipeline Jobs (in order):

1. **unit-tests**
   - Runs: `go test -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/... ./internal/... ./cmd/...`
   - Uploads coverage to Codecov
   - **CRITICAL**: Tests run with `-race` flag to detect race conditions

2. **linting**
   - Runs: `golangci-lint run --config .golangci.yml`
   - Uses `golangci/golangci-lint-action@v9` with latest version

3. **integration-tests** (depends on unit-tests + linting)
   - Builds binary first: `make build`
   - Runs: `go test -v -tags=integration -race ./tests/integration/...`

4. **docker-build** (only on main branch/tags, not PRs)
   - Multi-platform: linux/amd64, linux/arm64
   - Pushes to Docker Hub as `jdschulze/restinthemiddle`

5. **release-please** (only on main branch)
   - Creates release PRs and GitHub releases
   - Uses conventional commits for versioning

### Important CI Notes

- **Race Detector**: All tests run with `-race` flag in CI. If you see race condition failures in CI but not locally, run tests with `go test -race ./...`
- **Coverage Target**: Current coverage is ~87-94% across packages. Don't significantly decrease coverage.
- **PR Checks**: Only unit-tests, linting, and integration-tests run on PRs. Docker builds run only on main/tags.

## Project Structure

```
restinthemiddle/
├── cmd/restinthemiddle/          # Main application entry point
│   ├── main.go                    # CLI setup, flag parsing, app initialization
│   └── main_test.go               # App-level tests
├── internal/                      # Private application code
│   ├── version/                   # Version info (injected at build time)
│   │   ├── version.go             # Version strings, build metadata
│   │   └── version_test.go
│   └── zapwriter/                 # Zap logger integration for HTTP logging
│       ├── zapwriter.go           # Structured logging with zap
│       └── zapwriter_test.go
├── pkg/core/                      # Core proxy logic (public API)
│   ├── core.go                    # Main proxy server orchestration
│   ├── server.go                  # HTTP server interface
│   ├── writer.go                  # HTTP logging writer interface
│   ├── config/                    # Configuration management
│   │   ├── config.go              # Config parsing, validation, translation
│   │   └── config_test.go
│   ├── proxy/                     # Reverse proxy implementation
│   │   ├── proxy.go               # HTTP reverse proxy with header handling
│   │   └── proxy_test.go
│   └── transport/                 # HTTP transport layer
│       ├── transport.go           # Custom HTTP transport with timing
│       └── transport_test.go
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

### Key Architecture Points

- **Entry Flow**: `main.go` → `core.Run()` → `proxy.NewServer()` → HTTP handler
- **Configuration**: Viper-based with precedence: defaults < YAML < env vars < CLI flags
- **Logging**: Structured JSON logging via zap, configured in `internal/zapwriter`
- **Proxy Logic**: Standard library `httputil.ReverseProxy` with custom Director in `pkg/core/proxy`
- **Testing Strategy**: Unit tests for all packages + integration tests with real HTTP server

## Configuration Files

### `.golangci.yml` - Linter Configuration
- **Enabled Linters**: errcheck, gocognit, goconst, gocritic, godot, godox, gosec, govet, ineffassign, staticcheck, unconvert, unparam, unused, wastedassign, whitespace
- **godox Check**: Flags TODO, FIXME, HACK, BUG comments (these are NOT allowed in code)
- **Timeout**: 5 minutes

### `Dockerfile` - Multi-Stage Build
- **Base Image**: `golang:1.25.4-alpine` (build-env stage)
- **Final Image**: `alpine:3.22.2` (artifact stage)
- **Build Args**: VERSION, BUILD_DATE, GIT_COMMIT (injected via ldflags)
- **Entry Point**: `dumb-init` wrapper for proper signal handling
- **CGO**: Disabled (`CGO_ENABLED=0`)

### `Makefile` - Build Variables
- **VERSION**: Derived from `git describe --tags --always --dirty`
- **BUILD_DATE**: ISO 8601 format (`date -u +"%Y-%m-%dT%H:%M:%SZ"`)
- **GIT_COMMIT**: Current commit SHA from `git rev-parse HEAD`
- **LDFLAGS**: Inject version info into `internal/version` package variables

## Common Development Workflows

### Making Code Changes

1. **Always format before committing**: `make fmt`
2. **Build to check compilation**: `make build`
3. **Run tests**: `make test`
4. **Check lint**: `make lint`
5. **Run integration tests**: `make test-integration`

### Adding New Features

1. Identify the appropriate package (`pkg/core`, `internal/`, etc.)
2. Write tests FIRST (TDD approach preferred)
3. Implement feature
4. Run full test suite: `make test && make test-integration`
5. Check coverage: `make test-coverage` (aim for >85% coverage)
6. Lint: `make lint` (must show 0 issues)
7. Format: `make fmt`

### Debugging Failed Tests

- **Unit Test Failures**: Run specific package: `go test -v ./pkg/core/...`
- **Race Conditions**: Run with race detector: `go test -race ./...`
- **Integration Failures**: Check `tests/integration/integration_test.go` and `mock-server.go`
- **Coverage Issues**: Generate HTML report: `make test-coverage-html` and open `coverage.html`

### Building Docker Image

```bash
make docker
```
- Tags as `jdschulze/restinthemiddle:latest`
- Uses buildx for multi-platform support (configured in CI)

## Important Constraints & Gotchas

### Code Quality Requirements

1. **NO TODO/FIXME/HACK Comments**: The godox linter flags these. If you must add a temporary note, use a GitHub issue instead.
2. **Test Coverage**: Don't reduce coverage below current levels (~87-94%). Add tests for new code.
3. **Race Detector**: All CI tests run with `-race`. Test locally with `go test -race ./...` before pushing.
4. **Go Formatting**: Always run `make fmt` before committing. CI checks this.

### Build & Dependency Notes

1. **Go Modules**: Run `go mod download` before building (Makefile does this automatically)
2. **Binary Output**: Build outputs to `bin/restinthemiddle` (ignored by .gitignore)
3. **CGO**: Always disabled (`CGO_ENABLED=0`) for static binaries
4. **Version Injection**: Version info is injected via ldflags at build time (see Makefile EXTRA_LDFLAGS)

### Testing Notes

1. **Integration Tests**: Require the binary to be built first (`make build`)
2. **Build Tags**: Integration tests use `-tags=integration` build tag
3. **Mock Server**: Integration tests start a mock HTTP server on a random port
4. **Timing Sensitivity**: Some tests involve HTTP timing; avoid hardcoded timeouts

### Configuration System

1. **Precedence Order**: App defaults → `/etc/restinthemiddle/config.yaml` → `~/.restinthemiddle/config.yaml` → `./config.yaml` → Environment Variables → CLI Flags (last wins)
2. **Required Field**: `targetHostDsn` is REQUIRED (will fail without it)
3. **Headers**: Can only be set via config file or CLI flags (NOT environment variables)
4. **Timeouts**: Default to 0 (no timeout); production should set `readTimeout`, `writeTimeout`, `idleTimeout`, `readHeaderTimeout`

## Validation Checklist Before PR

Run this exact sequence:

```bash
# 1. Format code (REQUIRED)
make fmt

# 2. Build (catches compilation errors)
make build

# 3. Unit tests (REQUIRED)
make test

# 4. Linting (REQUIRED - must show "0 issues")
make lint

# 5. Integration tests (REQUIRED)
make test-integration

# 6. Optional: Coverage check
make test-coverage
```

**Expected Output**:
- `make fmt`: May show formatted files, no errors
- `make build`: `bin/restinthemiddle` created successfully
- `make test`: All tests PASS, no FAIL
- `make lint`: "0 issues."
- `make test-integration`: All integration tests PASS

**If any command fails**: Fix the issue before creating a PR. The CI will run the same checks.

## Trust These Instructions

These instructions are comprehensive and tested. Follow them exactly:
- Use the Makefile targets (`make test`, `make lint`, etc.) rather than custom go commands
- Run commands in the documented order
- Check for "0 issues" in lint output before submitting PRs
- Only search for additional information if these instructions are incomplete or incorrect

## Quick Reference

| Task | Command |
|------|---------|
| Format code | `make fmt` |
| Build binary | `make build` |
| Run unit tests | `make test` |
| Run tests with coverage | `make test-coverage` |
| Generate coverage HTML | `make test-coverage-html` |
| Run linter | `make lint` |
| Run integration tests | `make test-integration` |
| Build Docker image | `make docker` |
| Run binary | `./bin/restinthemiddle --help` |

**Go Version**: 1.25.4
**Lint Timeout**: 5 minutes
**Test Duration**: ~3-4 seconds (unit), ~5 seconds (integration)
**Current Coverage**: 87-94% across packages
