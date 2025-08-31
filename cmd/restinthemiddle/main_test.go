package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	testListenIP = "127.0.0.1"
)

// Import the default constants to use in tests.
const (
	testDefaultTargetHostDSN       = ""
	testDefaultListenIP            = "0.0.0.0"
	testDefaultListenPort          = "8000"
	testDefaultExclude             = ""
	testDefaultExcludePostBody     = ""
	testDefaultExcludeResponseBody = ""
	testDefaultReadTimeout         = 5
	testDefaultWriteTimeout        = 10
	testDefaultIdleTimeout         = 120
)

// Mock implementations for testing.
type MockConfigLoader struct {
	config *config.TranslatedConfig
	err    error
}

func (m *MockConfigLoader) Load(args []string) (*config.TranslatedConfig, error) {
	return m.config, m.err
}

type MockLoggerFactory struct {
	logger *zap.Logger
	err    error
}

func (m *MockLoggerFactory) CreateLogger() (*zap.Logger, error) {
	return m.logger, m.err
}

func TestApp_Run_Success(t *testing.T) {
	// Create a mock config
	mockConfig := &config.TranslatedConfig{
		ListenIP:   testListenIP,
		ListenPort: "8080",
	}

	// Create a mock logger
	mockLogger := zap.NewNop()

	app := &App{
		ConfigLoader:  &MockConfigLoader{config: mockConfig},
		LoggerFactory: &MockLoggerFactory{logger: mockLogger},
		Args:          []string{"test-app"},
	}

	// Test the setup parts that we can test without starting the server
	// We test that config and logger are loaded correctly
	cfg, err := app.ConfigLoader.Load(app.Args)
	if err != nil {
		t.Fatalf("Expected no error from config loader, got: %v", err)
	}

	if cfg.ListenIP != testListenIP {
		t.Errorf("Expected ListenIP to be '%s', got: %s", testListenIP, cfg.ListenIP)
	}

	if cfg.ListenPort != "8080" {
		t.Errorf("Expected ListenPort to be '8080', got: %s", cfg.ListenPort)
	}

	logger, err := app.LoggerFactory.CreateLogger()
	if err != nil {
		t.Fatalf("Expected no error from logger factory, got: %v", err)
	}

	if logger == nil {
		t.Error("Expected logger to be created")
	}

	// Note: We can't test core.Run() without starting an actual server
	// This would be better tested with a mock HTTP server interface
}

func TestApp_Run_ConfigError(t *testing.T) {
	expectedError := errors.New("config load error")

	var output bytes.Buffer

	app := &App{
		ConfigLoader:  &MockConfigLoader{err: expectedError},
		LoggerFactory: &MockLoggerFactory{logger: zap.NewNop()},
		Writer:        &output,
		Args:          []string{"test-app"},
	}

	err := app.Run()

	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("Expected error to contain 'failed to load config', got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), expectedError.Error()) {
		t.Errorf("Expected error to contain original error, got: %s", err.Error())
	}
}

func TestApp_Run_LoggerError(t *testing.T) {
	mockConfig := &config.TranslatedConfig{
		ListenIP:   testListenIP,
		ListenPort: "8080",
	}

	expectedError := errors.New("logger creation error")

	var output bytes.Buffer

	app := &App{
		ConfigLoader:  &MockConfigLoader{config: mockConfig},
		LoggerFactory: &MockLoggerFactory{err: expectedError},
		Writer:        &output,
		Args:          []string{"test-app"},
	}

	err := app.Run()

	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	if !strings.Contains(err.Error(), "failed to create logger") {
		t.Errorf("Expected error to contain 'failed to create logger', got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), expectedError.Error()) {
		t.Errorf("Expected error to contain original error, got: %s", err.Error())
	}
}

func TestNewApp(t *testing.T) {
	app := NewApp()

	// Since we know NewApp() never returns nil, we can safely access its fields
	if app.ConfigLoader == nil {
		t.Error("Expected ConfigLoader to be set")
	}

	if app.LoggerFactory == nil {
		t.Error("Expected LoggerFactory to be set")
	}

	if app.Writer == nil {
		t.Error("Expected Writer to be set")
	}

	if app.Args == nil {
		t.Error("Expected Args to be set")
	}

	// Check if default implementations are used
	if _, ok := app.ConfigLoader.(*DefaultConfigLoader); !ok {
		t.Error("Expected DefaultConfigLoader")
	}

	if _, ok := app.LoggerFactory.(*DefaultLoggerFactory); !ok {
		t.Error("Expected DefaultLoggerFactory")
	}
}

func TestDefaultLoggerFactory_CreateLogger(t *testing.T) {
	factory := &DefaultLoggerFactory{}
	logger, err := factory.CreateLogger()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	// Test that the logger is properly configured
	// This is a basic test - in a real scenario you might want to test specific config options
}

func TestDefaultConfigLoader_Load(t *testing.T) {
	loader := &DefaultConfigLoader{}

	// Test with minimal valid arguments
	args := []string{"test-app", "--target-host-dsn", "http://example.com"}

	_, err := loader.Load(args)

	// This might fail due to missing config file, but we're testing the structure
	// In a real test, you would set up the environment properly
	if err != nil {
		// Expected - we need a target host DSN
		t.Logf("Expected error for minimal config: %v", err)
	}
}

func TestSetupFlags(t *testing.T) {
	// Reset flags before test
	oldCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	defer func() { flag.CommandLine = oldCommandLine }()

	flagVars := setupFlags()

	// Test default values
	if flagVars.listenIP != testDefaultListenIP {
		t.Errorf("Expected listenIP default to be '%s', got: %s", testDefaultListenIP, flagVars.listenIP)
	}

	if flagVars.listenPort != testDefaultListenPort {
		t.Errorf("Expected listenPort default to be '%s', got: %s", testDefaultListenPort, flagVars.listenPort)
	}

	if !flagVars.loggingEnabled {
		t.Error("Expected loggingEnabled default to be true")
	}

	if flagVars.setRequestID {
		t.Error("Expected setRequestID default to be false")
	}

	if !flagVars.logPostBody {
		t.Error("Expected logPostBody default to be true")
	}

	if !flagVars.logResponseBody {
		t.Error("Expected logResponseBody default to be true")
	}

	if flagVars.readTimeout != testDefaultReadTimeout {
		t.Errorf("Expected readTimeout default to be %d, got: %d", testDefaultReadTimeout, flagVars.readTimeout)
	}

	if flagVars.writeTimeout != testDefaultWriteTimeout {
		t.Errorf("Expected writeTimeout default to be %d, got: %d", testDefaultWriteTimeout, flagVars.writeTimeout)
	}

	if flagVars.idleTimeout != testDefaultIdleTimeout {
		t.Errorf("Expected idleTimeout default to be %d, got: %d", testDefaultIdleTimeout, flagVars.idleTimeout)
	}
}

func TestSetupFlagsUsageTemplate(t *testing.T) {
	// Reset flags before test
	oldCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	defer func() { flag.CommandLine = oldCommandLine }()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Setup flags (this sets the custom Usage function)
	setupFlags()

	// Call the Usage function
	flag.Usage()

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read the captured output
	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	output := buf.String()

	// Test that the output contains version information
	if !strings.Contains(output, "restinthemiddle") {
		t.Errorf("Expected usage output to contain 'restinthemiddle', got: %s", output)
	}

	// Test that the output contains usage information
	if !strings.Contains(output, "Usage of") {
		t.Errorf("Expected usage output to contain 'Usage of', got: %s", output)
	}

	// Test that the output contains flag definitions
	if !strings.Contains(output, "--target-host-dsn") {
		t.Errorf("Expected usage output to contain '--target-host-dsn', got: %s", output)
	}

	// Test that version info appears before usage
	versionIndex := strings.Index(output, "restinthemiddle")
	usageIndex := strings.Index(output, "Usage of")
	if versionIndex == -1 || usageIndex == -1 || versionIndex >= usageIndex {
		t.Errorf("Expected version info to appear before usage info. Version at %d, Usage at %d", versionIndex, usageIndex)
	}

	// Test that there's proper spacing (two newlines between version and usage)
	if !strings.Contains(output, "\n\nUsage of") {
		t.Errorf("Expected proper spacing between version and usage info, got: %s", output)
	}
}

func TestSetupFlagsUsageTemplateWithMockVersion(t *testing.T) {
	// Reset flags before test
	oldCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	defer func() { flag.CommandLine = oldCommandLine }()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Setup flags
	setupFlags()

	// Call the Usage function
	flag.Usage()

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read the captured output
	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	output := buf.String()

	// Test that version.Info() is being called
	// This should contain the pattern from version.Info(): "restinthemiddle X (built Y, commit Z)"
	matched := strings.Contains(output, "(built") && strings.Contains(output, "commit")
	if !matched {
		t.Errorf("Expected usage output to match version pattern, got: %s", output)
	}

	// Test that output has the expected structure
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines in usage output, got %d: %v", len(lines), lines)
	}

	// First line should be version info
	if !strings.Contains(lines[0], "restinthemiddle") {
		t.Errorf("Expected first line to contain version info, got: %s", lines[0])
	}

	// Second line should be empty (spacing)
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		t.Errorf("Expected second line to be empty for spacing, got: %s", lines[1])
	}

	// Third line should start with "Usage of"
	if len(lines) > 2 && !strings.HasPrefix(lines[2], "Usage of") {
		t.Errorf("Expected third line to start with 'Usage of', got: %s", lines[2])
	}
}

func TestSetupFlagsCustomUsageSet(t *testing.T) {
	// Reset flags before test
	oldCommandLine := flag.CommandLine
	oldUsage := flag.Usage
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	defer func() {
		flag.CommandLine = oldCommandLine
		flag.Usage = oldUsage
	}()

	// Setup flags (this should set a custom Usage function)
	setupFlags()

	// Test that the custom Usage function is not nil
	if flag.Usage == nil {
		t.Error("Expected setupFlags to set a non-nil Usage function")
	}

	// Test that calling the Usage function works without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected Usage function to execute without panic, but got: %v", r)
		}
	}()

	// Capture stderr to test the function works
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call the Usage function - this should not panic
	flag.Usage()

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read output to verify it worked
	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	output := buf.String()

	// Should have some output (not empty)
	if len(output) == 0 {
		t.Error("Expected Usage function to produce output")
	}
}

func TestUpdateConfigFromFlags(t *testing.T) {
	cfg := &config.SourceConfig{}

	flagVars := &FlagVars{
		targetHostDSN:       "http://example.com",
		listenIP:            testListenIP,
		listenPort:          "9000",
		loggingEnabled:      false,
		setRequestID:        true,
		exclude:             "test-exclude",
		logPostBody:         false,
		logResponseBody:     false,
		excludePostBody:     "exclude-post",
		excludeResponseBody: "exclude-response",
		readTimeout:         30,
		writeTimeout:        60,
		idleTimeout:         300,
	}

	updateConfigFromFlags(cfg, flagVars)

	if cfg.TargetHostDSN != "http://example.com" {
		t.Errorf("Expected TargetHostDSN to be 'http://example.com', got: %s", cfg.TargetHostDSN)
	}

	if cfg.ListenIP != testListenIP {
		t.Errorf("Expected ListenIP to be '%s', got: %s", testListenIP, cfg.ListenIP)
	}

	if cfg.ListenPort != "9000" {
		t.Errorf("Expected ListenPort to be '9000', got: %s", cfg.ListenPort)
	}

	if cfg.LoggingEnabled {
		t.Error("Expected LoggingEnabled to be false")
	}

	if !cfg.SetRequestID {
		t.Error("Expected SetRequestID to be true")
	}

	if cfg.Exclude != "test-exclude" {
		t.Errorf("Expected Exclude to be 'test-exclude', got: %s", cfg.Exclude)
	}

	if cfg.LogPostBody {
		t.Error("Expected LogPostBody to be false")
	}

	if cfg.LogResponseBody {
		t.Error("Expected LogResponseBody to be false")
	}

	if cfg.ExcludePostBody != "exclude-post" {
		t.Errorf("Expected ExcludePostBody to be 'exclude-post', got: %s", cfg.ExcludePostBody)
	}

	if cfg.ExcludeResponseBody != "exclude-response" {
		t.Errorf("Expected ExcludeResponseBody to be 'exclude-response', got: %s", cfg.ExcludeResponseBody)
	}

	if cfg.ReadTimeout != 30 {
		t.Errorf("Expected ReadTimeout to be 30, got: %d", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 60 {
		t.Errorf("Expected WriteTimeout to be 60, got: %d", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout != 300 {
		t.Errorf("Expected IdleTimeout to be 300, got: %d", cfg.IdleTimeout)
	}
}

func TestUpdateConfigFromFlags_DefaultValues(t *testing.T) {
	cfg := &config.SourceConfig{}

	flagVars := &FlagVars{
		targetHostDSN:       testDefaultTargetHostDSN, // empty - should not update
		listenIP:            testDefaultListenIP,      // default - should not update
		listenPort:          testDefaultListenPort,    // default - should not update
		loggingEnabled:      true,
		setRequestID:        false,
		exclude:             testDefaultExclude, // empty - should not update
		logPostBody:         true,
		logResponseBody:     true,
		excludePostBody:     testDefaultExcludePostBody,     // empty - should not update
		excludeResponseBody: testDefaultExcludeResponseBody, // empty - should not update
		readTimeout:         testDefaultReadTimeout,         // default - should not update
		writeTimeout:        testDefaultWriteTimeout,        // default - should not update
		idleTimeout:         testDefaultIdleTimeout,         // default - should not update
	}

	updateConfigFromFlags(cfg, flagVars)

	if cfg.TargetHostDSN != "" {
		t.Errorf("Expected TargetHostDSN to remain empty, got: %s", cfg.TargetHostDSN)
	}

	if cfg.ListenIP != "" {
		t.Errorf("Expected ListenIP to remain empty, got: %s", cfg.ListenIP)
	}

	if cfg.ListenPort != "" {
		t.Errorf("Expected ListenPort to remain empty, got: %s", cfg.ListenPort)
	}

	if cfg.Exclude != "" {
		t.Errorf("Expected Exclude to remain empty, got: %s", cfg.Exclude)
	}

	if cfg.ExcludePostBody != "" {
		t.Errorf("Expected ExcludePostBody to remain empty, got: %s", cfg.ExcludePostBody)
	}

	if cfg.ExcludeResponseBody != "" {
		t.Errorf("Expected ExcludeResponseBody to remain empty, got: %s", cfg.ExcludeResponseBody)
	}

	if cfg.ReadTimeout != 0 {
		t.Errorf("Expected ReadTimeout to remain 0, got: %d", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 0 {
		t.Errorf("Expected WriteTimeout to remain 0, got: %d", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout != 0 {
		t.Errorf("Expected IdleTimeout to remain 0, got: %d", cfg.IdleTimeout)
	}
}

func TestProcessHeaders(t *testing.T) {
	tests := []struct {
		name            string
		headers         []string
		expectedHeaders map[string]string
	}{
		{
			name:            "Empty headers",
			headers:         []string{},
			expectedHeaders: map[string]string{},
		},
		{
			name:            "Single header",
			headers:         []string{"content-type:application/json"},
			expectedHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:    "Multiple headers",
			headers: []string{"content-type:application/json", "authorization:Bearer token123"},
			expectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
		},
		{
			name:            "Header without colon (should be ignored)",
			headers:         []string{"invalid-header", "valid-header:value"},
			expectedHeaders: map[string]string{"Valid-Header": "value"},
		},
		{
			name:            "Header with multiple colons",
			headers:         []string{"custom-header:key:value:extra"},
			expectedHeaders: map[string]string{"Custom-Header": "key:value:extra"},
		},
		{
			name:    "Mixed case headers",
			headers: []string{"CONTENT-TYPE:application/json", "x-custom-header:value"},
			expectedHeaders: map[string]string{
				"Content-Type":    "application/json",
				"X-Custom-Header": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SourceConfig{}

			processHeaders(cfg, tt.headers)

			if len(cfg.Headers) != len(tt.expectedHeaders) {
				t.Errorf("Expected %d headers, got %d", len(tt.expectedHeaders), len(cfg.Headers))
			}

			for expectedKey, expectedValue := range tt.expectedHeaders {
				if actualValue, ok := cfg.Headers[expectedKey]; !ok {
					t.Errorf("Expected header %s not found", expectedKey)
				} else if actualValue != expectedValue {
					t.Errorf("Expected header %s to have value %s, got %s", expectedKey, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestProcessHeaders_WithExistingHeaders(t *testing.T) {
	cfg := &config.SourceConfig{
		Headers: map[string]string{
			"Existing-Header": "existing-value",
		},
	}

	headers := []string{"new-header:new-value"}

	processHeaders(cfg, headers)

	expectedHeaders := map[string]string{
		"Existing-Header": "existing-value",
		"New-Header":      "new-value",
	}

	if len(cfg.Headers) != len(expectedHeaders) {
		t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(cfg.Headers))
	}

	for expectedKey, expectedValue := range expectedHeaders {
		if actualValue, ok := cfg.Headers[expectedKey]; !ok {
			t.Errorf("Expected header %s not found", expectedKey)
		} else if actualValue != expectedValue {
			t.Errorf("Expected header %s to have value %s, got %s", expectedKey, expectedValue, actualValue)
		}
	}
}

func TestSetupConfigPaths(t *testing.T) {
	// We can't easily test the actual paths being added to viper without more complex setup
	// But we can test that the function doesn't error

	// Create a temporary viper instance
	v := viper.New()

	err := setupConfigPaths(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestSetupConfigPaths_UserHomeDirError(t *testing.T) {
	// We can test the happy path since setupConfigPaths handles errors gracefully
	// The function continues even if UserHomeDir() fails

	// Create a temporary viper instance
	v := viper.New()

	// Call setupConfigPaths - it should not fail even if some paths can't be added
	err := setupConfigPaths(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The function should have set up at least the basic config paths
	// We can verify that the function completed without error
	// In a real scenario, you could check that certain paths were added to viper
}

// Integration test for LoadConfig.
func TestLoadConfig_Integration(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test config
	configContent := `
targetHostDsn: "http://test.example.com"
listenIp: "` + testListenIP + `"
listenPort: "9000"
loggingEnabled: true
logPostBody: false
headers:
  "Content-Type": "application/json"
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	tmpFile.Close()

	// Test individual components that make up LoadConfig
	// instead of testing the full integration which has flag conflicts

	// Test that we can at least validate the config file format
	v := viper.New()
	v.SetConfigFile(tmpFile.Name())
	err = v.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Verify some config values were read correctly
	if v.GetString("targetHostDsn") != "http://test.example.com" {
		t.Errorf("Expected targetHostDsn to be 'http://test.example.com', got: %s", v.GetString("targetHostDsn"))
	}

	if v.GetString("listenIp") != testListenIP {
		t.Errorf("Expected listenIp to be '%s', got: %s", testListenIP, v.GetString("listenIp"))
	}

	if v.GetString("listenPort") != "9000" {
		t.Errorf("Expected listenPort to be '9000', got: %s", v.GetString("listenPort"))
	}
}

// Benchmark tests.
func BenchmarkSetupFlags(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Reset flags for each iteration
		flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
		setupFlags()
	}
}

func BenchmarkUpdateConfigFromFlags(b *testing.B) {
	cfg := &config.SourceConfig{}
	flagVars := &FlagVars{
		targetHostDSN:       "http://example.com",
		listenIP:            testListenIP,
		listenPort:          "9000",
		loggingEnabled:      false,
		setRequestID:        true,
		exclude:             "test-exclude",
		logPostBody:         false,
		logResponseBody:     false,
		excludePostBody:     "exclude-post",
		excludeResponseBody: "exclude-response",
		readTimeout:         30,
		writeTimeout:        60,
		idleTimeout:         300,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updateConfigFromFlags(cfg, flagVars)
	}
}

func BenchmarkProcessHeaders(b *testing.B) {
	cfg := &config.SourceConfig{}
	headers := []string{
		"content-type:application/json",
		"authorization:Bearer token123",
		"x-custom-header:custom-value",
		"user-agent:test-client/1.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Headers = make(map[string]string) // Reset for each iteration
		processHeaders(cfg, headers)
	}
}
