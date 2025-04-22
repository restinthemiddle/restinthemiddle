package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

func TestNewServer(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}

	server := NewServer(cfg)
	if server == nil {
		t.Error("Server sollte nicht nil sein")
	}
	if server.cfg != cfg {
		t.Error("Konfiguration wurde nicht korrekt gesetzt")
	}
}

func TestServeHTTP(t *testing.T) {
	// Setup
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}
	server := NewServer(cfg)

	// Test Request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Mock Response
	server.SetModifyResponse(func(resp *http.Response) error {
		resp.StatusCode = http.StatusOK
		return nil
	})

	// Test
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Erwarteter Status 200, got %d", w.Code)
	}
}

func TestSetModifyResponse(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}
	server := NewServer(cfg)

	modifyCalled := false
	server.SetModifyResponse(func(resp *http.Response) error {
		modifyCalled = true
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if !modifyCalled {
		t.Error("ModifyResponse wurde nicht aufgerufen")
	}
}
