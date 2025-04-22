package core_config

import (
	"testing"
)

func TestNewTranslatedConfiguration(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN: "http://example.com",
		ListenIP:      "127.0.0.1",
		ListenPort:    "8080",
		Headers: map[string]string{
			"X-Test": "test",
		},
		LoggingEnabled: true,
		SetRequestID:   true,
	}

	translated := cfg.NewTranslatedConfiguration()
	if translated == nil {
		t.Error("TranslatedConfig sollte nicht nil sein")
	}

	if translated.TargetURL.String() != "http://example.com" {
		t.Errorf("Erwartete URL http://example.com, got %s", translated.TargetURL.String())
	}

	if translated.ListenIP != "127.0.0.1" {
		t.Errorf("Erwartete IP 127.0.0.1, got %s", translated.ListenIP)
	}

	if translated.ListenPort != "8080" {
		t.Errorf("Erwarteter Port 8080, got %s", translated.ListenPort)
	}

	if !translated.LoggingEnabled {
		t.Error("Logging sollte aktiviert sein")
	}

	if !translated.SetRequestID {
		t.Error("RequestID sollte gesetzt werden")
	}

	if translated.Headers["X-Test"] != "test" {
		t.Errorf("Erwarteter Header X-Test: test, got %s", translated.Headers["X-Test"])
	}
}

func TestNewTranslatedConfigurationWithInvalidURL(t *testing.T) {
	cfg := &SourceConfig{
		TargetHostDSN: "://invalid",
	}

	translated := cfg.NewTranslatedConfiguration()
	if translated != nil {
		t.Error("TranslatedConfig sollte nil sein bei ungültiger URL")
	}
}

func TestNewTranslatedConfigurationWithEmptyConfig(t *testing.T) {
	cfg := &SourceConfig{}

	translated := cfg.NewTranslatedConfiguration()
	if translated != nil {
		t.Error("TranslatedConfig sollte nil sein bei leerer Konfiguration")
	}
}
