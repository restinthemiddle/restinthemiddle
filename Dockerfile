ARG BASE_IMAGE_BUILD=golang:1.16-alpine
ARG BASE_IMAGE_RELEASE=alpine:3.14

FROM ${BASE_IMAGE_BUILD} AS build-env

RUN mkdir -p /src
WORKDIR /src

COPY go.* .
RUN go mod download

COPY . .
WORKDIR /src
RUN go build -o restinthemiddle

FROM ${BASE_IMAGE_RELEASE}

LABEL org.opencontainers.image.authors="Jens Schulze"

COPY --from=build-env /src/restinthemiddle /usr/local/bin/

CMD ["/usr/local/bin/restinthemiddle"]
