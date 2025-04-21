package zapwriter

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
	"github.com/restinthemiddle/restinthemiddle/pkg/core/transport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HTTPTiming contains several connection related time metrics.
type HTTPTiming struct {
	GetConn              time.Time
	GotConn              time.Time
	ConnEstDuration      time.Duration
	ConnectionStart      time.Time
	ConnectionEnd        time.Time
	ConnectionDuration   time.Duration
	RoundTripStart       time.Time
	RoundTripEnd         time.Time
	RoundTripDuration    time.Duration
	GotFirstResponseByte time.Time
	TLSHandshakeStart    time.Time
	TLSHandshakeDone     time.Time
	TLSHandshakeDuration time.Duration
}

// MarshalLogObject is used for the type safe JSON serialization of the HTTPTiming struct.
func (t HTTPTiming) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddTime("get_conn", t.GetConn)
	enc.AddTime("got_conn", t.GotConn)
	enc.AddDuration("conn_establish_duration", t.ConnEstDuration)
	enc.AddTime("connection_start", t.ConnectionStart)
	enc.AddTime("connection_end", t.ConnectionEnd)
	enc.AddDuration("connection_duration", t.ConnectionDuration)
	enc.AddTime("roundtrip_start", t.RoundTripStart)
	enc.AddTime("roundtrip_end", t.RoundTripEnd)
	enc.AddDuration("roundtrip_duration", t.RoundTripDuration)
	enc.AddTime("got_first_response_byte", t.GotFirstResponseByte)
	enc.AddTime("tls_handshake_start", t.TLSHandshakeStart)
	enc.AddTime("tls_handshake_done", t.TLSHandshakeDone)
	enc.AddDuration("tls_handshake_duration", t.TLSHandshakeDuration)
	return nil
}

// NewHTTPTimingFromCore yields a new, partially hydrated HTTPTiming struct from the eponymous core struct.
func NewHTTPTimingFromCore(ct *transport.HTTPTiming) (HTTPTiming, error) {
	t := HTTPTiming{
		GetConn:              ct.GetConn,
		GotConn:              ct.GotConn,
		ConnEstDuration:      ct.GotConn.Sub(ct.GetConn),
		GotFirstResponseByte: ct.GotFirstResponseByte,
		TLSHandshakeStart:    ct.TLSHandshakeStart,
		TLSHandshakeDone:     ct.TLSHandshakeDone,
		TLSHandshakeDuration: ct.TLSHandshakeDone.Sub(ct.TLSHandshakeStart),
	}

	return t, nil
}

// Writer is being used to print out logs via the zap library.
type Writer struct {
	Logger *zap.Logger
	Config *config.TranslatedConfig
}

// LogResponse is being called in the eponymous method in core.
func (w Writer) LogResponse(response *http.Response) (err error) {
	query := ""
	rawQuery := response.Request.URL.RawQuery
	if len(rawQuery) > 0 {
		query = fmt.Sprintf("?%s", rawQuery)
	}

	requestHeaders := make([]string, 0)
	for name, values := range response.Request.Header {
		for _, value := range values {
			requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", name, value))
		}
	}

	requestBodyString := ""
	if val := response.Request.Context().Value(transport.ProfilingContextKey("requestBodyString")); val != nil {
		if str, ok := val.(string); ok {
			requestBodyString = str
		}
	}

	timing, _ := NewHTTPTimingFromCore(response.Request.Context().Value(transport.ProfilingContextKey("timing")).(*transport.HTTPTiming))

	timing.ConnectionStart = response.Request.Context().Value(transport.ProfilingContextKey("connectionStart")).(time.Time)
	timing.ConnectionEnd = response.Request.Context().Value(transport.ProfilingContextKey("connectionEnd")).(time.Time)
	timing.ConnectionDuration = timing.ConnectionEnd.Sub(timing.ConnectionStart)

	timing.RoundTripStart = response.Request.Context().Value(transport.ProfilingContextKey("roundTripStart")).(time.Time)
	timing.RoundTripEnd = response.Request.Context().Value(transport.ProfilingContextKey("roundTripEnd")).(time.Time)
	timing.RoundTripDuration = timing.RoundTripEnd.Sub(timing.RoundTripStart)

	responseHeaders := make([]string, 0)
	for name, values := range response.Header {
		for _, value := range values {
			responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", name, value))
		}
	}

	responseBodyString := ""
	if w.Config.LogResponseBody {
		func() {
			if w.Config.ExcludeResponseBodyRegexp != nil && w.Config.ExcludeResponseBodyRegexp.String() != "" && w.Config.ExcludeResponseBodyRegexp.MatchString(response.Request.URL.Path) {
				return
			}

			responseBodyBytes, err := io.ReadAll(response.Body)
			if err != nil {
				log.Print(err)

				return
			}
			response.Body.Close()

			response.Body = io.NopCloser(bytes.NewBuffer(responseBodyBytes))
			responseBodyString = string(responseBodyBytes)
		}()
	}

	w.Logger.Info("",
		zap.String("request_method", response.Request.Method),
		zap.String("scheme", response.Request.URL.Scheme),
		zap.String("http_host", response.Request.URL.Host),
		zap.String("request", response.Request.URL.Path),
		zap.String("args", query),
		zap.Strings("request_headers", requestHeaders),
		zap.String("post_body", requestBodyString),
		zap.Int("status", response.StatusCode),
		zap.Strings("response_headers", responseHeaders),
		zap.Int64("body_bytes_sent", response.ContentLength),
		zap.String("response_body", responseBodyString),
		zap.Object("timing", timing),
	)

	return nil
}
