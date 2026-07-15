package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupOverProxy(t *testing.T, ts *httptest.Server, proto string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/setup", bytes.NewBufferString(testCreds))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if proto != "" {
		req.Header.Set("X-Forwarded-Proto", proto)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST setup: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: want 201, got %d", resp.StatusCode)
	}
	return resp
}

func sessionCookie(t *testing.T, resp *http.Response) string {
	t.Helper()
	for _, c := range resp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(c, "rivly_session=") {
			return c
		}
	}
	t.Fatal("no session cookie in response")
	return ""
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "frame-ancestors 'none'",
		"Referrer-Policy":         "no-referrer",
	}

	for _, path := range []string{"/api/health", "/api/v1/setup", "/api/v1/me", "/api/v1/nope"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_ = resp.Body.Close()
		for header, value := range want {
			if got := resp.Header.Get(header); got != value {
				t.Errorf("%s: %s = %q, want %q", path, header, got, value)
			}
		}
	}
}

func TestSecureCookiesMarksSessionCookieBehindTLSProxy(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	cookie := sessionCookie(t, setupOverProxy(t, ts, "https"))
	if !strings.Contains(cookie, "; Secure") {
		t.Fatalf("session cookie behind a TLS proxy must be Secure, got %q", cookie)
	}
}

func TestSecureCookiesLeavesPlainHTTPCookieUsable(t *testing.T) {
	ts := httptest.NewServer(newTestServer(t).Router())
	defer ts.Close()

	cookie := sessionCookie(t, setupOverProxy(t, ts, ""))
	if strings.Contains(cookie, "; Secure") {
		t.Fatalf("plain http cookie must not be Secure, got %q", cookie)
	}
}
