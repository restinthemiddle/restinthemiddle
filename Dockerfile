ARG BASE_IMAGE_BUILD=golang:1.18-alpine
ARG BASE_IMAGE_RELEASE=alpine:3.16

FROM ${BASE_IMAGE_BUILD} AS build-env

WORKDIR /src

COPY go.* .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o restinthemiddle

FROM ${BASE_IMAGE_RELEASE}

LABEL org.opencontainers.image.authors="Jens Schulze"

COPY --from=build-env /src/restinthemiddle /usr/local/bin/

CMD ["/usr/local/bin/restinthemiddle"]
