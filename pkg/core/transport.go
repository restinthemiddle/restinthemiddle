package core

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
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
	requestBodyString := ""

	if r.ContentLength > 0 {
		func() {
			requestBodyBytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Print(err)

				return
			}

			r.Body = ioutil.NopCloser(bytes.NewBuffer(requestBodyBytes))

			requestBodyString = string(requestBodyBytes)
		}()
	}

	ctxRequestBodyString := context.WithValue(r.Context(), ProfilingContextKey("requestBodyString"), requestBodyString)
	ctxRoundTripStart := context.WithValue(ctxRequestBodyString, ProfilingContextKey("roundTripStart"), time.Now())

	response, err := transport.roundTripper.RoundTrip(r.WithContext(ctxRoundTripStart))

	ctxConnectionStart := context.WithValue(response.Request.Context(), ProfilingContextKey("connectionStart"), transport.connectionStart)
	ctxConnectionEnd := context.WithValue(ctxConnectionStart, ProfilingContextKey("connectionEnd"), transport.connectionEnd)
	ctxRoundTripEnd := context.WithValue(ctxConnectionEnd, ProfilingContextKey("roundTripEnd"), time.Now())

	response.Request = response.Request.WithContext(ctxRoundTripEnd)

	return response, err
}

func (transport *ProfilingTransport) dial(network, addr string) (net.Conn, error) {
	transport.connectionStart = time.Now()

	connections, err := transport.dialer.Dial(network, addr)

	transport.connectionEnd = time.Now()

	return connections, err
}
