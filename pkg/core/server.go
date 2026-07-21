package core

import (
	"crypto/tls"
	"fmt"
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
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	// Config validation guarantees cert and key are either both set or both empty.
	if cfg.TLSCertFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate/key pair: %w", err)
		}
		server.TLSConfig = &tls.Config{
			MinVersion:   cfg.TLSMinVersion,
			Certificates: []tls.Certificate{cert},
		}
		return server.ListenAndServeTLS("", "")
	}
	return server.ListenAndServe()
}
