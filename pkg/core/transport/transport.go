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
	"time"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

// HTTPTiming contains several connection related time metrics
type HTTPTiming struct {
	GetConn              time.Time
	GotConn              time.Time
	GotFirstResponseByte time.Time
	TLSHandshakeStart    time.Time
	TLSHandshakeDone     time.Time
}

// The ProfilingTransport is a http.Transport with a http.RoundTripper
type ProfilingTransport struct {
	roundTripper    http.RoundTripper
	dialer          *net.Dialer
	connectionStart time.Time
	connectionEnd   time.Time
	cfg             *config.TranslatedConfig
}

// ProfilingContextKey is a special string type
type ProfilingContextKey string

// NewProfilingTransport creates a new profiling transport
func NewProfilingTransport(cfg *config.TranslatedConfig) (*ProfilingTransport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is nil")
	}

	transport := &ProfilingTransport{
		dialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		cfg: cfg,
	}
	transport.roundTripper = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                transport.dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return transport, nil
}

// RoundTrip facilitates several timing measurements
func (transport *ProfilingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	requestBodyString := ""

	if r.ContentLength > 0 {
		func() {
			if transport.cfg.ExcludePostBodyRegexp.String() != "" && transport.cfg.ExcludePostBodyRegexp.MatchString(r.URL.Path) {
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

	response, err := transport.roundTripper.RoundTrip(r.WithContext(ctxTrace))

	ctxConnectionStart := context.WithValue(response.Request.Context(), ProfilingContextKey("connectionStart"), transport.connectionStart)
	ctxConnectionEnd := context.WithValue(ctxConnectionStart, ProfilingContextKey("connectionEnd"), transport.connectionEnd)
	ctxRoundTripEnd := context.WithValue(ctxConnectionEnd, ProfilingContextKey("roundTripEnd"), time.Now())
	ctxTiming := context.WithValue(ctxRoundTripEnd, ProfilingContextKey("timing"), timing)

	response.Request = response.Request.WithContext(ctxTiming)

	return response, err
}

func (transport *ProfilingTransport) dial(network, addr string) (net.Conn, error) {
	transport.connectionStart = time.Now()
	connections, err := transport.dialer.Dial(network, addr)
	transport.connectionEnd = time.Now()
	return connections, err
}
