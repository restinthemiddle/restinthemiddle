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

## Usage

Typically you place the logging proxy between an application and an API:

```text
+-----------------+         +-----------------+         +-----------------+
|                 +-------->+                 +-------->+                 |
|   Application   |         | Restinthemiddle |         |       API       |
|                 +<--------+                 +<--------+                 |
+-----------------+         +-----------------+         +-----------------+
```

But there are cases where it makes sense to place it between your browser and the application. For example you could want to add custom headers to every request (kind of an off-label use, because no logging is needed):

```text
+-----------------+         +-----------------+         +-----------------+
|                 +-------->+                 +-------->+                 |
|     Browser     |         | Restinthemiddle |         |   Application   |
|                 +<--------+                 +<--------+                 |
+-----------------+         +-----------------+         +-----------------+
```

### Configuration

Restinthemiddle is intended for use in a dockerized environment. Therefore it is configurable entirely via environment variables.

#### Environment variables

* `TARGET_HOST_DSN` (required): The DSN of the target host in the form `schema://username:password@hostname:port/basepath`.
  * `schema` (required) is `http` or `https`
  * `username:password@` is optional and will be evaluated only if both values are set.
  * `hostname` (required)
  * `port` is optional. Standard ports are 80 (http) and 443 (https).
  * `basepath` is optional. Will be prefixed to any request URL path pointed at Restinthemiddle. See examples section.
* `PORT` (optional): The port on which Restinthemiddle will be listening to requests. Defaults to `8000`.
* `CONFIG` (optional): At the moment you can configure extra headers as a JSON string in the form:

```json
{
    "headers": {
        "X-App-Version": "3.0.0",
        "Another-Header": "Test"
    }
}
```

## Examples

### Basic

We want to log HTTP calls against `www.example.com` over an insecure connection.

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=http://www.example.com -p 8000:8000 jdschulze/restinthemiddle

# In another terminal window we make the API call against http://www.example.com/api/uptime
curl -i http://127.0.0.1:8000/api/uptime
```

### Advanced

We want to log HTTP calls against `www.example.com:4430` over a TLS connection (`https://…`). The API is protected by HTTP basic auth (username: `user`; password: `pass`). The base path always contains `api/`.

Note that we define a base path in `TARGET_HOST_DSN` that prefixes any subsequent calls!

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=https://user:pass@www.example.com:4430/api -p 8000:8000 jdschulze/restinthemiddle

# In another terminal window we make the API call against https://user:pass@www.example.com:4430/api/uptime
curl -i http://127.0.0.1:8000/uptime
```

### With configuration

We want to log HTTP calls against `www.example.com` over an insecure connection. Every request has to be enhanced with a custom header `X-App-Version: 3.0.0`.

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=http://www.example.com -e CONFIG='{"headers":{"X-App-Version":"3.0.0"}}' -p 8000:8000 jdschulze/restinthemiddle:latest

# In another terminal window we make the API call against http://www.example.com/home
curl -i http://127.0.0.1:8000/home
```
