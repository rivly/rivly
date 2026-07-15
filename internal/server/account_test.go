package server

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

func putStatus(t *testing.T, c *http.Client, url, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

func loginClient(t *testing.T, ts *httptest.Server, password string) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	body := `{"email":"admin@rivly.dev","password":"` + password + `"}`
	if code := postStatus(t, client, ts.URL+"/api/v1/login", body); code != http.StatusOK {
		t.Fatalf("login: want 200, got %d", code)
	}
	return client
}

func TestUpdateProfile(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()
	client := authedClient(t, ts)

	if code := putStatus(t, client, ts.URL+"/api/v1/me", `{"displayName":"Renamed"}`); code != http.StatusOK {
		t.Fatalf("update: want 200, got %d", code)
	}
	var me userResponse
	getJSON(t, client, ts.URL+"/api/v1/me", &me)
	if me.DisplayName != "Renamed" {
		t.Fatalf("display name: got %q", me.DisplayName)
	}

	if code := putStatus(t, client, ts.URL+"/api/v1/me", `{"displayName":"   "}`); code != http.StatusBadRequest {
		t.Fatalf("empty name: want 400, got %d", code)
	}
	if code := putStatus(t, client, ts.URL+"/api/v1/me", `{"displayName":"`+strings.Repeat("x", 101)+`"}`); code != http.StatusBadRequest {
		t.Fatalf("long name: want 400, got %d", code)
	}
	if code := putStatus(t, &http.Client{}, ts.URL+"/api/v1/me", `{"displayName":"x"}`); code != http.StatusUnauthorized {
		t.Fatalf("no auth: want 401, got %d", code)
	}
}

func TestChangePassword(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()
	client := authedClient(t, ts)

	const url = "/api/v1/me/password"

	if code := postStatus(t, client, ts.URL+url,
		`{"currentPassword":"wrong-password","newPassword":"brand-new-password"}`); code != http.StatusUnauthorized {
		t.Fatalf("wrong current: want 401, got %d", code)
	}
	if code := postStatus(t, client, ts.URL+url,
		`{"currentPassword":"s3cret-password","newPassword":"short"}`); code != http.StatusBadRequest {
		t.Fatalf("short new: want 400, got %d", code)
	}
	if code := postStatus(t, client, ts.URL+url,
		`{"currentPassword":"s3cret-password","newPassword":"brand-new-password"}`); code != http.StatusNoContent {
		t.Fatalf("change: want 204, got %d", code)
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/me"); code != http.StatusOK {
		t.Fatalf("session after change: want 200, got %d", code)
	}
	if code := postStatus(t, &http.Client{}, ts.URL+"/api/v1/login",
		`{"email":"admin@rivly.dev","password":"s3cret-password"}`); code != http.StatusUnauthorized {
		t.Fatalf("old password still works: want 401, got %d", code)
	}
	if code := postStatus(t, &http.Client{}, ts.URL+"/api/v1/login",
		`{"email":"admin@rivly.dev","password":"brand-new-password"}`); code != http.StatusOK {
		t.Fatalf("new password: want 200, got %d", code)
	}
}

func TestChangePasswordDestroysOtherSessions(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	owner := authedClient(t, ts)
	intruder := loginClient(t, ts, "s3cret-password")

	if code := getStatus(t, intruder, ts.URL+"/api/v1/me"); code != http.StatusOK {
		t.Fatalf("intruder before: want 200, got %d", code)
	}

	if code := postStatus(t, owner, ts.URL+"/api/v1/me/password",
		`{"currentPassword":"s3cret-password","newPassword":"brand-new-password"}`); code != http.StatusNoContent {
		t.Fatalf("change: want 204, got %d", code)
	}

	if code := getStatus(t, intruder, ts.URL+"/api/v1/me"); code != http.StatusUnauthorized {
		t.Fatalf("intruder session survived the password change: want 401, got %d", code)
	}
	if code := getStatus(t, owner, ts.URL+"/api/v1/me"); code != http.StatusOK {
		t.Fatalf("owner session was destroyed: want 200, got %d", code)
	}
}
