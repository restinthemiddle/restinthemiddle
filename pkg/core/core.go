package core

import (
	"fmt"
	"log"
	"net/http"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	proxy "github.com/restinthemiddle/restinthemiddle/pkg/core/proxy"
)

var cfg *config.TranslatedConfig
var wrt Writer
var proxyServer *proxy.Server
var server HTTPServer

// Run starts the proxy server
func Run(c *config.TranslatedConfig, w Writer, s HTTPServer) {
	cfg = c
	wrt = w
	server = s

	var err error
	proxyServer, err = proxy.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create proxy server: %v", err)
	}
	proxyServer.SetModifyResponse(logResponse)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)

	// Set the handler in the server
	if err := server.ListenAndServe(fmt.Sprintf("%s:%s", cfg.ListenIP, cfg.ListenPort), mux); err != nil {
		log.Fatalf("%v", err)
	}
}

func handleRequest(response http.ResponseWriter, request *http.Request) {
	proxyServer.ServeHTTP(response, request)
}

func logResponse(response *http.Response) (err error) {
	if !cfg.LoggingEnabled {
		return nil
	}

	if cfg.ExcludeRegexp != nil && cfg.ExcludeRegexp.String() != "" && cfg.ExcludeRegexp.MatchString(response.Request.URL.Path) {
		return nil
	}

	return wrt.LogResponse(response)
}
