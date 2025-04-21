package proxy

import (
	"net/http"
	"net/http/httputil"

	"github.com/restinthemiddle/restinthemiddle/pkg/core"
	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

// Writer interface moves to pkg/writer
type Writer interface {
	LogResponse(response *http.Response) error
}

type Server struct {
	cfg    *config.TranslatedConfig
	writer core.Writer
	proxy  *httputil.ReverseProxy
}

func NewServer(cfg *config.TranslatedConfig, writer Writer) *Server {
	return &Server{
		cfg:    cfg,
		writer: writer,
		proxy:  newReverseProxy(cfg),
	}
}

func newReverseProxy(cfg *config.TranslatedConfig) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = cfg.TargetURL.Scheme
			req.URL.Host = cfg.TargetURL.Host
			req.Host = cfg.TargetURL.Host
		},
	}
}
