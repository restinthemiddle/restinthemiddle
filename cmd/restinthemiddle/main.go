package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/restinthemiddle/restinthemiddle/internal/version"
	"github.com/restinthemiddle/restinthemiddle/internal/zapwriter"
	"github.com/restinthemiddle/restinthemiddle/pkg/core"
	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Default configuration values.
const (
	defaultTargetHostDSN       = ""
	defaultListenIP            = "0.0.0.0"
	defaultListenPort          = "8000"
	defaultLoggingEnabled      = true
	defaultSetRequestID        = false
	defaultExclude             = ""
	defaultLogPostBody         = true
	defaultLogResponseBody     = true
	defaultExcludePostBody     = ""
	defaultExcludeResponseBody = ""
	defaultReadTimeout         = 5
	defaultWriteTimeout        = 10
	defaultIdleTimeout         = 120
)

// App represents the application with configurable dependencies.
type App struct {
	ConfigLoader  ConfigLoader
	LoggerFactory LoggerFactory
	Writer        io.Writer
	Args          []string
}

// ConfigLoader defines the interface for loading configuration.
type ConfigLoader interface {
	Load(args []string) (*config.TranslatedConfig, error)
}

// LoggerFactory defines the interface for creating loggers.
type LoggerFactory interface {
	CreateLogger() (*zap.Logger, error)
}

// DefaultConfigLoader is the default implementation of ConfigLoader.
type DefaultConfigLoader struct{}

// DefaultLoggerFactory is the default implementation of LoggerFactory.
type DefaultLoggerFactory struct{}

// NewApp creates a new App with default dependencies.
func NewApp() *App {
	return &App{
		ConfigLoader:  &DefaultConfigLoader{},
		LoggerFactory: &DefaultLoggerFactory{},
		Writer:        os.Stdout,
		Args:          os.Args,
	}
}

// Run executes the main application logic.
func (a *App) Run() error {
	translatedConfig, err := a.ConfigLoader.Load(a.Args)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger, err := a.LoggerFactory.CreateLogger()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	w := zapwriter.Writer{Logger: logger, Config: translatedConfig}

	fmt.Fprintln(a.Writer, "restinthemiddle started.")

	core.Run(translatedConfig, w, &core.DefaultHTTPServer{})
	return nil
}

// CreateLogger creates a production logger with caller disabled.
func (f *DefaultLoggerFactory) CreateLogger() (*zap.Logger, error) {
	zapConfig := zap.NewProductionConfig()
	zapConfig.DisableCaller = true
	return zapConfig.Build()
}

// Load loads configuration from flags, environment variables, and config files.
func (l *DefaultConfigLoader) Load(args []string) (*config.TranslatedConfig, error) {
	return LoadConfig(args)
}

func main() {
	app := NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

// LoadConfig loads configuration from various sources.
func LoadConfig(args []string) (*config.TranslatedConfig, error) {
	flagVars := setupFlags()

	v := viper.New()
	setupViperDefaults(v)
	setupViperEnvBindings(v)

	// Bind all flags to viper
	if err := v.BindPFlags(flag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}

	// Parse arguments
	if err := flag.CommandLine.Parse(args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	// Setup config paths and read config file
	if err := setupConfigPaths(v); err != nil {
		return nil, err
	}

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

	// Update config with flag values
	updateConfigFromFlags(&cfg, flagVars)

	// Process headers
	processHeaders(&cfg, flagVars.headers)

	if cfg.TargetHostDSN == defaultTargetHostDSN {
		return nil, fmt.Errorf("no target host given")
	}

	cfg.PrintConfig()

	if configFile := v.ConfigFileUsed(); configFile != "" {
		fmt.Printf("Config File: %s\n", configFile)
	}

	translatedConfig, err := cfg.NewTranslatedConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to translate configuration: %w", err)
	}

	return translatedConfig, nil
}

// FlagVars holds all flag variables.
type FlagVars struct {
	headers             []string
	targetHostDSN       string
	listenIP            string
	listenPort          string
	loggingEnabled      bool
	setRequestID        bool
	exclude             string
	excludePostBody     string
	excludeResponseBody string
	logPostBody         bool
	logResponseBody     bool
	readTimeout         int
	writeTimeout        int
	idleTimeout         int
}

// setupFlags initializes all command line flags.
func setupFlags() *FlagVars {
	flagVars := &FlagVars{}

	// Set custom usage template with version information
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", version.Info())
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringSliceVar(&flagVars.headers, "header", []string{}, "HTTP header to set. You may use this flag multiple times.")
	flag.StringVar(&flagVars.targetHostDSN, "target-host-dsn", defaultTargetHostDSN, "Target host DSN to proxy requests to")
	flag.StringVar(&flagVars.listenIP, "listen-ip", defaultListenIP, "IP address to listen on")
	flag.StringVar(&flagVars.listenPort, "listen-port", defaultListenPort, "Port to listen on")
	flag.BoolVar(&flagVars.loggingEnabled, "logging-enabled", defaultLoggingEnabled, "Enable logging")
	flag.BoolVar(&flagVars.setRequestID, "set-request-id", defaultSetRequestID, "Set request ID")
	flag.StringVar(&flagVars.exclude, "exclude", defaultExclude, "Regex pattern to exclude from logging")
	flag.BoolVar(&flagVars.logPostBody, "log-post-body", defaultLogPostBody, "Log POST request body")
	flag.BoolVar(&flagVars.logResponseBody, "log-response-body", defaultLogResponseBody, "Log response body")
	flag.StringVar(&flagVars.excludePostBody, "exclude-post-body", defaultExcludePostBody, "Regex pattern to exclude from POST body logging")
	flag.StringVar(&flagVars.excludeResponseBody, "exclude-response-body", defaultExcludeResponseBody, "Regex pattern to exclude from response body logging")
	flag.IntVar(&flagVars.readTimeout, "read-timeout", defaultReadTimeout, "Read timeout in seconds")
	flag.IntVar(&flagVars.writeTimeout, "write-timeout", defaultWriteTimeout, "Write timeout in seconds")
	flag.IntVar(&flagVars.idleTimeout, "idle-timeout", defaultIdleTimeout, "Idle timeout in seconds")

	return flagVars
}

// setupViperDefaults sets up default values for viper.
func setupViperDefaults(v *viper.Viper) {
	defaults := map[string]interface{}{
		"targetHostDsn":       defaultTargetHostDSN,
		"listenIp":            defaultListenIP,
		"listenPort":          defaultListenPort,
		"headers":             make(map[string]string),
		"loggingEnabled":      defaultLoggingEnabled,
		"setRequestId":        defaultSetRequestID,
		"exclude":             defaultExclude,
		"logPostBody":         defaultLogPostBody,
		"logResponseBody":     defaultLogResponseBody,
		"excludePostBody":     defaultExcludePostBody,
		"excludeResponseBody": defaultExcludeResponseBody,
		"readTimeout":         defaultReadTimeout,
		"writeTimeout":        defaultWriteTimeout,
		"idleTimeout":         defaultIdleTimeout,
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

// setupViperEnvBindings sets up environment variable bindings for viper.
func setupViperEnvBindings(v *viper.Viper) {
	v.BindEnv("targetHostDsn", "TARGET_HOST_DSN")             //nolint:errcheck
	v.BindEnv("listenIp", "LISTEN_IP")                        //nolint:errcheck
	v.BindEnv("listenPort", "LISTEN_PORT")                    //nolint:errcheck
	v.BindEnv("headers", "HEADERS")                           //nolint:errcheck
	v.BindEnv("loggingEnabled", "LOGGING_ENABLED")            //nolint:errcheck
	v.BindEnv("setRequestId", "SET_REQUEST_ID")               //nolint:errcheck
	v.BindEnv("exclude", "EXCLUDE")                           //nolint:errcheck
	v.BindEnv("logPostBody", "LOG_POST_BODY")                 //nolint:errcheck
	v.BindEnv("logResponseBody", "LOG_RESPONSE_BODY")         //nolint:errcheck
	v.BindEnv("excludePostBody", "EXCLUDE_POST_BODY")         //nolint:errcheck
	v.BindEnv("excludeResponseBody", "EXCLUDE_RESPONSE_BODY") //nolint:errcheck
	v.BindEnv("readTimeout", "READ_TIMEOUT")                  //nolint:errcheck
	v.BindEnv("writeTimeout", "WRITE_TIMEOUT")                //nolint:errcheck
	v.BindEnv("idleTimeout", "IDLE_TIMEOUT")                  //nolint:errcheck
}

// setupConfigPaths sets up configuration file paths for viper.
func setupConfigPaths(v *viper.Viper) error {
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	v.AddConfigPath("/etc/restinthemiddle")
	v.AddConfigPath(homeDir + "/.restinthemiddle")
	v.AddConfigPath(".")

	return nil
}

// updateConfigFromFlags updates the configuration with flag values.
func updateConfigFromFlags(cfg *config.SourceConfig, flagVars *FlagVars) {
	if flagVars.targetHostDSN != defaultTargetHostDSN {
		cfg.TargetHostDSN = flagVars.targetHostDSN
	}
	if flagVars.listenIP != defaultListenIP {
		cfg.ListenIP = flagVars.listenIP
	}
	if flagVars.listenPort != defaultListenPort {
		cfg.ListenPort = flagVars.listenPort
	}
	cfg.LoggingEnabled = flagVars.loggingEnabled
	cfg.SetRequestID = flagVars.setRequestID
	if flagVars.exclude != defaultExclude {
		cfg.Exclude = flagVars.exclude
	}
	cfg.LogPostBody = flagVars.logPostBody
	cfg.LogResponseBody = flagVars.logResponseBody
	if flagVars.excludePostBody != defaultExcludePostBody {
		cfg.ExcludePostBody = flagVars.excludePostBody
	}
	if flagVars.excludeResponseBody != defaultExcludeResponseBody {
		cfg.ExcludeResponseBody = flagVars.excludeResponseBody
	}
	if flagVars.readTimeout != defaultReadTimeout {
		cfg.ReadTimeout = flagVars.readTimeout
	}
	if flagVars.writeTimeout != defaultWriteTimeout {
		cfg.WriteTimeout = flagVars.writeTimeout
	}
	if flagVars.idleTimeout != defaultIdleTimeout {
		cfg.IdleTimeout = flagVars.idleTimeout
	}
}

// processHeaders processes header flags and updates the configuration.
func processHeaders(cfg *config.SourceConfig, headers []string) {
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
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
}
