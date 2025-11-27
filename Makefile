.PHONY: docker, docker-build-env, build, fmt, lint, test, test-coverage, test-coverage-html, test-integration

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

EXTRA_LDFLAGS := -X github.com/restinthemiddle/restinthemiddle/internal/version.Version=$(VERSION) -X github.com/restinthemiddle/restinthemiddle/internal/version.BuildDate=$(BUILD_DATE) -X github.com/restinthemiddle/restinthemiddle/internal/version.GitCommit=$(GIT_COMMIT)

docker:
	docker buildx build --pull --progress=plain \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t jdschulze/restinthemiddle:latest .

docker-build-env:
	docker buildx build --pull --progress=plain \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		--target=build-env -t jdschulze/restinthemiddle:build-env .

bin/restinthemiddle: $(shell find . -name "*.go" -not -path "./vendor/*") go.mod go.sum
	@mkdir -p bin
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w $(EXTRA_LDFLAGS)' -trimpath -o bin/restinthemiddle ./cmd/restinthemiddle/main.go

build: bin/restinthemiddle

fmt:
	go fmt ./...

lint:
	golangci-lint run --config .golangci.yml --timeout 5m

test:
	go test -race -v ./...

test-coverage:
	go test -v -cover ./...

coverage.out: $(shell find . -name "*.go" -not -path "./vendor/*") go.mod go.sum
	go test -coverprofile=coverage.out ./...

test-coverage-html: coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration:
	@if [ -d "tests/integration" ] && [ -n "$$(find tests/integration -name '*.go' 2>/dev/null)" ]; then \
		echo "Running integration tests..."; \
		go test -race -v -tags integration ./tests/integration/...; \
	else \
		echo "No integration tests found in tests/integration/"; \
	fi
