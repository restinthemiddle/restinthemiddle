package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

// HTTPServer defines the interface for an HTTP server.
type HTTPServer interface {
	ListenAndServe(addr string, handler http.Handler) error
}

// DefaultHTTPServer is the default implementation of the HTTPServer interface.
type DefaultHTTPServer struct {
	cfg    *config.TranslatedConfig
	mu     sync.Mutex
	server *http.Server
}

// NewDefaultHTTPServer creates a DefaultHTTPServer for the given configuration.
func NewDefaultHTTPServer(cfg *config.TranslatedConfig) *DefaultHTTPServer {
	return &DefaultHTTPServer{cfg: cfg}
}

// newServer builds the http.Server from the configuration, including TLS
// settings when a certificate is configured.
func (s *DefaultHTTPServer) newServer(addr string, handler http.Handler) (*http.Server, error) {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       s.cfg.ReadTimeout,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}

	// Config validation guarantees cert and key are either both set or both empty.
	if s.cfg.TLSCertFile != "" {
		cert, err := tls.LoadX509KeyPair(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate/key pair: %w", err)
		}
		server.TLSConfig = &tls.Config{
			MinVersion:   s.cfg.TLSMinVersion,
			Certificates: []tls.Certificate{cert},
		}
	}

	return server, nil
}

// ListenAndServe implements the HTTPServer interface.
func (s *DefaultHTTPServer) ListenAndServe(addr string, handler http.Handler) error {
	server, err := s.newServer(addr, handler)
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
