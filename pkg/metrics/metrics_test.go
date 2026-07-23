package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/restinthemiddle/restinthemiddle/internal/version"
)

func TestInit(t *testing.T) {
	Init()

	// Check that uptime is being tracked
	if ProcessUptimeSeconds == nil {
		t.Fatal("ProcessUptimeSeconds should be initialized")
	}

	// Check that build info is set
	if BuildInfo == nil {
		t.Fatal("BuildInfo should be initialized")
	}

	// Build info gauge must carry the injected build metadata with value 1.
	buildInfo := testutil.ToFloat64(BuildInfo.WithLabelValues(version.Version, version.BuildDate, version.GitCommit))
	if buildInfo != 1.0 {
		t.Errorf("Expected build_info gauge value 1.0, got %f", buildInfo)
	}

	// The uptime goroutine ticks once per second; wait for the first update.
	deadline := time.Now().Add(3 * time.Second)
	for testutil.ToFloat64(ProcessUptimeSeconds) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("ProcessUptimeSeconds was not updated within 3s")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if uptime := testutil.ToFloat64(ProcessUptimeSeconds); uptime <= 0 {
		t.Errorf("Expected positive uptime, got %f", uptime)
	}
}

// histogramSnapshot returns the sample count and sum of a histogram child.
func histogramSnapshot(t *testing.T, o prometheus.Observer) (uint64, float64) {
	t.Helper()
	m := &dto.Metric{}
	metric, ok := o.(prometheus.Metric)
	if !ok {
		t.Fatal("observer does not implement prometheus.Metric")
	}
	if err := metric.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetHistogram().GetSampleCount(), m.GetHistogram().GetSampleSum()
}

func TestHistogramsRecordObservations(t *testing.T) {
	cases := []struct {
		name     string
		observer prometheus.Observer
		value    float64
	}{
		{"http_request_duration_seconds", HTTPRequestDuration.WithLabelValues("PUT", "204", "histo.example.com"), 0.25},
		{"http_request_size_bytes", HTTPRequestSizeBytes.WithLabelValues("PUT", "histo.example.com"), 512},
		{"http_response_size_bytes", HTTPResponseSizeBytes.WithLabelValues("PUT", "204", "histo.example.com"), 4096},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			countBefore, sumBefore := histogramSnapshot(t, tc.observer)
			tc.observer.Observe(tc.value)
			countAfter, sumAfter := histogramSnapshot(t, tc.observer)

			if countAfter != countBefore+1 {
				t.Errorf("Expected sample count %d, got %d", countBefore+1, countAfter)
			}
			if sumAfter != sumBefore+tc.value {
				t.Errorf("Expected sample sum %f, got %f", sumBefore+tc.value, sumAfter)
			}
		})
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
