package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

func TestClassifyErrorNil(t *testing.T) {
	if result := classifyError(nil); result != "none" {
		t.Errorf("classifyError(nil) = %s, expected none", result)
	}
}

func TestClassifyErrorConnectionRefused(t *testing.T) {
	err := errors.New("dial tcp: connection refused")
	if result := classifyError(err); result != "connection_refused" {
		t.Errorf("classifyError(%v) = %s, expected connection_refused", err, result)
	}
}

func TestClassifyErrorTimeout(t *testing.T) {
	err := errors.New("dial tcp: i/o timeout")
	if result := classifyError(err); result != "timeout" {
		t.Errorf("classifyError(%v) = %s, expected timeout", err, result)
	}
}

func TestClassifyErrorDNS(t *testing.T) {
	err := errors.New("dial tcp: lookup example.invalid: no such host")
	if result := classifyError(err); result != "dns_error" {
		t.Errorf("classifyError(%v) = %s, expected dns_error", err, result)
	}
}

func TestClassifyErrorEOF(t *testing.T) {
	err := errors.New("read tcp: EOF")
	if result := classifyError(err); result != "eof" {
		t.Errorf("classifyError(%v) = %s, expected eof", err, result)
	}
}

func TestClassifyErrorTLS(t *testing.T) {
	err := errors.New("TLS handshake failed")
	if result := classifyError(err); result != "tls_error" {
		t.Errorf("classifyError(%v) = %s, expected tls_error", err, result)
	}
}

func TestClassifyErrorContextCanceled(t *testing.T) {
	err := errors.New("net/http: request canceled: context canceled")
	if result := classifyError(err); result != "context_canceled" {
		t.Errorf("classifyError(%v) = %s, expected context_canceled", err, result)
	}
}

func TestClassifyErrorContextDeadline(t *testing.T) {
	err := errors.New("net/http: request canceled: context deadline exceeded")
	if result := classifyError(err); result != "context_deadline_exceeded" {
		t.Errorf("classifyError(%v) = %s, expected context_deadline_exceeded", err, result)
	}
}

func TestClassifyErrorOther(t *testing.T) {
	err := errors.New("some other error")
	if result := classifyError(err); result != "other" {
		t.Errorf("classifyError(%v) = %s, expected other", err, result)
	}
}

func TestNewProfilingTransport(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
		return
	}

	if transport == nil {
		t.Error("Expected transport to be created")
		return
	}

	if transport.cfg != cfg {
		t.Error("Expected transport to store config reference")
	}

	if transport.roundTripper == nil {
		t.Error("Expected roundTripper to be initialized")
	}
}

func TestNewProfilingTransportWithNilConfig(t *testing.T) {
	transport, err := NewProfilingTransport(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	if transport != nil {
		t.Error("Expected nil transport for nil config")
	}
}

func TestRoundTripWithMetricsEnabled(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test response")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer testServer.Close()

	targetURL, _ := url.Parse(testServer.URL)
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
		LogPostBody:    false,
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

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
}

func TestRoundTripWithError(t *testing.T) {
	targetURL, _ := url.Parse("http://invalid.local.invalid:99999")
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
		LogPostBody:    false,
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	req, err := http.NewRequest("GET", "http://invalid.local.invalid:99999/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Error("Expected RoundTrip to fail with invalid host")
	}

	if err != nil {
		errorType := classifyError(err)
		if errorType == "" {
			t.Error("Expected error to be classified")
		}
	}
}

func TestRoundTripWithoutMetrics(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test response")); err != nil {
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
}

func TestRoundTripErrorConnectionRefused(t *testing.T) {
	targetURL, _ := url.Parse("http://127.0.0.1:1")
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
		LogPostBody:    false,
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	req, _ := http.NewRequest("GET", "http://127.0.0.1:1", nil)
	resp, err := transport.RoundTrip(req)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Error("Expected error, got nil")
	} else if classifyError(err) == "" {
		t.Error("Expected error to be classified")
	}
}

func TestRoundTripErrorInvalidHost(t *testing.T) {
	targetURL, _ := url.Parse("http://invalid.local.test:99999")
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
		LogPostBody:    false,
	}

	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	req, _ := http.NewRequest("GET", "http://invalid.local.test:99999", nil)
	resp, err := transport.RoundTrip(req)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Error("Expected error, got nil")
	}
}
