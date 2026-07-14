package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContainerActions(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	body := `{"action":"stop","ids":["abc","def"]}`
	if code := postStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers/actions", body); code != http.StatusUnauthorized {
		t.Fatalf("actions without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var ok struct {
		Results []actionResult `json:"results"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/containers/actions", body, &ok)
	if len(ok.Results) != 2 || !ok.Results[0].OK || !ok.Results[1].OK {
		t.Fatalf("actions: got %+v", ok.Results)
	}
}

func TestContainerActionsInvalid(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/containers/actions", `{"action":"explode","ids":["abc"]}`); code != http.StatusBadRequest {
		t.Fatalf("invalid action: want 400, got %d", code)
	}
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/containers/actions", `{"action":"stop","ids":[]}`); code != http.StatusBadRequest {
		t.Fatalf("empty ids: want 400, got %d", code)
	}
}

func TestContainerActionsPartialFailure(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{actionErr: errors.New("boom")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var out struct {
		Results []actionResult `json:"results"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/containers/actions", `{"action":"remove","ids":["abc"]}`, &out)
	if len(out.Results) != 1 || out.Results[0].OK || out.Results[0].Error == "" {
		t.Fatalf("partial failure: got %+v", out.Results)
	}
}
