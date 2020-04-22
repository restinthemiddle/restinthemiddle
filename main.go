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

	logRequest(request)

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = logResponse
	request.URL.Host = url.Host
	request.URL.Scheme = url.Scheme
	request.Header.Set("X-Forwarded-Host", request.Header.Get("Host"))
	request.Host = url.Host
	request.Header.Set("User-Agent", "Rest in the middle logging proxy")

	proxy.ServeHTTP(response, request)
}

func logRequest(request *http.Request) (err error) {
	title := fmt.Sprintf("REQUEST - Method: %s; Path: %s\n", request.Method, request.URL.Path)

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
