# Prometheus Metrics

Restinthemiddle exposes Prometheus metrics on a separate HTTP endpoint to monitor proxy performance and health.

## Configuration

Metrics can be configured via command-line flags, environment variables, or YAML configuration:

### Command-line flags

```shell
./restinthemiddle \
  --target-host-dsn http://api.example.com \
  --metrics-enabled=true \
  --metrics-port=9090
```

### Environment variables

```shell
export METRICS_ENABLED=true
export METRICS_PORT=9090
./restinthemiddle --target-host-dsn http://api.example.com
```

### YAML configuration

```yaml
targetHostDsn: http://api.example.com
metricsEnabled: true
metricsPort: "9090"
```

## Default Values

- **metricsEnabled**: `true` (enabled by default)
- **metricsPort**: `9090`

## Accessing Metrics

Once the proxy is running, metrics are available at:

```text
http://<LISTEN_IP>:<METRICS_PORT>/metrics
```

Example:

```shell
curl http://127.0.0.1:9090/metrics
```

## Available Metrics

### Build Information

```promql
build_info
````

- **Type**: Gauge (always 1)
- **Labels**: `version`, `build_date`, `git_commit`
- **Description**: Build information including version, build date, and git commit
- **Example**:

  ```promql
  build_info{version="1.0.0",build_date="2024-01-15T10:30:00Z",git_commit="abc123"} 1
  ```

### Process Uptime

```promql
process_uptime_seconds
```

- **Type**: Gauge
- **Description**: Time in seconds since the process started

### HTTP Request Metrics

#### Total Requests

```promql
http_requests_total{method, status_code, target_host}
```

- **Type**: Counter
- **Labels**: `method` (GET, POST, etc.), `status_code` (200, 404, etc.), `target_host`
- **Description**: Total number of HTTP requests proxied

#### Request Duration

```promql
http_request_duration_seconds{method, status_code, target_host}
```

- **Type**: Histogram
- **Labels**: `method`, `status_code`, `target_host`
- **Buckets**: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10 (seconds)
- **Description**: HTTP request duration in seconds

#### Request Size

```promql
http_request_size_bytes{method, target_host}
```

- **Type**: Histogram
- **Labels**: `method`, `target_host`
- **Buckets**: 100, 1000, 10000, 100000, 1000000, 10000000 (bytes)
- **Description**: HTTP request size in bytes

#### Response Size

```promql
http_response_size_bytes{method, status_code, target_host}
```

- **Type**: Histogram
- **Labels**: `method`, `status_code`, `target_host`
- **Buckets**: 100, 1000, 10000, 100000, 1000000, 10000000 (bytes)
- **Description**: HTTP response size in bytes

#### In-Flight Requests

```promql
http_requests_in_flight{target_host}
```

- **Type**: Gauge
- **Labels**: `target_host`
- **Description**: Number of HTTP requests currently being processed

### Error Metrics

#### Upstream Errors

```promql
http_upstream_errors_total{target_host, error_type}
```

- **Type**: Counter
- **Labels**: `target_host`, `error_type` (connection_refused, timeout, dns_error, etc.)
- **Description**: Total number of upstream connection errors

#### Proxy Failures

```promql
http_proxy_failures_total{target_host, reason}
```

- **Type**: Counter
- **Labels**: `target_host`, `reason` (connection_refused, timeout, etc.)
- **Description**: Total number of failed proxy attempts

#### Proxy Timeouts

```promql
http_proxy_timeouts_total{timeout_type, target_host}
```

- **Type**: Counter
- **Labels**: `timeout_type` (read, write, idle), `target_host`
- **Description**: Total number of proxy timeouts

### Go Runtime Metrics

Standard Go runtime metrics are also exported automatically, including:

- `go_goroutines` - Number of goroutines
- `go_memstats_*` - Memory statistics
- `go_gc_*` - Garbage collection statistics
- `process_*` - Process statistics (CPU, memory, file descriptors)

## Example Queries

### Request rate by status code

```promql
sum(rate(http_requests_total[5m])) by (status_code)
```

### 95th percentile request duration

```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Error rate

```promql
sum(rate(http_upstream_errors_total[5m])) by (error_type)
```

### Current in-flight requests

```promql
sum(http_requests_in_flight) by (target_host)
```

## Grafana Dashboard

You can create a Grafana dashboard using these metrics. Example panels:

1. **Request Rate**: `sum(rate(http_requests_total[5m]))`
2. **Error Rate**: `sum(rate(http_upstream_errors_total[5m]))`
3. **Latency**: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
4. **In-Flight Requests**: `sum(http_requests_in_flight)`
5. **Uptime**: `process_uptime_seconds`

## Disabling Metrics

The `/metrics` endpoint is exposed by default. To disable the metrics endpoint:

```shell
./restinthemiddle --target-host-dsn http://api.example.com --metrics-enabled=false
```

Or via environment variable:

```shell
export METRICS_ENABLED=false
```

Or in YAML:

```yaml
targetHostDsn: http://api.example.com
metricsEnabled: false
```
