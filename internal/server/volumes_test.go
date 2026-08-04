package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

func TestListVolumes(t *testing.T) {
	srv := newTestServer(t, fakeDocker{
		volumes: []docker.Volume{
			{Name: "app_data", Driver: "local", Created: 1700000000, InUse: true},
		},
	}, fakeCompose{})
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
	srv := newTestServer(t, fakeDocker{}, fakeCompose{})
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

func TestCreateVolume(t *testing.T) {
	srv := newTestServer(t, fakeDocker{
		volumeCreated: docker.Volume{Name: "app_data", Driver: "local", Mountpoint: "/data"},
	}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := postStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/volumes", `{"name":"app_data"}`); code != http.StatusUnauthorized {
		t.Fatalf("create without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var vol volumeResponse
	postJSONCreated(t, client, ts.URL+"/api/v1/environments/1/volumes", `{"name":"app_data","driver":"local"}`, &vol)
	if vol.Name != "app_data" || vol.Driver != "local" {
		t.Fatalf("create volume: got %+v", vol)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/volumes", `{"name":"bad name!"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid volume name: want 400, got %d", code)
	}
}

func TestCreateVolumeUnreachable(t *testing.T) {
	srv := newTestServer(t, fakeDocker{volumeCreateErr: errors.New("cannot connect")}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/volumes", `{"name":"app_data"}`); code != http.StatusBadGateway {
		t.Fatalf("create unreachable: want 502, got %d", code)
	}
}

func TestVolumeDetail(t *testing.T) {
	srv := newTestServer(t, fakeDocker{volumeDetail: docker.VolumeDetail{
		Name: "app_data", Driver: "local", Mountpoint: "/data", Scope: "local",
		Containers: []docker.VolumeContainer{{ID: "c1", Name: "web"}},
	}}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/volumes/app_data"); code != http.StatusUnauthorized {
		t.Fatalf("detail without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	var got volumeDetailResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/volumes/app_data", &got)
	if got.Name != "app_data" || got.Driver != "local" || len(got.Containers) != 1 || got.Containers[0].Name != "web" {
		t.Fatalf("volume detail: got %+v", got)
	}
}

func TestVolumeDetailUnreachable(t *testing.T) {
	srv := newTestServer(t, fakeDocker{volumeDetailErr: errors.New("no such volume")}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/volumes/nope"); code != http.StatusBadGateway {
		t.Fatalf("detail unreachable: want 502, got %d", code)
	}
}

func TestListVolumesUnreachable(t *testing.T) {
	srv := newTestServer(t, fakeDocker{volumesErr: errors.New("cannot connect")}, fakeCompose{})
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/volumes"); code != http.StatusBadGateway {
		t.Fatalf("volumes unreachable: want 502, got %d", code)
	}
}
