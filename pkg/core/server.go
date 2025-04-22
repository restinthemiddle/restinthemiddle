package core

import "net/http"

// HTTPServer definiert das Interface für einen HTTP-Server
type HTTPServer interface {
	ListenAndServe(addr string, handler http.Handler) error
}

// DefaultHTTPServer ist die Standard-Implementierung des HTTPServer-Interfaces
type DefaultHTTPServer struct{}

// ListenAndServe implementiert das HTTPServer-Interface
func (s *DefaultHTTPServer) ListenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}
