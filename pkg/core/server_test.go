package core

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/restinthemiddle/restinthemiddle/internal/testutil"
	config "github.com/restinthemiddle/restinthemiddle/pkg/core/config"
)

func TestDefaultHTTPServerTLSInvalidKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(certFile, []byte("not a cert"), 0600); err != nil {
		t.Fatalf("write dummy cert: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte("not a key"), 0600); err != nil {
		t.Fatalf("write dummy key: %v", err)
	}

	targetURL, _ := url.Parse("http://example.com")
	cfg = &config.TranslatedConfig{
		TargetURL:   targetURL,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}

	server := &DefaultHTTPServer{}
	err := server.ListenAndServe("127.0.0.1:0", http.NewServeMux())
	if err == nil {
		t.Fatal("expected error for invalid TLS key pair, got nil")
	}
}

func TestDefaultHTTPServerTLSServe(t *testing.T) {
	certFile, keyFile := testutil.GenerateSelfSignedCert(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	targetURL, _ := url.Parse("http://example.com")
	cfg = &config.TranslatedConfig{
		TargetURL:     targetURL,
		TLSCertFile:   certFile,
		TLSKeyFile:    keyFile,
		TLSMinVersion: tls.VersionTLS12,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &DefaultHTTPServer{}
	go server.ListenAndServe(addr, mux) //nolint:errcheck

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed test cert
		},
	}

	var lastErr error
	for i := 0; i < 30; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/", addr))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", resp.StatusCode)
			}
			return
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("HTTPS server did not become ready: %v", lastErr)
}
