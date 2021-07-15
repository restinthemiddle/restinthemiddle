package core

import (
	"encoding/json"
	"log"

	yaml "gopkg.in/yaml.v3"
)

// Config holds the core configuration
type Config struct {
	TargetHostDsn  string            `json:"targetHostDsn"`
	ListenAddress  string            `json:"listenAddress"`
	Headers        map[string]string `json:"headers"`
	LoggingEnabled bool              `json:"loggingEnabled"`
	Exclude        string            `json:"exclude"`
}

// PrintConfig logs the env variables required for a reverse proxy
func (c *Config) PrintConfig() {
	log.Printf("Listening on: %s\n", c.ListenAddress)
	log.Printf("Targeting server on: %s\n", c.TargetHostDsn)

	if c.Exclude != "" {
		log.Printf("Exclude pattern: %s\n", c.Exclude)
	}

	log.Printf("Logging enabled: %s",
		func() string {
			if c.LoggingEnabled {
				return "true"
			}

			return "false"
		}())

	log.Println("Overwriting headers:")
	for key, value := range c.Headers {
		log.Printf("  %s: %s", key, value)
	}

	jsonString, _ := json.Marshal(c)
	log.Printf("JSON: %s\n", string(jsonString))

	yamlString, _ := yaml.Marshal(c)
	log.Printf("YAML: %s\n", string(yamlString))
}
