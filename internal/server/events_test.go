package server

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventsStreamRequiresAuth(t *testing.T) {
	srv := newTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("events without auth: want 401, got %d", resp.StatusCode)
	}
}

func TestEventsStreamDelivers(t *testing.T) {
	srv := newTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	resp, err := client.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("events: want 200, got %d", resp.StatusCode)
	}

	lines := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if line := scanner.Text(); strings.HasPrefix(line, "data: ") {
				lines <- line
				return
			}
		}
	}()

	srv.events.Publish("environment.updated", map[string]int{"id": 42})

	select {
	case line := <-lines:
		if !strings.Contains(line, `"environment.updated"`) || !strings.Contains(line, `"id":42`) {
			t.Fatalf("unexpected event line: %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}
