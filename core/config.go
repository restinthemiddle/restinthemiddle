package core

import (
	"fmt"
	"log"

	yaml "gopkg.in/yaml.v3"
)

// Config holds the core configuration
type Config struct {
	TargetHostDsn  string
	ListenAddress  string
	Headers        map[string]string
	LoggingEnabled bool
	Exclude        string
}

// PrintConfig logs the env variables required for a reverse proxy
func (c *Config) PrintConfig() {
	log.Println("Restinthemiddle configuration")
	fmt.Printf("Listening on: %s\n", c.ListenAddress)
	fmt.Printf("Targeting server on: %s\n", c.TargetHostDsn)

	if c.Exclude != "" {
		fmt.Printf("Exclude pattern: %s\n", c.Exclude)
	}

	fmt.Printf("Logging enabled: %s",
		func() string {
			if c.LoggingEnabled {
				return "true"
			}

			return "false"
		}())

	fmt.Println("Overwriting headers:")
	for key, value := range c.Headers {
		fmt.Printf("  %s: %s", key, value)
	}

	yamlString, _ := yaml.Marshal(c)
	fmt.Printf("YAML:\n%s\n", string(yamlString))
}
