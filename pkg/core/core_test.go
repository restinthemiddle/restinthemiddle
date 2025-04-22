package core

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

// MockHTTPServer is a mock implementation of the HTTPServer interface
type MockHTTPServer struct {
	handler http.Handler
}

func (s *MockHTTPServer) ListenAndServe(addr string, handler http.Handler) error {
	s.handler = handler
	return nil
}

// MockWriter is a mock implementation of the Writer interface
type MockWriter struct {
	lastResponse *http.Response
}

func (w *MockWriter) LogResponse(response *http.Response) error {
	w.lastResponse = response
	return nil
}

func TestRun(t *testing.T) {
	// Setup
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		ListenIP:                  "127.0.0.1",
		ListenPort:                "8080",
		TargetURL:                 targetURL,
		LoggingEnabled:            true,
		SetRequestID:              true,
		Headers:                   make(map[string]string),
		LogPostBody:               true,
		LogResponseBody:           true,
		ExcludeRegexp:             regexp.MustCompile(""),
		ExcludePostBodyRegexp:     regexp.MustCompile(""),
		ExcludeResponseBodyRegexp: regexp.MustCompile(""),
	}
	mockServer := &MockHTTPServer{}
	mockWriter := &MockWriter{}

	// Run
	Run(cfg, mockWriter, mockServer)

	// Test
	if mockServer.handler == nil {
		t.Error("Handler was not set")
	}
}
