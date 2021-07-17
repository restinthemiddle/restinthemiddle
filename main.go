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

	viper.RegisterAlias("targetHostDsn", "target_host_dsn")
	viper.RegisterAlias("listenAddress", "listen_address")
	viper.RegisterAlias("loggingEnabled", "logging_enabled")

	viper.SetDefault("targetHostDsn", "http://127.0.0.1:8081")
	viper.SetDefault("listenAddress", "0.0.0.0:8000")
	viper.SetDefault("headers", map[string]string{"User-Agent": "Rest in the middle logging proxy"})
	viper.SetDefault("loggingEnabled", true)
	viper.SetDefault("exclude", "")

	viper.AutomaticEnv()

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
