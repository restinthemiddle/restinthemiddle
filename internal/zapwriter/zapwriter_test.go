package zapwriter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	"github.com/restinthemiddle/restinthemiddle/pkg/core/transport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestHTTPTiming_MarshalLogObject(t *testing.T) {
	timing := HTTPTiming{
		GetConn:                  time.Date(2025, 7, 9, 10, 0, 0, 0, time.UTC),
		GotConn:                  time.Date(2025, 7, 9, 10, 0, 1, 0, time.UTC),
		ConnEstDuration:          time.Second,
		TCPConnectionStart:       time.Date(2025, 7, 9, 10, 0, 0, 500000000, time.UTC),
		TCPConnectionEstablished: time.Date(2025, 7, 9, 10, 0, 0, 800000000, time.UTC),
		TCPConnectionDuration:    300 * time.Millisecond,
		RoundTripStart:           time.Date(2025, 7, 9, 10, 0, 1, 0, time.UTC),
		RoundTripEnd:             time.Date(2025, 7, 9, 10, 0, 2, 0, time.UTC),
		RoundTripDuration:        time.Second,
		GotFirstResponseByte:     time.Date(2025, 7, 9, 10, 0, 1, 500000000, time.UTC),
		TLSHandshakeStart:        time.Date(2025, 7, 9, 10, 0, 0, 200000000, time.UTC),
		TLSHandshakeDone:         time.Date(2025, 7, 9, 10, 0, 0, 400000000, time.UTC),
		TLSHandshakeDuration:     200 * time.Millisecond,
	}

	// Create a test encoder to capture the fields
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// Log the timing object
	logger.Info("test", zap.Object("timing", timing))

	// Verify that the log was recorded
	if len(recorded.All()) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(recorded.All()))
	}

	entry := recorded.All()[0]
	if len(entry.Context) != 1 {
		t.Fatalf("Expected 1 context field, got %d", len(entry.Context))
	}

	// Verify the timing field exists
	timingField := entry.Context[0]
	if timingField.Key != "timing" {
		t.Errorf("Expected field key 'timing', got %s", timingField.Key)
	}

	if timingField.Type != zapcore.ObjectMarshalerType {
		t.Errorf("Expected field type ObjectMarshalerType, got %v", timingField.Type)
	}
}

func TestNewHTTPTimingFromCore(t *testing.T) {
	coreTime := time.Date(2025, 7, 9, 10, 0, 0, 0, time.UTC)
	coreTiming := &transport.HTTPTiming{
		GetConn:              coreTime,
		GotConn:              coreTime.Add(time.Second),
		GotFirstResponseByte: coreTime.Add(2 * time.Second),
		TLSHandshakeStart:    coreTime.Add(100 * time.Millisecond),
		TLSHandshakeDone:     coreTime.Add(300 * time.Millisecond),
	}

	timing, err := NewHTTPTimingFromCore(coreTiming)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if timing.GetConn != coreTime {
		t.Errorf("Expected GetConn %v, got %v", coreTime, timing.GetConn)
	}

	if timing.GotConn != coreTime.Add(time.Second) {
		t.Errorf("Expected GotConn %v, got %v", coreTime.Add(time.Second), timing.GotConn)
	}

	if timing.ConnEstDuration != time.Second {
		t.Errorf("Expected ConnEstDuration %v, got %v", time.Second, timing.ConnEstDuration)
	}

	if timing.TLSHandshakeDuration != 200*time.Millisecond {
		t.Errorf("Expected TLSHandshakeDuration %v, got %v", 200*time.Millisecond, timing.TLSHandshakeDuration)
	}
}

func TestWriter_LogResponse_BasicRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/path?query=value", nil)
	req.Header.Set("User-Agent", "test-agent")

	// Add timing context
	timing := &transport.HTTPTiming{
		GetConn: time.Now(),
		GotConn: time.Now().Add(time.Millisecond),
	}
	ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("timing"), timing)
	req = req.WithContext(ctx)

	resp := &http.Response{
		StatusCode:    200,
		Request:       req,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader("response body")),
		ContentLength: 13,
	}
	resp.Header.Set("Content-Type", "text/plain")

	// Create a test logger with observer
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	writer := Writer{
		Logger: logger,
		Config: &config.TranslatedConfig{
			LogResponseBody: true,
		},
	}

	err := writer.LogResponse(resp)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if len(recorded.All()) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(recorded.All()))
	}

	entry := recorded.All()[0]
	expectedFields := []string{
		"request_method", "scheme", "http_host", "request", "args",
		"request_headers", "post_body", "status", "response_headers",
		"body_bytes_sent", "response_body", "timing",
	}

	fieldNames := make([]string, len(entry.Context))
	for i, field := range entry.Context {
		fieldNames[i] = field.Key
	}

	for _, expectedField := range expectedFields {
		found := false
		for _, fieldName := range fieldNames {
			if fieldName == expectedField {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field %s not found in log entry", expectedField)
		}
	}
}

func TestWriter_LogResponse_POSTRequest(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://example.com/api", strings.NewReader("post data"))
	req.Header.Set("Content-Type", "application/json")

	// Add request body context
	ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("requestBodyString"), "post data")

	// Add timing context
	timing := &transport.HTTPTiming{
		GetConn: time.Now(),
		GotConn: time.Now().Add(time.Millisecond),
	}
	ctx = context.WithValue(ctx, transport.ProfilingContextKey("timing"), timing)
	req = req.WithContext(ctx)

	resp := &http.Response{
		StatusCode:    201,
		Request:       req,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader("created")),
		ContentLength: 7,
	}

	// Create a test logger with observer
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	writer := Writer{
		Logger: logger,
		Config: &config.TranslatedConfig{
			LogResponseBody: true,
		},
	}

	err := writer.LogResponse(resp)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if len(recorded.All()) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(recorded.All()))
	}
}

func TestWriter_LogResponse_BodyLoggingDisabled(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/path", nil)

	timing := &transport.HTTPTiming{
		GetConn: time.Now(),
		GotConn: time.Now().Add(time.Millisecond),
	}
	ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("timing"), timing)
	req = req.WithContext(ctx)

	resp := &http.Response{
		StatusCode: 200,
		Request:    req,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("response body")),
	}

	// Create a test logger with observer
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	writer := Writer{
		Logger: logger,
		Config: &config.TranslatedConfig{
			LogResponseBody: false,
		},
	}

	err := writer.LogResponse(resp)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if len(recorded.All()) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(recorded.All()))
	}
}

func TestWriter_LogResponse_MissingTimingContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/path", nil)
	resp := &http.Response{
		StatusCode: 200,
		Request:    req,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("response body")),
	}

	// Create a test logger with observer
	core, _ := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	writer := Writer{
		Logger: logger,
		Config: &config.TranslatedConfig{
			LogResponseBody: true,
		},
	}

	err := writer.LogResponse(resp)
	if err == nil {
		t.Fatal("Expected error, but got none")
	}
}

func TestWriter_extractRequestData(t *testing.T) {
	tests := []struct {
		name            string
		setupRequest    func() *http.Request
		expectedQuery   string
		expectedBody    string
		expectedHeaders int
	}{
		{
			name: "GET request with query parameters",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/path?param1=value1&param2=value2", nil)
				req.Header.Set("User-Agent", "test-agent")
				req.Header.Set("Accept", "application/json")
				return req
			},
			expectedQuery:   "?param1=value1&param2=value2",
			expectedBody:    "",
			expectedHeaders: 2,
		},
		{
			name: "POST request with body",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("POST", "https://example.com/api", strings.NewReader("post data"))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("requestBodyString"), "post data")
				req = req.WithContext(ctx)
				return req
			},
			expectedQuery:   "",
			expectedBody:    "post data",
			expectedHeaders: 1,
		},
		{
			name: "Request without query or body",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				return req
			},
			expectedQuery:   "",
			expectedBody:    "",
			expectedHeaders: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := Writer{}
			response := &http.Response{
				Request: tt.setupRequest(),
			}

			data := writer.extractRequestData(response)

			if data.Query != tt.expectedQuery {
				t.Errorf("Expected query %s, got %s", tt.expectedQuery, data.Query)
			}

			if data.Body != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, data.Body)
			}

			if len(data.Headers) != tt.expectedHeaders {
				t.Errorf("Expected %d headers, got %d", tt.expectedHeaders, len(data.Headers))
			}
		})
	}
}

func TestWriter_extractResponseData(t *testing.T) {
	tests := []struct {
		name            string
		setupResponse   func() *http.Response
		setupConfig     func() *config.TranslatedConfig
		expectedBody    string
		expectedHeaders int
		expectEmptyBody bool
	}{
		{
			name: "Response with body logging enabled",
			setupResponse: func() *http.Response {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				resp := &http.Response{
					Request: req,
					Header:  make(http.Header),
					Body:    io.NopCloser(strings.NewReader("response body")),
				}
				resp.Header.Set("Content-Type", "text/plain")
				resp.Header.Set("Content-Length", "13")
				return resp
			},
			setupConfig: func() *config.TranslatedConfig {
				return &config.TranslatedConfig{
					LogResponseBody: true,
				}
			},
			expectedBody:    "response body",
			expectedHeaders: 2,
			expectEmptyBody: false,
		},
		{
			name: "Response with body logging disabled",
			setupResponse: func() *http.Response {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				resp := &http.Response{
					Request: req,
					Header:  make(http.Header),
					Body:    io.NopCloser(strings.NewReader("response body")),
				}
				resp.Header.Set("Content-Type", "text/plain")
				return resp
			},
			setupConfig: func() *config.TranslatedConfig {
				return &config.TranslatedConfig{
					LogResponseBody: false,
				}
			},
			expectedBody:    "",
			expectedHeaders: 1,
			expectEmptyBody: true,
		},
		{
			name: "Response with excluded path",
			setupResponse: func() *http.Response {
				req, _ := http.NewRequest("GET", "https://example.com/excluded", nil)
				resp := &http.Response{
					Request: req,
					Header:  make(http.Header),
					Body:    io.NopCloser(strings.NewReader("response body")),
				}
				return resp
			},
			setupConfig: func() *config.TranslatedConfig {
				excludeRegex := regexp.MustCompile("/excluded")
				return &config.TranslatedConfig{
					LogResponseBody:           true,
					ExcludeResponseBodyRegexp: excludeRegex,
				}
			},
			expectedBody:    "",
			expectedHeaders: 0,
			expectEmptyBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := Writer{
				Config: tt.setupConfig(),
			}

			response := tt.setupResponse()
			data := writer.extractResponseData(response)

			if data.Body != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, data.Body)
			}

			if len(data.Headers) != tt.expectedHeaders {
				t.Errorf("Expected %d headers, got %d", tt.expectedHeaders, len(data.Headers))
			}

			if tt.expectEmptyBody && data.Body != "" {
				t.Error("Expected empty body but got content")
			}
		})
	}
}

func TestWriter_extractResponseBody(t *testing.T) {
	tests := []struct {
		name          string
		setupResponse func() *http.Response
		setupConfig   func() *config.TranslatedConfig
		expectedBody  string
	}{
		{
			name: "Normal response body extraction",
			setupResponse: func() *http.Response {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				return &http.Response{
					Request: req,
					Body:    io.NopCloser(strings.NewReader("response body")),
				}
			},
			setupConfig: func() *config.TranslatedConfig {
				return &config.TranslatedConfig{}
			},
			expectedBody: "response body",
		},
		{
			name: "Excluded path returns empty body",
			setupResponse: func() *http.Response {
				req, _ := http.NewRequest("GET", "https://example.com/excluded", nil)
				return &http.Response{
					Request: req,
					Body:    io.NopCloser(strings.NewReader("response body")),
				}
			},
			setupConfig: func() *config.TranslatedConfig {
				excludeRegex := regexp.MustCompile("/excluded")
				return &config.TranslatedConfig{
					ExcludeResponseBodyRegexp: excludeRegex,
				}
			},
			expectedBody: "",
		},
		{
			name: "Body with read error",
			setupResponse: func() *http.Response {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				return &http.Response{
					Request: req,
					Body:    &errorReader{},
				}
			},
			setupConfig: func() *config.TranslatedConfig {
				return &config.TranslatedConfig{}
			},
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := Writer{
				Config: tt.setupConfig(),
			}

			response := tt.setupResponse()
			body := writer.extractResponseBody(response)

			if body != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, body)
			}
		})
	}
}

func TestWriter_extractTiming(t *testing.T) {
	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectError   bool
		expectedError string
	}{
		{
			name: "Valid timing context",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				timing := &transport.HTTPTiming{
					GetConn: time.Now(),
					GotConn: time.Now().Add(time.Millisecond),
				}
				ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("timing"), timing)
				return req.WithContext(ctx)
			},
			expectError: false,
		},
		{
			name: "Missing timing context",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				return req
			},
			expectError:   true,
			expectedError: "timing information not available in request context",
		},
		{
			name: "Timing context with additional values",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/path", nil)
				timing := &transport.HTTPTiming{
					GetConn: time.Now(),
					GotConn: time.Now().Add(time.Millisecond),
				}
				ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("timing"), timing)
				ctx = context.WithValue(ctx, transport.ProfilingContextKey("tcpConnectionStart"), time.Now())
				ctx = context.WithValue(ctx, transport.ProfilingContextKey("tcpConnectionEstablished"), time.Now().Add(time.Millisecond))
				ctx = context.WithValue(ctx, transport.ProfilingContextKey("roundTripStart"), time.Now())
				ctx = context.WithValue(ctx, transport.ProfilingContextKey("roundTripEnd"), time.Now().Add(time.Millisecond))
				return req.WithContext(ctx)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := Writer{}
			response := &http.Response{
				Request: tt.setupRequest(),
			}

			timing, err := writer.extractTiming(response)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain %s, got %s", tt.expectedError, err.Error())
				}
				if timing != nil {
					t.Error("Expected nil timing when error occurs")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if timing == nil {
					t.Error("Expected timing object but got nil")
				}
			}
		})
	}
}

func TestWriter_populateConnectionTiming(t *testing.T) {
	writer := Writer{}
	timing := &HTTPTiming{}

	tcpStart := time.Now()
	tcpEstablished := tcpStart.Add(100 * time.Millisecond)

	req, _ := http.NewRequest("GET", "https://example.com/path", nil)
	ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("tcpConnectionStart"), tcpStart)
	ctx = context.WithValue(ctx, transport.ProfilingContextKey("tcpConnectionEstablished"), tcpEstablished)
	req = req.WithContext(ctx)

	response := &http.Response{Request: req}

	writer.populateConnectionTiming(response, timing)

	if timing.TCPConnectionStart != tcpStart {
		t.Errorf("Expected TCPConnectionStart %v, got %v", tcpStart, timing.TCPConnectionStart)
	}

	if timing.TCPConnectionEstablished != tcpEstablished {
		t.Errorf("Expected TCPConnectionEstablished %v, got %v", tcpEstablished, timing.TCPConnectionEstablished)
	}

	expectedDuration := tcpEstablished.Sub(tcpStart)
	if timing.TCPConnectionDuration != expectedDuration {
		t.Errorf("Expected TCPConnectionDuration %v, got %v", expectedDuration, timing.TCPConnectionDuration)
	}
}

func TestWriter_populateRoundTripTiming(t *testing.T) {
	writer := Writer{}
	timing := &HTTPTiming{}

	roundTripStart := time.Now()
	roundTripEnd := roundTripStart.Add(200 * time.Millisecond)

	req, _ := http.NewRequest("GET", "https://example.com/path", nil)
	ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("roundTripStart"), roundTripStart)
	ctx = context.WithValue(ctx, transport.ProfilingContextKey("roundTripEnd"), roundTripEnd)
	req = req.WithContext(ctx)

	response := &http.Response{Request: req}

	writer.populateRoundTripTiming(response, timing)

	if timing.RoundTripStart != roundTripStart {
		t.Errorf("Expected RoundTripStart %v, got %v", roundTripStart, timing.RoundTripStart)
	}

	if timing.RoundTripEnd != roundTripEnd {
		t.Errorf("Expected RoundTripEnd %v, got %v", roundTripEnd, timing.RoundTripEnd)
	}

	expectedDuration := roundTripEnd.Sub(roundTripStart)
	if timing.RoundTripDuration != expectedDuration {
		t.Errorf("Expected RoundTripDuration %v, got %v", expectedDuration, timing.RoundTripDuration)
	}
}

// errorReader is a test helper that always returns an error when reading.
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (r *errorReader) Close() error {
	return nil
}

// BenchmarkWriter_LogResponse benchmarks the LogResponse method.
func BenchmarkWriter_LogResponse(b *testing.B) {
	// Create a test logger that discards output
	logger := zap.NewNop()

	writer := Writer{
		Logger: logger,
		Config: &config.TranslatedConfig{
			LogResponseBody: true,
		},
	}

	// Setup a test response
	req, _ := http.NewRequest("GET", "https://example.com/path?query=value", nil)
	req.Header.Set("User-Agent", "test-agent")

	timing := &transport.HTTPTiming{
		GetConn: time.Now(),
		GotConn: time.Now().Add(time.Millisecond),
	}
	ctx := context.WithValue(req.Context(), transport.ProfilingContextKey("timing"), timing)
	req = req.WithContext(ctx)

	response := &http.Response{
		StatusCode:    200,
		Request:       req,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader("response body")),
		ContentLength: 13,
	}
	response.Header.Set("Content-Type", "text/plain")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset the body for each iteration
		response.Body = io.NopCloser(strings.NewReader("response body"))
		err := writer.LogResponse(response)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHTTPTiming_MarshalLogObject benchmarks the MarshalLogObject method.
func BenchmarkHTTPTiming_MarshalLogObject(b *testing.B) {
	timing := HTTPTiming{
		GetConn:                  time.Now(),
		GotConn:                  time.Now().Add(time.Second),
		ConnEstDuration:          time.Second,
		TCPConnectionStart:       time.Now(),
		TCPConnectionEstablished: time.Now().Add(time.Millisecond),
		TCPConnectionDuration:    time.Millisecond,
		RoundTripStart:           time.Now(),
		RoundTripEnd:             time.Now().Add(time.Second),
		RoundTripDuration:        time.Second,
		GotFirstResponseByte:     time.Now(),
		TLSHandshakeStart:        time.Now(),
		TLSHandshakeDone:         time.Now().Add(time.Millisecond),
		TLSHandshakeDuration:     time.Millisecond,
	}

	logger := zap.NewNop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test", zap.Object("timing", timing))
	}
}
