FROM golang:alpine AS build-env
COPY . /src/
RUN cd /src && go build -o restinthemiddle

FROM alpine:latest
ENV TARGET_HOST_DSN=http://host.docker.internal:8081 \
    PORT=8000
COPY --from=build-env /src/restinthemiddle /usr/local/bin/
CMD ["/usr/local/bin/restinthemiddle"]
