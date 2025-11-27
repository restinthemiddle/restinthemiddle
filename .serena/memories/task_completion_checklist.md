# Task Completion Checklist

## Before Creating a Pull Request

### 1. Code Formatting (MANDATORY)

```bash
make fmt
```

- **Must run** before any commit
- Formats all Go code with `go fmt`
- CI will reject unformatted code

### 2. Build Validation

```bash
make build
```

- Ensures code compiles
- Creates `bin/restinthemiddle` binary
- Duration: ~3-5 seconds
- **Must succeed** without errors

### 3. Unit Tests (MANDATORY)

```bash
make test
```

- Runs all unit tests
- **Must pass** with 0 failures
- Duration: ~3-4 seconds
- Coverage should be ~87-94%

### 4. Linting (MANDATORY)

```bash
make lint
```

- **Must show "0 issues"**
- Checks for:
  - Code quality issues
  - Security vulnerabilities (gosec)
  - TODO/FIXME/HACK comments (NOT allowed)
  - Unused code
  - Unchecked errors
- Timeout: 5 minutes

### 5. Integration Tests (MANDATORY)

```bash
make test-integration
```

- Tests end-to-end scenarios
- **Must pass** all tests
- Duration: ~5 seconds
- Requires binary to be built first

### 6. Coverage Check (RECOMMENDED)

```bash
make test-coverage
```

- Verify coverage hasn't decreased
- Target: 85%+ coverage
- **Do not** significantly reduce coverage

## Complete Pre-Commit Command

Run all checks in one command:

```bash
make fmt && make build && make test && make lint && make test-integration
```

## Expected Outputs

### Success Indicators

- `make fmt`: Files may be formatted, no errors
- `make build`: Binary created at `bin/restinthemiddle`
- `make test`: All tests show "PASS", none show "FAIL"
- `make lint`: Displays "0 issues."
- `make test-integration`: All integration tests PASS

### Failure Indicators

- `make fmt`: Should never fail (only formats)
- `make build`: Compilation errors
- `make test`: Test failures or panics
- `make lint`: Issues count > 0
- `make test-integration`: Test failures

## CI/CD Pipeline Checks

Your PR will be tested with these jobs (must all pass):

### 1. unit-tests

- Runs: `go test -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/... ./internal/... ./cmd/...`
- **Includes race detector** (`-race` flag)
- Uploads coverage to Codecov
- **Critical**: If CI fails with race conditions but local tests pass, run `go test -race ./...` locally

### 2. linting

- Runs: `golangci-lint run --config .golangci.yml`
- Uses latest golangci-lint version
- **Must show 0 issues**

### 3. integration-tests

- Depends on: unit-tests + linting passing
- Builds binary first: `make build`
- Runs: `go test -v -tags=integration -race ./tests/integration/...`

### 4. docker-build (only on main/tags, not PRs)

- Multi-platform: linux/amd64, linux/arm64
- Pushes to Docker Hub

### 5. release-please (only on main)

- Creates release PRs based on conventional commits
- Generates CHANGELOG

## Common Failure Scenarios

### Race Conditions

**Symptom**: Tests pass locally but fail in CI with race detector
**Solution**:

```bash
go test -race ./...
```

- CI always runs with `-race` flag
- Test locally with race detector before pushing

### TODO/FIXME/HACK Comments

**Symptom**: Linter fails with godox errors
**Solution**:

- Remove all TODO/FIXME/HACK comments
- Use GitHub issues to track work instead
- Never leave these markers in code

### Coverage Decrease

**Symptom**: Coverage drops below acceptable threshold
**Solution**:

- Add tests for new code
- Use `make test-coverage-html` to identify gaps
- Aim for 85%+ coverage

### Build Failures

**Symptom**: `make build` fails
**Solution**:

- Check compilation errors
- Ensure `go mod download` succeeds
- Verify Go version (1.25.4+)

### Formatting Issues

**Symptom**: CI complains about formatting
**Solution**:

- Always run `make fmt` before commit
- Configure editor to run `go fmt` on save

## Quick Validation

Minimal checks before committing:

```bash
# Quick validation (< 15 seconds total)
make fmt && make lint && make test
```

Full validation before PR:

```bash
# Complete validation (< 30 seconds total)
make fmt && make build && make test && make lint && make test-integration
```

## Notes

1. **Always format first**: `make fmt` should be the first command
2. **Lint must show 0 issues**: No exceptions
3. **All tests must pass**: No flaky tests accepted
4. **Race detector**: CI runs with `-race`, test locally with it
5. **Coverage**: Don't significantly reduce existing coverage
6. **Build time**: All commands should complete in < 30 seconds total
7. **TODO comments**: Use GitHub issues instead

## If All Checks Pass

You're ready to create a PR! The CI will run the same checks and should pass.
