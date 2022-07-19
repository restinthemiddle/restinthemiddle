package core

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

var cfg *TranslatedConfig
var wrt Writer

var proxy *httputil.ReverseProxy

func handleRequest(response http.ResponseWriter, request *http.Request) {
	proxy.ServeHTTP(response, request)
}

func logResponse(response *http.Response) (err error) {
	if !cfg.LoggingEnabled {
		return nil
	}

	if cfg.ExcludeRegexp.String() != "" && cfg.ExcludeRegexp.MatchString(response.Request.URL.Path) {
		return nil
	}

	return wrt.LogResponse(response)
}

func Run(c *TranslatedConfig, w Writer) {
	cfg = c
	wrt = w

	proxy = newSingleHostReverseProxy(cfg.TargetURL)
	proxy.ModifyResponse = logResponse

	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.ListenIp, cfg.ListenPort), nil); err != nil {
		log.Fatalf(err.Error())
	}
}

func newSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		if cfg.SetRequestId && req.Header.Get("X-Request-Id") == "" {
			requestId := uuid.Must(uuid.NewRandom())

			req.Header.Set("X-Request-Id", requestId.String())
		}

		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", target.Host)
		}

		if req.Header.Get("X-Forwarded-Proto") == "" {
			req.Header.Set("X-Forwarded-Proto", target.Scheme)
		}

		if req.Header.Get("X-Forwarded-Port") == "" {
			if target.Port() != "" {
				req.Header.Set("X-Forwarded-Port", target.Port())
			} else {
				if target.Scheme == "https" {
					req.Header.Set("X-Forwarded-Port", "443")
				} else {
					req.Header.Set("X-Forwarded-Port", "80")
				}
			}
		}

		if req.Header.Get("X-Forwarded-For") == "" {
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		}

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

		for key, value := range cfg.Headers {
			req.Header.Set(key, value)
		}
	}

	transport := newProfilingTransport()

	return &httputil.ReverseProxy{Director: director, Transport: transport}
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
