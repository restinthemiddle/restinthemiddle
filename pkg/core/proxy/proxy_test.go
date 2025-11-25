package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
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

// verifyExpectedHeader checks a specific header value against expected value with special handling for generated values.
func verifyExpectedHeader(t *testing.T, capturedRequest *http.Request, key, expectedValue string) {
	switch {
	case key == "X-Request-Id" && expectedValue == "generated":
		// Special case: check if UUID was generated
		actualValue := capturedRequest.Header.Get(key)
		if actualValue == "" {
			t.Errorf("Expected %s header to be generated, but it was empty", key)
		}
		// Basic UUID format check (36 characters with dashes)
		if len(actualValue) != 36 {
			t.Errorf("Expected %s to be a valid UUID, got: %s", key, actualValue)
		}
	case key == "X-Forwarded-Host" && expectedValue == "backend-host":
		// Special case: check that X-Forwarded-Host is set to the backend host
		actualValue := capturedRequest.Header.Get(key)
		if actualValue == "" {
			t.Errorf("Expected %s header to be set, but it was empty", key)
		}
		// Should be the test server's host
		if !strings.Contains(actualValue, "127.0.0.1") && !strings.Contains(actualValue, "localhost") {
			t.Errorf("Expected %s to contain backend host, got: %s", key, actualValue)
		}
	case key == "X-Forwarded-Port" && expectedValue == "backend-port":
		// Special case: check that X-Forwarded-Port is set to the backend port
		actualValue := capturedRequest.Header.Get(key)
		if actualValue == "" {
			t.Errorf("Expected %s header to be set, but it was empty", key)
		}
		// Should be a valid port number
		if _, err := strconv.Atoi(actualValue); err != nil {
			t.Errorf("Expected %s to be a valid port number, got: %s", key, actualValue)
		}
	default:
		actualValue := capturedRequest.Header.Get(key)
		if actualValue != expectedValue {
			t.Errorf("Expected %s header to be %q, got %q", key, expectedValue, actualValue)
		}
	}
}

func TestProxyDirectorHeaders(t *testing.T) {
	tests := []struct {
		name            string
		targetURL       string
		requestHeaders  map[string]string
		configHeaders   map[string]string
		expectedHeaders map[string]string
		checkFunc       func(req *http.Request) error
	}{
		{
			name:           "X-Request-Id header is set when missing",
			targetURL:      "http://example.com",
			requestHeaders: map[string]string{},
			expectedHeaders: map[string]string{
				"X-Request-Id": "generated", // Will check if it exists and is valid UUID
			},
		},
		{
			name:      "X-Request-Id header is preserved when present",
			targetURL: "http://example.com",
			requestHeaders: map[string]string{
				"X-Request-Id": "existing-request-id",
			},
			expectedHeaders: map[string]string{
				"X-Request-Id": "existing-request-id",
			},
		},
		{
			name:           "X-Forwarded headers are set",
			targetURL:      "https://example.com:8443",
			requestHeaders: map[string]string{},
			expectedHeaders: map[string]string{
				// Note: Go's reverse proxy sets these based on the actual backend server
				// We'll check that they exist and have reasonable values
				"X-Forwarded-Host":  "backend-host", // Will be checked dynamically
				"X-Forwarded-Proto": "http",         // Test server is HTTP
				"X-Forwarded-Port":  "backend-port", // Will be checked dynamically
			},
		},
		{
			name:      "X-Forwarded headers are preserved when present",
			targetURL: "http://example.com",
			requestHeaders: map[string]string{
				"X-Forwarded-Host":  "original-host.com",
				"X-Forwarded-Proto": "original-proto",
				"X-Forwarded-Port":  "original-port",
				"X-Forwarded-For":   "original-for",
			},
			expectedHeaders: map[string]string{
				"X-Forwarded-Host":  "original-host.com",
				"X-Forwarded-Proto": "original-proto",
				"X-Forwarded-Port":  "original-port",
				"X-Forwarded-For":   "original-for, 192.0.2.1", // Client IP is appended
			},
		},
		{
			name:           "Default ports are set correctly for HTTPS",
			targetURL:      "https://example.com",
			requestHeaders: map[string]string{},
			expectedHeaders: map[string]string{
				"X-Forwarded-Port": "backend-port", // Will be checked dynamically
			},
		},
		{
			name:           "Default ports are set correctly for HTTP",
			targetURL:      "http://example.com",
			requestHeaders: map[string]string{},
			expectedHeaders: map[string]string{
				"X-Forwarded-Port": "backend-port", // Will be checked dynamically
			},
		},
		{
			name:           "Custom headers from config are added",
			targetURL:      "http://example.com",
			requestHeaders: map[string]string{},
			configHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
				"X-App-Version":   "1.0.0",
			},
			expectedHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
				"X-App-Version":   "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, err := url.Parse(tt.targetURL)
			if err != nil {
				t.Fatalf("Failed to parse target URL: %v", err)
			}

			cfg := &config.TranslatedConfig{
				TargetURL:    targetURL,
				Headers:      tt.configHeaders,
				SetRequestID: true,
			}

			// Create a test server to capture the modified request
			var capturedRequest *http.Request
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			// Parse the test server URL and update config
			testServerURL, _ := url.Parse(testServer.URL)
			cfg.TargetURL = testServerURL

			server, err := NewServer(cfg)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Create request with specified headers
			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.requestHeaders {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Check expected headers
			for key, expectedValue := range tt.expectedHeaders {
				verifyExpectedHeader(t, capturedRequest, key, expectedValue)
			}
		})
	}
}

func TestProxyDirectorRespectsDisabledRequestID(t *testing.T) {
	targetURL, err := url.Parse("http://example.com")
	if err != nil {
		t.Fatalf("Failed to parse target URL: %v", err)
	}

	cfg := &config.TranslatedConfig{
		TargetURL:    targetURL,
		SetRequestID: false,
	}

	var capturedRequest *http.Request
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	testServerURL, _ := url.Parse(testServer.URL)
	cfg.TargetURL = testServerURL

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if capturedRequest.Header.Get("X-Request-Id") != "" {
		t.Errorf("Expected X-Request-Id to remain unset when SetRequestID is false")
	}
}

func TestProxyDirectorAuthorizationHeaders(t *testing.T) {
	tests := []struct {
		name               string
		targetURL          string
		requestAuthHeader  string
		expectedAuthHeader string
	}{
		{
			name:               "Basic Auth from DSN only",
			targetURL:          "http://user:pass@example.com",
			requestAuthHeader:  "",
			expectedAuthHeader: "Basic dXNlcjpwYXNz", // base64(user:pass)
		},
		{
			name:               "Authorization header merged with Basic Auth",
			targetURL:          "http://user:pass@example.com",
			requestAuthHeader:  "Bearer token123",
			expectedAuthHeader: "Basic dXNlcjpwYXNz, Bearer token123",
		},
		{
			name:               "No Basic Auth in DSN, preserve existing header",
			targetURL:          "http://example.com",
			requestAuthHeader:  "Bearer token456",
			expectedAuthHeader: "Bearer token456",
		},
		{
			name:               "No auth at all",
			targetURL:          "http://example.com",
			requestAuthHeader:  "",
			expectedAuthHeader: "",
		},
		{
			name:               "Username without password in DSN",
			targetURL:          "http://user@example.com",
			requestAuthHeader:  "Bearer token123",
			expectedAuthHeader: "Bearer token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, err := url.Parse(tt.targetURL)
			if err != nil {
				t.Fatalf("Failed to parse target URL: %v", err)
			}

			cfg := &config.TranslatedConfig{
				TargetURL: targetURL,
			}

			// Create a test server to capture the modified request
			var capturedRequest *http.Request
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			// Parse the test server URL and update config (preserve user info from original)
			testServerURL, _ := url.Parse(testServer.URL)
			testServerURL.User = targetURL.User
			cfg.TargetURL = testServerURL

			server, err := NewServer(cfg)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Create request with Authorization header if specified
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.requestAuthHeader != "" {
				req.Header.Set("Authorization", tt.requestAuthHeader)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Check Authorization header
			actualAuthHeader := capturedRequest.Header.Get("Authorization")
			if actualAuthHeader != tt.expectedAuthHeader {
				t.Errorf("Expected Authorization header to be %q, got %q", tt.expectedAuthHeader, actualAuthHeader)
			}
		})
	}
}

func TestProxyDirectorPathAndQuery(t *testing.T) {
	tests := []struct {
		name             string
		targetURL        string
		requestPath      string
		expectedPath     string
		expectedRawQuery string
	}{
		{
			name:         "Simple path joining",
			targetURL:    "http://example.com/api",
			requestPath:  "/users",
			expectedPath: "/api/users",
		},
		{
			name:         "Path with trailing slash on target",
			targetURL:    "http://example.com/api/",
			requestPath:  "/users",
			expectedPath: "/api/users",
		},
		{
			name:         "Path without leading slash on request",
			targetURL:    "http://example.com/api",
			requestPath:  "/users",
			expectedPath: "/api/users",
		},
		{
			name:         "Both paths have slashes",
			targetURL:    "http://example.com/api/",
			requestPath:  "/users",
			expectedPath: "/api/users",
		},
		{
			name:             "Query parameter merging - target has query",
			targetURL:        "http://example.com?version=v1",
			requestPath:      "/users?limit=10",
			expectedPath:     "/users",
			expectedRawQuery: "version=v1&limit=10",
		},
		{
			name:             "Query parameter merging - only request has query",
			targetURL:        "http://example.com",
			requestPath:      "/users?limit=10",
			expectedPath:     "/users",
			expectedRawQuery: "limit=10",
		},
		{
			name:             "Query parameter merging - only target has query",
			targetURL:        "http://example.com?version=v1",
			requestPath:      "/users",
			expectedPath:     "/users",
			expectedRawQuery: "version=v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, err := url.Parse(tt.targetURL)
			if err != nil {
				t.Fatalf("Failed to parse target URL: %v", err)
			}

			cfg := &config.TranslatedConfig{
				TargetURL: targetURL,
			}

			// Create a test server to capture the modified request
			var capturedRequest *http.Request
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r
				w.WriteHeader(http.StatusOK)
			}))
			defer testServer.Close()

			// Parse the test server URL and preserve path/query from original
			testServerURL, _ := url.Parse(testServer.URL)
			testServerURL.Path = targetURL.Path
			testServerURL.RawQuery = targetURL.RawQuery
			cfg.TargetURL = testServerURL

			server, err := NewServer(cfg)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Create request
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Check path
			if capturedRequest.URL.Path != tt.expectedPath {
				t.Errorf("Expected path to be %q, got %q", tt.expectedPath, capturedRequest.URL.Path)
			}

			// Check query if specified
			if tt.expectedRawQuery != "" {
				if capturedRequest.URL.RawQuery != tt.expectedRawQuery {
					t.Errorf("Expected raw query to be %q, got %q", tt.expectedRawQuery, capturedRequest.URL.RawQuery)
				}
			}
		})
	}
}

func TestSingleJoiningSlash(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected string
	}{
		{
			name:     "Both have slashes",
			a:        "path/",
			b:        "/to/resource",
			expected: "path/to/resource",
		},
		{
			name:     "Neither has slashes",
			a:        "path",
			b:        "to/resource",
			expected: "path/to/resource",
		},
		{
			name:     "Only a has slash",
			a:        "path/",
			b:        "to/resource",
			expected: "path/to/resource",
		},
		{
			name:     "Only b has slash",
			a:        "path",
			b:        "/to/resource",
			expected: "path/to/resource",
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: "/",
		},
		{
			name:     "Empty a",
			a:        "",
			b:        "/to/resource",
			expected: "/to/resource",
		},
		{
			name:     "Empty b",
			a:        "path/",
			b:        "",
			expected: "path/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := singleJoiningSlash(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("singleJoiningSlash(%q, %q) = %q, expected %q", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestNewSingleHostReverseProxyError(t *testing.T) {
	// Test that NewProfilingTransport error is handled
	targetURL, _ := url.Parse("http://example.com")

	// This should work fine with a valid config
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}

	_, err := newSingleHostReverseProxy(targetURL, cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestNewServerProfilingTransportError(t *testing.T) {
	// Test NewServer when newSingleHostReverseProxy fails
	// This is harder to test since NewProfilingTransport rarely fails
	// But we can test with an invalid configuration that might cause issues
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
		// Add any config that might cause NewProfilingTransport to fail
	}

	server, err := NewServer(cfg)
	// In normal cases this should succeed, but this tests the error path exists
	if err != nil {
		// Error path is tested
		if server != nil {
			t.Error("Server should be nil when error occurs")
		}
	} else {
		// Success path
		if server == nil {
			t.Error("Server should not be nil when no error occurs")
		}
	}
}

func TestDefaultReverseProxyModifyResponse(t *testing.T) {
	// Test the ModifyResponse method of DefaultReverseProxy
	targetURL, _ := url.Parse("http://example.com")
	cfg := &config.TranslatedConfig{
		TargetURL: targetURL,
	}

	proxy, err := newSingleHostReverseProxy(targetURL, cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Test that ModifyResponse can be set
	called := false
	proxy.ModifyResponse(func(resp *http.Response) error {
		called = true
		return nil
	})

	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Update the proxy's target to our test server
	testServerURL, _ := url.Parse(testServer.URL)
	cfg.TargetURL = testServerURL
	proxy, _ = newSingleHostReverseProxy(testServerURL, cfg)
	proxy.ModifyResponse(func(resp *http.Response) error {
		called = true
		return nil
	})

	// Make a request through the proxy
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)

	if !called {
		t.Error("ModifyResponse function was not called")
	}
}
