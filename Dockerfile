FROM golang:1.25.0-alpine AS build-env

WORKDIR /src

RUN apk -U upgrade \
    && apk add --no-cache dumb-init

COPY go.* .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o restinthemiddle ./cmd/restinthemiddle/main.go

FROM alpine:3.22.1 AS artifact

LABEL org.opencontainers.image.authors="Jens Schulze"

ENV TZ=UTC

RUN apk -U upgrade \
    && apk add --no-cache ca-certificates tzdata \
    && rm -rf /var/cache/apk/*

COPY --from=build-env /src/restinthemiddle /usr/bin/restinthemiddle

COPY --from=build-env /usr/bin/dumb-init /usr/bin/dumb-init

ENTRYPOINT ["dumb-init", "--"]
CMD ["restinthemiddle"]
