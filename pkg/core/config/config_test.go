package core_config

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewTranslatedConfiguration(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN: "http://example.com",
		ListenIP:      "127.0.0.1",
		ListenPort:    "8080",
		Headers: map[string]string{
			"X-Test":  "test",
			"X-Test2": "test2",
		},
		LoggingEnabled:  true,
		SetRequestID:    true,
		LogPostBody:     false,
		LogResponseBody: false,
		ReadTimeout:     10,
		WriteTimeout:    20,
		IdleTimeout:     300,
	}

	translated, err := cfg.NewTranslatedConfiguration()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if translated.TargetURL.String() != "http://example.com" {
		t.Errorf("Expected URL http://example.com, got %s", translated.TargetURL.String())
	}

	if translated.ListenIP != "127.0.0.1" {
		t.Errorf("Expected IP 127.0.0.1, got %s", translated.ListenIP)
	}

	if translated.ListenPort != "8080" {
		t.Errorf("Expected port 8080, got %s", translated.ListenPort)
	}

	if !translated.LoggingEnabled {
		t.Error("Logging should be enabled")
	}

	if translated.LogPostBody {
		t.Error("POST body should not be logged")
	}

	if translated.LogResponseBody {
		t.Error("Response body should not be logged")
	}

	if !translated.SetRequestID {
		t.Error("RequestID should be set")
	}

	if translated.Headers["X-Test"] != "test" {
		t.Errorf("Expected header X-Test: test, got %s", translated.Headers["X-Test"])
	}

	if translated.Headers["X-Test2"] != "test2" {
		t.Errorf("Expected header X-Test2: test2, got %s", translated.Headers["X-Test2"])
	}

	if translated.ReadTimeout != 10*time.Second {
		t.Errorf("Expected ReadTimeout 10s, got %v", translated.ReadTimeout)
	}

	if translated.WriteTimeout != 20*time.Second {
		t.Errorf("Expected WriteTimeout 20s, got %v", translated.WriteTimeout)
	}

	if translated.IdleTimeout != 300*time.Second {
		t.Errorf("Expected IdleTimeout 300s, got %v", translated.IdleTimeout)
	}
}

func TestNewTranslatedConfigurationWithDefaultTimeouts(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN: "http://example.com",
	}

	translated, err := cfg.NewTranslatedConfiguration()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if translated.ReadTimeout != 5*time.Second {
		t.Errorf("Expected default ReadTimeout 5s, got %v", translated.ReadTimeout)
	}

	if translated.WriteTimeout != 10*time.Second {
		t.Errorf("Expected default WriteTimeout 10s, got %v", translated.WriteTimeout)
	}

	if translated.IdleTimeout != 120*time.Second {
		t.Errorf("Expected default IdleTimeout 120s, got %v", translated.IdleTimeout)
	}
}

func TestNewTranslatedConfigurationWithInvalidURL(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN: "://invalid",
	}

	translated, err := cfg.NewTranslatedConfiguration()
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	if translated != nil {
		t.Error("TranslatedConfig should be nil for invalid URL")
	}
}

func TestNewTranslatedConfigurationWithEmptyConfig(t *testing.T) {
	cfg := &SourceConfig{}

	translated, err := cfg.NewTranslatedConfiguration()
	if err == nil {
		t.Error("Expected error for empty configuration")
	}
	if translated != nil {
		t.Error("TranslatedConfig should be nil for empty configuration")
	}
}

func TestGetExcludeRegexp(t *testing.T) {
	tests := []struct {
		name           string
		exclude        string
		expectedRegexp *regexp.Regexp
		shouldMatch    []string
		shouldNotMatch []string
	}{
		{
			name:           "Empty exclude string",
			exclude:        "",
			expectedRegexp: nil,
		},
		{
			name:           "Valid regexp pattern",
			exclude:        `^/api/.*`,
			expectedRegexp: regexp.MustCompile(`^/api/.*`),
			shouldMatch:    []string{"/api/users", "/api/posts"},
			shouldNotMatch: []string{"/web/users", "/static/css"},
		},
		{
			name:           "Complex regexp pattern",
			exclude:        `\.(css|js|png|jpg)$`,
			expectedRegexp: regexp.MustCompile(`\.(css|js|png|jpg)$`),
			shouldMatch:    []string{"/style.css", "/script.js", "/image.png"},
			shouldNotMatch: []string{"/api/users", "/index.html"},
		},
		{
			name:           "Invalid regexp pattern",
			exclude:        `[invalid`,
			expectedRegexp: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExcludeRegexp(tt.exclude)

			if tt.expectedRegexp == nil {
				if result != nil {
					t.Errorf("Expected nil regexp, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected regexp, got nil")
				return
			}

			if result.String() != tt.expectedRegexp.String() {
				t.Errorf("Expected regexp %s, got %s", tt.expectedRegexp.String(), result.String())
			}

			// Test matching behavior
			for _, text := range tt.shouldMatch {
				if !result.MatchString(text) {
					t.Errorf("Expected %s to match pattern %s", text, tt.exclude)
				}
			}

			for _, text := range tt.shouldNotMatch {
				if result.MatchString(text) {
					t.Errorf("Expected %s to NOT match pattern %s", text, tt.exclude)
				}
			}
		})
	}
}

func TestPrintConfig(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &SourceConfig{
		TargetHostDSN:       "http://example.com",
		ListenIP:            "127.0.0.1",
		ListenPort:          "8080",
		Headers:             map[string]string{"X-Test": "test"},
		LoggingEnabled:      true,
		SetRequestID:        true,
		Exclude:             "^/api/.*",
		LogPostBody:         false,
		LogResponseBody:     false,
		ExcludePostBody:     "^/secure/.*",
		ExcludeResponseBody: "^/private/.*",
		ReadTimeout:         10,
		WriteTimeout:        20,
		IdleTimeout:         300,
	}

	cfg.PrintConfig()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy output: %v", err)
	}
	output := buf.String()

	// Check that the output contains expected content
	expectedContents := []string{
		"restinthemiddle", // Version info should be present
		"YAML configuration:",
		"targetHostDsn: http://example.com",
		"listenIp: 127.0.0.1",
		"listenPort: \"8080\"",
		"loggingEnabled: true",
		"setRequestId: true",
		"exclude: ^/api/.*",
		"logPostBody: false",
		"logResponseBody: false",
		"excludePostBody: ^/secure/.*",
		"excludeResponseBody: ^/private/.*",
		"readTimeout: 10",
		"writeTimeout: 20",
		"idleTimeout: 300",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", expected, output)
		}
	}
}

func TestNewTranslatedConfigurationWithRegexps(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN:       "http://example.com",
		Exclude:             "^/api/.*",
		ExcludePostBody:     "^/secure/.*",
		ExcludeResponseBody: "^/private/.*",
	}

	translated, err := cfg.NewTranslatedConfiguration()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test ExcludeRegexp
	if translated.ExcludeRegexp == nil {
		t.Error("Expected ExcludeRegexp to be set")
	} else {
		if !translated.ExcludeRegexp.MatchString("/api/users") {
			t.Error("Expected ExcludeRegexp to match /api/users")
		}
		if translated.ExcludeRegexp.MatchString("/web/users") {
			t.Error("Expected ExcludeRegexp to NOT match /web/users")
		}
	}

	// Test ExcludePostBodyRegexp
	if translated.ExcludePostBodyRegexp == nil {
		t.Error("Expected ExcludePostBodyRegexp to be set")
	} else {
		if !translated.ExcludePostBodyRegexp.MatchString("/secure/data") {
			t.Error("Expected ExcludePostBodyRegexp to match /secure/data")
		}
		if translated.ExcludePostBodyRegexp.MatchString("/public/data") {
			t.Error("Expected ExcludePostBodyRegexp to NOT match /public/data")
		}
	}

	// Test ExcludeResponseBodyRegexp
	if translated.ExcludeResponseBodyRegexp == nil {
		t.Error("Expected ExcludeResponseBodyRegexp to be set")
	} else {
		if !translated.ExcludeResponseBodyRegexp.MatchString("/private/info") {
			t.Error("Expected ExcludeResponseBodyRegexp to match /private/info")
		}
		if translated.ExcludeResponseBodyRegexp.MatchString("/public/info") {
			t.Error("Expected ExcludeResponseBodyRegexp to NOT match /public/info")
		}
	}
}

func TestNewTranslatedConfigurationWithInvalidRegexps(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN:       "http://example.com",
		Exclude:             "[invalid",
		ExcludePostBody:     "[invalid",
		ExcludeResponseBody: "[invalid",
	}

	translated, err := cfg.NewTranslatedConfiguration()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Invalid regexps should result in nil
	if translated.ExcludeRegexp != nil {
		t.Error("Expected ExcludeRegexp to be nil for invalid regexp")
	}

	if translated.ExcludePostBodyRegexp != nil {
		t.Error("Expected ExcludePostBodyRegexp to be nil for invalid regexp")
	}

	if translated.ExcludeResponseBodyRegexp != nil {
		t.Error("Expected ExcludeResponseBodyRegexp to be nil for invalid regexp")
	}
}

func TestNewTranslatedConfigurationWithEmptyRegexps(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN:       "http://example.com",
		Exclude:             "",
		ExcludePostBody:     "",
		ExcludeResponseBody: "",
	}

	translated, err := cfg.NewTranslatedConfiguration()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Empty regexps should result in nil
	if translated.ExcludeRegexp != nil {
		t.Error("Expected ExcludeRegexp to be nil for empty regexp")
	}

	if translated.ExcludePostBodyRegexp != nil {
		t.Error("Expected ExcludePostBodyRegexp to be nil for empty regexp")
	}

	if translated.ExcludeResponseBodyRegexp != nil {
		t.Error("Expected ExcludeResponseBodyRegexp to be nil for empty regexp")
	}
}
