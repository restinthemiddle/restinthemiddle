.PHONY: docker, docker-build-env, build

docker:
	DOCKER_BUILDKIT=1 docker build --pull -t jdschulze/restinthemiddle:latest .

docker-build-env:
	DOCKER_BUILDKIT=1 docker build --pull --target=build-env -t jdschulze/restinthemiddle:build-env .

build:
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o bin/restinthemiddle
