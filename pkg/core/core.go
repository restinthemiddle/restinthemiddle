package core

import (
	"fmt"
	"log"
	"net/http"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	proxy "github.com/restinthemiddle/restinthemiddle/pkg/core/proxy"
)

// Proxy wires the reverse proxy with its configuration and log writer.
type Proxy struct {
	cfg         *config.TranslatedConfig
	writer      Writer
	proxyServer *proxy.Server
}

// NewProxy creates a Proxy from the given configuration and writer.
func NewProxy(c *config.TranslatedConfig, w Writer) (*Proxy, error) {
	proxyServer, err := proxy.NewServer(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy server: %w", err)
	}

	p := &Proxy{cfg: c, writer: w, proxyServer: proxyServer}
	proxyServer.SetModifyResponse(p.logResponse)

	return p, nil
}

// Handler returns the HTTP handler serving the proxy.
func (p *Proxy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRequest)
	return mux
}

func (p *Proxy) handleRequest(response http.ResponseWriter, request *http.Request) {
	p.proxyServer.ServeHTTP(response, request)
}

func (p *Proxy) logResponse(response *http.Response) error {
	if !p.cfg.LoggingEnabled {
		return nil
	}

	if p.cfg.ExcludeRegexp != nil && p.cfg.ExcludeRegexp.MatchString(response.Request.URL.Path) {
		return nil
	}

	return p.writer.LogResponse(response)
}

// Run starts the proxy server and terminates the process on failure.
func Run(c *config.TranslatedConfig, w Writer, s HTTPServer) {
	if err := run(c, w, s); err != nil {
		log.Fatalf("%v", err)
	}
}

func run(c *config.TranslatedConfig, w Writer, s HTTPServer) error {
	p, err := NewProxy(c, w)
	if err != nil {
		return err
	}

	return s.ListenAndServe(fmt.Sprintf("%s:%s", c.ListenIP, c.ListenPort), p.Handler())
}
