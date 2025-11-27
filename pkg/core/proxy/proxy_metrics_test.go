package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

func TestServeHTTPWithMetrics(t *testing.T) {
	// Create a test backend server
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
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestServeHTTPWithoutMetrics(t *testing.T) {
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
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestResponseWriterWrapper(t *testing.T) {
	w := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     0,
		responseSize:   0,
	}

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusCreated)
	if wrapper.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, wrapper.statusCode)
	}

	// Test Write
	testData := []byte("test data")
	n, err := wrapper.Write(testData)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), n)
	}
	if wrapper.responseSize != len(testData) {
		t.Errorf("Expected response size %d, got %d", len(testData), wrapper.responseSize)
	}

	// Test accumulated Write
	moreData := []byte(" more data")
	n2, err := wrapper.Write(moreData)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedSize := len(testData) + len(moreData)
	if wrapper.responseSize != expectedSize {
		t.Errorf("Expected accumulated response size %d, got %d (n2=%d)", expectedSize, wrapper.responseSize, n2)
	}
}

func TestComputeApproximateRequestSizeGET(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	size := computeApproximateRequestSize(req)
	if size < 20 {
		t.Errorf("Expected size >= 20, got %d", size)
	}
}

func TestComputeApproximateRequestSizePOST(t *testing.T) {
	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "http://example.com/api/users", body)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 15
	size := computeApproximateRequestSize(req)
	if size < 50 {
		t.Errorf("Expected size >= 50, got %d", size)
	}
}

func TestComputeApproximateRequestSizeHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Accept", "application/json")
	size := computeApproximateRequestSize(req)
	if size < 80 {
		t.Errorf("Expected size >= 80, got %d", size)
	}
}

func TestComputeApproximateRequestSizeLongURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/very/long/path/to/resource?param1=value1&param2=value2&param3=value3", nil)
	req.Header.Set("Accept", "text/html")
	size := computeApproximateRequestSize(req)
	if size < 100 {
		t.Errorf("Expected size >= 100, got %d", size)
	}
}

func TestComputeApproximateRequestSizeEdgeCases(t *testing.T) {
	// Nil URL
	req := &http.Request{
		Method: "GET",
		URL:    nil,
		Header: http.Header{},
	}
	size := computeApproximateRequestSize(req)
	if size <= 0 {
		t.Errorf("Expected size > 0 even with nil URL, got %d", size)
	}

	// Large Content-Length
	req2 := httptest.NewRequest("POST", "http://example.com/", nil)
	req2.ContentLength = 10000000
	size2 := computeApproximateRequestSize(req2)
	if size2 < 10000000 {
		t.Errorf("Expected size >= 10000000, got %d", size2)
	}
}

func TestServeHTTPMetricsStatus200(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	targetURL, _ := url.Parse(testServer.URL)
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
	}

	server, _ := NewServer(cfg)
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestServeHTTPMetricsStatus404(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	targetURL, _ := url.Parse(testServer.URL)
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
	}

	server, _ := NewServer(cfg)
	req := httptest.NewRequest("GET", "http://example.com/missing", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestServeHTTPMetricsStatus500(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	targetURL, _ := url.Parse(testServer.URL)
	cfg := &config.TranslatedConfig{
		TargetURL:      targetURL,
		MetricsEnabled: true,
	}

	server, _ := NewServer(cfg)
	req := httptest.NewRequest("GET", "http://example.com/error", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
