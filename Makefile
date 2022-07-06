BASE_IMAGE_BUILD := golang:1.18-alpine
BASE_IMAGE_RELEASE := alpine:3.16

docker:
	docker pull $(BASE_IMAGE_BUILD)
	docker pull $(BASE_IMAGE_RELEASE)
	DOCKER_BUILDKIT=1 docker build --build-arg BASE_IMAGE_BUILD=$(BASE_IMAGE_BUILD) --build-arg BASE_IMAGE_RELEASE=$(BASE_IMAGE_RELEASE) -t jdschulze/restinthemiddle:latest .

native:
	go mod download
	CGO_ENABLED=0 go build -ldflags '-s -w' -o bin/restinthemiddle
