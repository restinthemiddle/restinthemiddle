package core

import (
	"context"
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

func TestDefaultHTTPServerShutdownWithoutStart(t *testing.T) {
	server := &DefaultHTTPServer{}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown on never-started server: %v", err)
	}
}

// freeAddr returns an address that was free at the time of the call.
// There is a small window in which another process could grab the port.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// serveAndRequest starts srv.ListenAndServe on addr in the background,
// performs one GET with the given client and shuts the server down.
func serveAndRequest(t *testing.T, srv *DefaultHTTPServer, addr string, client *http.Client, scheme string) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	go srv.ListenAndServe(addr, mux) //nolint:errcheck
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	var lastErr error
	for i := 0; i < 30; i++ {
		resp, err := client.Get(fmt.Sprintf("%s://%s/", scheme, addr))
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
	t.Fatalf("server did not become ready: %v", lastErr)
}

func TestDefaultHTTPServerServe(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	cfg = &config.TranslatedConfig{
		TargetURL: targetURL,
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	serveAndRequest(t, &DefaultHTTPServer{}, freeAddr(t), client, "http")
}

func TestDefaultHTTPServerTLSServe(t *testing.T) {
	certFile, keyFile := testutil.GenerateSelfSignedCert(t)

	targetURL, _ := url.Parse("http://example.com")
	cfg = &config.TranslatedConfig{
		TargetURL:     targetURL,
		TLSCertFile:   certFile,
		TLSKeyFile:    keyFile,
		TLSMinVersion: tls.VersionTLS12,
	}

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed test cert
		},
	}
	serveAndRequest(t, &DefaultHTTPServer{}, freeAddr(t), client, "https")
}
