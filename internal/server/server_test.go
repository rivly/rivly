package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rivly/rivly/internal/auth"
	"github.com/rivly/rivly/internal/config"
	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	sqlDB, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	queries := db.New(sqlDB)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger, queries, auth.NewSessionManager(sqlDB), auth.NewLocal(queries), fakeDocker{up: true}, config.Config{})
}

func TestAuthFlow(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	const creds = `{"email":"admin@rivly.dev","password":"s3cret-password","displayName":"Admin"}`

	var status struct {
		NeedsSetup bool `json:"needsSetup"`
	}
	getJSON(t, client, ts.URL+"/api/v1/setup", &status)
	if !status.NeedsSetup {
		t.Fatal("expected needsSetup=true before setup")
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/me"); code != http.StatusUnauthorized {
		t.Fatalf("me before auth: want 401, got %d", code)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/setup", creds); code != http.StatusCreated {
		t.Fatalf("setup: want 201, got %d", code)
	}

	var me userResponse
	getJSON(t, client, ts.URL+"/api/v1/me", &me)
	if me.Email != "admin@rivly.dev" || me.Role != "admin" {
		t.Fatalf("me after setup: got %+v", me)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/setup", creds); code != http.StatusConflict {
		t.Fatalf("second setup: want 409, got %d", code)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/logout", ""); code != http.StatusNoContent {
		t.Fatalf("logout: want 204, got %d", code)
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/me"); code != http.StatusUnauthorized {
		t.Fatalf("me after logout: want 401, got %d", code)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/login", `{"email":"admin@rivly.dev","password":"nope"}`); code != http.StatusUnauthorized {
		t.Fatalf("login wrong password: want 401, got %d", code)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/login", `{"email":"admin@rivly.dev","password":"s3cret-password"}`); code != http.StatusOK {
		t.Fatalf("login: want 200, got %d", code)
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/me"); code != http.StatusOK {
		t.Fatalf("me after login: want 200, got %d", code)
	}
}

func getJSON(t *testing.T, c *http.Client, url string, dst any) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func getStatus(t *testing.T, c *http.Client, url string) int {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

func postStatus(t *testing.T, c *http.Client, url, body string) int {
	t.Helper()
	resp, err := c.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}
