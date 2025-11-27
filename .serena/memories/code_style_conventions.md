# Code Style and Conventions

## Go Style Guide

### General Conventions

- **Standard Go formatting**: Use `go fmt` (enforced via `make fmt`)
- **Naming**: CamelCase for exported, camelCase for unexported
- **Package names**: Lowercase, single word when possible
- **File naming**: Lowercase with underscores (e.g., `core_test.go`)

### Code Organization

#### Package Structure

- `cmd/`: Main applications (entry points)
- `internal/`: Private application code (not importable)
- `pkg/`: Public libraries (can be imported by external projects)
- `tests/`: Integration and e2e tests

#### Naming Patterns

- **Interfaces**: Named by capability (e.g., `Writer`, `HTTPServer`, `ConfigLoader`)
- **Structs**: Descriptive nouns (e.g., `App`, `Server`, `TranslatedConfig`)
- **Functions**: Verb-based (e.g., `Run`, `NewServer`, `LoadConfig`)
- **Test files**: `*_test.go` suffix
- **Integration tests**: Use `-tags=integration` build tag

### Testing Conventions

#### Test Structure

- **Table-driven tests**: Use subtests with `t.Run()`
- **Test naming**: `Test<FunctionName>` or `Test<StructName>_<MethodName>`
- **Coverage target**: Maintain 85%+ coverage
- **Mock interfaces**: Use for dependency injection in tests

#### Test Files

```go
// Unit tests in same package
package core

func TestRun(t *testing.T) { ... }

// Integration tests with build tag
//go:build integration
package integration_test
```

### Documentation

#### Comments

- **Package comments**: At top of one file per package
- **Function comments**: Start with function name
- **Exported items**: Must have doc comments
- **TODO/FIXME/HACK**: **NOT ALLOWED** (godox linter flags these)
  - Use GitHub issues instead for tracking work

#### Example Doc Comment

```go
// Run starts the proxy server with the given configuration.
// It initializes the reverse proxy, sets up logging, and starts
// the HTTP server on the configured IP and port.
func Run(c *config.TranslatedConfig, w Writer, s HTTPServer) {
    ...
}
```

### Error Handling

#### Patterns

- **Early returns**: Check errors immediately
- **Fatal errors**: Use `log.Fatalf()` for unrecoverable errors
- **Wrapped errors**: Use `fmt.Errorf("context: %w", err)` when appropriate
- **No panic**: Prefer returning errors over panic

#### Example

```go
proxyServer, err := proxy.NewServer(cfg)
if err != nil {
    log.Fatalf("Failed to create proxy server: %v", err)
}
```

### Configuration

#### Viper Precedence (last wins)

1. Application defaults
2. `/etc/restinthemiddle/config.yaml`
3. `~/.restinthemiddle/config.yaml`
4. `./config.yaml`
5. Environment variables
6. CLI flags

#### Required Fields

- `targetHostDsn`: REQUIRED - will fail without it
- All other fields: Optional with sensible defaults

### Dependency Management

#### Go Modules

- **go.mod**: Declares Go 1.25
- **Dependencies**: Minimal external deps
  - spf13/viper (config)
  - go.uber.org/zap (logging)
  - spf13/pflag (CLI flags)
  - google/uuid (request IDs)

#### Build Flags

- **CGO_ENABLED=0**: Always for static binaries
- **-trimpath**: Remove file paths from binary
- **-ldflags '-s -w'**: Strip debug info for smaller binary
- **Version injection**: Via ldflags at build time

### Linting

#### Enabled Linters (.golangci.yml)

- `errcheck`: Check for unchecked errors
- `gocognit`: Complexity checking
- `goconst`: Find repeated strings
- `gocritic`: Comprehensive checks
- `godot`: Check comment periods
- `godox`: **Flags TODO/FIXME/HACK** (STRICT)
- `gosec`: Security checks
- `govet`: Standard Go vet
- `ineffassign`: Detect ineffectual assignments
- `staticcheck`: Advanced static analysis
- `unconvert`: Unnecessary conversions
- `unparam`: Unused function parameters
- `unused`: Unused code
- `wastedassign`: Wasted assignments
- `whitespace`: Whitespace issues

#### Lint Timeout

- 5 minutes (configured in `.golangci.yml`)

### Git Conventions

#### Commit Messages

- Use conventional commits (release-please requirement)
- Format: `type(scope): description`
- Types: feat, fix, docs, style, refactor, test, build, ci, perf, revert

#### Examples

```
feat(proxy): add support for custom timeout configuration
fix(config): handle empty targetHostDsn gracefully
docs(readme): update installation instructions
test(core): add race detector tests
```

### File Headers

#### Version Information

- Version, build date, git commit injected via ldflags
- Located in `internal/version/version.go`
- Accessible via `--version` flag

#### License

- MIT License (see LICENSE file)
- Some packages have separate LICENSE file (e.g., pkg/core/LICENSE)

### Design Patterns

#### Dependency Injection

- Interfaces for testability (Writer, HTTPServer, ConfigLoader)
- Constructor functions (NewApp, NewServer)
- Factory pattern for logger creation

#### Configuration

- Struct-based config (Config → TranslatedConfig)
- Validation at translation time
- Regexp compilation during config load

#### Middleware Pattern

- ModifyResponse function in reverse proxy
- Request/response logging as middleware
- Header manipulation in Director function
