//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// MockServer represents an HTTP server used for testing
type MockServer struct {
	server       *http.Server
	port         string
	requests     []RequestRecord
	responses    map[string]ResponseConfig
	defaultResp  ResponseConfig
	mu           sync.Mutex
	started      bool
	requestWg    *sync.WaitGroup
	requestCount int
}

// RequestRecord stores information about a received request
type RequestRecord struct {
	Method      string
	Path        string
	QueryParams map[string][]string
	Headers     map[string][]string
	Body        string
	Timestamp   time.Time
}

// ResponseConfig defines how the mock server should respond to a request
type ResponseConfig struct {
	StatusCode  int
	Headers     map[string]string
	Body        string
	ContentType string
	Delay       time.Duration
}

// NewMockServer creates a new mock server
func NewMockServer(port string) *MockServer {
	return &MockServer{
		port:      port,
		requests:  make([]RequestRecord, 0),
		responses: make(map[string]ResponseConfig),
		defaultResp: ResponseConfig{
			StatusCode:  http.StatusOK,
			Headers:     map[string]string{},
			Body:        `{"status":"ok"}`,
			ContentType: "application/json",
		},
		requestWg: &sync.WaitGroup{},
	}
}

// Start starts the mock server
func (m *MockServer) Start() error {
	if m.started {
		return fmt.Errorf("server already started")
	}

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", m.port),
		Handler: m,
	}

	// Try to start the server directly and catch port binding errors
	listener, err := net.Listen("tcp", m.server.Addr)
	if err != nil {
		return err // Port binding error will be passed to StartMockServer
	}

	m.started = true
	go func() {
		err := m.server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			// Only log critical runtime errors (not port binding errors)
			fmt.Printf("Mock-Server runtime error: %v\n", err)
		}
	}()

	// Wait briefly for the server to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop stops the mock server
func (m *MockServer) Stop() error {
	if !m.started {
		return nil
	}
	m.started = false

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.server.Shutdown(ctx)
}

// SetResponse configures a specific response for a given path
func (m *MockServer) SetResponse(path string, response ResponseConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[path] = response
}

// SetDefaultResponse sets the default response for paths without specific configuration
func (m *MockServer) SetDefaultResponse(response ResponseConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultResp = response
}

// WaitForRequests waits until a certain number of requests have been received
func (m *MockServer) WaitForRequests(count int, timeout time.Duration) bool {
	// A context with timeout would be better here, but this is sufficient for simple tests
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		current := len(m.requests)
		m.mu.Unlock()
		if current >= count {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// GetRequests returns all requests recorded so far
func (m *MockServer) GetRequests() []RequestRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests
}

// GetLastRequest returns the last received request or nil if none exists
func (m *MockServer) GetLastRequest() *RequestRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		return nil
	}
	req := m.requests[len(m.requests)-1]
	return &req
}

// GetPort returns the port the mock server is running on
func (m *MockServer) GetPort() string {
	return m.port
}

// ServeHTTP implements the http.Handler interface
func (m *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()

	// Record the request
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	record := RequestRecord{
		Method:      r.Method,
		Path:        r.URL.Path,
		QueryParams: r.URL.Query(),
		Headers:     make(map[string][]string),
		Body:        string(body),
		Timestamp:   time.Now(),
	}

	// Copy headers
	for name, values := range r.Header {
		record.Headers[name] = values
	}

	// Add request to the list
	m.requests = append(m.requests, record)
	m.requestCount++

	// Determine which response to send
	response, exists := m.responses[r.URL.Path]
	if !exists {
		response = m.defaultResp
	}

	m.mu.Unlock()

	// Add delay if configured
	if response.Delay > 0 {
		time.Sleep(response.Delay)
	}

	// Set headers
	for name, value := range response.Headers {
		w.Header().Set(name, value)
	}

	// Set Content-Type
	if response.ContentType != "" {
		w.Header().Set("Content-Type", response.ContentType)
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Send body
	w.Write([]byte(response.Body))
}

// StartMockServer is a helper function to set up a MockServer with default values
func StartMockServer() (*MockServer, error) {
	// Try to find an available port starting from 18080, with a wider range
	var mock *MockServer
	var err error

	for port := 18080; port <= 18120; port++ {
		mock = NewMockServer(fmt.Sprintf("%d", port))
		err = mock.Start()
		if err == nil {
			// Successfully started - only log if we didn't get the first port
			if port > 18080 {
				fmt.Printf("Mock server successfully started on port %d (after conflicts with port(s) 18080-%d)\n", port, port-1)
			}
			break // Successfully started on this port
		}
		// For port conflicts, show which port will be tried next
		if port < 18120 {
			fmt.Printf("Mock server port %d occupied, trying port %d\n", port, port+1)
		} else {
			fmt.Printf("Mock server port %d occupied, no more ports available\n", port)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start mock server on any port between 18080-18120: %w", err)
	}

	// Default test endpoint
	mock.SetResponse("/test", ResponseConfig{
		StatusCode:  http.StatusOK,
		Body:        `{"status":"ok","message":"test endpoint"}`,
		ContentType: "application/json",
	})

	// Set up GET endpoint
	mock.SetResponse("/api/items", ResponseConfig{
		StatusCode:  http.StatusOK,
		Body:        `{"items":[{"id":1,"name":"Item 1"},{"id":2,"name":"Item 2"}]}`,
		ContentType: "application/json",
	})

	// Set up POST endpoint
	mock.SetResponse("/api/submit", ResponseConfig{
		StatusCode:  http.StatusCreated,
		Body:        `{"status":"created","id":123}`,
		ContentType: "application/json",
	})

	return mock, nil
}
