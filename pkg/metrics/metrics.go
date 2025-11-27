package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/restinthemiddle/restinthemiddle/internal/version"
)

var (
	// BuildInfo exposes version, build date, and git commit.
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "build_info",
			Help: "Build information including version, build date, and git commit",
		},
		[]string{"version", "build_date", "git_commit"},
	)

	// ProcessUptimeSeconds tracks the uptime of the process in seconds.
	ProcessUptimeSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "process_uptime_seconds",
			Help: "Process uptime in seconds",
		},
	)

	// HTTPUpstreamErrorsTotal counts upstream connection errors.
	HTTPUpstreamErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_upstream_errors_total",
			Help: "Total number of upstream connection errors",
		},
		[]string{"target_host", "error_type"},
	)

	// HTTPProxyTimeoutsTotal counts proxy timeouts by type.
	HTTPProxyTimeoutsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_proxy_timeouts_total",
			Help: "Total number of proxy timeouts",
		},
		[]string{"timeout_type", "target_host"},
	)

	// HTTPProxyFailuresTotal counts failed proxy attempts.
	HTTPProxyFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_proxy_failures_total",
			Help: "Total number of failed proxy attempts",
		},
		[]string{"target_host", "reason"},
	)

	// HTTPRequestsTotal counts total HTTP requests proxied.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests proxied",
		},
		[]string{"method", "status_code", "target_host"},
	)

	// HTTPRequestDuration measures request duration in seconds.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "status_code", "target_host"},
	)

	// HTTPRequestSizeBytes measures request size in bytes.
	HTTPRequestSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "target_host"},
	)

	// HTTPResponseSizeBytes measures response size in bytes.
	HTTPResponseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "status_code", "target_host"},
	)

	// HTTPRequestsInFlight tracks active in-flight requests.
	HTTPRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"target_host"},
	)

	startTime time.Time
)

// Init initializes the metrics with build information and starts the uptime tracker.
func Init() {
	startTime = time.Now()

	// Set build info as a constant gauge with value 1
	BuildInfo.WithLabelValues(
		version.Version,
		version.BuildDate,
		version.GitCommit,
	).Set(1)

	// Start a goroutine to update uptime periodically
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ProcessUptimeSeconds.Set(time.Since(startTime).Seconds())
		}
	}()
}
