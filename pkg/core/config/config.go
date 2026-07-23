package core_config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/restinthemiddle/restinthemiddle/internal/version"
	yaml "gopkg.in/yaml.v3"
)

// Default configuration values.
const (
	DefaultTargetHostDSN       = ""
	DefaultListenIP            = "0.0.0.0"
	DefaultListenPort          = "8000"
	DefaultMetricsEnabled      = true
	DefaultMetricsPort         = "9090"
	DefaultLoggingEnabled      = true
	DefaultSetRequestID        = false
	DefaultExclude             = ""
	DefaultLogPostBody         = true
	DefaultLogResponseBody     = true
	DefaultExcludePostBody     = ""
	DefaultExcludeResponseBody = ""
	DefaultReadTimeout         = 0
	DefaultReadHeaderTimeout   = 0
	DefaultWriteTimeout        = 0
	DefaultIdleTimeout         = 0
	TLSVersion12               = "1.2"
	TLSVersion13               = "1.3"
)

// Common configuration errors.
var (
	ErrEmptyTargetHostDSN       = errors.New("target host DSN is empty")
	ErrInvalidTargetHostDSN     = errors.New("invalid target host DSN")
	ErrTLSFilePairIncomplete    = errors.New("both TLSCertFile and TLSKeyFile must be provided for TLS")
	ErrUnsupportedTLSMinVersion = errors.New("unsupported TLS minimum version")
)

// SourceConfig holds the raw core configuration.
type SourceConfig struct {
	TargetHostDSN       string            `yaml:"targetHostDsn"`
	ListenIP            string            `yaml:"listenIp"`
	ListenPort          string            `yaml:"listenPort"`
	MetricsEnabled      bool              `yaml:"metricsEnabled"`
	MetricsPort         string            `yaml:"metricsPort"`
	Headers             map[string]string `yaml:"headers,omitempty"`
	LoggingEnabled      bool              `yaml:"loggingEnabled"`
	SetRequestID        bool              `yaml:"setRequestId"`
	Exclude             string            `yaml:"exclude"`
	LogPostBody         bool              `yaml:"logPostBody"`
	LogResponseBody     bool              `yaml:"logResponseBody"`
	ExcludePostBody     string            `yaml:"excludePostBody"`
	ExcludeResponseBody string            `yaml:"excludeResponseBody"`
	ReadTimeout         int               `yaml:"readTimeout"`
	WriteTimeout        int               `yaml:"writeTimeout"`
	IdleTimeout         int               `yaml:"idleTimeout"`
	ReadHeaderTimeout   int               `yaml:"readHeaderTimeout"`
	TLSCertFile         string            `yaml:"tlsCertFile"`
	TLSKeyFile          string            `yaml:"tlsKeyFile"`
	TLSMinVersion       string            `yaml:"tlsMinVersion"`
}

// TranslatedConfig holds the compiled core configuration.
type TranslatedConfig struct {
	TargetURL                 *url.URL
	ListenIP                  string
	ListenPort                string
	MetricsEnabled            bool
	MetricsPort               string
	Headers                   map[string]string
	LoggingEnabled            bool
	SetRequestID              bool
	ExcludeRegexp             *regexp.Regexp
	LogPostBody               bool
	LogResponseBody           bool
	ExcludePostBodyRegexp     *regexp.Regexp
	ExcludeResponseBodyRegexp *regexp.Regexp
	ReadTimeout               time.Duration
	WriteTimeout              time.Duration
	IdleTimeout               time.Duration
	ReadHeaderTimeout         time.Duration
	TLSCertFile               string
	TLSKeyFile                string
	TLSMinVersion             uint16
}

func (s *SourceConfig) NewTranslatedConfiguration() (*TranslatedConfig, error) {
	if s.TargetHostDSN == "" {
		return nil, ErrEmptyTargetHostDSN
	}

	if (s.TLSCertFile != "" && s.TLSKeyFile == "") || (s.TLSCertFile == "" && s.TLSKeyFile != "") {
		return nil, ErrTLSFilePairIncomplete
	}

	targetURL := getTargetURL(s.TargetHostDSN)
	if targetURL == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTargetHostDSN, s.TargetHostDSN)
	}

	// Use configured timeout values directly.
	// Note: A value of 0 means no timeout (same as net/http.Server default).
	readTimeout := s.ReadTimeout
	writeTimeout := s.WriteTimeout
	idleTimeout := s.IdleTimeout
	readHeaderTimeout := s.ReadHeaderTimeout

	// TLS Min Version mapping
	var minVersion uint16
	switch s.TLSMinVersion {
	case "", TLSVersion12:
		minVersion = tls.VersionTLS12
	case TLSVersion13:
		minVersion = tls.VersionTLS13
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedTLSMinVersion, s.TLSMinVersion)
	}

	return &TranslatedConfig{
		TargetURL:                 targetURL,
		ListenIP:                  s.ListenIP,
		ListenPort:                s.ListenPort,
		MetricsEnabled:            s.MetricsEnabled,
		MetricsPort:               s.MetricsPort,
		Headers:                   s.Headers,
		LoggingEnabled:            s.LoggingEnabled,
		SetRequestID:              s.SetRequestID,
		ExcludeRegexp:             getExcludeRegexp(s.Exclude),
		LogPostBody:               s.LogPostBody,
		LogResponseBody:           s.LogResponseBody,
		ExcludePostBodyRegexp:     getExcludeRegexp(s.ExcludePostBody),
		ExcludeResponseBodyRegexp: getExcludeRegexp(s.ExcludeResponseBody),
		ReadTimeout:               time.Duration(readTimeout) * time.Second,
		WriteTimeout:              time.Duration(writeTimeout) * time.Second,
		IdleTimeout:               time.Duration(idleTimeout) * time.Second,
		ReadHeaderTimeout:         time.Duration(readHeaderTimeout) * time.Second,
		TLSCertFile:               s.TLSCertFile,
		TLSKeyFile:                s.TLSKeyFile,
		TLSMinVersion:             minVersion,
	}, nil
}

func getExcludeRegexp(exclude string) *regexp.Regexp {
	if exclude == "" {
		return nil
	}
	regex, err := regexp.Compile(exclude)
	if err != nil {
		return nil
	}
	return regex
}

func getTargetURL(targetHostDsn string) *url.URL {
	url, err := url.Parse(targetHostDsn)
	if err != nil {
		return nil
	}
	return url
}

// PrintConfig logs the configuration and version information.
func (s *SourceConfig) PrintConfig() {
	fmt.Printf("%s\n\n", version.Info())
	fmt.Println("YAML configuration:")
	yamlString, _ := yaml.Marshal(s)
	fmt.Printf("%s\n", string(yamlString))
}
