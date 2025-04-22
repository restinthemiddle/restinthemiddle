package transport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

func TestNewProfilingTransport(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}

	transport := NewProfilingTransport(cfg)
	if transport == nil {
		t.Error("Transport sollte nicht nil sein")
	}
	if transport.cfg != cfg {
		t.Error("Konfiguration wurde nicht korrekt gesetzt")
	}
}

func TestRoundTrip(t *testing.T) {
	// Setup
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}
	transport := NewProfilingTransport(cfg)

	// Test Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test Request
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)

	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Erwarteter Status 200, got %d", resp.StatusCode)
	}

	// Überprüfe Timing-Informationen
	timing := resp.Request.Context().Value(ProfilingContextKey("timing")).(*HTTPTiming)
	if timing.GetConn.IsZero() {
		t.Error("GetConn Zeit wurde nicht gesetzt")
	}
	if timing.GotConn.IsZero() {
		t.Error("GotConn Zeit wurde nicht gesetzt")
	}
	if timing.GotFirstResponseByte.IsZero() {
		t.Error("GotFirstResponseByte Zeit wurde nicht gesetzt")
	}
}

func TestDial(t *testing.T) {
	// Setup
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}
	transport := NewProfilingTransport(cfg)

	// Test Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test Connection
	conn, err := transport.dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	defer conn.Close()

	if transport.connectionStart.IsZero() {
		t.Error("connectionStart Zeit wurde nicht gesetzt")
	}
	if transport.connectionEnd.IsZero() {
		t.Error("connectionEnd Zeit wurde nicht gesetzt")
	}
	if transport.connectionEnd.Before(transport.connectionStart) {
		t.Error("connectionEnd sollte nach connectionStart sein")
	}
}
