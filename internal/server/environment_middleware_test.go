package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var environmentScopedPaths = []string{
	"/api/v1/environments/%s",
	"/api/v1/environments/%s/stacks",
	"/api/v1/environments/%s/containers",
	"/api/v1/environments/%s/containers/abc",
	"/api/v1/environments/%s/images",
	"/api/v1/environments/%s/images/abc",
	"/api/v1/environments/%s/volumes",
	"/api/v1/environments/%s/volumes/abc",
	"/api/v1/environments/%s/networks",
	"/api/v1/environments/%s/networks/abc",
}

func TestEnvironmentScopedRoutesRejectABadIdConsistently(t *testing.T) {
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := authedClient(t, ts)

	cases := map[string]struct {
		id   string
		want int
	}{
		"not a number":        {"abc", http.StatusBadRequest},
		"overflows an int64":  {"99999999999999999999", http.StatusBadRequest},
		"negative":            {"-1", http.StatusNotFound},
		"unknown environment": {"4242", http.StatusNotFound},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for _, pattern := range environmentScopedPaths {
				url := ts.URL + fmt.Sprintf(pattern, tc.id)
				if got := getStatus(t, client, url); got != tc.want {
					t.Errorf("GET %s: want %d, got %d", fmt.Sprintf(pattern, tc.id), tc.want, got)
				}
			}
		})
	}
}

func TestEnvironmentScopedStreamsRejectAnUnknownEnvironment(t *testing.T) {
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := authedClient(t, ts)

	for _, path := range []string{
		"/api/v1/environments/4242/containers/abc/logs",
		"/api/v1/environments/4242/containers/abc/stats",
		"/api/v1/environments/4242/images/pull?ref=nginx",
	} {
		if got := getStatus(t, client, ts.URL+path); got != http.StatusNotFound {
			t.Errorf("GET %s: want 404, got %d", path, got)
		}
	}
}
