package zapwriter

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/restinthemiddle/restinthemiddle/pkg/core"
	"go.uber.org/zap"
)

type Writer struct {
	Logger *zap.Logger
	Config *core.Config
}

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

	requestBodyString := response.Request.Context().Value(core.ProfilingContextKey("requestBodyString")).(string)

	connectionStart := response.Request.Context().Value(core.ProfilingContextKey("connectionStart")).(time.Time)
	connectionEnd := response.Request.Context().Value(core.ProfilingContextKey("connectionEnd")).(time.Time)
	connectionDuration := connectionEnd.Sub(connectionStart)

	roundTripStart := response.Request.Context().Value(core.ProfilingContextKey("roundTripStart")).(time.Time)
	roundTripEnd := response.Request.Context().Value(core.ProfilingContextKey("roundTripEnd")).(time.Time)
	roundTripDuration := roundTripEnd.Sub(roundTripStart)

	responseHeaders := make([]string, 0)
	for name, values := range response.Header {
		for _, value := range values {
			responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", name, value))
		}
	}

	responseBodyString := ""
	if w.Config.LogResponseBody && response.ContentLength > 0 {
		responseBodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Print(err)

			return err
		}
		response.Body = ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes))

		responseBodyString = string(responseBodyBytes)
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
		zap.Time("connection_start", connectionStart),
		zap.Time("connection_end", connectionEnd),
		zap.Duration("connection_duration", connectionDuration),
		zap.Time("roundtrip_start", roundTripStart),
		zap.Time("roundtrip_end", roundTripEnd),
		zap.Duration("roundtrip_duration", roundTripDuration),
		zap.Strings("response_headers", responseHeaders),
		zap.Int64("body_bytes_sent", response.ContentLength),
		zap.String("response_body", responseBodyString),
	)

	return nil
}
