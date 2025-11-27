//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "restinthemiddle")

	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/restinthemiddle")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}
	return binPath
}

type proxyInstance struct {
	cmd  *exec.Cmd
	bin  string
	port string
	mock *MockServer
	out  *bytes.Buffer
	err  *bytes.Buffer
}

func getFreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
}

func startProxy(t *testing.T, bin string, env map[string]string, args []string, workDir string) *proxyInstance {
	t.Helper()

	if env == nil {
		env = map[string]string{}
	}

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock server: %v", err)
	}

	listenPort := env["LISTEN_PORT"]
	if listenPort == "" {
		listenPort = getFreePort(t)
		env["LISTEN_PORT"] = listenPort
	}

	if _, ok := env["TARGET_HOST_DSN"]; !ok {
		env["TARGET_HOST_DSN"] = fmt.Sprintf("http://127.0.0.1:%s", mock.GetPort())
	}

	cmd := exec.Command(bin)
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	if workDir != "" {
		cmd.Dir = workDir
	}

	if err := cmd.Start(); err != nil {
		mock.Stop()
		t.Fatalf("start proxy: %v", err)
	}

	// Wait for readiness by hitting the mock's /test endpoint via the proxy.
	client := http.Client{Timeout: 500 * time.Millisecond}
	url := fmt.Sprintf("http://127.0.0.1:%s/test", listenPort)
	var lastErr error
	for i := 0; i < 30; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			lastErr = nil
			break
		}
		lastErr = err
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		cmd.Process.Kill() //nolint:errcheck
		mock.Stop()
		t.Fatalf("proxy did not become ready: %v\nstdout:\n%s\nstderr:\n%s", lastErr, stdoutBuf.String(), stderrBuf.String())
	}

	return &proxyInstance{cmd: cmd, bin: bin, port: listenPort, mock: mock, out: stdoutBuf, err: stderrBuf}
}

func (p *proxyInstance) stop() {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		p.cmd.Wait() //nolint:errcheck
	}
	if p.mock != nil {
		_ = p.mock.Stop()
	}
}

func TestFlagOverridesEnvAndConfigForTargetHost(t *testing.T) {
	bin := buildBinary(t)

	configMock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start config mock: %v", err)
	}
	defer configMock.Stop()
	envMock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start env mock: %v", err)
	}
	defer envMock.Stop()
	flagMock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start flag mock: %v", err)
	}
	defer flagMock.Stop()

	configDir := t.TempDir()
	configContent := fmt.Sprintf("targetHostDsn: http://127.0.0.1:%s\n", configMock.GetPort())
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	env := map[string]string{
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s", envMock.GetPort()),
		"LISTEN_PORT":     getFreePort(t),
	}

	args := []string{
		"--target-host-dsn", fmt.Sprintf("http://127.0.0.1:%s", flagMock.GetPort()),
	}

	proxy := startProxy(t, bin, env, args, configDir)
	defer proxy.stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/test", proxy.port))
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	if len(flagMock.GetRequests()) == 0 {
		t.Fatalf("flag target did not receive request")
	}
	if len(envMock.GetRequests()) != 0 {
		t.Errorf("env target should not receive request")
	}
	if len(configMock.GetRequests()) != 0 {
		t.Errorf("config target should not receive request")
	}
}

func TestHeadersFromFlagsAndConfig(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	configDir := t.TempDir()
	configContent := fmt.Sprintf(`
targetHostDsn: http://127.0.0.1:%s
headers:
  X-Config-Only: config
`, mock.GetPort())
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s", mock.GetPort()),
	}
	args := []string{"--header", "X-Flag-Only:flag"}

	proxy := startProxy(t, bin, env, args, configDir)
	defer proxy.stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/test", proxy.port))
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	req := mock.GetLastRequest()
	if req == nil {
		t.Fatalf("no request received at mock")
	}

	assertHeader := func(name, want string) {
		t.Helper()
		if got := req.Headers[name]; len(got) == 0 || got[0] != want {
			t.Fatalf("header %s = %v, want %s", name, got, want)
		}
	}
	assertHeader("X-Config-Only", "config")
	assertHeader("X-Flag-Only", "flag")
}

func TestSetRequestIDRespected(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s", mock.GetPort()),
		"SET_REQUEST_ID":  "false",
	}

	proxy := startProxy(t, bin, env, nil, "")
	defer proxy.stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/test", proxy.port))
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	req := mock.GetLastRequest()
	if req == nil {
		t.Fatalf("no request received at mock")
	}

	if _, exists := req.Headers["X-Request-Id"]; exists {
		t.Fatalf("X-Request-Id header should not be set when SET_REQUEST_ID=false")
	}
}

func TestPathAndQueryForwarding(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s/api?token=123", mock.GetPort()),
	}

	proxy := startProxy(t, bin, env, nil, "")
	defer proxy.stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/users?limit=10", proxy.port))
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	req := mock.GetLastRequest()
	if req == nil {
		t.Fatalf("no request received at mock")
	}

	if req.Path != "/api/users" {
		t.Fatalf("path = %s, want /api/users", req.Path)
	}
	token := req.QueryParams["token"]
	if len(token) == 0 || token[0] != "123" {
		t.Fatalf("token query missing or wrong: %v", token)
	}
	limit := req.QueryParams["limit"]
	if len(limit) == 0 || limit[0] != "10" {
		t.Fatalf("limit query missing or wrong: %v", limit)
	}
}

func TestAuthMergingFromDSNAndHeader(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://user:pass@127.0.0.1:%s", mock.GetPort()),
	}

	proxy := startProxy(t, bin, env, nil, "")
	defer proxy.stop()

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%s/test", proxy.port), nil)
	req.Header.Set("Authorization", "Bearer TOKEN")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	rec := mock.GetLastRequest()
	if rec == nil {
		t.Fatalf("no request received at mock")
	}

	auth := rec.Headers["Authorization"]
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")) + ", Bearer TOKEN"
	if len(auth) == 0 || auth[0] != expected {
		t.Fatalf("Authorization header = %v, want %s", auth, expected)
	}
}

func TestMetricsEndpointEnabled(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	metricsPort := getFreePort(t)
	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s", mock.GetPort()),
		"METRICS_ENABLED": "true",
		"METRICS_PORT":    metricsPort,
	}

	proxy := startProxy(t, bin, env, nil, "")
	defer proxy.stop()

	// Make a request through the proxy to generate some metrics
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/test", proxy.port))
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	// Wait a bit for metrics to be updated
	time.Sleep(100 * time.Millisecond)

	// Now check the metrics endpoint
	metricsURL := fmt.Sprintf("http://127.0.0.1:%s/metrics", metricsPort)
	client := &http.Client{Timeout: 2 * time.Second}

	metricsResp, err := client.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}
	defer metricsResp.Body.Close()

	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("metrics endpoint status = %d, want %d", metricsResp.StatusCode, http.StatusOK)
	}

	// Read the metrics content
	buf := new(bytes.Buffer)
	buf.ReadFrom(metricsResp.Body)
	metricsContent := buf.String()

	// Verify expected metrics are present
	expectedMetrics := []string{
		"build_info",
		"http_requests_total",
		"http_requests_in_flight",
		"http_request_duration_seconds",
		"http_request_size_bytes",
		"http_response_size_bytes",
	}

	for _, metric := range expectedMetrics {
		if !bytes.Contains([]byte(metricsContent), []byte(metric)) {
			t.Errorf("metrics output missing expected metric: %s", metric)
		}
	}

	// Verify the request we made is counted
	if !bytes.Contains([]byte(metricsContent), []byte(`http_requests_total{`)) {
		t.Error("metrics should contain http_requests_total counter")
	}
}

func TestMetricsEndpointDisabled(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	metricsPort := getFreePort(t)
	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s", mock.GetPort()),
		"METRICS_ENABLED": "false",
		"METRICS_PORT":    metricsPort,
	}

	proxy := startProxy(t, bin, env, nil, "")
	defer proxy.stop()

	// Make a request through the proxy to ensure it's working
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/test", proxy.port))
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Try to access the metrics endpoint - it should not be available
	metricsURL := fmt.Sprintf("http://127.0.0.1:%s/metrics", metricsPort)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	_, err = client.Get(metricsURL)
	if err == nil {
		t.Fatal("metrics endpoint should not be accessible when METRICS_ENABLED=false")
	}
	// We expect a connection error since the metrics server shouldn't be running
}

func TestMetricsCustomPort(t *testing.T) {
	bin := buildBinary(t)

	mock, err := StartMockServer()
	if err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer mock.Stop()

	customMetricsPort := getFreePort(t)
	env := map[string]string{
		"LISTEN_PORT":     getFreePort(t),
		"TARGET_HOST_DSN": fmt.Sprintf("http://127.0.0.1:%s", mock.GetPort()),
		"METRICS_ENABLED": "true",
		"METRICS_PORT":    customMetricsPort,
	}

	proxy := startProxy(t, bin, env, nil, "")
	defer proxy.stop()

	// Verify metrics are available on the custom port
	metricsURL := fmt.Sprintf("http://127.0.0.1:%s/metrics", customMetricsPort)
	client := &http.Client{Timeout: 2 * time.Second}

	metricsResp, err := client.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to get metrics on custom port %s: %v", customMetricsPort, err)
	}
	defer metricsResp.Body.Close()

	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("metrics endpoint status = %d, want %d", metricsResp.StatusCode, http.StatusOK)
	}
}
