package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

type stubRoundTripper struct {
	req *http.Request
}

func (s *stubRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	s.req = r
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
		Request:    r,
	}, nil
}

func TestProfilingTransportRespectsLogPostBody(t *testing.T) {
	cfg := &config.TranslatedConfig{
		LogPostBody: false,
	}
	transport := &ProfilingTransport{
		cfg: cfg,
	}
	stub := &stubRoundTripper{}
	transport.roundTripper = stub

	req := httptest.NewRequest("POST", "http://example.com/test", strings.NewReader("secret"))
	req.ContentLength = int64(len("secret"))

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if val, _ := resp.Request.Context().Value(ProfilingContextKey("requestBodyString")).(string); val != "" {
		t.Errorf("Expected requestBodyString to be empty when LogPostBody is false, got %q", val)
	}

	bodyBytes, _ := io.ReadAll(stub.req.Body)
	if string(bodyBytes) != "secret" {
		t.Errorf("Expected upstream to receive original body, got %q", string(bodyBytes))
	}
}
