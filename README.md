# restinthemiddle

This Go program acts as a lightweight HTTP logging proxy for developing and staging environments. If you put it between an API client and the API you can easily monitor requests and responses.

## Installation

### Docker (recommended)

```bash
docker pull jdschulze/restinthemiddle
```

### Build the Docker image yourself

```bash
./build
```

### Build the binary yourself

```bash
go build -o restinthemiddle
```

## Example

```bash
docker run -e TARGET_HOST_DSN=https://www.example.com:4430 -p 8000:8000 jdschulze/restinthemiddle

curl -i http://127.0.0.1:8000/check
```
