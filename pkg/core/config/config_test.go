package core_config

import (
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
