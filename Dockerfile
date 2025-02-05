FROM golang:1.23.6-alpine AS build-env

WORKDIR /src

RUN apk -U upgrade \
    && apk add --no-cache dumb-init

COPY go.* .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o restinthemiddle

FROM alpine:3.21 AS artifact

LABEL org.opencontainers.image.authors="Jens Schulze"

RUN apk -U upgrade

COPY --from=build-env /src/restinthemiddle /usr/bin/restinthemiddle

COPY --from=build-env /usr/bin/dumb-init /usr/bin/dumb-init

ENTRYPOINT ["dumb-init", "--"]
CMD ["restinthemiddle"]
