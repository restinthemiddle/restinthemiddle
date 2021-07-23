ARG BASE_IMAGE_BUILD=golang:alpine
ARG BASE_IMAGE_RELEASE=alpine:latest

FROM ${BASE_IMAGE_BUILD} AS build-env

RUN mkdir -p /src
WORKDIR /src

COPY go.* .
RUN go mod download

COPY . .
RUN cd /src && go build -o restinthemiddle

FROM ${BASE_IMAGE_RELEASE}

COPY --from=build-env /src/restinthemiddle /usr/local/bin/

CMD ["/usr/local/bin/restinthemiddle"]
