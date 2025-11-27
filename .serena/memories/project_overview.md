# Project Overview

## Purpose

**restinthemiddle** is a lightweight HTTP logging proxy server designed for development and staging environments. It acts as a transparent middleware between HTTP clients and servers, logging all requests and responses for debugging and monitoring purposes.

## Key Features

- HTTP/HTTPS reverse proxy with comprehensive logging
- Request and response body logging with filtering capabilities
- Custom header injection
- Request ID generation (UUID v4)
- Configurable timeouts for security
- Structured JSON logging via zap
- Multi-platform Docker support (amd64, arm64)

## Tech Stack

- **Language**: Go 1.25.4
- **Configuration**: spf13/viper (supports YAML, env vars, CLI flags)
- **Logging**: go.uber.org/zap (structured JSON logging)
- **HTTP**: Standard library net/http with custom reverse proxy
- **UUID**: Google UUID for request IDs
- **Build**: Multi-stage Docker builds with Alpine Linux
- **CI/CD**: GitHub Actions with release-please

## Project Size

- Approximately 7,500 lines of code
- Small, focused codebase
- Well-tested (~87-94% coverage across packages)

## Use Cases

1. **API Debugging**: Place between application and API to monitor HTTP traffic
2. **Header Injection**: Add custom headers to requests (off-label use)
3. **Development Proxy**: Alternative entrypoint for applications in staging
