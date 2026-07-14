package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

func TestListVolumes(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{
		volumes: []docker.Volume{
			{Name: "app_data", Driver: "local", Created: 1700000000, InUse: true},
		},
	}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/volumes"); code != http.StatusUnauthorized {
		t.Fatalf("volumes without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var volumes []volumeResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/volumes", &volumes)
	if len(volumes) != 1 || volumes[0].Name != "app_data" || !volumes[0].InUse {
		t.Fatalf("volumes: got %+v", volumes)
	}
}

func TestVolumeActions(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var out struct {
		Results []actionResult `json:"results"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/volumes/actions", `{"action":"remove","ids":["app_data"]}`, &out)
	if len(out.Results) != 1 || !out.Results[0].OK {
		t.Fatalf("volume actions: got %+v", out.Results)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/volumes/actions", `{"action":"nuke","ids":["x"]}`); code != http.StatusBadRequest {
		t.Fatalf("invalid volume action: want 400, got %d", code)
	}
}

func TestListVolumesUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{volumesErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/volumes"); code != http.StatusBadGateway {
		t.Fatalf("volumes unreachable: want 502, got %d", code)
	}
}
