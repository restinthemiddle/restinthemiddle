package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/restinthemiddle/restinthemiddle/pkg/core"
	"github.com/restinthemiddle/restinthemiddle/pkg/zapwriter"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func main() {
	var targetHostDsn string
	var listenIp string
	var listenPort string
	var headers []string
	var loggingEnabled bool
	var setRequestId bool
	var exclude string
	var logPostBody bool
	var logResponseBody bool

	flag.StringVar(&targetHostDsn, "target-host-dsn", "", "The DSN of the target host in the form schema://username:password@hostname:port/basepath?query")
	flag.StringVar(&listenIp, "listen-ip", "0.0.0.0", "The IP on which Restinthemiddle listens for requests.")
	flag.StringVar(&listenPort, "listen-port", "8000", "The port on which Restinthemiddle listens for to requests.")
	flag.StringArrayVar(&headers, "header", make([]string, 0), "HTTP header to set. You may use this flag multiple times.")
	flag.BoolVar(&loggingEnabled, "logging-enabled", true, "")
	flag.BoolVar(&setRequestId, "set-request-id", false, "If not already present in the request, add an X-Request-Id header with a version 4 UUID.")
	flag.StringVar(&exclude, "exclude", "", "If the given URL path matches this Regular Expression the request/response will not be logged.")
	flag.BoolVar(&logPostBody, "log-post-body", true, "If the given URL path matches this Regular Expression the request/response will not be logged.")
	flag.BoolVar(&logResponseBody, "log-response-body", true, "If the given URL path matches this Regular Expression the request/response will not be logged.")

	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf(err.Error())
	}

	viper.SetDefault("targetHostDsn", "")
	viper.SetDefault("listenIp", "0.0.0.0")
	viper.SetDefault("listenPort", "8000")
	viper.SetDefault("headers", make(map[string]string, 0))
	viper.SetDefault("loggingEnabled", true)
	viper.SetDefault("setRequestId", false)
	viper.SetDefault("exclude", "")
	viper.SetDefault("logPostBody", true)
	viper.SetDefault("logResponseBody", true)

	viper.BindEnv("targetHostDsn", "TARGET_HOST_DSN")
	viper.BindEnv("listenIp", "LISTEN_IP")
	viper.BindEnv("listenPort", "LISTEN_PORT")
	viper.BindEnv("loggingEnabled", "LOGGING_ENABLED")
	viper.BindEnv("setRequestId", "SET_REQUEST_ID")
	viper.BindEnv("exclude", "EXCLUDE")
	viper.BindEnv("logPostBody", "LOG_POST_BODY")
	viper.BindEnv("logResponseBody", "LOG_RESPONSE_BODY")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("/etc/restinthemiddle")
	viper.AddConfigPath("/restinthemiddle")
	viper.AddConfigPath(".")
	viper.AddConfigPath(homeDir + "/restinthemiddle")

	viper.BindPFlag("targetHostDsn", flag.Lookup("target-host-dsn"))
	viper.BindPFlag("listenIp", flag.Lookup("listen-ip"))
	viper.BindPFlag("listenPort", flag.Lookup("listen-port"))
	viper.BindPFlag("loggingEnabled", flag.Lookup("logging-enabled"))
	viper.BindPFlag("setRequestId", flag.Lookup("set-request-id"))
	viper.BindPFlag("exclude", flag.Lookup("exclude"))
	viper.BindPFlag("logPostBody", flag.Lookup("log-post-body"))
	viper.BindPFlag("logResponseBody", flag.Lookup("log-response-body"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf(err.Error())
		}
	}

	config := core.Config{}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	for _, item := range headers {
		k, v, found := strings.Cut(item, ":")
		if found {
			config.Headers[k] = v
		}
	}

	titleCaser := cases.Title(language.AmericanEnglish)
	headersProcessed := make(map[string]string, 0)
	for k, v := range config.Headers {
		headersProcessed[titleCaser.String(strings.ToLower(k))] = v
	}
	config.Headers = headersProcessed

	if config.TargetHostDsn == "" {
		log.Fatalf("No target host given.")
	}

	config.PrintConfig()

	configFileUsed := viper.ConfigFileUsed()
	if len(configFileUsed) > 0 {
		fmt.Printf("Config File: %s\n", configFileUsed)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf(err.Error())
	}

	defer logger.Sync()

	w := zapwriter.Writer{Logger: logger, Config: &config}

	core.Run(&config, &w)
}
