docker:
	DOCKER_BUILDKIT=1 docker build --pull -t jdschulze/restinthemiddle:latest .

build:
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w' -o bin/restinthemiddle
