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
}

func (w Writer) LogRequest(request *http.Request) (err error) {
	query := ""
	rawQuery := request.URL.RawQuery
	if len(rawQuery) > 0 {
		query = fmt.Sprintf("?%s", rawQuery)
	}

	headers := make([]string, 0)
	for name, values := range request.Header {
		for _, value := range values {
			headers = append(headers, fmt.Sprintf("%s: %s", name, value))
		}
	}

	bodyString := ""
	if request.ContentLength > 0 {
		bodyBytes, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}

		request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		bodyString = string(bodyBytes)
	}

	w.Logger.Info("",
		zap.String("request_method", request.Method),
		zap.String("scheme", request.URL.Scheme),
		zap.String("http_host", request.URL.Host),
		zap.String("request", request.URL.Path),
		zap.String("args", query),
		zap.Strings("headers", headers),
		zap.String("body", bodyString),
	)

	return nil

	// requestRow := row{time.Now().Format(time.RFC3339Nano), request.Method, request.URL.Scheme, request.URL.Host, request.URL.Path, query, headers, bodyString}

	// m, err := json.Marshal(requestRow)
	// if err != nil {
	// 	log.Fatal(err)
	// 	panic(err)
	// }

	// fmt.Println(string(m))

	// return err
}

func (w Writer) LogResponse(response *http.Response) (err error) {
	// request := response.Request
	// w.LogRequest(request)

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
	if response.Request.ContentLength > 0 {
		requestBodyBytes, err := ioutil.ReadAll(response.Request.Body)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}

		response.Request.Body = ioutil.NopCloser(bytes.NewBuffer(requestBodyBytes))

		requestBodyString = string(requestBodyBytes)
	}

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
	if response.ContentLength > 0 {
		responseBodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
			panic(err)
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
		zap.String("status", response.Status),
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
