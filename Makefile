BASE_IMAGE_BUILD := golang:1.19-alpine
BASE_IMAGE_RELEASE := scratch

docker:
	DOCKER_BUILDKIT=1 docker build --pull --build-arg BASE_IMAGE_BUILD=$(BASE_IMAGE_BUILD) --build-arg BASE_IMAGE_RELEASE=$(BASE_IMAGE_RELEASE) -t jdschulze/restinthemiddle:latest .

build:
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w' -o bin/restinthemiddle
