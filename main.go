package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jensschulze/restinthemiddle/core"
	"github.com/jensschulze/restinthemiddle/logwriter"
	"github.com/spf13/viper"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	viper.SetDefault("targetHostDsn", "http://host.docker.internal:8081")
	viper.SetDefault("listenIp", "0.0.0.0")
	viper.SetDefault("listenPort", "8000")
	viper.SetDefault("headers", map[string]string{"User-Agent": "Rest in the middle logging proxy"})
	viper.SetDefault("loggingEnabled", true)
	viper.SetDefault("setRequestId", false)
	viper.SetDefault("exclude", "")

	viper.BindEnv("targetHostDsn", "TARGET_HOST_DSN")
	viper.BindEnv("listenIp", "LISTEN_IP")
	viper.BindEnv("listenPort", "LISTEN_PORT", "PORT")
	viper.BindEnv("loggingEnabled", "LOGGING_ENABLED")
	viper.BindEnv("setRequestId", "SET_REQUEST_ID")
	viper.BindEnv("excluded", "EXCLUDED")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("/etc/restinthemiddle")
	viper.AddConfigPath("/restinthemiddle")
	viper.AddConfigPath(".")
	viper.AddConfigPath(homeDir + "/restinthemiddle")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	config := core.Config{}

	if err := viper.Unmarshal(&config); err != nil {
		log.Panicf("unable to decode into struct, %v", err)
	}

	headersProcessed := map[string]string{"User-Agent": "Rest in the middle logging proxy"}
	for k, v := range config.Headers {
		headersProcessed[strings.Title(strings.ToLower(k))] = v
	}
	config.Headers = headersProcessed

	config.PrintConfig()

	configFileUsed := viper.ConfigFileUsed()
	if len(configFileUsed) > 0 {
		fmt.Printf("Config File: %s\n", configFileUsed)
	}

	w := logwriter.Writer{}

	core.Run(&config, &w)
}
