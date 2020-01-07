# restinthemiddle

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
docker run -e TARGET_HOST_DSN=https://www.example.com:8088 -p 8000:8000 jdschulze/restinthemiddle
```
