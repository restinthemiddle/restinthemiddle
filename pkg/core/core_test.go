package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

// MockHTTPServer is a mock implementation of the HTTPServer interface.
type MockHTTPServer struct {
	handler     http.Handler
	lastAddress string
}

func (s *MockHTTPServer) ListenAndServe(addr string, handler http.Handler) error {
	s.handler = handler
	s.lastAddress = addr
	return nil
}

// MockWriter is a mock implementation of the Writer interface.
type MockWriter struct {
	lastResponse *http.Response
}

func (w *MockWriter) LogResponse(response *http.Response) error {
	w.lastResponse = response
	return nil
}

// MockWriterWithError is a mock implementation that returns an error.
type MockWriterWithError struct{}

func (w *MockWriterWithError) LogResponse(response *http.Response) error {
	return &MockError{message: "mock writer error"}
}

// MockError is a simple error implementation.
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// MockHTTPServerWithError is a mock that returns an error.
type MockHTTPServerWithError struct{}

func (s *MockHTTPServerWithError) ListenAndServe(addr string, handler http.Handler) error {
	return &MockError{message: "mock server error"}
}

func testConfig() *config.TranslatedConfig {
	targetURL, _ := url.Parse("http://example.com")
	return &config.TranslatedConfig{
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
}

func TestRun(t *testing.T) {
	mockServer := &MockHTTPServer{}
	mockWriter := &MockWriter{}

	Run(testConfig(), mockWriter, mockServer)

	if mockServer.handler == nil {
		t.Error("Handler was not set")
	}
}

func TestNewProxy(t *testing.T) {
	cfg := testConfig()
	mockWriter := &MockWriter{}

	p, err := NewProxy(cfg, mockWriter)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if p.cfg != cfg {
		t.Error("cfg was not set correctly")
	}
	if p.writer != Writer(mockWriter) {
		t.Error("writer was not set correctly")
	}
	if p.proxyServer == nil {
		t.Error("proxyServer was not set")
	}
}

func TestNewProxyWithProxyCreationError(t *testing.T) {
	cfg := testConfig()
	cfg.TargetURL = nil // causes proxy.NewServer to fail

	_, err := NewProxy(cfg, &MockWriter{})
	if err == nil {
		t.Fatal("Expected error when proxy creation fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create proxy server") {
		t.Errorf("Expected proxy creation error, got: %v", err)
	}
}

func TestHandleRequest(t *testing.T) {
	p, err := NewProxy(testConfig(), &MockWriter{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	p.Handler().ServeHTTP(w, req)

	// The handler delegates to proxyServer.ServeHTTP. The upstream host is
	// not reachable in tests; we only verify the handler completes.
	if w.Code == 0 {
		t.Log("No response code set, which is expected in this test setup")
	}
}

// TestHandleRequestWithDifferentMethods tests the handler with various HTTP methods.
func TestHandleRequestWithDifferentMethods(t *testing.T) {
	p, err := NewProxy(testConfig(), &MockWriter{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	handler := p.Handler()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			// The handler should complete without panic for all methods.
		})
	}
}

func TestLogResponse(t *testing.T) {
	tests := []struct {
		name           string
		loggingEnabled bool
		excludeRegexp  *regexp.Regexp
		requestPath    string
		shouldLog      bool
		expectError    bool
	}{
		{
			name:           "Logging disabled",
			loggingEnabled: false,
			excludeRegexp:  nil,
			requestPath:    "/test",
			shouldLog:      false,
			expectError:    false,
		},
		{
			name:           "Logging enabled, no exclusion",
			loggingEnabled: true,
			excludeRegexp:  nil,
			requestPath:    "/test",
			shouldLog:      true,
			expectError:    false,
		},
		{
			name:           "Logging enabled, path excluded",
			loggingEnabled: true,
			excludeRegexp:  regexp.MustCompile("^/api/.*"),
			requestPath:    "/api/users",
			shouldLog:      false,
			expectError:    false,
		},
		{
			name:           "Logging enabled, path not excluded",
			loggingEnabled: true,
			excludeRegexp:  regexp.MustCompile("^/api/.*"),
			requestPath:    "/web/users",
			shouldLog:      true,
			expectError:    false,
		},
		{
			name:           "Logging enabled, empty exclude regexp matches all",
			loggingEnabled: true,
			excludeRegexp:  regexp.MustCompile(""),
			requestPath:    "/test",
			shouldLog:      false, // Empty regexp matches everything, so should exclude
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, _ := url.Parse("http://example.com")
			mockWriter := &MockWriter{}
			p := &Proxy{
				cfg: &config.TranslatedConfig{
					LoggingEnabled: tt.loggingEnabled,
					ExcludeRegexp:  tt.excludeRegexp,
					TargetURL:      targetURL,
				},
				writer: mockWriter,
			}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			resp := &http.Response{
				StatusCode: 200,
				Request:    req,
				Header:     make(http.Header),
			}

			err := p.logResponse(resp)

			if tt.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.shouldLog {
				if mockWriter.lastResponse != resp {
					t.Errorf("Expected response to be logged, but it wasn't")
				}
			} else {
				if mockWriter.lastResponse == resp {
					t.Errorf("Expected response not to be logged, but it was")
				}
			}
		})
	}
}

func TestLogResponseWithWriterError(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	p := &Proxy{
		cfg: &config.TranslatedConfig{
			LoggingEnabled: true,
			ExcludeRegexp:  nil,
			TargetURL:      targetURL,
		},
		writer: &MockWriterWithError{},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	resp := &http.Response{
		StatusCode: 200,
		Request:    req,
		Header:     make(http.Header),
	}

	err := p.logResponse(resp)

	if err == nil {
		t.Error("Expected error from writer, got nil")
	}
	if err.Error() != "mock writer error" {
		t.Errorf("Expected 'mock writer error', got %v", err)
	}
}

func TestDefaultHTTPServer(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	server := NewDefaultHTTPServer(&config.TranslatedConfig{
		ReadTimeout:  10,
		WriteTimeout: 20,
		IdleTimeout:  120,
		TargetURL:    targetURL,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Invalid address, so no server is actually started.
	err := server.ListenAndServe("invalid:address:format", handler)
	if err == nil {
		t.Error("Expected error for invalid address format")
	}
}

func TestRunWithDifferentConfigurations(t *testing.T) {
	tests := []struct {
		name       string
		listenIP   string
		listenPort string
	}{
		{
			name:       "Standard configuration",
			listenIP:   "127.0.0.1",
			listenPort: "8080",
		},
		{
			name:       "Different IP",
			listenIP:   "0.0.0.0",
			listenPort: "9090",
		},
		{
			name:       "Localhost",
			listenIP:   "localhost",
			listenPort: "3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.ListenIP = tt.listenIP
			cfg.ListenPort = tt.listenPort

			mockServer := &MockHTTPServer{}

			Run(cfg, &MockWriter{}, mockServer)

			expectedAddr := tt.listenIP + ":" + tt.listenPort
			if mockServer.lastAddress != expectedAddr {
				t.Errorf("Expected address %s, got %s", expectedAddr, mockServer.lastAddress)
			}
		})
	}
}

// TestRunWithEmptyConfiguration tests Run with minimal configuration.
func TestRunWithEmptyConfiguration(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		ListenIP:   "127.0.0.1",
		ListenPort: "8080",
		TargetURL:  targetURL,
		Headers:    make(map[string]string),
	}

	mockServer := &MockHTTPServer{}

	Run(cfg, &MockWriter{}, mockServer)

	if mockServer.handler == nil {
		t.Error("Handler was not set")
	}

	expectedAddr := "127.0.0.1:8080"
	if mockServer.lastAddress != expectedAddr {
		t.Errorf("Expected address %s, got %s", expectedAddr, mockServer.lastAddress)
	}
}

// TestRunWithComplexConfiguration tests Run with complex configurations.
func TestRunWithComplexConfiguration(t *testing.T) {
	targetURL, _ := url.Parse("https://api.example.com:8443/v1")
	cfg := testConfig()
	cfg.ListenIP = "0.0.0.0"
	cfg.ListenPort = "9999"
	cfg.TargetURL = targetURL
	cfg.Headers = map[string]string{
		"X-Custom-Header": "CustomValue",
		"Authorization":   "Bearer token123",
	}
	cfg.ExcludeRegexp = regexp.MustCompile("^/health")
	cfg.ExcludePostBodyRegexp = regexp.MustCompile("password")
	cfg.ExcludeResponseBodyRegexp = regexp.MustCompile("secret")

	mockServer := &MockHTTPServer{}

	Run(cfg, &MockWriter{}, mockServer)

	if mockServer.handler == nil {
		t.Error("Handler was not set")
	}

	expectedAddr := "0.0.0.0:9999"
	if mockServer.lastAddress != expectedAddr {
		t.Errorf("Expected address %s, got %s", expectedAddr, mockServer.lastAddress)
	}
}

// TestRunErrorWithProxyCreationError tests the proxy creation error path of run.
func TestRunErrorWithProxyCreationError(t *testing.T) {
	cfg := testConfig()
	cfg.TargetURL = nil // causes proxy.NewServer to fail

	err := run(cfg, &MockWriter{}, &MockHTTPServer{})
	if err == nil {
		t.Error("Expected error when proxy creation fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create proxy server") {
		t.Errorf("Expected proxy creation error, got: %v", err)
	}
}

// TestRunErrorWithServerError tests the server error path of run.
func TestRunErrorWithServerError(t *testing.T) {
	err := run(testConfig(), &MockWriter{}, &MockHTTPServerWithError{})
	if err == nil {
		t.Error("Expected error when server fails, got nil")
	}
	if !strings.Contains(err.Error(), "mock server error") {
		t.Errorf("Expected server error, got: %v", err)
	}
}
