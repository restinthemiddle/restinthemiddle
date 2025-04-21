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

	_, err := NewServer(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewServerWithNilConfig(t *testing.T) {
	server, err := NewServer(nil)
	if err == nil {
		t.Error("Expected error for nil configuration")
	}
	if server != nil {
		t.Error("Server should be nil for nil configuration")
	}
}

func TestNewServerWithNilTargetURL(t *testing.T) {
	cfg := &config.TranslatedConfig{
		TargetURL: nil,
	}

	server, err := NewServer(cfg)
	if err == nil {
		t.Error("Expected error for nil target URL")
	}
	if server != nil {
		t.Error("Server should be nil for nil target URL")
	}
}

func TestServeHTTP(t *testing.T) {
	// Setup
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

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
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSetModifyResponse(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	modifyCalled := false
	server.SetModifyResponse(func(resp *http.Response) error {
		modifyCalled = true
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if !modifyCalled {
		t.Error("ModifyResponse was not called")
	}
}
