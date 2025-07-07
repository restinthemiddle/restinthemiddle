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
	GetConn                  time.Time
	GotConn                  time.Time
	ConnEstDuration          time.Duration
	TCPConnectionStart       time.Time
	TCPConnectionEstablished time.Time
	TCPConnectionDuration    time.Duration
	RoundTripStart           time.Time
	RoundTripEnd             time.Time
	RoundTripDuration        time.Duration
	GotFirstResponseByte     time.Time
	TLSHandshakeStart        time.Time
	TLSHandshakeDone         time.Time
	TLSHandshakeDuration     time.Duration
}

// MarshalLogObject is used for the type safe JSON serialization of the HTTPTiming struct.
func (t HTTPTiming) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddTime("get_conn", t.GetConn)
	enc.AddTime("got_conn", t.GotConn)
	enc.AddDuration("conn_establish_duration", t.ConnEstDuration)
	enc.AddTime("tcp_connection_start", t.TCPConnectionStart)
	enc.AddTime("tcp_connection_established", t.TCPConnectionEstablished)
	enc.AddDuration("tcp_connection_duration", t.TCPConnectionDuration)
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
	timing, err := w.extractTiming(response)
	if err != nil {
		return err
	}

	requestData := w.extractRequestData(response)
	responseData := w.extractResponseData(response)

	w.Logger.Info("",
		zap.String("request_method", response.Request.Method),
		zap.String("scheme", response.Request.URL.Scheme),
		zap.String("http_host", response.Request.URL.Host),
		zap.String("request", response.Request.URL.Path),
		zap.String("args", requestData.Query),
		zap.Strings("request_headers", requestData.Headers),
		zap.String("post_body", requestData.Body),
		zap.Int("status", response.StatusCode),
		zap.Strings("response_headers", responseData.Headers),
		zap.Int64("body_bytes_sent", response.ContentLength),
		zap.String("response_body", responseData.Body),
		zap.Object("timing", timing),
	)

	return nil
}

// requestData holds extracted request information.
type requestData struct {
	Query   string
	Headers []string
	Body    string
}

// responseData holds extracted response information.
type responseData struct {
	Headers []string
	Body    string
}

// extractRequestData extracts request information from the response.
func (w Writer) extractRequestData(response *http.Response) requestData {
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

	return requestData{
		Query:   query,
		Headers: requestHeaders,
		Body:    requestBodyString,
	}
}

// extractResponseData extracts response information.
func (w Writer) extractResponseData(response *http.Response) responseData {
	responseHeaders := make([]string, 0)
	for name, values := range response.Header {
		for _, value := range values {
			responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", name, value))
		}
	}

	responseBodyString := ""
	if w.Config.LogResponseBody {
		responseBodyString = w.extractResponseBody(response)
	}

	return responseData{
		Headers: responseHeaders,
		Body:    responseBodyString,
	}
}

// extractResponseBody extracts the response body if logging is enabled.
func (w Writer) extractResponseBody(response *http.Response) string {
	if w.Config.ExcludeResponseBodyRegexp != nil && w.Config.ExcludeResponseBodyRegexp.MatchString(response.Request.URL.Path) {
		return ""
	}

	responseBodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Print(err)
		return ""
	}
	response.Body.Close()

	response.Body = io.NopCloser(bytes.NewBuffer(responseBodyBytes))
	return string(responseBodyBytes)
}

// extractTiming extracts and processes timing information from the request context.
func (w Writer) extractTiming(response *http.Response) (*HTTPTiming, error) {
	// Get timing info from context with defensive null checks
	timingValue := response.Request.Context().Value(transport.ProfilingContextKey("timing"))
	if timingValue == nil {
		// If timing is not available, skip timing-related logging
		return nil, fmt.Errorf("timing information not available in request context")
	}

	timing, err := NewHTTPTimingFromCore(timingValue.(*transport.HTTPTiming))
	if err != nil {
		return nil, err
	}

	w.populateConnectionTiming(response, &timing)
	w.populateRoundTripTiming(response, &timing)

	return &timing, nil
}

// populateConnectionTiming adds TCP connection timing to the timing object.
func (w Writer) populateConnectionTiming(response *http.Response, timing *HTTPTiming) {
	// Safely get TCP connection timing values
	if tcpConnectionStartValue := response.Request.Context().Value(transport.ProfilingContextKey("tcpConnectionStart")); tcpConnectionStartValue != nil {
		if tcpConnectionStart, ok := tcpConnectionStartValue.(time.Time); ok {
			timing.TCPConnectionStart = tcpConnectionStart
		}
	}

	if tcpConnectionEstablishedValue := response.Request.Context().Value(transport.ProfilingContextKey("tcpConnectionEstablished")); tcpConnectionEstablishedValue != nil {
		if tcpConnectionEstablished, ok := tcpConnectionEstablishedValue.(time.Time); ok {
			timing.TCPConnectionEstablished = tcpConnectionEstablished
			timing.TCPConnectionDuration = timing.TCPConnectionEstablished.Sub(timing.TCPConnectionStart)
		}
	}
}

// populateRoundTripTiming adds round trip timing to the timing object.
func (w Writer) populateRoundTripTiming(response *http.Response, timing *HTTPTiming) {
	// Safely get round trip timing values
	if roundTripStartValue := response.Request.Context().Value(transport.ProfilingContextKey("roundTripStart")); roundTripStartValue != nil {
		if roundTripStart, ok := roundTripStartValue.(time.Time); ok {
			timing.RoundTripStart = roundTripStart
		}
	}

	if roundTripEndValue := response.Request.Context().Value(transport.ProfilingContextKey("roundTripEnd")); roundTripEndValue != nil {
		if roundTripEnd, ok := roundTripEndValue.(time.Time); ok {
			timing.RoundTripEnd = roundTripEnd
			timing.RoundTripDuration = timing.RoundTripEnd.Sub(timing.RoundTripStart)
		}
	}
}
