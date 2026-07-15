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
