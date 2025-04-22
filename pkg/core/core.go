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

// Run startet den Proxy-Server
func Run(c *config.TranslatedConfig, w Writer, s HTTPServer) {
	cfg = c
	wrt = w
	server = s

	proxyServer = proxy.NewServer(cfg)
	proxyServer.SetModifyResponse(logResponse)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)

	// Setze den Handler im Server
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
