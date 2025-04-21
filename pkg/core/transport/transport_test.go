package transport

import (
	"net/http/httptest"
	"regexp"
	"testing"

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

func TestDial(t *testing.T) {
	cfg := &config.TranslatedConfig{
		ExcludePostBodyRegexp: regexp.MustCompile(""),
	}
	transport, err := NewProfilingTransport(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	conn, err := transport.dial("tcp", "example.com:80")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if conn == nil {
		t.Error("Connection should not be nil")
	}
}
