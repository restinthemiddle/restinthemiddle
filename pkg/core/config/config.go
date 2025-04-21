package core_config

import (
	"fmt"
	"log"
	"net/url"
	"regexp"

	yaml "gopkg.in/yaml.v3"
)

// SourceConfig holds the raw core configuration
type SourceConfig struct {
	TargetHostDsn       string            `yaml:"targetHostDsn"`
	ListenIp            string            `yaml:"listenIp"`
	ListenPort          string            `yaml:"listenPort"`
	Headers             map[string]string `yaml:"headers,omitempty"`
	LoggingEnabled      bool              `yaml:"loggingEnabled"`
	SetRequestId        bool              `yaml:"setRequestId"`
	Exclude             string            `yaml:"exclude"`
	LogPostBody         bool              `yaml:"logPostBody"`
	LogResponseBody     bool              `yaml:"logResponseBody"`
	ExcludePostBody     string            `yaml:"excludePostBody"`
	ExcludeResponseBody string            `yaml:"excludeResponseBody"`
}

// TranslatedConfig holds the compiled core configuration
type TranslatedConfig struct {
	TargetURL                 *url.URL
	ListenIp                  string
	ListenPort                string
	Headers                   map[string]string
	LoggingEnabled            bool
	SetRequestId              bool
	ExcludeRegexp             *regexp.Regexp
	LogPostBody               bool
	LogResponseBody           bool
	ExcludePostBodyRegexp     *regexp.Regexp
	ExcludeResponseBodyRegexp *regexp.Regexp
}

func (s *SourceConfig) NewTranslatedConfiguration() *TranslatedConfig {
	return &TranslatedConfig{
		TargetURL:                 getTargetURL(s.TargetHostDsn),
		ListenIp:                  s.ListenIp,
		ListenPort:                s.ListenPort,
		Headers:                   s.Headers,
		LoggingEnabled:            s.LoggingEnabled,
		SetRequestId:              s.SetRequestId,
		ExcludeRegexp:             getExcludeRegexp(s.Exclude),
		LogPostBody:               s.LogPostBody,
		LogResponseBody:           s.LogResponseBody,
		ExcludePostBodyRegexp:     getExcludeRegexp(s.ExcludePostBody),
		ExcludeResponseBodyRegexp: getExcludeRegexp(s.ExcludeResponseBody),
	}
}

func getExcludeRegexp(exclude string) *regexp.Regexp {
	regex, err := regexp.Compile(exclude)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	return regex
}

func getTargetURL(targetHostDsn string) *url.URL {
	url, err := url.Parse(targetHostDsn)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	return url
}

// PrintConfig logs the env variables required for a reverse proxy
func (s *SourceConfig) PrintConfig() {
	fmt.Println("YAML configuration:")
	yamlString, _ := yaml.Marshal(s)
	fmt.Printf("%s\n", string(yamlString))
}
