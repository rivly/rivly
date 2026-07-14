package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

func TestListImages(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{
		images: []docker.Image{
			{ID: "abc123", Tags: []string{"nginx:latest"}, Size: 142000000, Created: 1700000000},
		},
	}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/images"); code != http.StatusUnauthorized {
		t.Fatalf("images without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var images []imageResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/images", &images)
	if len(images) != 1 || images[0].Tags[0] != "nginx:latest" {
		t.Fatalf("images: got %+v", images)
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/environments/999/images"); code != http.StatusNotFound {
		t.Fatalf("missing env images: want 404, got %d", code)
	}
}

func TestImageActions(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var out struct {
		Results []actionResult `json:"results"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/images/actions", `{"action":"remove","ids":["abc","def"]}`, &out)
	if len(out.Results) != 2 || !out.Results[0].OK {
		t.Fatalf("image actions: got %+v", out.Results)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/images/actions", `{"action":"pull","ids":["abc"]}`); code != http.StatusBadRequest {
		t.Fatalf("invalid image action: want 400, got %d", code)
	}
}

func TestListImagesUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{imagesErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/images"); code != http.StatusBadGateway {
		t.Fatalf("images unreachable: want 502, got %d", code)
	}
}
