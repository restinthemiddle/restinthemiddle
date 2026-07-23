package transport

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

type stubRoundTripper struct {
	req *http.Request
}

func (s *stubRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	s.req = r
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
		Request:    r,
	}, nil
}

func TestProfilingTransportRespectsLogPostBody(t *testing.T) {
	cfg := &config.TranslatedConfig{
		LogPostBody: false,
	}
	transport := &ProfilingTransport{
		cfg: cfg,
	}
	stub := &stubRoundTripper{}
	transport.roundTripper = stub

	req := httptest.NewRequest("POST", "http://example.com/test", strings.NewReader("secret"))
	req.ContentLength = int64(len("secret"))

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if val, _ := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")).(string); val != "" {
		t.Errorf("Expected requestBodyString to be empty when LogPostBody is false, got %q", val)
	}

	bodyBytes, _ := io.ReadAll(stub.req.Body)
	if string(bodyBytes) != "secret" {
		t.Errorf("Expected upstream to receive original body, got %q", string(bodyBytes))
	}
}

func TestRoundTripLogsPostBody(t *testing.T) {
	cfg := &config.TranslatedConfig{
		LogPostBody: true,
	}
	transport := &ProfilingTransport{
		cfg: cfg,
	}
	stub := &stubRoundTripper{}
	transport.roundTripper = stub

	req := httptest.NewRequest("POST", "http://example.com/test", strings.NewReader("payload"))
	req.ContentLength = int64(len("payload"))

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if val, _ := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")).(string); val != "payload" {
		t.Errorf("Expected requestBodyString to be %q, got %q", "payload", val)
	}

	bodyBytes, _ := io.ReadAll(stub.req.Body)
	if string(bodyBytes) != "payload" {
		t.Errorf("Expected upstream to receive original body, got %q", string(bodyBytes))
	}
}

func TestRoundTripExcludePostBodyRegexpMatch(t *testing.T) {
	cfg := &config.TranslatedConfig{
		LogPostBody:           true,
		ExcludePostBodyRegexp: regexp.MustCompile(`^/secret`),
	}
	transport := &ProfilingTransport{
		cfg: cfg,
	}
	stub := &stubRoundTripper{}
	transport.roundTripper = stub

	req := httptest.NewRequest("POST", "http://example.com/secret/path", strings.NewReader("confidential"))
	req.ContentLength = int64(len("confidential"))

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if val, _ := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")).(string); val != "" {
		t.Errorf("Expected requestBodyString to be empty for excluded path, got %q", val)
	}

	bodyBytes, _ := io.ReadAll(stub.req.Body)
	if string(bodyBytes) != "confidential" {
		t.Errorf("Expected upstream to receive original body, got %q", string(bodyBytes))
	}
}

func TestRoundTripExcludePostBodyRegexpNoMatch(t *testing.T) {
	cfg := &config.TranslatedConfig{
		LogPostBody:           true,
		ExcludePostBodyRegexp: regexp.MustCompile(`^/secret`),
	}
	transport := &ProfilingTransport{
		cfg: cfg,
	}
	stub := &stubRoundTripper{}
	transport.roundTripper = stub

	req := httptest.NewRequest("POST", "http://example.com/public", strings.NewReader("visible"))
	req.ContentLength = int64(len("visible"))

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if val, _ := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")).(string); val != "visible" {
		t.Errorf("Expected requestBodyString to be %q, got %q", "visible", val)
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("read failure")
}

func TestRoundTripPostBodyReadError(t *testing.T) {
	cfg := &config.TranslatedConfig{
		LogPostBody: true,
	}
	transport := &ProfilingTransport{
		cfg: cfg,
	}
	stub := &stubRoundTripper{}
	transport.roundTripper = stub

	req := httptest.NewRequest("POST", "http://example.com/test", io.NopCloser(&errorReader{}))
	req.ContentLength = 42

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if val, _ := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")).(string); val != "" {
		t.Errorf("Expected requestBodyString to be empty on body read error, got %q", val)
	}
}

func TestRoundTripTLSHandshakeTiming(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("tls response")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer testServer.Close()

	targetURL, _ := url.Parse(testServer.URL)
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: false,
		LogPostBody:    false,
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	// Trust the test server's certificate.
	transport.roundTripper.(*http.Transport).TLSClientConfig = testServer.Client().Transport.(*http.Transport).TLSClientConfig

	req, err := http.NewRequest("GET", testServer.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	timing, ok := resp.Request.Context().Value(ProfilingContextKey("timing")).(*HTTPTiming)
	if !ok {
		t.Fatal("Expected timing to be stored in request context")
	}

	if timing.TLSHandshakeStart.IsZero() {
		t.Error("Expected TLSHandshakeStart to be set for HTTPS request")
	}
	if timing.TLSHandshakeDone.IsZero() {
		t.Error("Expected TLSHandshakeDone to be set for HTTPS request")
	}
	if timing.TLSHandshakeDone.Before(timing.TLSHandshakeStart) {
		t.Error("Expected TLSHandshakeDone to be at or after TLSHandshakeStart")
	}
}
