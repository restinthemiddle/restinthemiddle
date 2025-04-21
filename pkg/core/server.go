package core

import (
	"net/http"
)

// HTTPServer defines the interface for an HTTP server.
type HTTPServer interface {
	ListenAndServe(addr string, handler http.Handler) error
}

// DefaultHTTPServer is the default implementation of the HTTPServer interface.
type DefaultHTTPServer struct{}

// ListenAndServe implements the HTTPServer interface.
func (s *DefaultHTTPServer) ListenAndServe(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return server.ListenAndServe()
}
