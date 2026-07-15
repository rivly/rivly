package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeployStackRejectsOversizedBody(t *testing.T) {
	srv := newTestServer(t)
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	body := fmt.Sprintf(`{"name":"demo","content":%q}`, strings.Repeat("x", maxRequestBody+1))
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/stacks", body); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized stack body: want 413, got %d", code)
	}
}

func TestDeployStackRejectsUnknownField(t *testing.T) {
	srv := newTestServer(t)
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	body := `{"name":"demo","content":"services: {}","typoedField":true}`
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/stacks", body); code != http.StatusBadRequest {
		t.Fatalf("unknown field: want 400, got %d", code)
	}
}

func TestImagePruneAcceptsEmptyBody(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/images/prune", ""); code != http.StatusOK {
		t.Fatalf("prune with an empty body: want 200, got %d", code)
	}
}
