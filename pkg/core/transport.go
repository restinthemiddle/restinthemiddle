package core

import (
	"context"
	"net"
	"net/http"
	"time"
)

type ProfilingTransport struct {
	roundTripper    http.RoundTripper
	dialer          *net.Dialer
	connectionStart time.Time
	connectionEnd   time.Time
}

type ProfilingContextKey string

func newProfilingTransport() *ProfilingTransport {
	transport := &ProfilingTransport{
		dialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}
	transport.roundTripper = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                transport.dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return transport
}

func (transport *ProfilingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctxRoundTripStart := context.WithValue(r.Context(), ProfilingContextKey("roundTripStart"), time.Now())

	response, err := transport.roundTripper.RoundTrip(r.WithContext(ctxRoundTripStart))

	ctxRoundTripEnd := context.WithValue(response.Request.Context(), ProfilingContextKey("roundTripEnd"), time.Now())
	ctxConnectionStart := context.WithValue(ctxRoundTripEnd, ProfilingContextKey("connectionStart"), transport.connectionStart)
	ctxConnectionEnd := context.WithValue(ctxConnectionStart, ProfilingContextKey("connectionEnd"), transport.connectionEnd)

	response.Request = response.Request.WithContext(ctxConnectionEnd)

	return response, err
}

func (transport *ProfilingTransport) dial(network, addr string) (net.Conn, error) {
	transport.connectionStart = time.Now()

	connections, err := transport.dialer.Dial(network, addr)

	transport.connectionEnd = time.Now()

	return connections, err
}
