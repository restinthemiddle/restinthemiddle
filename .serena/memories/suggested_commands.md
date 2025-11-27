# Development Commands

## Essential Commands (Run in Order Before PR)

### 1. Format Code (REQUIRED)

```bash
make fmt
```

- Runs `go fmt ./...`
- Must be run before any commit
- CI will check formatting

### 2. Build Binary

```bash
make build
```

- Output: `bin/restinthemiddle`
- Runs `go mod download` automatically
- CGO_ENABLED=0 for static binary
- Injects version via ldflags
- Duration: ~3-5 seconds

### 3. Run Unit Tests (REQUIRED)

```bash
make test
```

- Runs: `go test -v ./...`
- Tests: `./pkg/...`, `./internal/...`, `./cmd/...`
- Duration: ~3-4 seconds
- Must pass with 0 failures

### 4. Run Linter (REQUIRED)

```bash
make lint
```

- Runs: `golangci-lint run --config .golangci.yml --timeout 5m`
- **Must show "0 issues"**
- Flags TODO/FIXME/HACK comments (godox linter)
- Timeout: 5 minutes

### 5. Run Integration Tests (REQUIRED)

```bash
make test-integration
```

- Runs: `go test -v -tags=integration ./tests/integration/...`
- Duration: ~5 seconds
- Tests end-to-end scenarios with mock HTTP server

## Coverage Commands

```bash
# Show coverage percentages
make test-coverage

# Generate HTML coverage report
make test-coverage-html
# Opens coverage.html
```

## Complete Pre-Commit Workflow

```bash
make fmt && make build && make test && make lint && make test-integration
```

**Expected Results:**

- `make fmt`: May show formatted files, no errors
- `make build`: `bin/restinthemiddle` created
- `make test`: All tests PASS
- `make lint`: "0 issues."
- `make test-integration`: All integration tests PASS

## Docker Commands

```bash
# Build Docker image
make docker
# Tags as jdschulze/restinthemiddle:latest

# Build Docker build environment only
make docker-build-env
```

## Running the Binary

```bash
# Show help
./bin/restinthemiddle --help

# Run with config
./bin/restinthemiddle --target-host-dsn=http://example.com

# Run with config file
./bin/restinthemiddle --config=/path/to/config.yaml
```

## Direct Go Commands (Avoid - Use Makefile)

```bash
# Run tests with race detector (CI does this)
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Run specific package tests
go test -v ./pkg/core/...

# Run specific test
go test -v -run TestFunctionName ./pkg/core/...
```

## System Utilities (macOS/Darwin)

```bash
# Common Darwin commands
ls -la                    # List files with details
find . -name "*.go"       # Find Go files
grep -r "pattern" .       # Search recursively
git status                # Check git status
git log --oneline         # View commit history

# Package management (if needed)
brew install golangci-lint
brew upgrade go
```

## Validation Checklist

Before creating a PR, ensure:

1. ✅ Code is formatted (`make fmt`)
2. ✅ Build succeeds (`make build`)
3. ✅ All unit tests pass (`make test`)
4. ✅ Linter shows 0 issues (`make lint`)
5. ✅ Integration tests pass (`make test-integration`)
6. ✅ Coverage doesn't decrease significantly (`make test-coverage`)
