package core_config

import (
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
}

func (s *SourceConfig) NewTranslatedConfiguration() (*TranslatedConfig, error) {
	if s.TargetHostDSN == "" {
		return nil, fmt.Errorf("target host DSN is empty")
	}

	targetURL := getTargetURL(s.TargetHostDSN)
	if targetURL == nil {
		return nil, fmt.Errorf("invalid target host DSN: %s", s.TargetHostDSN)
	}

	// Use configured timeout values directly.
	// Note: A value of 0 means no timeout (same as net/http.Server default).
	readTimeout := s.ReadTimeout
	writeTimeout := s.WriteTimeout
	idleTimeout := s.IdleTimeout
	readHeaderTimeout := s.ReadHeaderTimeout

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
