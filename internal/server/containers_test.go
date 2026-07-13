package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

func TestListContainers(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{
		containers: []docker.Container{
			{
				ID:      "abc123",
				Name:    "web",
				Image:   "nginx:latest",
				State:   "running",
				Status:  "Up 2 hours",
				Created: 1700000000,
			},
		},
	}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers"); code != http.StatusUnauthorized {
		t.Fatalf("containers without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var containers []containerResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/containers", &containers)
	if len(containers) != 1 {
		t.Fatalf("containers: want 1, got %d", len(containers))
	}
	if containers[0].Name != "web" || containers[0].State != "running" {
		t.Fatalf("container: got %+v", containers[0])
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/environments/999/containers"); code != http.StatusNotFound {
		t.Fatalf("missing env containers: want 404, got %d", code)
	}
}

func TestListContainersUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{containersErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/containers"); code != http.StatusBadGateway {
		t.Fatalf("containers unreachable: want 502, got %d", code)
	}
}
