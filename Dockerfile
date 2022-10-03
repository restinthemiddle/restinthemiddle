FROM golang:1.19-alpine AS build-env

WORKDIR /src

COPY go.* .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o restinthemiddle

FROM scratch

LABEL org.opencontainers.image.authors="Jens Schulze"

COPY --from=build-env /src/restinthemiddle /

ENTRYPOINT ["/restinthemiddle"]
