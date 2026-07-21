package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// HTTPServer defines the interface for an HTTP server.
type HTTPServer interface {
	ListenAndServe(addr string, handler http.Handler) error
}

// DefaultHTTPServer is the default implementation of the HTTPServer interface.
type DefaultHTTPServer struct {
	mu     sync.Mutex
	server *http.Server
}

// newServer builds the http.Server from the translated configuration,
// including TLS settings when a certificate is configured.
func newServer(addr string, handler http.Handler) (*http.Server, error) {
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
			return nil, fmt.Errorf("failed to load TLS certificate/key pair: %w", err)
		}
		server.TLSConfig = &tls.Config{
			MinVersion:   cfg.TLSMinVersion,
			Certificates: []tls.Certificate{cert},
		}
	}

	return server, nil
}

// ListenAndServe implements the HTTPServer interface.
func (s *DefaultHTTPServer) ListenAndServe(addr string, handler http.Handler) error {
	server, err := newServer(addr, handler)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.server = server
	s.mu.Unlock()

	if server.TLSConfig != nil {
		return server.ServeTLS(listener, "", "")
	}
	return server.Serve(listener)
}

// Shutdown gracefully stops a running server. It is a no-op if the server
// was never started.
func (s *DefaultHTTPServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	server := s.server
	s.mu.Unlock()

	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}
