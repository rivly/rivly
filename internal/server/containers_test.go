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

func TestContainerDetail(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{detail: docker.ContainerDetail{
		ID: "abc", Name: "web", Image: "nginx:latest", State: "running",
		Env: []string{"FOO=bar"}, RestartPolicy: "unless-stopped",
	}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers/abc"); code != http.StatusUnauthorized {
		t.Fatalf("detail without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	var got containerDetailResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1/containers/abc", &got)
	if got.Name != "web" || got.State != "running" || len(got.Env) != 1 || got.RestartPolicy != "unless-stopped" {
		t.Fatalf("detail: got %+v", got)
	}
}

func TestCreateContainer(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{createdContainerID: "newid123"}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	body := `{"name":"web","image":"nginx:latest","command":"nginx -g 'daemon off;'","env":[{"key":"FOO","value":"bar"}],"ports":[{"hostPort":"8080","containerPort":"80","proto":"tcp"}],"mounts":[{"source":"app_data","target":"/data"}],"network":"bridge","restartPolicy":"unless-stopped","start":true}`

	if code := postStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers", body); code != http.StatusUnauthorized {
		t.Fatalf("create without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var out struct {
		ID string `json:"id"`
	}
	postJSONStatus(t, client, ts.URL+"/api/v1/environments/1/containers", body, http.StatusCreated, &out)
	if out.ID != "newid123" {
		t.Fatalf("create container: got %+v", out)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/containers", `{"image":""}`); code != http.StatusBadRequest {
		t.Fatalf("missing image: want 400, got %d", code)
	}
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/containers", `{"image":"nginx","restartPolicy":"sometimes"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid restart policy: want 400, got %d", code)
	}
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/containers", `{"image":"nginx","name":"bad name!"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid name: want 400, got %d", code)
	}
}

func TestCreateContainerUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{createContainerErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := postStatus(t, client, ts.URL+"/api/v1/environments/1/containers", `{"image":"nginx:latest"}`); code != http.StatusBadGateway {
		t.Fatalf("create unreachable: want 502, got %d", code)
	}
}

func TestContainerStatsStream(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{statsData: []docker.Stats{{CPUPercent: 12.5, MemUsage: 100, MemLimit: 1000, MemPercent: 10}}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers/abc/stats"); code != http.StatusUnauthorized {
		t.Fatalf("stats without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	resp, err := client.Get(ts.URL + "/api/v1/environments/1/containers/abc/stats")
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read stats: %v", err)
	}
	stream := string(body)
	if !strings.Contains(stream, `"cpuPercent":12.5`) {
		t.Fatalf("stats missing cpu sample: %q", stream)
	}
	if !strings.Contains(stream, "event: end") {
		t.Fatalf("stats missing end event: %q", stream)
	}
}

func TestContainerLogsStream(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{logLines: []docker.LogLine{
		{Stream: "stdout", Message: "starting up"},
		{Stream: "stderr", Message: "a warning"},
	}}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments/1/containers/abc/logs"); code != http.StatusUnauthorized {
		t.Fatalf("logs without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)
	resp, err := client.Get(ts.URL + "/api/v1/environments/1/containers/abc/logs")
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logs: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("logs content-type: got %q", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read logs: %v", err)
	}
	stream := string(body)
	if !strings.Contains(stream, `"message":"starting up"`) {
		t.Fatalf("logs missing stdout line: %q", stream)
	}
	if !strings.Contains(stream, `"stream":"stderr"`) {
		t.Fatalf("logs missing stderr line: %q", stream)
	}
	if !strings.Contains(stream, "event: end") {
		t.Fatalf("logs missing end event: %q", stream)
	}
}

func TestContainerLogsUnreachable(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{logErr: errors.New("no such container")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)
	if code := getStatus(t, client, ts.URL+"/api/v1/environments/1/containers/abc/logs"); code != http.StatusBadGateway {
		t.Fatalf("logs unreachable: want 502, got %d", code)
	}
}
