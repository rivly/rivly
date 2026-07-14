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

func TestCreateNetwork(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{networkCreated: docker.CreatedNetwork{ID: "net456"}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := postStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/networks", `{"name":"app_net"}`); code != http.StatusUnauthorized {
		t.Fatalf("create without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	postJSONStatus(t, client, ts.URL+"/api/v1/environments/1/networks", `{"name":"app_net","driver":"bridge","subnet":"172.20.0.0/16"}`, http.StatusCreated, &out)
	if out.ID != "net456" || out.Name != "app_net" {
		t.Fatalf("create network: got %+v", out)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/networks", `{"name":"app_net","subnet":"not-a-cidr"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid subnet: want 400, got %d", code)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/networks", `{"name":""}`); code != http.StatusBadRequest {
		t.Fatalf("empty name: want 400, got %d", code)
	}
}

func TestCreateNetworkUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{networkCreateErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/networks", `{"name":"app_net"}`); code != http.StatusBadGateway {
		t.Fatalf("create unreachable: want 502, got %d", code)
	}
}

func TestNetworkDetail(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{networkDetail: docker.NetworkDetail{
		ID: "net123", Name: "app_net", Driver: "bridge", Scope: "local",
		Subnets:    []docker.NetworkSubnet{{Subnet: "172.20.0.0/16", Gateway: "172.20.0.1"}},
		Containers: []docker.NetworkContainer{{ID: "c1", Name: "web", IPv4: "172.20.0.2/16"}},
	}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/networks/net123"); code != http.StatusUnauthorized {
		t.Fatalf("detail without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	var got networkDetailResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/networks/net123", &got)
	if got.Name != "app_net" || len(got.Subnets) != 1 || got.Subnets[0].Subnet != "172.20.0.0/16" || len(got.Containers) != 1 || got.Containers[0].Name != "web" {
		t.Fatalf("network detail: got %+v", got)
	}
}

func TestNetworkDetailUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{networkDetailErr: errors.New("no such network")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/networks/nope"); code != http.StatusBadGateway {
		t.Fatalf("detail unreachable: want 502, got %d", code)
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
