package core

import (
	"fmt"
	"log"
	"net/http"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	"github.com/restinthemiddle/restinthemiddle/pkg/core/proxy"
)

var cfg *config.TranslatedConfig
var wrt Writer

var proxyServer *proxy.Server

func handleRequest(response http.ResponseWriter, request *http.Request) {
	proxyServer.ServeHTTP(response, request)
}

func logResponse(response *http.Response) (err error) {
	if !cfg.LoggingEnabled {
		return nil
	}

	if cfg.ExcludeRegexp.String() != "" && cfg.ExcludeRegexp.MatchString(response.Request.URL.Path) {
		return nil
	}

	return wrt.LogResponse(response)
}

func Run(c *config.TranslatedConfig, w Writer) {
	cfg = c
	wrt = w

	proxyServer = proxy.NewServer(cfg)
	proxyServer.SetModifyResponse(logResponse)

	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.ListenIP, cfg.ListenPort), nil); err != nil {
		log.Fatalf("%s", err.Error())
	}
}
