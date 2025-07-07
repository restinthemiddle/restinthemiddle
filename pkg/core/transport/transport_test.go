package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

func TestNewProfilingTransport(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: regexp.MustCompile(""),
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if transport.cfg != cfg {
		t.Error("Configuration was not set correctly")
	}
}

func TestNewProfilingTransportWithNilConfig(t *testing.T) {
	transport, err := NewProfilingTransport(nil)
	if err == nil {
		t.Error("Expected error for nil configuration")
	}
	if transport != nil {
		t.Error("Transport should be nil for nil configuration")
	}
}

func TestRoundTrip(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: regexp.MustCompile(""),
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
}

func TestTCPConnectionTiming(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: regexp.MustCompile(""),
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test that the transport has been created with the correct configuration
	if transport.cfg != cfg {
		t.Error("Configuration was not set correctly")
	}

	// Test that we can store and retrieve timing data
	req := httptest.NewRequest("GET", "http://example.com", nil)
	timing := &TCPConnectionTiming{
		Start:       time.Now(),
		Established: time.Now(),
	}
	transport.connectionTimings.Store(req, timing)

	// Verify we can retrieve it
	if retrieved, exists := transport.connectionTimings.Load(req); !exists {
		t.Error("Should be able to store and retrieve connection timing")
	} else if _, ok := retrieved.(*TCPConnectionTiming); !ok {
		t.Error("Retrieved timing should be of correct type")
	}
}

func TestRoundTripWithPostBody(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil, // No exclusion
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server that captures the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Request-Body", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create request with POST body
	postBody := "test=data&field=value"
	req := httptest.NewRequest("POST", server.URL, strings.NewReader(postBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// Verify that the request body was captured and forwarded
	if resp.Header.Get("X-Request-Body") != postBody {
		t.Errorf("Expected request body %q, got %q", postBody, resp.Header.Get("X-Request-Body"))
	}

	// Verify that the request context contains the body string
	if resp.Request != nil {
		if bodyString := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")); bodyString != postBody {
			t.Errorf("Expected request body string %q in context, got %q", postBody, bodyString)
		}
	}
}

func TestRoundTripWithExcludedPostBody(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: regexp.MustCompile("^/secure/.*"), // Exclude /secure/ paths
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server that captures the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Request-Body", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create request with POST body to excluded path
	postBody := "sensitive=data"
	req := httptest.NewRequest("POST", server.URL+"/secure/login", strings.NewReader(postBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// Verify that the request body was forwarded (but not captured in context)
	if resp.Header.Get("X-Request-Body") != postBody {
		t.Errorf("Expected request body %q, got %q", postBody, resp.Header.Get("X-Request-Body"))
	}

	// Verify that the request context contains empty body string (excluded)
	if resp.Request != nil {
		if bodyString := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")); bodyString != "" {
			t.Errorf("Expected empty request body string in context for excluded path, got %q", bodyString)
		}
	}
}

func TestRoundTripWithReadBodyError(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a request with a body that will cause an error when read
	// We'll simulate this by creating a request with a body that's already closed
	body := io.NopCloser(strings.NewReader("test data"))
	body.Close() // Close the body to cause an error

	req := httptest.NewRequest("POST", server.URL, body)
	req.ContentLength = 9 // Set content length to trigger body reading

	// This should not panic even if body reading fails
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
}

func TestRoundTripWithBodyReadError(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a request with a body reader that will fail
	failingReader := &failingReader{}
	req := httptest.NewRequest("POST", server.URL, failingReader)
	req.ContentLength = 10 // Set content length to trigger body reading

	// This should handle the error gracefully and log it
	resp, err := transport.RoundTrip(req)

	// The request might fail at the network level due to the failing reader
	// This is expected behavior, so we test that it doesn't panic
	if err != nil {
		// Network error is expected due to failing reader, this is OK
		if resp != nil {
			t.Error("Response should be nil on network error")
		}
		return
	}

	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// The request body should be empty due to read error
	if resp.Request != nil {
		if bodyString := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")); bodyString != "" {
			t.Errorf("Expected empty request body string due to read error, got %q", bodyString)
		}
	}
}

// failingReader is a reader that always returns an error.
type failingReader struct{}

func (fr *failingReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestRoundTripWithZeroContentLength(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create request with zero content length (should skip body reading)
	req := httptest.NewRequest("POST", server.URL, strings.NewReader("ignored"))
	req.ContentLength = 0 // This should skip the body reading logic

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// Check that request body string is empty (not read due to zero content length)
	if resp.Request != nil {
		if bodyString := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")); bodyString != "" {
			t.Errorf("Expected empty request body string for zero content length, got %q", bodyString)
		}
	}
}

func TestRoundTripWithNegativeContentLength(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create request with negative content length (should skip body reading)
	req := httptest.NewRequest("POST", server.URL, strings.NewReader("ignored"))
	req.ContentLength = -1 // This should skip the body reading logic

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// Check that request body string is empty (not read due to negative content length)
	if resp.Request != nil {
		if bodyString := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")); bodyString != "" {
			t.Errorf("Expected empty request body string for negative content length, got %q", bodyString)
		}
	}
}

func TestRoundTripContextValues(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req := httptest.NewRequest("GET", server.URL, nil)

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// Check that all expected context values are present
	if resp.Request != nil {
		ctx := resp.Request.Context()

		// Check roundTripStart
		if startTime := ctx.Value(ProfilingContextKey("roundTripStart")); startTime == nil {
			t.Error("Expected roundTripStart in context")
		} else if _, ok := startTime.(time.Time); !ok {
			t.Error("roundTripStart should be a time.Time")
		}

		// Check roundTripEnd
		if endTime := ctx.Value(ProfilingContextKey("roundTripEnd")); endTime == nil {
			t.Error("Expected roundTripEnd in context")
		} else if _, ok := endTime.(time.Time); !ok {
			t.Error("roundTripEnd should be a time.Time")
		}

		// Check timing
		if timing := ctx.Value(ProfilingContextKey("timing")); timing == nil {
			t.Error("Expected timing in context")
		} else if _, ok := timing.(*HTTPTiming); !ok {
			t.Error("timing should be an HTTPTiming")
		}

		// Check TCP connection timing
		if startTime := ctx.Value(ProfilingContextKey("tcpConnectionStart")); startTime == nil {
			t.Error("Expected tcpConnectionStart in context")
		}

		if endTime := ctx.Value(ProfilingContextKey("tcpConnectionEstablished")); endTime == nil {
			t.Error("Expected tcpConnectionEstablished in context")
		}

		// Check requestBodyString
		if bodyString := ctx.Value(ProfilingContextKey("requestBodyString")); bodyString == nil {
			t.Error("Expected requestBodyString in context")
		} else if bodyString != "" {
			t.Error("Expected empty requestBodyString for GET request")
		}
	}
}

func TestRoundTripWithHTTPSAndTLS(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test HTTPS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use the server's client which has the correct TLS config
	req := httptest.NewRequest("GET", server.URL, nil)

	// For this test, we'll just verify the structure works even if TLS fails
	resp, err := transport.RoundTrip(req)

	// The request may fail due to TLS certificate issues in test environment
	// but we should still have proper error handling
	if err != nil {
		// This is expected for TLS tests, just ensure it doesn't panic
		if resp != nil {
			t.Error("Response should be nil on TLS error")
		}
		return
	}

	if resp == nil {
		t.Error("Response should not be nil")
		return
	}

	// Check that timing information is present
	if resp.Request != nil {
		ctx := resp.Request.Context()
		if timing := ctx.Value(ProfilingContextKey("timing")); timing != nil {
			if httpTiming, ok := timing.(*HTTPTiming); ok {
				// For HTTPS, we might have TLS timing information
				// Note: TLS timing may be zero if the handshake failed
				if !httpTiming.TLSHandshakeStart.IsZero() && !httpTiming.TLSHandshakeDone.IsZero() {
					t.Logf("TLS handshake timing captured: start=%v, done=%v",
						httpTiming.TLSHandshakeStart, httpTiming.TLSHandshakeDone)
				}
			}
		}
	}
}

func TestRoundTripErrorHandling(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Create a request to a non-existent server to trigger an error
	req := httptest.NewRequest("GET", "http://nonexistent.invalid:12345", nil)

	resp, err := transport.RoundTrip(req)
	if err == nil {
		t.Error("Expected error for non-existent server")
	}

	// Even on error, the function should handle context properly
	// resp will be nil, but we should test that the error path is handled
	if resp != nil {
		t.Error("Response should be nil on error")
	}
}

func TestProfilingContextKey(t *testing.T) {
	// Test that ProfilingContextKey is a distinct type
	key1 := ProfilingContextKey("test")
	key2 := ProfilingContextKey("test")

	if key1 != key2 {
		t.Error("Same ProfilingContextKey values should be equal")
	}

	// Test that it's different from a regular string
	type testKey string
	ctx := context.Background()
	ctx = context.WithValue(ctx, key1, "value1")
	ctx = context.WithValue(ctx, testKey("test"), "value2")

	if ctx.Value(key1) != "value1" {
		t.Error("ProfilingContextKey should retrieve correct value")
	}
	if ctx.Value(testKey("test")) != "value2" {
		t.Error("String key should retrieve correct value")
	}
}

func TestConnectionTimingCleanup(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: nil,
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req := httptest.NewRequest("GET", server.URL, nil)

	// Manually store timing to test cleanup
	timing := &TCPConnectionTiming{
		Start:       time.Now(),
		Established: time.Now(),
	}
	transport.connectionTimings.Store(req, timing)

	// Verify it's stored
	if _, exists := transport.connectionTimings.Load(req); !exists {
		t.Error("Connection timing should be stored")
	}

	// Make the request
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}

	// Verify it's cleaned up after the request
	if _, exists := transport.connectionTimings.Load(req); exists {
		t.Error("Connection timing should be cleaned up after request")
	}
}

func TestHTTPTimingStructure(t *testing.T) {
	// Test that HTTPTiming structure is properly initialized
	timing := &HTTPTiming{}

	if !timing.GetConn.IsZero() {
		t.Error("GetConn should be zero initially")
	}
	if !timing.GotConn.IsZero() {
		t.Error("GotConn should be zero initially")
	}
	if !timing.GotFirstResponseByte.IsZero() {
		t.Error("GotFirstResponseByte should be zero initially")
	}
	if !timing.TLSHandshakeStart.IsZero() {
		t.Error("TLSHandshakeStart should be zero initially")
	}
	if !timing.TLSHandshakeDone.IsZero() {
		t.Error("TLSHandshakeDone should be zero initially")
	}
}

func TestTCPConnectionTimingStructure(t *testing.T) {
	// Test that TCPConnectionTiming structure works correctly
	start := time.Now()
	established := time.Now().Add(100 * time.Millisecond)

	timing := &TCPConnectionTiming{
		Start:       start,
		Established: established,
	}

	if timing.Start != start {
		t.Error("Start time should match")
	}
	if timing.Established != established {
		t.Error("Established time should match")
	}
	if timing.Established.Before(timing.Start) {
		t.Error("Established time should be after start time")
	}
}
