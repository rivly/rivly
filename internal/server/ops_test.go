package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReportsTheDatabase(t *testing.T) {
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	var body struct {
		Status string `json:"status"`
	}
	getJSON(t, &http.Client{}, ts.URL+"/api/health", &body)
	if body.Status != "ok" {
		t.Fatalf("status = %q, want ok", body.Status)
	}
}

func TestHealthFailsWhenTheDatabaseIsGone(t *testing.T) {
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	closeTestDatabase(t, srv)

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/health"); code != http.StatusServiceUnavailable {
		t.Fatalf("health with a dead database: want 503, got %d", code)
	}
}

func TestVersionRequiresAuthentication(t *testing.T) {
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/version"); code != http.StatusUnauthorized {
		t.Fatalf("version without a session: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	var body versionResponse
	getJSON(t, client, ts.URL+"/api/v1/version", &body)
	if body.Version == "" {
		t.Error("version must never be empty")
	}
	if body.Go == "" {
		t.Error("the go version must be reported")
	}
}
