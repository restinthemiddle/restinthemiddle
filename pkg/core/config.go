package core

import (
	"fmt"
	"log"

	yaml "gopkg.in/yaml.v3"
)

// Config holds the core configuration
type Config struct {
	TargetHostDsn   string            `yaml:"targetHostDsn"`
	ListenIp        string            `yaml:"listenIp"`
	ListenPort      string            `yaml:"listenPort"`
	Headers         map[string]string `yaml:"headers,omitempty"`
	LoggingEnabled  bool              `yaml:"loggingEnabled"`
	SetRequestId    bool              `yaml:"setRequestId"`
	Exclude         string            `yaml:"exclude"`
	LogPostBody     bool              `yaml:"logPostBody"`
	LogResponseBody bool              `yaml:"logResponseBody"`
}

// PrintConfig logs the env variables required for a reverse proxy
func (c *Config) PrintConfig() {
	log.Println("restinthemiddle started")
	fmt.Println("YAML configuration:")
	yamlString, _ := yaml.Marshal(c)
	fmt.Printf("%s\n", string(yamlString))
}
