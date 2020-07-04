package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getListenAddress() string {
	port := getEnv("PORT", "8000")
	return ":" + port
}

func getTargetHostDsn() string {
	return getEnv("TARGET_HOST_DSN", "127.0.0.1:8081")
}

func getTargetURL() (*url.URL, error) {
	return url.Parse(getTargetHostDsn())
}

// Log the env variables required for a reverse proxy
func logSetup() {
	log.Printf("Listening on: %s\n", getListenAddress())
	log.Printf("Targeting server on: %s\n", getTargetHostDsn())
}

func handleRequest(response http.ResponseWriter, request *http.Request) {
	url, _ := getTargetURL()

	// logRequest(request)

	proxy := newSingleHostReverseProxy(url)
	proxy.ModifyResponse = logResponse
	// request.URL.Host = url.Host
	// request.URL.Scheme = url.Scheme
	// request.Header.Set("X-Forwarded-Host", request.Header.Get("Host"))
	// request.Host = url.Host
	// request.Header.Set("User-Agent", "Rest in the middle logging proxy")

	proxy.ServeHTTP(response, request)
}

func logRequest(request *http.Request) (err error) {
	query := ""
	rawQuery := request.URL.RawQuery
	if len(rawQuery) > 0 {
		query = fmt.Sprintf("?%s", rawQuery)
	}

	title := fmt.Sprintf("REQUEST - Method: %s; URL: %s://%s; Path: %s%s\n", request.Method, request.URL.Scheme, request.URL.Host, request.URL.Path, query)

	headers := ""
	for key, element := range request.Header {
		headers += fmt.Sprintf("%s: %s\n", key, element)
	}

	bodyString := ""
	if "POST" == request.Method || "PUT" == request.Method || "PATCH" == request.Method {
		bodyBytes, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}

		request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		bodyString = fmt.Sprintf("Content: %s\n", string(bodyBytes))
	}

	log.Printf("%s%s%s", title, headers, bodyString)

	return err
}

func logResponse(response *http.Response) (err error) {
	title := fmt.Sprintf("RESPONSE - Code: %d\n", response.StatusCode)

	headers := ""
	for key, element := range response.Header {
		headers += fmt.Sprintf("%s: %s\n", key, element)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	response.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	bodyString := fmt.Sprintf("Content: %s\n", string(bodyBytes))

	log.Printf("%s%s%s", title, headers, bodyString)

	return err
}

func main() {
	logSetup()

	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}

func newSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		password, passwordIsSet := target.User.Password()
		if passwordIsSet {
			req.SetBasicAuth(target.User.Username(), password)
		}

		req.Header.Set("User-Agent", "Rest in the middle logging proxy")
		logRequest(req)
	}

	return &httputil.ReverseProxy{Director: director}
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