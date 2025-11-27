# CI/CD Pipeline Details

## GitHub Actions Workflow

**File**: `.github/workflows/github-pipeline.yml`

## Pipeline Jobs

### 1. unit-tests

**Runs on**: All pushes and PRs
**Duration**: ~3-4 seconds

**Steps**:

1. Checkout code
2. Set up Go (stable version)
3. Install dependencies: `go mod download`
4. Run tests with race detector and coverage:

   ```bash
   go test -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/... ./internal/... ./cmd/...
   ```

5. Generate HTML coverage: `go tool cover -html=coverage.out -o coverage.html`
6. Upload coverage to Codecov (with token)
7. Upload coverage as GitHub artifact
8. Comment coverage on PR (if PR event)

**Key Points**:

- **Always uses `-race` flag** for race condition detection
- Tests only: `./pkg/...`, `./internal/...`, `./cmd/...`
- Coverage uploaded to Codecov
- Coverage artifacts available for download

### 2. linting

**Runs on**: All pushes and PRs
**Duration**: Variable (timeout: 5 minutes)

**Steps**:

1. Checkout code
2. Set up Go (stable version)
3. Run golangci-lint:

   ```bash
   golangci-lint run --config .golangci.yml
   ```

**Key Points**:

- Uses `golangci/golangci-lint-action@v9`
- Latest golangci-lint version
- Configuration from `.golangci.yml`
- Must show "0 issues" to pass

### 3. integration-tests

**Runs on**: All pushes and PRs
**Depends on**: unit-tests + linting (must both pass)
**Duration**: ~5 seconds

**Steps**:

1. Checkout code
2. Set up Go (stable version)
3. Build binary: `make build`
4. Run integration tests:

   ```bash
   go test -v -tags=integration -race ./tests/integration/...
   ```

**Key Points**:

- Requires binary to be built first
- Uses `-tags=integration` build tag
- Also uses `-race` flag
- Tests HTTP proxy end-to-end scenarios

### 4. docker-build

**Runs on**: main branch pushes and tags ONLY (not PRs)
**Depends on**: unit-tests + linting + integration-tests
**Duration**: Several minutes

**Steps**:

1. Checkout code
2. Set up Docker Buildx
3. Login to Docker Hub (uses secrets)
4. Determine build variables:
   - VERSION: `git describe --tags --always --dirty`
   - BUILD_DATE: ISO 8601 format
   - GIT_COMMIT: `git rev-parse HEAD`
5. Extract Docker metadata (tags, labels)
6. Build and push multi-platform image:
   - Platforms: linux/amd64, linux/arm64
   - Registry: docker.io
   - Image: jdschulze/restinthemiddle
7. Update Docker Hub README (on tag pushes)

**Tags Created**:

- `latest` (on main branch)
- `edge` (on main branch)
- `sha-<short>` (short commit SHA)
- Semantic versions on tags (e.g., `2`, `2.2`, `2.2.2`)

**Key Points**:

- **Only runs on main branch or tags**
- **Never runs on PRs**
- Multi-platform build (amd64, arm64)
- Requires Docker Hub credentials (secrets)

### 5. coverage-summary

**Runs on**: All pushes and PRs
**Depends on**: unit-tests
**Duration**: ~1 second

**Steps**:

1. Checkout code
2. Download coverage artifacts
3. Set up Go
4. Generate coverage summary in GitHub Step Summary

**Key Points**:

- Displays coverage percentage
- Shows top uncovered functions
- Links to detailed coverage reports

### 6. release-please

**Runs on**: main branch pushes ONLY
**Duration**: Variable

**Steps**:

1. Checkout code
2. Run release-please action
3. If release created:
   - Set up Go
   - Build binary
   - Upload binary as release asset

**Key Points**:

- **Only runs on main branch**
- Creates release PRs based on conventional commits
- Generates CHANGELOG automatically
- Uploads binary to GitHub releases

## Required Secrets

- `CODECOV_TOKEN`: For uploading coverage to Codecov
- `DOCKERHUB_USERNAME`: Docker Hub username
- `DOCKERHUB_TOKEN`: Docker Hub access token
- `RELEASE_PLEASE_TOKEN`: Token for release-please
- `GITHUB_TOKEN`: Automatic (provided by GitHub)

## Permissions

- `contents: read` (default)
- `contents: write` (release-please)
- `pull-requests: write` (release-please)
- `packages: write` (docker-build)

## Important Notes

### Race Detector

- **All tests run with `-race` flag in CI**
- If tests pass locally but fail in CI, run `go test -race ./...` locally
- Race conditions are common in concurrent HTTP code

### Coverage Requirements

- Current coverage: ~87-94% across packages
- Don't significantly decrease coverage
- Coverage uploaded to Codecov for tracking

### PR Checks vs Main Checks

**On PRs**:

- unit-tests ✅
- linting ✅
- integration-tests ✅
- coverage-summary ✅

**Only on main/tags**:

- docker-build (main branch and tags)
- release-please (main branch only)

### Build Artifacts

- Coverage reports (coverage.out, coverage.html)
- Binary (on releases only)

## Troubleshooting CI Failures

### Race Condition Failures

**Error**: `WARNING: DATA RACE`
**Solution**: Run `go test -race ./...` locally to reproduce

### Lint Failures

**Error**: `x issues`
**Solution**: Run `make lint` locally, must show "0 issues"

### Integration Test Failures

**Error**: HTTP request failures, timeouts
**Solution**: Check `tests/integration/integration_test.go` and mock server

### Docker Build Failures

**Error**: Platform build errors
**Solution**: Test locally with `make docker` (single platform)

### Coverage Upload Failures

**Error**: Codecov upload failed
**Note**: This doesn't fail the build (fail_ci_if_error: false)
