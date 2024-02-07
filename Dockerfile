FROM golang:1.22-alpine AS build-env

WORKDIR /src

RUN apk update \
    && apk upgrade \
    && apk add --no-cache ca-certificates dumb-init

COPY go.* .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o restinthemiddle

FROM busybox:latest as artifact

LABEL org.opencontainers.image.authors="Jens Schulze"

COPY --from=build-env /src/restinthemiddle /usr/bin/restinthemiddle

COPY --from=build-env /etc/ssl /etc/ssl
COPY --from=build-env /usr/bin/dumb-init /usr/bin/dumb-init

ENTRYPOINT ["dumb-init", "--"]
CMD ["restinthemiddle"]