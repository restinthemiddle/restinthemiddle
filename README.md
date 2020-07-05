# restinthemiddle

This Go program acts as a lightweight HTTP logging proxy for developing and staging environments. If you put it between an API client and the API you can easily monitor requests and responses.

## Installation

### Docker (recommended)

Pull the [Docker image](https://hub.docker.com/repository/docker/jdschulze/restinthemiddle) from Docker Hub

```bash
docker pull jdschulze/restinthemiddle
```

### Build the Docker image yourself

Clone this repository and run the `build` script.

```bash
git clone https://github.com/jensschulze/restinthemiddle.git
cd restinthemiddle
./build
```

### Build the binary yourself

Clone this repository and run `go build`.

```bash
git clone https://github.com/jensschulze/restinthemiddle.git
cd restinthemiddle
go build -o restinthemiddle
```

## Example

We want to log HTTP calls against `www.example.com:4430` over a TLS connection (`https://…`). The API is protected by HTTP basic auth (user: `user`; password: `password`). The base path always contains `api/`.

Note that we define a base path in `TARGET_HOST_DSN` that prefixes any subsequent calls!

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=https://user:password@www.example.com:4430/api -p 8000:8000 jdschulze/restinthemiddle

# In another terminal window we make the API call against https://user:password@www.example.com:4430/api/test
curl -i http://127.0.0.1:8000/test
```
