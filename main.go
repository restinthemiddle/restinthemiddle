package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// Config represents the configuration
type Config struct {
	TargetHostDsn  string
	ListenAddress  string
	Headers        map[string]string
	LoggingEnabled bool
}

var config Config
var targetURL *url.URL

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
	return getEnv("TARGET_HOST_DSN", "http://127.0.0.1:8081")
}

func getTargetURL(targetHostDsn string) *url.URL {
	url, err := url.Parse(targetHostDsn)
	if err != nil {
		log.Panic(err)
	}

	return url
}

func getLoggingEnabled() bool {
	value := getEnv("LOGGING_ENABLED", "true")
	if strings.ToLower(value) == "false" {
		return false
	}

	return true
}
func readConfig() {
	config.Headers = make(map[string]string)

	// Set default values
	config.Headers["User-Agent"] = "Rest in the middle logging proxy"
	config.LoggingEnabled = true

	// Read configuration
	configString := getEnv("CONFIG", "")
	json.Unmarshal([]byte(configString), &config)

	// Read environment variables
	config.TargetHostDsn = getTargetHostDsn()
	config.ListenAddress = getListenAddress()
	config.LoggingEnabled = getLoggingEnabled()

	targetURL = getTargetURL(config.TargetHostDsn)
}

// Log the env variables required for a reverse proxy
func logSetup() {
	log.Printf("Listening on: %s\n", config.ListenAddress)
	log.Printf("Targeting server on: %s\n", config.TargetHostDsn)
	log.Printf("Logging enabled: %s",
		func() string {
			if config.LoggingEnabled {
				return "true"
			}

			return "false"
		}())
	log.Println("Overwriting headers:")
	for key, value := range config.Headers {
		// Each value is an interface{} type, that is type asserted as a string
		log.Printf("  %s: %s", key, value)
	}

}

func handleRequest(response http.ResponseWriter, request *http.Request) {
	proxy := newSingleHostReverseProxy(targetURL)
	proxy.ModifyResponse = logResponse

	proxy.ServeHTTP(response, request)
}

func logRequest(request *http.Request) (err error) {
	if !config.LoggingEnabled {
		return nil
	}

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
	if request.ContentLength > 0 {
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
	if !config.LoggingEnabled {
		return nil
	}

	title := fmt.Sprintf("RESPONSE - Code: %d\n", response.StatusCode)

	headers := ""
	for key, element := range response.Header {
		headers += fmt.Sprintf("%s: %s\n", key, element)
	}

	bodyString := ""
	if response.ContentLength > 0 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}

		response.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		bodyString = fmt.Sprintf("Content: %s\n", string(bodyBytes))
	}

	log.Printf("%s%s%s", title, headers, bodyString)

	return err
}

func main() {
	readConfig()
	logSetup()

	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
		log.Panic(err)
	}
}

func newSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Header.Set("X-Forwarded-Host", target.Host)
		req.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		// Store the current "Authorization" header(s)
		he := req.Header.Get("Authorization")

		password, passwordIsSet := target.User.Password()
		if passwordIsSet {
			// Setting HTTP Basic Auth overwrites the current "Authorization" header(s)
			req.SetBasicAuth(target.User.Username(), password)

			if he != "" {
				// Merge Authorization header(s)
				req.Header.Set("Authorization", fmt.Sprintf("%s, %s", req.Header.Get("Authorization"), he))
			}
		}

		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}

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
