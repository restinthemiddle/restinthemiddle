.PHONY: docker, docker-build-env, build

docker:
	docker buildx build --pull --progress=plain -t jdschulze/restinthemiddle:latest .

docker-build-env:
	docker buildx --pull --progress=plain --target=build-env -t jdschulze/restinthemiddle:build-env .

build:
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o bin/restinthemiddle ./cmd/restinthemiddle/main.go
