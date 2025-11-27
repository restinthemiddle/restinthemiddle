package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestInit(t *testing.T) {
	Init()

	// Wait a moment for the uptime goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Check that uptime is being tracked
	if ProcessUptimeSeconds == nil {
		t.Fatal("ProcessUptimeSeconds should be initialized")
	}

	// Check that build info is set
	if BuildInfo == nil {
		t.Fatal("BuildInfo should be initialized")
	}
}

func TestHTTPRequestsTotal(t *testing.T) {
	counter := HTTPRequestsTotal.WithLabelValues("GET", "200", "example.com")
	counter.Inc()

	value := testutil.ToFloat64(counter)
	if value < 1.0 {
		t.Errorf("Expected counter value >= 1.0, got %f", value)
	}
}

func TestHTTPRequestsInFlight(t *testing.T) {
	gauge := HTTPRequestsInFlight.WithLabelValues("example.com")
	initialValue := testutil.ToFloat64(gauge)

	gauge.Inc()
	value := testutil.ToFloat64(gauge)
	if value != initialValue+1.0 {
		t.Errorf("Expected gauge value %f, got %f", initialValue+1.0, value)
	}

	gauge.Dec()
	value = testutil.ToFloat64(gauge)
	if value != initialValue {
		t.Errorf("Expected gauge value %f, got %f", initialValue, value)
	}
}

func TestHTTPUpstreamErrorsTotal(t *testing.T) {
	counter := HTTPUpstreamErrorsTotal.WithLabelValues("example.com", "connection_refused")
	counter.Inc()

	value := testutil.ToFloat64(counter)
	if value < 1.0 {
		t.Errorf("Expected counter value >= 1.0, got %f", value)
	}
}

func TestHTTPProxyTimeoutsTotal(t *testing.T) {
	counter := HTTPProxyTimeoutsTotal.WithLabelValues("read", "example.com")
	counter.Inc()

	value := testutil.ToFloat64(counter)
	if value < 1.0 {
		t.Errorf("Expected counter value >= 1.0, got %f", value)
	}
}

func TestHTTPProxyFailuresTotal(t *testing.T) {
	counter := HTTPProxyFailuresTotal.WithLabelValues("example.com", "bad_gateway")
	counter.Inc()

	value := testutil.ToFloat64(counter)
	if value < 1.0 {
		t.Errorf("Expected counter value >= 1.0, got %f", value)
	}
}

func TestHTTPRequestDuration(t *testing.T) {
	// Just test that we can observe a value without errors
	HTTPRequestDuration.WithLabelValues("POST", "201", "example.com").Observe(0.5)
}

func TestHTTPRequestSizeBytes(t *testing.T) {
	// Just test that we can observe a value without errors
	HTTPRequestSizeBytes.WithLabelValues("GET", "example.com").Observe(1024)
}

func TestHTTPResponseSizeBytes(t *testing.T) {
	// Just test that we can observe a value without errors
	HTTPResponseSizeBytes.WithLabelValues("GET", "200", "example.com").Observe(2048)
}
