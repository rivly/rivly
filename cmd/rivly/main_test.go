package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
}

func startRivly(t *testing.T, overrides map[string]string) (string, *syncBuffer) {
	t.Helper()

	dir := t.TempDir()
	addr := freePort(t)
	env := map[string]string{
		"RIVLY_ADDR":        addr,
		"RIVLY_DATABASE":    filepath.Join(dir, "rivly.db"),
		"RIVLY_DATA":        filepath.Join(dir, "data"),
		"DOCKER_HOST":       "unix://" + filepath.Join(dir, "absent.sock"),
		"RIVLY_COMPOSE_BIN": filepath.Join(dir, "no-compose"),
		"RIVLY_SETUP_TOKEN": "token-from-the-environment",
	}
	for k, v := range overrides {
		env[k] = v
	}

	out := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- run(ctx, func(k string) string { return env[k] }, out) }()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("run returned %v", err)
			}
		case <-time.After(20 * time.Second):
			t.Error("run did not return after its context was cancelled")
		}
	})

	waitForReady(t, "http://"+addr+"/api/health", done)
	return "http://" + addr, out
}

func waitForReady(t *testing.T, url string, done <-chan error) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("run exited before becoming ready: %v", err)
		default:
		}

		resp, err := http.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("rivly never became ready")
}

func TestRunServesAWorkingInstance(t *testing.T) {
	base, logs := startRivly(t, nil)

	var health struct {
		Status string `json:"status"`
	}
	getJSON(t, base+"/api/health", &health)
	if health.Status != "ok" {
		t.Fatalf("health status = %q", health.Status)
	}

	var setup struct {
		NeedsSetup bool `json:"needsSetup"`
	}
	getJSON(t, base+"/api/v1/setup", &setup)
	if !setup.NeedsSetup {
		t.Fatal("a fresh instance must ask for setup")
	}

	if !strings.Contains(logs.String(), "setup is open") {
		t.Error("the operator must be told how to claim the instance")
	}
	if !strings.Contains(logs.String(), "no docker compose executable found") {
		t.Error("a missing compose executable must be reported, not silently ignored")
	}
}

func TestRunRefusesAnInvalidConfiguration(t *testing.T) {
	err := run(
		context.Background(),
		func(k string) string {
			if k == "RIVLY_POLL_INTERVAL" {
				return "every friday"
			}
			return ""
		},
		io.Discard,
	)
	if err == nil {
		t.Fatal("an invalid configuration must abort startup")
	}
	if !strings.Contains(err.Error(), "RIVLY_POLL_INTERVAL") {
		t.Errorf("the error must name the offending variable, got %v", err)
	}
}

func TestRunClaimsTheInstanceWithThePinnedToken(t *testing.T) {
	base, _ := startRivly(t, nil)

	body := `{"email":"admin@rivly.dev","password":"s3cret-password","displayName":"Admin","token":"token-from-the-environment"}`
	resp, err := http.Post(base+"/api/v1/setup", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST setup: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: want 201, got %d", resp.StatusCode)
	}

	var status struct {
		NeedsSetup bool `json:"needsSetup"`
	}
	getJSON(t, base+"/api/v1/setup", &status)
	if status.NeedsSetup {
		t.Fatal("the instance must be claimed after a successful setup")
	}
}

func getJSON(t *testing.T, url string, dst any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %s", url, fmt.Sprint(resp.StatusCode))
	}
}
