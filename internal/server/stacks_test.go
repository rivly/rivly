package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

func TestListStacks(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{
		stacks: []docker.Stack{
			{Name: "green-roots", Type: "external", Services: 3, Running: 3, Total: 3, State: "running", WorkingDir: "/home/me/green-roots"},
		},
	}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/stacks"); code != http.StatusUnauthorized {
		t.Fatalf("stacks without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var stacks []stackResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/stacks", &stacks)
	if len(stacks) != 1 || stacks[0].Name != "green-roots" || stacks[0].State != "running" || stacks[0].Services != 3 {
		t.Fatalf("stacks: got %+v", stacks)
	}
}

func TestStackActions(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var out struct {
		Results []actionResult `json:"results"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/stacks/actions", `{"action":"stop","ids":["green-roots"]}`, &out)
	if len(out.Results) != 1 || !out.Results[0].OK {
		t.Fatalf("stack actions: got %+v", out.Results)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/stacks/actions", `{"action":"pause","ids":["x"]}`); code != http.StatusBadRequest {
		t.Fatalf("invalid stack action: want 400, got %d", code)
	}
}
