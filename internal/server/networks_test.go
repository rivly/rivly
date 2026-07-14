package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

func TestListNetworks(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{
		networks: []docker.Network{
			{ID: "net123", Name: "bridge", Driver: "bridge", Scope: "local", InUse: true},
		},
	}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/networks"); code != http.StatusUnauthorized {
		t.Fatalf("networks without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var networks []networkResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/networks", &networks)
	if len(networks) != 1 || networks[0].Name != "bridge" || !networks[0].InUse {
		t.Fatalf("networks: got %+v", networks)
	}
}

func TestNetworkActions(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var out struct {
		Results []actionResult `json:"results"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/networks/actions", `{"action":"remove","ids":["net123"]}`, &out)
	if len(out.Results) != 1 || !out.Results[0].OK {
		t.Fatalf("network actions: got %+v", out.Results)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/networks/actions", `{"action":"prune","ids":["x"]}`); code != http.StatusBadRequest {
		t.Fatalf("invalid network action: want 400, got %d", code)
	}
}

func TestListNetworksUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{networksErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/networks"); code != http.StatusBadGateway {
		t.Fatalf("networks unreachable: want 502, got %d", code)
	}
}
