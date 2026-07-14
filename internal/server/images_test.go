package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestImagePull(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{pullData: []docker.PullProgress{
		{Status: "Pulling from library/nginx", ID: "latest"},
		{Status: "Downloading", ID: "abc123", Current: 500, Total: 1000},
	}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/images/pull?ref=nginx"); code != http.StatusUnauthorized {
		t.Fatalf("pull without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/images/pull"); code != http.StatusBadRequest {
		t.Fatalf("pull without ref: want 400, got %d", code)
	}

	resp, err := client.Get(ts.URL + "/api/v1/environments/1/images/pull?ref=nginx:latest")
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	stream := string(body)
	if !strings.Contains(stream, `"status":"Downloading"`) || !strings.Contains(stream, "event: end") {
		t.Fatalf("pull stream: %q", stream)
	}
}

func TestImagePrune(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{pruneResult: docker.PruneResult{ImagesDeleted: 3, SpaceReclaimed: 4096}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	var out struct {
		ImagesDeleted  int    `json:"imagesDeleted"`
		SpaceReclaimed uint64 `json:"spaceReclaimed"`
	}
	postJSON(t, client, ts.URL+"/api/v1/environments/1/images/prune", `{"all":true}`, &out)
	if out.ImagesDeleted != 3 || out.SpaceReclaimed != 4096 {
		t.Fatalf("prune: got %+v", out)
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
