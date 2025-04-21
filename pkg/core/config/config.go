package core_config

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// SourceConfig holds the raw core configuration.
type SourceConfig struct {
	TargetHostDSN       string            `yaml:"targetHostDsn"`
	ListenIP            string            `yaml:"listenIp"`
	ListenPort          string            `yaml:"listenPort"`
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
}

// TranslatedConfig holds the compiled core configuration.
type TranslatedConfig struct {
	TargetURL                 *url.URL
	ListenIP                  string
	ListenPort                string
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
}

func (s *SourceConfig) NewTranslatedConfiguration() (*TranslatedConfig, error) {
	if s.TargetHostDSN == "" {
		return nil, fmt.Errorf("target host DSN is empty")
	}

	targetURL := getTargetURL(s.TargetHostDSN)
	if targetURL == nil {
		return nil, fmt.Errorf("invalid target host DSN: %s", s.TargetHostDSN)
	}

	// Set default timeouts if not specified.
	readTimeout := 5
	if s.ReadTimeout > 0 {
		readTimeout = s.ReadTimeout
	}
	writeTimeout := 10
	if s.WriteTimeout > 0 {
		writeTimeout = s.WriteTimeout
	}
	idleTimeout := 120
	if s.IdleTimeout > 0 {
		idleTimeout = s.IdleTimeout
	}

	return &TranslatedConfig{
		TargetURL:                 targetURL,
		ListenIP:                  s.ListenIP,
		ListenPort:                s.ListenPort,
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

// PrintConfig logs the env variables required for a reverse proxy.
func (s *SourceConfig) PrintConfig() {
	fmt.Println("YAML configuration:")
	yamlString, _ := yaml.Marshal(s)
	fmt.Printf("%s\n", string(yamlString))
}
