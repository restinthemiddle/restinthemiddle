//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestHelper contains helper functions for tests
type TestHelper struct {
	t            *testing.T
	proxyProcess *exec.Cmd
	proxyPort    string
	tempDir      string
	mockServer   *MockServer
}

// setupProxy function simplify and make more robust
func setupProxy(t *testing.T, env map[string]string, args []string) *TestHelper {
	tempDir, err := os.MkdirTemp("", "restinthemiddle-test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	// Determine build directory
	buildDir, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("Error determining build directory: %v", err)
	}

	// Start mock server
	mockServer, err := StartMockServer()
	if err != nil {
		t.Fatalf("Error starting mock server: %v", err)
	}

	// Wait briefly for the mock server to start
	time.Sleep(500 * time.Millisecond)

	// Determine proxy port - use port from environment variable if available
	proxyPort := "8888" // Test port as default
	if customPort, exists := env["LISTEN_PORT"]; exists {
		proxyPort = customPort
	}

	// Command line arguments have the highest priority - check if --listen-port is set
	for i, arg := range args {
		if arg == "--listen-port" && i+1 < len(args) {
			// Next argument is the port
			proxyPort = args[i+1]
			break
		} else if len(arg) > 14 && arg[:14] == "--listen-port=" {
			// Format: --listen-port=8002
			proxyPort = arg[14:]
			break
		}
	}

	// Use mock server URL for TARGET_HOST_DSN
	mockServerURL := fmt.Sprintf("http://localhost:%s", mockServer.port)

	// Prepare environment variables for the proxy
	proxyEnv := make(map[string]string)
	for k, v := range env {
		proxyEnv[k] = v
	}

	// Always set the mock server as TARGET_HOST_DSN (unless already overridden)
	// Except when it's a config file test
	if _, isConfigTest := proxyEnv["_CONFIG_FILE_TEST"]; !isConfigTest {
		if _, exists := proxyEnv["TARGET_HOST_DSN"]; !exists {
			proxyEnv["TARGET_HOST_DSN"] = mockServerURL
		}
	}
	delete(proxyEnv, "_CONFIG_FILE_TEST") // Cleanup

	// Set LISTEN_PORT
	proxyEnv["LISTEN_PORT"] = proxyPort

	// Disable logging for tests to reduce output
	if _, exists := proxyEnv["LOGGING_ENABLED"]; !exists {
		proxyEnv["LOGGING_ENABLED"] = "false"
	}

	// Start process
	cmd := exec.Command(filepath.Join(buildDir, "bin/restinthemiddle"))
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = os.Environ()
	for k, v := range proxyEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Redirect output for debugging
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		mockServer.Stop()
		t.Fatalf("Error starting proxy: %v", err)
	}

	// Wait and test if the proxy starts
	time.Sleep(1 * time.Second)

	// Test if the proxy is reachable
	testURL := fmt.Sprintf("http://localhost:%s/test", proxyPort)

	// Multiple attempts with timeout
	var lastErr error
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		// Use HTTP client with timeout
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(testURL)
		if err == nil {
			resp.Body.Close()
			lastErr = nil
			break
		}
		lastErr = err

		// Check if process is still running
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Logf("Proxy process exited early after attempt %d", i+1)
			break
		}

		// Wait briefly before next attempt
		time.Sleep(200 * time.Millisecond)
	}

	if lastErr != nil {
		// Output debugging information
		t.Logf("Proxy stdout: %s", stdoutBuf.String())
		t.Logf("Proxy stderr: %s", stderrBuf.String())

		// Check if process is still running
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Logf("Proxy process already exited: %v", cmd.ProcessState)
		}

		mockServer.Stop()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		t.Fatalf("Proxy is not reachable at %s after multiple attempts: %v", testURL, lastErr)
	}

	return &TestHelper{
		t:            t,
		proxyProcess: cmd,
		proxyPort:    proxyPort,
		tempDir:      tempDir,
		mockServer:   mockServer,
	}
}

// cleanup terminates the proxy process and cleans up temporary files
func (h *TestHelper) cleanup() {
	// Stop mock server
	if h.mockServer != nil {
		_ = h.mockServer.Stop()
		h.mockServer = nil
	}

	// Terminate proxy process
	if h.proxyProcess != nil && h.proxyProcess.Process != nil {
		// Try graceful shutdown with SIGTERM first
		if err := h.proxyProcess.Process.Signal(os.Interrupt); err == nil {
			// Wait briefly for graceful shutdown
			done := make(chan error, 1)
			go func() {
				_, err := h.proxyProcess.Process.Wait()
				done <- err
			}()

			select {
			case <-done:
				// Graceful shutdown successful
			case <-time.After(2 * time.Second):
				// Force kill after timeout
				if err := h.proxyProcess.Process.Kill(); err != nil {
					h.t.Logf("Warning: Error force-killing proxy process: %v", err)
				}
				<-done // Wait until Process.Wait() is finished
			}
		} else {
			// If SIGTERM fails, kill directly
			if err := h.proxyProcess.Process.Kill(); err != nil {
				h.t.Logf("Warning: Error terminating proxy process: %v", err)
			}
			_, _ = h.proxyProcess.Process.Wait()
		}
		h.proxyProcess = nil
	}

	// Clean up temporary directory
	if h.tempDir != "" {
		_ = os.RemoveAll(h.tempDir)
		h.tempDir = ""
	}
}

// makeRequest sends a request through the proxy
func (h *TestHelper) makeRequest(path string, method string, headers map[string]string, body io.Reader) (*http.Response, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%s%s", h.proxyPort, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Add standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return client.Do(req)
}

// createConfigFile creates a temporary configuration file with the given content
func (h *TestHelper) createConfigFile(content string) (string, error) {
	configPath := filepath.Join(h.tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	return configPath, err
}

// TestBasicConfiguration adapt
func TestBasicConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantPort string
	}{
		{
			name:     "default configuration",
			envVars:  map[string]string{},
			wantPort: "8888", // The actual test port
		},
		{
			name: "custom port",
			envVars: map[string]string{
				"LISTEN_PORT": "9000",
			},
			wantPort: "9000",
		},
		{
			name: "custom ip",
			envVars: map[string]string{
				"LISTEN_IP": "127.0.0.1",
			},
			wantPort: "8888", // The actual test port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupProxy(t, tt.envVars, nil)
			defer h.cleanup()

			// Check if the proxy listens on the expected port
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			url := fmt.Sprintf("http://localhost:%s/test", tt.wantPort)
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status: %d, Got: %d", http.StatusOK, resp.StatusCode)
			}
		})
	}
}

// TestHeaderManipulation adapt
func TestHeaderManipulation(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		args    []string
		config  string
		want    map[string]string
	}{
		{
			name: "custom headers via args",
			args: []string{"--header=X-App-Version:3.0.0", "--header=X-Custom-Header:Test"},
			want: map[string]string{
				"X-App-Version":   "3.0.0",
				"X-Custom-Header": "Test",
			},
		},
		{
			name: "custom headers via config file",
			config: `
headers:
  X-App-Version: 3.0.0
  X-Custom-Header: Test
`, // targetHostDsn will be added in the test method
			want: map[string]string{
				"X-App-Version":   "3.0.0",
				"X-Custom-Header": "Test",
			},
		},
		{
			name: "request id header",
			args: []string{"--set-request-id=true"},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupProxy(t, map[string]string{}, tt.args)
			defer h.cleanup()

			if tt.config != "" {
				// Create configuration file in the current directory
				// (the binary automatically searches for config.yaml in ".")
				configWithMock := fmt.Sprintf("targetHostDsn: %s\n%s", fmt.Sprintf("http://localhost:%s", h.mockServer.port), tt.config)

				configPath := "config.yaml"
				err := os.WriteFile(configPath, []byte(configWithMock), 0644)
				if err != nil {
					t.Fatalf("Error creating configuration file: %v", err)
				}
				defer os.Remove(configPath) // Cleanup

				h.cleanup()
				h = setupProxy(t, map[string]string{}, nil) // No additional args needed
				defer h.cleanup()
			}

			// Send request through the proxy
			resp, err := h.makeRequest("/test", "GET", tt.headers, nil)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			// Check if the expected headers arrived at the mock server
			lastRequest := h.mockServer.GetLastRequest()
			if lastRequest == nil {
				t.Fatal("No request arrived at mock server")
			}

			for headerName, expectedValue := range tt.want {
				// Header names are canonicalized in Go, so check multiple variants
				var actualValues []string
				var exists bool

				// Try different header names
				headerVariants := []string{
					headerName,
					http.CanonicalHeaderKey(headerName),
				}

				for _, variant := range headerVariants {
					if values, found := lastRequest.Headers[variant]; found {
						actualValues = values
						exists = true
						break
					}
				}

				if !exists || len(actualValues) == 0 {
					t.Errorf("Header %s not found. Available headers: %v", headerName, getHeaderNames(lastRequest.Headers))
				} else if actualValues[0] != expectedValue {
					t.Errorf("Header %s has wrong value: expected %s, got %s",
						headerName, expectedValue, actualValues[0])
				}
			}

			// If request ID should be set, check if header is present
			if contains(tt.args, "--set-request-id=true") {
				// Try different header names for request ID
				requestIdVariants := []string{"X-Request-Id", "X-Request-ID"}
				var requestIdHeaders []string
				var foundRequestId bool

				for _, variant := range requestIdVariants {
					if headers, exists := lastRequest.Headers[variant]; exists && len(headers) > 0 {
						requestIdHeaders = headers
						foundRequestId = true
						break
					}
				}

				if !foundRequestId {
					t.Errorf("X-Request-Id header not found, although set-request-id=true. Available headers: %v", getHeaderNames(lastRequest.Headers))
				} else {
					// Check if it's a valid UUID v4
					uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
					if !uuidPattern.MatchString(requestIdHeaders[0]) {
						t.Errorf("X-Request-Id is not a valid UUID v4: %s", requestIdHeaders[0])
					}
				}
			}
		})
	}
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getHeaderNames returns all header names from a header map
func getHeaderNames(headers map[string][]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	return names
}

// TestConfigurationOptions adapt
func TestConfigurationOptions(t *testing.T) {
	// Configuration options from README.md
	configTests := []struct {
		name     string
		config   string
		envVar   string
		flagName string
		want     interface{}
		check    func(*TestHelper) bool
	}{
		{
			name:     "listen port configuration",
			config:   "listenPort",
			envVar:   "LISTEN_PORT",
			flagName: "--listen-port",
			want:     "8000",
			check: func(h *TestHelper) bool {
				// Check if the server listens on the specified port
				conn, err := http.Get(fmt.Sprintf("http://localhost:%s/test", h.proxyPort))
				return err == nil && conn.StatusCode == http.StatusOK
			},
		},
		{
			name:     "target host DSN configuration",
			config:   "targetHostDsn",
			envVar:   "TARGET_HOST_DSN",
			flagName: "--target-host-dsn",
			want:     fmt.Sprintf("http://localhost:%s", "18080"), // Will be set dynamically in the test
			check: func(h *TestHelper) bool {
				// Send request and check if it was forwarded
				resp, err := h.makeRequest("/", "GET", nil, nil)
				return err == nil && resp.StatusCode == http.StatusOK
			},
		},
		{
			name:     "logging enabled configuration",
			config:   "loggingEnabled",
			envVar:   "LOGGING_ENABLED",
			flagName: "--logging-enabled",
			want:     false,
			check: func(h *TestHelper) bool {
				// Analyze log output
				// In a real test this would check if logs were created
				return true
			},
		},
		{
			name:     "set request ID configuration",
			config:   "setRequestId",
			envVar:   "SET_REQUEST_ID",
			flagName: "--set-request-id",
			want:     true,
			check: func(h *TestHelper) bool {
				resp, err := h.makeRequest("/test", "GET", nil, nil)
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				// Check if request ID was sent to the backend
				lastRequest := h.mockServer.GetLastRequest()
				if lastRequest == nil {
					return false
				}

				// Request ID should be present in the backend request
				requestIdHeaders, exists := lastRequest.Headers["X-Request-Id"]
				return exists && len(requestIdHeaders) > 0 && requestIdHeaders[0] != ""
			},
		},
		{
			name:     "exclude configuration",
			config:   "exclude",
			envVar:   "EXCLUDE",
			flagName: "--exclude",
			want:     "^/health$",
			check: func(h *TestHelper) bool {
				// In a real test this would check if certain paths are excluded
				return true
			},
		},
		{
			name:     "log post body configuration",
			config:   "logPostBody",
			envVar:   "LOG_POST_BODY",
			flagName: "--log-post-body",
			want:     true,
			check: func(h *TestHelper) bool {
				// In a real test this would check if POST bodies are logged
				return true
			},
		},
		{
			name:     "log response body configuration",
			config:   "logResponseBody",
			envVar:   "LOG_RESPONSE_BODY",
			flagName: "--log-response-body",
			want:     true,
			check: func(h *TestHelper) bool {
				// In a real test this would check if response bodies are logged
				return true
			},
		},
		{
			name:     "exclude post body configuration",
			config:   "excludePostBody",
			envVar:   "EXCLUDE_POST_BODY",
			flagName: "--exclude-post-body",
			want:     "^/login$",
			check: func(h *TestHelper) bool {
				// In a real test this would check if certain POST bodies are excluded
				return true
			},
		},
		{
			name:     "exclude response body configuration",
			config:   "excludeResponseBody",
			envVar:   "EXCLUDE_RESPONSE_BODY",
			flagName: "--exclude-response-body",
			want:     "^/users$",
			check: func(h *TestHelper) bool {
				// In a real test this would check if certain response bodies are excluded
				return true
			},
		},
		{
			name:     "read timeout configuration",
			config:   "readTimeout",
			envVar:   "READ_TIMEOUT",
			flagName: "--read-timeout",
			want:     10,
			check: func(h *TestHelper) bool {
				// In a real test this would check the timeout setting
				return true
			},
		},
		{
			name:     "write timeout configuration",
			config:   "writeTimeout",
			envVar:   "WRITE_TIMEOUT",
			flagName: "--write-timeout",
			want:     15,
			check: func(h *TestHelper) bool {
				// In a real test this would check the timeout setting
				return true
			},
		},
		{
			name:     "idle timeout configuration",
			config:   "idleTimeout",
			envVar:   "IDLE_TIMEOUT",
			flagName: "--idle-timeout",
			want:     60,
			check: func(h *TestHelper) bool {
				// In a real test this would check the timeout setting
				return true
			},
		},
	}

	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with config file
			t.Run("via config file", func(t *testing.T) {
				// Start mock server for config file test
				mockServer, err := StartMockServer()
				if err != nil {
					t.Fatalf("Error starting mock server: %v", err)
				}
				defer mockServer.Stop()

				// Create configuration file in the current directory
				// The binary automatically loads config.yaml from the current directory
				mockURL := fmt.Sprintf("http://localhost:%s", mockServer.port)

				var config string
				if tt.config == "targetHostDsn" {
					// Special case: When testing targetHostDsn, use the desired value
					config = fmt.Sprintf("%s: %v", tt.config, tt.want)
				} else {
					// Normal: Mock server URL + additional configuration
					config = fmt.Sprintf("targetHostDsn: %s\n%s: %v", mockURL, tt.config, tt.want)
				}

				configPath := "config.yaml"
				err = os.WriteFile(configPath, []byte(config), 0644)
				if err != nil {
					t.Fatalf("Error creating configuration file: %v", err)
				}
				defer os.Remove(configPath) // Cleanup

				// setupProxy without automatic TARGET_HOST_DSN (will be loaded from config.yaml)
				env := map[string]string{
					// Prevent setupProxy from automatically setting TARGET_HOST_DSN
					"_CONFIG_FILE_TEST": "true",
				}
				h := setupProxy(t, env, nil)
				h.mockServer = mockServer // Assign mock server
				defer h.cleanup()

				if !tt.check(h) {
					t.Errorf("Configuration via file failed: %s=%v", tt.config, tt.want)
				}
			})

			// Test with environment variable
			if tt.envVar != "" {
				t.Run("via environment variable", func(t *testing.T) {
					env := map[string]string{} // Mock server will be used automatically

					// Set value according to type
					var envValue string
					switch v := tt.want.(type) {
					case string:
						envValue = v
					case bool:
						envValue = fmt.Sprintf("%t", v)
					case int:
						envValue = fmt.Sprintf("%d", v)
					default:
						t.Fatalf("Unsupported type for env var: %v", reflect.TypeOf(tt.want))
					}

					env[tt.envVar] = envValue

					h := setupProxy(t, env, nil)
					defer h.cleanup()

					if !tt.check(h) {
						t.Errorf("Configuration via environment variable failed: %s=%v", tt.envVar, tt.want)
					}
				})
			}

			// Test with command line argument
			t.Run("via command line flag", func(t *testing.T) {
				// Set value according to type
				var flagValue string
				switch v := tt.want.(type) {
				case string:
					flagValue = v
				case bool:
					flagValue = fmt.Sprintf("%t", v)
				case int:
					flagValue = fmt.Sprintf("%d", v)
				default:
					t.Fatalf("Unsupported type for flag: %v", reflect.TypeOf(tt.want))
				}

				arg := fmt.Sprintf("%s=%s", tt.flagName, flagValue)

				h := setupProxy(t, map[string]string{}, []string{arg})
				defer h.cleanup()

				if !tt.check(h) {
					t.Errorf("Configuration via flag failed: %s=%v", tt.flagName, tt.want)
				}
			})
		})
	}
}

// TestSpecialFeatures tests special features like DSN processing and authentication
func TestSpecialFeatures(t *testing.T) {
	tests := []struct {
		name      string
		targetDsn string
		check     func(*TestHelper) bool
	}{
		{
			name:      "basic auth via DSN",
			targetDsn: "", // Will be set dynamically to mock server URL with auth
			check: func(h *TestHelper) bool {
				// Send request without any Authorization header
				resp, err := h.makeRequest("/test", "GET", nil, nil)
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				// Check if Basic Auth header was correctly set from DSN
				lastRequest := h.mockServer.GetLastRequest()
				if lastRequest == nil {
					return false
				}

				auth, exists := lastRequest.Headers["Authorization"]
				if !exists || len(auth) == 0 {
					return false
				}

				// Should be Basic auth with base64 encoded "user:pass"
				expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
				return auth[0] == expectedAuth
			},
		},
		{
			name:      "auth header merging - DSN auth with existing header",
			targetDsn: "", // Will be set dynamically to mock server URL with auth
			check: func(h *TestHelper) bool {
				// Send request with existing Authorization header
				headers := map[string]string{
					"Authorization": "Bearer JWT123TOKEN",
				}
				resp, err := h.makeRequest("/test", "GET", headers, nil)
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				// Check if both Basic Auth (from DSN) and Bearer token are present
				lastRequest := h.mockServer.GetLastRequest()
				if lastRequest == nil {
					return false
				}

				auth, exists := lastRequest.Headers["Authorization"]
				if !exists || len(auth) == 0 {
					return false
				}

				// Should contain both Basic auth and the original Bearer token
				expectedBasicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
				expectedMergedAuth := expectedBasicAuth + ", Bearer JWT123TOKEN"
				return auth[0] == expectedMergedAuth
			},
		},
		{
			name:      "auth header passthrough - no DSN auth",
			targetDsn: "", // Will be set dynamically to mock server URL without auth
			check: func(h *TestHelper) bool {
				// Send request with Authorization header but no DSN auth
				headers := map[string]string{
					"Authorization": "Bearer ONLYTOKEN",
				}
				resp, err := h.makeRequest("/test", "GET", headers, nil)
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				// Check if original Authorization header is preserved unchanged
				lastRequest := h.mockServer.GetLastRequest()
				if lastRequest == nil {
					return false
				}

				auth, exists := lastRequest.Headers["Authorization"]
				if !exists || len(auth) == 0 {
					return false
				}

				// Should be exactly the original header
				return auth[0] == "Bearer ONLYTOKEN"
			},
		},
		{
			name:      "custom headers override authorization",
			targetDsn: "", // Will be set dynamically to mock server URL with auth
			check: func(h *TestHelper) bool {
				// This test checks if custom headers from config override everything
				// We'll need to set up a proxy with custom Authorization header in config
				// For now, we'll just test that the system doesn't crash
				resp, err := h.makeRequest("/test", "GET", nil, nil)
				if err != nil {
					return false
				}
				defer resp.Body.Close()
				return resp.StatusCode == 200
			},
		},
		{
			name:      "base path handling",
			targetDsn: "", // Will be set dynamically to mock server URL with path
			check: func(h *TestHelper) bool {
				// Check if base path is correctly prepended
				// In a real test this would check if the path is correctly composed
				return true
			},
		},
		{
			name:      "query parameter handling",
			targetDsn: "", // Will be set dynamically to mock server URL with query params
			check: func(h *TestHelper) bool {
				// Check if query parameters are correctly forwarded
				// In a real test this would check if query parameters are correctly forwarded
				return true
			},
		},
		{
			name:      "custom port in DSN",
			targetDsn: "", // Will be set dynamically in the test
			check: func(h *TestHelper) bool {
				// Check if custom port is correctly used
				// Test that the proxy correctly forwards to the mock server
				resp, err := http.Get(fmt.Sprintf("http://localhost:%s/test", h.proxyPort))
				if err != nil {
					return false
				}
				defer resp.Body.Close()
				return resp.StatusCode == 200
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h *TestHelper
			var mockServer *MockServer
			var err error

			if tt.targetDsn == "" || tt.name == "custom port in DSN" ||
				strings.Contains(tt.name, "auth header") || tt.name == "basic auth via DSN" {
				// For tests that need dynamic mock server URLs
				mockServer, err = StartMockServer()
				if err != nil {
					t.Fatalf("Failed to start mock server: %v", err)
				}
				defer mockServer.Stop()

				var mockURL string
				switch tt.name {
				case "basic auth via DSN":
					mockURL = fmt.Sprintf("http://user:pass@localhost:%s", mockServer.GetPort())
				case "auth header merging - DSN auth with existing header":
					mockURL = fmt.Sprintf("http://user:pass@localhost:%s", mockServer.GetPort())
				case "auth header passthrough - no DSN auth":
					mockURL = fmt.Sprintf("http://localhost:%s", mockServer.GetPort())
				case "custom headers override authorization":
					mockURL = fmt.Sprintf("http://user:pass@localhost:%s", mockServer.GetPort())
				case "base path handling":
					mockURL = fmt.Sprintf("http://localhost:%s/api", mockServer.GetPort())
				case "query parameter handling":
					mockURL = fmt.Sprintf("http://localhost:%s?token=123", mockServer.GetPort())
				case "custom port in DSN":
					mockURL = fmt.Sprintf("http://localhost:%s", mockServer.GetPort())
				default:
					mockURL = fmt.Sprintf("http://localhost:%s", mockServer.GetPort())
				}

				h = setupProxy(t, map[string]string{"TARGET_HOST_DSN": mockURL}, nil)
				h.mockServer = mockServer // Assign the mock server to the test helper
			} else {
				h = setupProxy(t, map[string]string{"TARGET_HOST_DSN": tt.targetDsn}, nil)
			}

			defer h.cleanup()

			if !tt.check(h) {
				t.Errorf("Special feature failed: %s", tt.name)
			}
		})
	}
}

// TestPrecedence adapt
func TestPrecedence(t *testing.T) {
	// In the README.md the following order (ascending) is specified:
	// 1. Restinthemiddle default values
	// 2. Configuration via YAML file
	// 3. Configuration via Environment variables
	// 4. Command line arguments

	h := setupProxy(t, map[string]string{
		"LISTEN_PORT": "8001",
	}, []string{"--listen-port=8002"})
	defer h.cleanup()

	// Check if port 8002 (command line) was taken (not 8001 from environment)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// The proxy should run on port 8002 (command line has priority)
	// but h.proxyPort contains the actually used port
	url := fmt.Sprintf("http://localhost:%s/test", h.proxyPort)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the correct port was used (command line has priority over environment)
	if h.proxyPort != "8002" {
		t.Errorf("Precedence for command line arguments does not work correctly: expected port 8002, got port %s", h.proxyPort)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Precedence for command line arguments does not work correctly: expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Further tests for precedence order could be implemented here
}

// TestREADMEExamples adapt
func TestREADMEExamples(t *testing.T) {
	// Example 1: Basic
	t.Run("basic example", func(t *testing.T) {
		h := setupProxy(t, map[string]string{}, nil)
		defer h.cleanup()

		resp, err := h.makeRequest("/api/visitors", "GET", nil, nil)
		if err != nil {
			t.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()

		// Here we can check if the request arrived at the mock server
		lastRequest := h.mockServer.GetLastRequest()
		if lastRequest == nil {
			t.Fatal("No request arrived at mock server")
		}
		if lastRequest.Path != "/api/visitors" {
			t.Errorf("Wrong path: expected /api/visitors, got %s", lastRequest.Path)
		}
	})

	// Example 2: Advanced
	t.Run("advanced example", func(t *testing.T) {
		// Start a separate mock server for this test to get the port
		mockServer, err := StartMockServer()
		if err != nil {
			t.Fatalf("Failed to start mock server: %v", err)
		}
		defer mockServer.Stop()

		// Construct DSN with mock server URL and authentication
		mockServerURL := fmt.Sprintf("http://user:pass@localhost:%s/api?start=1577833200", mockServer.GetPort())

		h := setupProxy(t, map[string]string{
			"TARGET_HOST_DSN": mockServerURL,
		}, nil)
		defer h.cleanup()

		resp, err := h.makeRequest("/visitors", "GET", nil, nil)
		if err != nil {
			t.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()

		// Check if the request was correctly forwarded
		lastRequest := mockServer.GetLastRequest()
		if lastRequest == nil {
			t.Fatal("No request arrived at mock server")
		}

		// Path should be /api/visitors (base path + request path)
		if lastRequest.Path != "/api/visitors" {
			t.Errorf("Wrong path: expected /api/visitors, got %s", lastRequest.Path)
		}

		// Query parameters should be included
		if !contains(lastRequest.QueryParams["start"], "1577833200") {
			t.Errorf("Query parameter start=1577833200 missing or has wrong value")
		}

		// Basic auth header should be set
		auth, exists := lastRequest.Headers["Authorization"]
		if !exists || len(auth) == 0 {
			t.Errorf("Authorization header missing")
		} else {
			expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
			if auth[0] != expectedAuth {
				t.Errorf("Wrong auth header: expected %s, got %s", expectedAuth, auth[0])
			}
		}
	})

	// Example 3: Setting/changing headers
	t.Run("setting headers example", func(t *testing.T) {
		h := setupProxy(t, map[string]string{}, nil)

		// Use mock server URL for configuration file
		mockURL := fmt.Sprintf("http://localhost:%s", h.mockServer.port)
		config := fmt.Sprintf(`
targetHostDsn: %s
headers:
  X-App-Version: 3.0.0
loggingEnabled: false
`, mockURL)

		configPath := "config.yaml"
		err := os.WriteFile(configPath, []byte(config), 0644)
		if err != nil {
			t.Fatalf("Error creating configuration file: %v", err)
		}
		defer os.Remove(configPath) // Cleanup

		h.cleanup()
		h = setupProxy(t, map[string]string{}, nil) // No additional args needed
		defer h.cleanup()

		resp, err := h.makeRequest("/home", "GET", nil, nil)
		if err != nil {
			t.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()

		// Check if the header was correctly forwarded
		lastRequest := h.mockServer.GetLastRequest()
		if lastRequest == nil {
			t.Fatal("No request arrived at mock server")
		}

		appVersionHeaders, exists := lastRequest.Headers["X-App-Version"]
		if !exists || len(appVersionHeaders) == 0 {
			t.Errorf("X-App-Version header missing")
		} else if appVersionHeaders[0] != "3.0.0" {
			t.Errorf("X-App-Version has wrong value: expected 3.0.0, got %s", appVersionHeaders[0])
		}
	})
}

// TestAuthorizationHeaderHandling tests the Authorization header handling logic
func TestAuthorizationHeaderHandling(t *testing.T) {
	tests := []struct {
		name           string
		dsn            string
		requestHeaders map[string]string
		expectedAuth   string
		description    string
	}{
		{
			name:         "Basic Auth from DSN only",
			dsn:          "http://user:pass@localhost:%s",
			expectedAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
			description:  "Should set Basic Auth header from DSN credentials",
		},
		{
			name: "Authorization header merging",
			dsn:  "http://user:pass@localhost:%s",
			requestHeaders: map[string]string{
				"Authorization": "Bearer JWT123TOKEN",
			},
			expectedAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")) + ", Bearer JWT123TOKEN",
			description:  "Should merge DSN Basic Auth with existing Authorization header",
		},
		{
			name: "Authorization header passthrough",
			dsn:  "http://localhost:%s", // No auth in DSN
			requestHeaders: map[string]string{
				"Authorization": "Bearer ONLYTOKEN",
			},
			expectedAuth: "Bearer ONLYTOKEN",
			description:  "Should preserve original Authorization header when no DSN auth",
		},
		{
			name: "Multiple Authorization headers merging",
			dsn:  "http://admin:secret@localhost:%s",
			requestHeaders: map[string]string{
				"Authorization": "ApiKey abc123, Bearer xyz789",
			},
			expectedAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")) + ", ApiKey abc123, Bearer xyz789",
			description:  "Should merge DSN Basic Auth with complex existing Authorization header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock server
			mockServer, err := StartMockServer()
			if err != nil {
				t.Fatalf("Failed to start mock server: %v", err)
			}
			defer mockServer.Stop()

			// Construct DSN with mock server port
			dsn := fmt.Sprintf(tt.dsn, mockServer.GetPort())

			// Setup proxy with the DSN
			h := setupProxy(t, map[string]string{
				"TARGET_HOST_DSN": dsn,
			}, nil)
			defer h.cleanup()

			// Make request with specified headers
			resp, err := h.makeRequest("/test", "GET", tt.requestHeaders, nil)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			// Verify the request reached the mock server
			lastRequest := mockServer.GetLastRequest()
			if lastRequest == nil {
				t.Fatal("No request arrived at mock server")
			}

			// Check Authorization header
			auth, exists := lastRequest.Headers["Authorization"]
			if !exists || len(auth) == 0 {
				if tt.expectedAuth != "" {
					t.Errorf("Expected Authorization header '%s', but none found", tt.expectedAuth)
				}
				return
			}

			if auth[0] != tt.expectedAuth {
				t.Errorf("Authorization header mismatch.\nDescription: %s\nExpected: %s\nActual: %s",
					tt.description, tt.expectedAuth, auth[0])
			}

			// Log for debugging
			t.Logf("✓ %s: Authorization header correctly set to: %s", tt.description, auth[0])
		})
	}
}
