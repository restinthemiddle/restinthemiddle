# Important Constraints and Gotchas

## Critical Rules

### 1. NO TODO/FIXME/HACK Comments

**Rule**: The godox linter flags TODO, FIXME, HACK, and BUG comments
**Why**: These comments are not allowed in the codebase
**Solution**: Use GitHub issues to track work items instead
**Example**:

```go
// ❌ BAD
// TODO: Implement retry logic

// ✅ GOOD
// See issue #123 for retry logic implementation
```

### 2. Race Detector is MANDATORY

**Rule**: All CI tests run with `-race` flag
**Why**: Detect race conditions in concurrent code
**Solution**: Always test with race detector before pushing:

```bash
go test -race ./...
```

**Note**: Tests may pass locally but fail in CI if not tested with `-race`

### 3. Code Formatting is REQUIRED

**Rule**: All code must be formatted with `go fmt`
**Why**: CI checks formatting, unformatted code fails
**Solution**: Always run `make fmt` before committing
**Tip**: Configure your editor to run `go fmt` on save

### 4. Coverage Must Not Decrease

**Rule**: Maintain 85%+ coverage (currently 87-94%)
**Why**: Coverage regression indicates untested code
**Solution**: Add tests for new features
**Check**: Run `make test-coverage-html` to identify gaps

## Build Constraints

### 1. CGO Must Be Disabled

**Rule**: Always build with `CGO_ENABLED=0`
**Why**: Creates static binaries for Docker
**Implementation**: Makefile sets this automatically
**Don't**: Manually run `go build` without setting CGO_ENABLED=0

### 2. Version Injection via ldflags

**Rule**: Version info injected at build time
**Why**: Enables `--version` flag functionality
**Location**: `internal/version/version.go`
**Build variables**:

- VERSION: From `git describe --tags --always --dirty`
- BUILD_DATE: ISO 8601 timestamp
- GIT_COMMIT: Current commit SHA

### 3. Binary Output Location

**Rule**: Binary built to `bin/restinthemiddle`
**Why**: Ignored by `.gitignore`, keeps repo clean
**Don't**: Commit binaries to git

## Testing Constraints

### 1. Integration Tests Require Build

**Rule**: Run `make build` before `make test-integration`
**Why**: Integration tests start the actual binary
**Solution**: Use `make test-integration` (handles build automatically)

### 2. Build Tags for Integration Tests

**Rule**: Integration tests use `-tags=integration`
**Why**: Separates unit tests from integration tests
**Implementation**: Tests have `//go:build integration` at top of file

### 3. Mock Server Uses Random Port

**Rule**: Integration tests bind to random port
**Why**: Avoids port conflicts in CI
**Don't**: Hardcode port numbers in integration tests

### 4. Timing-Sensitive Tests

**Rule**: Avoid hardcoded timeouts
**Why**: CI environment may be slower
**Solution**: Use generous timeouts or retry logic

## Configuration Constraints

### 1. targetHostDsn is REQUIRED

**Rule**: Must provide targetHostDsn or app fails
**Why**: Core functionality requires target host
**Error**: App will `log.Fatalf()` without it
**Solution**: Always set via CLI, env var, or config file

### 2. Headers Only via Config/CLI

**Rule**: Headers cannot be set via environment variables
**Why**: Viper limitation with nested YAML
**Solution**: Use `--header` flag or config.yaml headers section

### 3. Configuration Precedence

**Order** (last wins):

1. App defaults
2. `/etc/restinthemiddle/config.yaml`
3. `~/.restinthemiddle/config.yaml`
4. `./config.yaml`
5. Environment variables
6. CLI flags

**Gotcha**: Environment variables can override config file
**Solution**: Use CLI flags for highest precedence

### 4. Timeout Defaults are Zero (No Timeout)

**Rule**: Default timeouts are 0 (unlimited)
**Why**: Go's net/http.Server default behavior
**Gotcha**: Production should set timeouts
**Recommendation**:

- readHeaderTimeout: 10s
- readTimeout: 30s
- writeTimeout: 30s
- idleTimeout: 120s

## Dependency Constraints

### 1. Go Modules Required

**Rule**: Always run `go mod download` before build
**Why**: Ensures dependencies are available
**Implementation**: Makefile does this automatically
**Don't**: Delete `go.sum` or commit without it

### 2. Minimal External Dependencies

**Rule**: Avoid adding unnecessary dependencies
**Why**: Keep binary small and secure
**Current deps**: viper, zap, pflag, uuid (only essentials)

## Git/CI Constraints

### 1. Conventional Commits Required

**Rule**: Use conventional commit format for main branch
**Why**: release-please generates CHANGELOG from commits
**Format**: `type(scope): description`
**Types**: feat, fix, docs, style, refactor, test, build, ci, perf, revert

### 2. PR Checks Must Pass

**Rule**: All CI checks must pass before merge
**Checks**:

- unit-tests (with race detector)
- linting (0 issues)
- integration-tests

### 3. Docker Builds Only on Main

**Rule**: Docker builds don't run on PRs
**Why**: Saves CI time, prevents unauthorized pushes
**When**: Only on main branch pushes and tags

## macOS/Darwin Specific

### 1. System Commands Differ from Linux

**Examples**:

- `date -u +"%Y-%m-%dT%H:%M:%SZ"` (works on macOS)
- `ls -la` (macOS has different flags than GNU ls)
- `find` (macOS find vs GNU find differences)

**Solution**: Makefile handles cross-platform differences

### 2. golangci-lint Installation

**macOS**: Install via Homebrew

```bash
brew install golangci-lint
```

**Verify**: `which golangci-lint` should show path

## Security Constraints

### 1. Default Timeouts are Insecure

**Rule**: Production must configure timeouts
**Why**: Vulnerable to Slowloris and DoS attacks without timeouts
**Solution**: Always set readHeaderTimeout, readTimeout, writeTimeout, idleTimeout in production

### 2. Logging May Expose Secrets

**Rule**: Be careful what you log
**Why**: Request/response bodies may contain tokens, passwords
**Solution**: Use excludePostBody and excludeResponseBody regex patterns

## Performance Constraints

### 1. Structured Logging Overhead

**Rule**: Logging adds latency to requests
**Why**: JSON serialization and I/O
**Mitigation**: Use exclude patterns for high-traffic paths

### 2. Body Logging Memory Usage

**Rule**: Logging large bodies consumes memory
**Why**: Full request/response buffered for logging
**Solution**: Use excludePostBody/excludeResponseBody for large payloads

## Common Pitfalls

### 1. Forgetting make fmt

**Symptom**: CI fails with formatting errors
**Solution**: Always `make fmt` first

### 2. Not Testing with -race

**Symptom**: CI fails with race conditions
**Solution**: Run `go test -race ./...` locally

### 3. Adding TODO Comments

**Symptom**: Lint fails with godox errors
**Solution**: Use GitHub issues instead

### 4. Not Running Full Test Suite

**Symptom**: PR fails integration tests
**Solution**: Run complete pre-commit workflow:

```bash
make fmt && make build && make test && make lint && make test-integration
```

### 5. Decreasing Coverage

**Symptom**: Coverage drops below threshold
**Solution**: Add tests for new code, check with `make test-coverage-html`
