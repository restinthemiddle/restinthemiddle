.PHONY: docker, docker-build-env, build, fmt, lint, test, test-coverage, test-coverage-html, test-integration

docker:
	docker buildx build --pull --progress=plain -t jdschulze/restinthemiddle:latest .

docker-build-env:
	docker buildx --pull --progress=plain --target=build-env -t jdschulze/restinthemiddle:build-env .

bin/restinthemiddle: $(shell find . -name "*.go" -not -path "./vendor/*") go.mod go.sum
	@mkdir -p bin
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o bin/restinthemiddle ./cmd/restinthemiddle/main.go

build: bin/restinthemiddle

fmt:
	go fmt ./...

lint:
	golangci-lint run --config .golangci.yml --timeout 5m

test:
	go test -v ./...

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
		go test -v -tags integration ./tests/integration/...; \
	else \
		echo "No integration tests found in tests/integration/"; \
	fi
