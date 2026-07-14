package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestContainerExecAuth(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers/abc/exec"); code != http.StatusUnauthorized {
		t.Fatalf("exec without auth: want 401, got %d", code)
	}
}

func TestContainerExecMissingEnv(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/999/containers/abc/exec"); code != http.StatusNotFound {
		t.Fatalf("exec missing env: want 404, got %d", code)
	}
}

func TestContainerExecWebsocketError(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{execErr: errors.New("no shell")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	u, _ := url.Parse(ts.URL)
	var parts []string
	for _, c := range client.Jar.Cookies(u) {
		parts = append(parts, c.String())
	}
	header := http.Header{}
	header.Set("Cookie", strings.Join(parts, "; "))
	header.Set("Origin", ts.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/environments/1/containers/abc/exec"
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer func() { _ = conn.CloseNow() }()

	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	if typ != websocket.MessageText || !strings.Contains(string(data), `"type":"error"`) {
		t.Fatalf("want error frame, got type=%v data=%q", typ, data)
	}
}
