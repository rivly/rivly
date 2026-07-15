package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rivly/rivly/internal/auth"
	"github.com/rivly/rivly/internal/config"
	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/events"
	"github.com/rivly/rivly/internal/gitcred"
	"github.com/rivly/rivly/internal/registry"
	"github.com/rivly/rivly/internal/secret"
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
	cipher, err := secret.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("secret: %v", err)
	}
	registries := registry.NewStore(queries, cipher)
	gitCredentials := gitcred.NewStore(queries, cipher)
	return New(logger, queries, auth.NewSessionManager(sqlDB), auth.NewLocal(queries, sqlDB), fakeDocker{}, fakeCompose{}, events.NewHub(), registries, gitCredentials, config.Config{SetupToken: testSetupToken})
}

func TestAuthFlow(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	const creds = testCreds

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

func TestSetupRequiresTheSetupToken(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	noToken := `{"email":"admin@rivly.dev","password":"s3cret-password","displayName":"Admin"}`
	if code := postStatus(t, client, ts.URL+"/api/v1/setup", noToken); code != http.StatusForbidden {
		t.Fatalf("setup without a token: want 403, got %d", code)
	}

	wrongToken := `{"email":"admin@rivly.dev","password":"s3cret-password","token":"nope"}`
	if code := postStatus(t, client, ts.URL+"/api/v1/setup", wrongToken); code != http.StatusForbidden {
		t.Fatalf("setup with a wrong token: want 403, got %d", code)
	}

	var status struct {
		NeedsSetup bool `json:"needsSetup"`
	}
	getJSON(t, client, ts.URL+"/api/v1/setup", &status)
	if !status.NeedsSetup {
		t.Fatal("a rejected setup must leave the instance unclaimed")
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/setup", testCreds); code != http.StatusCreated {
		t.Fatalf("setup with the right token: want 201, got %d", code)
	}
}

func TestSetupRejectsAnOversizedDisplayName(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	body := fmt.Sprintf(
		`{"email":"admin@rivly.dev","password":"s3cret-password","displayName":%q,"token":%q}`,
		strings.Repeat("a", maxDisplayName+1), testSetupToken,
	)
	if code := postStatus(t, client, ts.URL+"/api/v1/setup", body); code != http.StatusBadRequest {
		t.Fatalf("setup with an oversized display name: want 400, got %d", code)
	}

	var status struct {
		NeedsSetup bool `json:"needsSetup"`
	}
	getJSON(t, client, ts.URL+"/api/v1/setup", &status)
	if !status.NeedsSetup {
		t.Fatal("a rejected setup must not create the account")
	}
}

func TestSetupStaysClosedWhenNoTokenIsConfigured(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.SetupToken = ""
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	empty := `{"email":"admin@rivly.dev","password":"s3cret-password","token":""}`
	if code := postStatus(t, client, ts.URL+"/api/v1/setup", empty); code != http.StatusForbidden {
		t.Fatalf("empty token must never match: want 403, got %d", code)
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

func postJSON(t *testing.T, c *http.Client, url, body string, dst any) {
	t.Helper()
	resp, err := c.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: want 200, got %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func postJSONStatus(t *testing.T, c *http.Client, url, body string, want int, dst any) {
	t.Helper()
	resp, err := c.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != want {
		t.Fatalf("POST %s: want %d, got %d", url, want, resp.StatusCode)
	}
	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			t.Fatalf("decode %s: %v", url, err)
		}
	}
}
