package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func dashboardAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html":              {Data: []byte("<!doctype html><div id=root></div>")},
		"favicon.png":             {Data: []byte("png")},
		"assets/index-abc123.js":  {Data: []byte("console.log(1)")},
		"assets/index-abc123.css": {Data: []byte(".a{}")},
	}
}

func dashboardServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	ts := httptest.NewServer(srv.dashboardHandler(dashboardAssets()))
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, ts *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestDashboardServesTheIndex(t *testing.T) {
	ts := dashboardServer(t)

	resp := get(t, ts, "/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /: want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<!doctype html><div id=root></div>" {
		t.Fatalf("body = %q", body)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("index Cache-Control = %q, a cached index would pin users to an old build", got)
	}
}

func TestDashboardFallsBackForClientRoutes(t *testing.T) {
	ts := dashboardServer(t)

	for _, path := range []string{"/login", "/environments/1/containers", "/environments/1/stacks/app/edit"} {
		resp := get(t, ts, path)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: want 200 so the client router can take over, got %d", path, resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Errorf("GET %s: content type = %q", path, got)
		}
	}
}

func TestDashboardNeverSwallowsApiRoutes(t *testing.T) {
	ts := dashboardServer(t)

	resp := get(t, ts, "/api/v1/does-not-exist")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("an unknown API route must 404, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("an unknown API route must stay JSON, got %q", got)
	}
}

func TestDashboardCachesHashedAssetsForever(t *testing.T) {
	ts := dashboardServer(t)

	resp := get(t, ts, "/assets/index-abc123.js")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET asset: want 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Cache-Control"); got != immutableCache {
		t.Errorf("hashed asset Cache-Control = %q, want %q", got, immutableCache)
	}

	resp = get(t, ts, "/favicon.png")
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("an unhashed file must not be cached forever, got %q", got)
	}
}

func TestDashboardRefusesWrites(t *testing.T) {
	ts := dashboardServer(t)

	resp, err := ts.Client().Post(ts.URL+"/login", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("a write to a client route must not serve the index, got %d", resp.StatusCode)
	}
}

func TestSecurityHeadersDeclareAStrictPolicy(t *testing.T) {
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	resp := get(t, ts, "/api/health")
	csp := resp.Header.Get("Content-Security-Policy")

	for _, directive := range []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP is missing %q\ngot: %s", directive, csp)
		}
	}
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") || strings.Contains(csp, "unsafe-eval") {
		t.Errorf("scripts must never run inline or be evaluated\ngot: %s", csp)
	}
}
