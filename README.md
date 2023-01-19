# Restinthemiddle v2

![pulls](https://img.shields.io/docker/pulls/jdschulze/restinthemiddle?style=flat)
![semver](https://img.shields.io/github/v/tag/jensschulze/restinthemiddle?style=flat&sort=semver)
![size_amd64](https://img.shields.io/docker/image-size/jdschulze/restinthemiddle/2?style=flat&arch=amd64)
![size_arm64](https://img.shields.io/docker/image-size/jdschulze/restinthemiddle/2?style=flat&arch=arm64)

This lightweight application acts as a HTTP logging proxy for developing and staging environments. If you put it between an HTTP client and the respective server you can easily monitor requests and responses.

## Installation

### Docker (recommended)

Pull the [Docker image](https://hub.docker.com/repository/docker/jdschulze/restinthemiddle) from Docker Hub

```bash
docker pull jdschulze/restinthemiddle:2
```

Pinning the version to (at least) the major version is highly recommended. Use `latest` at your own risk. The `latest` tag is always the `HEAD` of the `main` branch **or** the latest _git tag_. Most recent wins, so `latest` is absolutely insane.

### Build the Docker image yourself

Clone this repository and run `make docker`.

```bash
git clone https://github.com/restinthemiddle/restinthemiddle.git
cd restinthemiddle
git checkout v2
make docker
```

### Build the binary yourself

Clone this repository and run `make build`.

```bash
git clone https://github.com/restinthemiddle/restinthemiddle.git
cd restinthemiddle
git checkout v2
make build
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

You may as well use Restinthemiddle as an alternative entrypoint for your application.

### Configuration

Configuration is handled by [spf13/viper](https://pkg.go.dev/github.com/spf13/viper).

Restinthemiddle is intended for use in a containerized environment. Therefore it is configurable entirely via environment variables - almost!
Headers have to be set via command line arguments or the configuration file.

The ascending order of precedence (last wins) is:

* restinthemiddle default values
* Configuration via YAML file
* Configuration via Environment variables
* Command line arguments

Example configuration file:

```yaml
targetHostDsn: www.example.com
listenIp: 0.0.0.0
listenPort: "8000"
headers:
    X-My-Header: myexamplevalue
loggingEnabled: true
setRequestId: false
exclude: ""
logPostBody: true
logResponseBody: true
excludePostBody: ""
excludeResponseBody: ""
```

There are several file locations where configuration is being searched for. The ascending order of precedence (last wins) is:

* `/etc/restinthemiddle/config.yaml`
* `$HOME/.restinthemiddle/config.yaml`
* `./config.yaml`

#### Keys

| Configuration key                | Environment variable    | Command line flag       | Description                                                                                                                                                  | Default value                                  |
|----------------------------------|-------------------------|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------|
| `targetHostDsn` (required)       | `TARGET_HOST_DSN`       | --target-host-dsn       | The DSN of the target host in the form `schema://username:password@hostname:port/basepath?query`. Find a [detailed description](#the-target-host-dsn) below. | -                                              |
| `listenIp` (optional)            | `LISTEN_IP`             | --listen-ip             | The IP on which Restinthemiddle listens for requests.                                                                                                        | `0.0.0.0`                                      |
| `listenPort` (optional)          | `LISTEN_PORT`           | --listen-port           | The port on which Restinthemiddle listens for to requests.                                                                                                   | `8000`                                         |
| `headers` (optional)             | -                       | --headers               | A dictionary of HTTP headers.                                                                                                                                | `User-Agent: Rest in the middle logging proxy` |
| `loggingEnabled` (optional)      | `LOGGING_ENABLED`       | --logging-enabled       | Enable logging.                                                                                                                                              | `true`                                         |
| `setRequestId` (optional)        | `SET_REQUEST_ID`        | --set-request-id        | If not already present in the request, add an `X-Request-Id` header with a version 4 UUID.                                                                   | `false`                                        |
| `exclude` (optional)             | `EXCLUDE`               | --exclude               | If the given URL path matches this Regular Expression this request+response will not be logged.                                                              | `""`                                           |
| `logPostBody` (optional)         | `LOG_POST_BODY`         | --log-post-body         | Log the request's body.                                                                                                                                      | `""`                                           |
| `logResponseBody` (optional)     | `LOG_RESPONSE_BODY`     | --log-response-body     | Log the response's body.                                                                                                                                     | `""`                                           |
| `excludePostBody` (optional)     | `EXCLUDE_POST_BODY`     | --exclude-post-body     | If the given URL path matches this Regular Expression the request body (post) is set empty.                                                                  | `""`                                           |
| `excludeResponseBody` (optional) | `EXCLUDE_RESPONSE_BODY` | --exclude-response-body | If the given URL path matches this Regular Expression the response body is set emtpy.                                                                        | `""`                                           |

##### The target host DSN

`schema://username:password@hostname:port/basepath?query`

* `schema` (required) is `http` or `https`
* `username:password@` is optional and will be evaluated only if both values are set.
* `hostname` (required)
* `port` is optional. Standard ports are `80` (http) and `443` (https).
* `basepath` is optional. Will be prefixed to any request URL path pointed at Restinthemiddle. See examples section.
* `query` is optional. If set, `query` will precede the actual request's query.

##### Headers

If a header is defined multiple times, the last assignment wins.

If you need to make a HTTP Basic Authentication **and** need to send another Authorization header at the same time (e.g. a JWT) we have got you covered. Just put the HTTP Basic Auth credentials into the _target host DSN_ string:

```bash
docker run -it --rm -e TARGET_HOST_DSN=http://user:password@www.example.com -p 8000:8000 jdschulze/restinthemiddle:2 --header="Authorization:Bearer ABCD1234"
```

## Examples

### Basic

We want to log HTTP calls against `www.example.com` over an insecure connection.

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=http://www.example.com -p 8000:8000 jdschulze/restinthemiddle:2

# In another terminal window we make the API call against http://www.example.com/api/visitors
curl -i http://127.0.0.1:8000/api/visitors
```

### Advanced

We want to log HTTP calls against `www.example.com:4430` over a TLS connection (`https://…`). The API is protected by HTTP basic auth (username: `user`; password: `pass`). The base path always starts with `api/`.

Note that the base path defined in `TARGET_HOST_DSN` prefixes any subsequent calls!

```bash
# Set up the proxy
docker run -it --rm -e TARGET_HOST_DSN=https://user:pass@www.example.com:4430/api?start=1577833200 -p 8000:8000 jdschulze/restinthemiddle:2

# In another terminal window we make the API call against https://user:pass@www.example.com:4430/api/visitors?start=1577833200
curl -i http://127.0.0.1:8000/visitors
```

### Setting/changing headers

We want to log HTTP calls against `www.example.com` over an insecure connection. Every request has to be enhanced with a custom header `X-App-Version: 3.0.0`. No logging shall take place.

#### With configuration file

##### config.yaml

```yaml
targetHostDsn: http://www.example.com
headers:
    X-App-Version: '3.0.0'
loggingEnabled: false
```

```bash
# Set up the proxy
docker run -it --rm -v ./config.yaml:/restinthemiddle/config.yaml -p 8000:8000 jdschulze/restinthemiddle:2

# In another terminal window we make the API call against http://www.example.com/home
curl -i http://127.0.0.1:8000/home
```

#### With command line arguments

```bash
# Set up the proxy
docker run -it --rm -p 8000:8000 jdschulze/restinthemiddle:2 --target-host-dsn=http://www.example.com --header=x-app-version:3.0.0
```

### Helm Chart for Kubernetes

There is a Helm Chart for Restinthemiddle at [https://github.com/restinthemiddle/helm](https://github.com/restinthemiddle/helm).
You may want to add the restinthemiddle Helm repository:

```shell
helm repo add restinthemiddle https://restinthemiddle.github.io/helm
helm repo update
```

In most cases you will use Restinthemiddle as a [conditional dependency](https://helm.sh/docs/chart_best_practices/dependencies/#conditions-and-tags) in your charts.
