package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/restinthemiddle/restinthemiddle/internal/zapwriter"
	"github.com/restinthemiddle/restinthemiddle/pkg/core"
	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func main() {
	translatedConfig, err := Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer logger.Sync()

	w := zapwriter.Writer{Logger: logger, Config: translatedConfig}

	log.Println("restinthemiddle started.")

	core.Run(translatedConfig, w)
}

func Load() (*config.TranslatedConfig, error) {
	var headers []string
	var targetHostDsn, listenIp, listenPort string
	var loggingEnabled, setRequestId bool
	var exclude, excludePostBody, excludeResponseBody string
	var logPostBody, logResponseBody bool

	// Define flags
	flag.StringSliceVar(&headers, "header", []string{}, "HTTP header to set. You may use this flag multiple times.")
	flag.StringVar(&targetHostDsn, "target-host-dsn", "", "Target host DSN to proxy requests to")
	flag.StringVar(&listenIp, "listen-ip", "0.0.0.0", "IP address to listen on")
	flag.StringVar(&listenPort, "listen-port", "8000", "Port to listen on")
	flag.BoolVar(&loggingEnabled, "logging-enabled", true, "Enable logging")
	flag.BoolVar(&setRequestId, "set-request-id", false, "Set request ID")
	flag.StringVar(&exclude, "exclude", "", "Regex pattern to exclude from logging")
	flag.BoolVar(&logPostBody, "log-post-body", true, "Log POST request body")
	flag.BoolVar(&logResponseBody, "log-response-body", true, "Log response body")
	flag.StringVar(&excludePostBody, "exclude-post-body", "", "Regex pattern to exclude from POST body logging")
	flag.StringVar(&excludeResponseBody, "exclude-response-body", "", "Regex pattern to exclude from response body logging")

	// Define configuration defaults and bind environment variables in one go
	defaults := map[string]interface{}{
		"targetHostDsn":       "",
		"listenIp":            "0.0.0.0",
		"listenPort":          "8000",
		"headers":             make(map[string]string),
		"loggingEnabled":      true,
		"setRequestId":        false,
		"exclude":             "",
		"logPostBody":         true,
		"logResponseBody":     true,
		"excludePostBody":     "",
		"excludeResponseBody": "",
	}

	v := viper.New()

	// Set defaults and bind environment variables
	for key, value := range defaults {
		v.SetDefault(key, value)
		v.BindEnv(key, strings.ToUpper(key))
	}

	// Bind all flags to viper
	if err := v.BindPFlags(flag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}

	flag.Parse()

	// Set config name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	v.AddConfigPath("/etc/restinthemiddle")
	v.AddConfigPath(homeDir + "/.restinthemiddle")
	v.AddConfigPath(".")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg config.SourceConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Update config with flag values if they are set
	if targetHostDsn != "" {
		cfg.TargetHostDsn = targetHostDsn
	}
	if listenIp != "0.0.0.0" {
		cfg.ListenIp = listenIp
	}
	if listenPort != "8000" {
		cfg.ListenPort = listenPort
	}
	cfg.LoggingEnabled = loggingEnabled
	cfg.SetRequestId = setRequestId
	if exclude != "" {
		cfg.Exclude = exclude
	}
	cfg.LogPostBody = logPostBody
	cfg.LogResponseBody = logResponseBody
	if excludePostBody != "" {
		cfg.ExcludePostBody = excludePostBody
	}
	if excludeResponseBody != "" {
		cfg.ExcludeResponseBody = excludeResponseBody
	}

	// Process headers from command line
	for _, item := range headers {
		k, v, found := strings.Cut(item, ":")
		if found {
			cfg.Headers[k] = v
		}
	}

	// Process header cases
	titleCaser := cases.Title(language.AmericanEnglish)
	headersProcessed := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headersProcessed[titleCaser.String(strings.ToLower(k))] = v
	}
	cfg.Headers = headersProcessed

	if cfg.TargetHostDsn == "" {
		return nil, fmt.Errorf("no target host given")
	}

	cfg.PrintConfig()

	if configFile := v.ConfigFileUsed(); configFile != "" {
		fmt.Printf("Config File: %s\n", configFile)
	}

	return cfg.NewTranslatedConfiguration(), nil
}
