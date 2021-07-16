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
	log.Println("starting restinthemiddle")
	fmt.Println("YAML configuration:")
	yamlString, _ := yaml.Marshal(c)
	fmt.Printf("YAML:\n%s\n", string(yamlString))
}
