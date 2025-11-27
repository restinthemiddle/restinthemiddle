package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	"github.com/restinthemiddle/restinthemiddle/pkg/core/transport"
	"github.com/restinthemiddle/restinthemiddle/pkg/metrics"
)

// ReverseProxy defines the interface for a reverse proxy.
type ReverseProxy interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	ModifyResponse(func(*http.Response) error)
}

// Server represents a reverse proxy server.
type Server struct {
	cfg   *config.TranslatedConfig
	proxy ReverseProxy
}

// NewServer creates a new reverse proxy server.
func NewServer(cfg *config.TranslatedConfig) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is nil")
	}
	if cfg.TargetURL == nil {
		return nil, fmt.Errorf("target URL is nil")
	}

	proxy, err := newSingleHostReverseProxy(cfg.TargetURL, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create reverse proxy: %w", err)
	}

	return &Server{
		cfg:   cfg,
		proxy: proxy,
	}, nil
}

// ServeHTTP handles incoming HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.MetricsEnabled {
		s.proxy.ServeHTTP(w, r)
		return
	}

	// Instrument with metrics
	targetHost := s.cfg.TargetURL.Host
	method := r.Method

	// Track in-flight requests
	metrics.HTTPRequestsInFlight.WithLabelValues(targetHost).Inc()
	defer metrics.HTTPRequestsInFlight.WithLabelValues(targetHost).Dec()

	// Track request size
	requestSize := float64(computeApproximateRequestSize(r))
	metrics.HTTPRequestSizeBytes.WithLabelValues(method, targetHost).Observe(requestSize)

	// Wrap response writer to capture status code and response size
	wrappedWriter := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Start timer for request duration
	start := time.Now()

	// Serve the request
	s.proxy.ServeHTTP(wrappedWriter, r)

	// Record metrics
	duration := time.Since(start).Seconds()
	statusCode := strconv.Itoa(wrappedWriter.statusCode)

	metrics.HTTPRequestsTotal.WithLabelValues(method, statusCode, targetHost).Inc()
	metrics.HTTPRequestDuration.WithLabelValues(method, statusCode, targetHost).Observe(duration)
	metrics.HTTPResponseSizeBytes.WithLabelValues(method, statusCode, targetHost).Observe(float64(wrappedWriter.responseSize))
}

// SetModifyResponse sets the response modifier function.
func (s *Server) SetModifyResponse(f func(*http.Response) error) {
	s.proxy.ModifyResponse(f)
}

// DefaultReverseProxy is the default implementation of the ReverseProxy interface.
type DefaultReverseProxy struct {
	*httputil.ReverseProxy
}

func (p *DefaultReverseProxy) ModifyResponse(f func(*http.Response) error) {
	p.ReverseProxy.ModifyResponse = f
}

func newSingleHostReverseProxy(target *url.URL, cfg *config.TranslatedConfig) (ReverseProxy, error) { //nolint:gocognit
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		if cfg.SetRequestID && req.Header.Get("X-Request-Id") == "" {
			requestID := uuid.Must(uuid.NewRandom())
			req.Header.Set("X-Request-Id", requestID.String())
		}

		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", target.Host)
		}

		if req.Header.Get("X-Forwarded-Proto") == "" {
			req.Header.Set("X-Forwarded-Proto", target.Scheme)
		}

		if req.Header.Get("X-Forwarded-Port") == "" {
			if target.Port() != "" {
				req.Header.Set("X-Forwarded-Port", target.Port())
			} else {
				if target.Scheme == "https" {
					req.Header.Set("X-Forwarded-Port", "443")
				} else {
					req.Header.Set("X-Forwarded-Port", "80")
				}
			}
		}

		if req.Header.Get("X-Forwarded-For") == "" {
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		}

		req.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		// Store the current "Authorization" header(s)
		he := req.Header.Get("Authorization")

		password, passwordIsSet := target.User.Password()
		if passwordIsSet {
			// Setting HTTP Basic Auth overwrites the current "Authorization" header(s).
			req.SetBasicAuth(target.User.Username(), password)

			if he != "" {
				// Merge Authorization header(s).
				req.Header.Set("Authorization", fmt.Sprintf("%s, %s", req.Header.Get("Authorization"), he))
			}
		}

		// Add custom headers from configuration
		for key, value := range cfg.Headers {
			req.Header.Set(key, value)
		}
	}

	profilingTransport, err := transport.NewProfilingTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create profiling transport: %w", err)
	}

	return &DefaultReverseProxy{
		ReverseProxy: &httputil.ReverseProxy{
			Director:  director,
			Transport: profilingTransport,
		},
	}, nil
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code and response size.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	responseSize int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseSize += size
	return size, err
}

// computeApproximateRequestSize computes the approximate size of the HTTP request.
func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// Add Content-Length if available
	if r.ContentLength > 0 {
		s += int(r.ContentLength)
	}

	return s
}
