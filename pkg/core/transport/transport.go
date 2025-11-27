package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	"github.com/restinthemiddle/restinthemiddle/pkg/metrics"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// Context keys for storing HTTP request data.
const (
	httpRequestKey contextKey = "http-request"
)

// HTTPTiming contains several connection related time metrics.
type HTTPTiming struct {
	GetConn              time.Time
	GotConn              time.Time
	GotFirstResponseByte time.Time
	TLSHandshakeStart    time.Time
	TLSHandshakeDone     time.Time
}

// TCPConnectionTiming holds TCP connection establishment timing (before TLS).
type TCPConnectionTiming struct {
	Start       time.Time // TCP dial start
	Established time.Time // TCP connection established (before TLS handshake)
}

// The ProfilingTransport is a http.Transport with a http.RoundTripper.
type ProfilingTransport struct {
	roundTripper      http.RoundTripper
	cfg               *config.TranslatedConfig
	connectionTimings sync.Map // map[*http.Request]*TCPConnectionTiming
}

// ProfilingContextKey is a special string type.
type ProfilingContextKey string

// NewProfilingTransport creates a new profiling transport.
func NewProfilingTransport(cfg *config.TranslatedConfig) (*ProfilingTransport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is nil")
	}

	transport := &ProfilingTransport{
		cfg: cfg,
	}

	// Create a custom transport with our custom dialer
	httpTransport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Get the original request from context to store timing info
			if req := ctx.Value(httpRequestKey); req != nil {
				if httpReq, ok := req.(*http.Request); ok {
					connectionStart := time.Now()
					transport.connectionTimings.Store(httpReq, &TCPConnectionTiming{Start: connectionStart})
				}
			}

			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			conn, err := dialer.DialContext(ctx, network, addr)

			// Store connection end time
			if req := ctx.Value(httpRequestKey); req != nil {
				if httpReq, ok := req.(*http.Request); ok {
					if timing, exists := transport.connectionTimings.Load(httpReq); exists {
						if connTiming, ok := timing.(*TCPConnectionTiming); ok {
							connTiming.Established = time.Now()
						}
					}
				}
			}

			return conn, err
		},
	}

	transport.roundTripper = httpTransport
	return transport, nil
}

// RoundTrip facilitates several timing measurements.
func (transport *ProfilingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	requestBodyString := ""

	if r.ContentLength > 0 && transport.cfg.LogPostBody {
		func() {
			if transport.cfg.ExcludePostBodyRegexp != nil && transport.cfg.ExcludePostBodyRegexp.MatchString(r.URL.Path) {
				return
			}

			requestBodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				log.Print(err)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))
			requestBodyString = string(requestBodyBytes)
		}()
	}

	ctxRequestBodyString := context.WithValue(r.Context(), ProfilingContextKey("requestBodyString"), requestBodyString)
	ctxRoundTripStart := context.WithValue(ctxRequestBodyString, ProfilingContextKey("roundTripStart"), time.Now())

	timing := &HTTPTiming{}

	trace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			timing.GetConn = time.Now()
		},
		GotConn: func(httptrace.GotConnInfo) {
			timing.GotConn = time.Now()
		},
		GotFirstResponseByte: func() {
			timing.GotFirstResponseByte = time.Now()
		},
		TLSHandshakeStart: func() {
			timing.TLSHandshakeStart = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			timing.TLSHandshakeDone = time.Now()
		},
	}

	ctxTrace := httptrace.WithClientTrace(ctxRoundTripStart, trace)

	// Add the request to the context so our DialContext can access it
	ctxWithRequest := context.WithValue(ctxTrace, httpRequestKey, r)

	response, err := transport.roundTripper.RoundTrip(r.WithContext(ctxWithRequest))

	// Get connection timing from our map
	var connectionStart, connectionEnd time.Time
	if timing, exists := transport.connectionTimings.Load(r); exists {
		if connTiming, ok := timing.(*TCPConnectionTiming); ok {
			connectionStart = connTiming.Start
			connectionEnd = connTiming.Established
		}
		// Clean up the timing info to prevent memory leaks
		transport.connectionTimings.Delete(r)
	}

	// Set timing context keys regardless of error status
	// We need to handle the case where response might be nil on error
	var ctx context.Context
	if response != nil && response.Request != nil {
		ctx = response.Request.Context()
	} else {
		// Use the original request context as fallback
		ctx = ctxWithRequest
	}

	ctxConnectionStart := context.WithValue(ctx, ProfilingContextKey("tcpConnectionStart"), connectionStart)
	ctxConnectionEnd := context.WithValue(ctxConnectionStart, ProfilingContextKey("tcpConnectionEstablished"), connectionEnd)
	ctxRoundTripEnd := context.WithValue(ctxConnectionEnd, ProfilingContextKey("roundTripEnd"), time.Now())
	ctxTiming := context.WithValue(ctxRoundTripEnd, ProfilingContextKey("timing"), timing)

	// Update the request context if we have a valid response
	if response != nil && response.Request != nil {
		response.Request = response.Request.WithContext(ctxTiming)
	}

	// Track errors in metrics if enabled
	if transport.cfg.MetricsEnabled && err != nil {
		targetHost := transport.cfg.TargetURL.Host
		errorType := classifyError(err)
		metrics.HTTPUpstreamErrorsTotal.WithLabelValues(targetHost, errorType).Inc()

		// Also track as proxy failure
		metrics.HTTPProxyFailuresTotal.WithLabelValues(targetHost, errorType).Inc()
	}

	return response, err
}

// classifyError categorizes the error type for metrics.
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "no such host"):
		return "dns_error"
	case strings.Contains(errStr, "EOF"):
		return "eof"
	case strings.Contains(errStr, "TLS"):
		return "tls_error"
	case strings.Contains(errStr, "context canceled"):
		return "context_canceled"
	case strings.Contains(errStr, "context deadline exceeded"):
		return "context_deadline_exceeded"
	default:
		return "other"
	}
}
