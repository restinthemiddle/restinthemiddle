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

Clone this repository and run the `build_native` script.

```bash
git clone https://github.com/jensschulze/restinthemiddle.git
cd restinthemiddle
./build_native
```

## Usage

Typically you place the logging proxy between an application and an API. This is the use case Restinthemiddle was developed for.

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

You may as well use Restinthemiddle as an alternative entrypoint for your Application.

### Configuration

Configuration is handled by [spf13/viper](https://pkg.go.dev/github.com/spf13/viper).

Restinthemiddle is intended for use in a containerized environment. Therefore it is configurable entirely via environment variables.

The ascending order of precedence (last wins) is:

* restinthemiddle default values
* Configuration via YAML file
* Configuration via Environment variables

Of course you may provide an incomplete configuration. This is the default configuration:

```yaml
targethostdsn: http://host.docker.internal:8081
listenaddress: 0.0.0.0:8000
headers:
    User-Agent: Rest in the middle logging proxy
loggingenabled: true
exclude: ""
```

#### Environment variables

* `EXCLUDE` (optional): If the given URL path matches this Regular Expression the request/response will not be logged.
* `LISTEN_ADDRESS` (optional): The (ip and) port on which Restinthemiddle will be listening to requests. Defaults to `0.0.0.0:8000`.
* `LOGGING_ENABLED` (optional): Defaults to `true`.
* `TARGET_HOST_DSN` (required): The DSN of the target host in the form `schema://username:password@hostname:port/basepath?query`.
  * `schema` (required) is `http` or `https`
  * `username:password@` is optional and will be evaluated only if both values are set.
  * `hostname` (required)
  * `port` is optional. Standard ports are 80 (http) and 443 (https).
  * `basepath` is optional. Will be prefixed to any request URL path pointed at Restinthemiddle. See examples section.
  * `query` is optional. If set, `query` will precede the actual request’s query.

**Note:** It is not possible to populate the `headers` dictionary via environment variable. You have to use a configuration file at least for `headers`.

## Examples

### Basic

We want to log HTTP calls against `www.example.com` over an insecure connection.

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=http://www.example.com -p 8000:8000 jdschulze/restinthemiddle

# In another terminal window we make the API call against http://www.example.com/api/visitors
curl -i http://127.0.0.1:8000/api/visitors
```

### Advanced

We want to log HTTP calls against `www.example.com:4430` over a TLS connection (`https://…`). The API is protected by HTTP basic auth (username: `user`; password: `pass`). The base path always starts with `api/`.

Note that the base path defined in `TARGET_HOST_DSN` prefixes any subsequent calls!

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=https://user:pass@www.example.com:4430/api?start=1577833200 -p 8000:8000 jdschulze/restinthemiddle

# In another terminal window we make the API call against https://user:pass@www.example.com:4430/api/visitors?start=1577833200
curl -i http://127.0.0.1:8000/visitors
```

### With configuration

We want to log HTTP calls against `www.example.com` over an insecure connection. Every request has to be enhanced with a custom header `X-App-Version: 3.0.0`. No logging shall take place.

#### config.yaml

```yaml
targetHostDsn: http://www.example.com
headers:
    X-App-Version: '3.0.0'
loggingEnabled: false
```

```bash
# Set up the proxy
docker run -it --rm -v ./config.yaml:/restinthemiddle/config.yaml -p 8000:8000 jdschulze/restinthemiddle:latest

# In another terminal window we make the API call against http://www.example.com/home
curl -i http://127.0.0.1:8000/home
```
