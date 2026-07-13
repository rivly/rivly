package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type fakeDocker struct {
	up      bool
	info    docker.SystemInfo
	infoErr error
}

func (f fakeDocker) Ping(_ context.Context, _ int64, _ string) docker.Status {
	return docker.Status{Up: f.up}
}

func (f fakeDocker) Info(_ context.Context, _ int64, _ string) (docker.SystemInfo, error) {
	return f.info, f.infoErr
}

const testCreds = `{"email":"admin@rivly.dev","password":"s3cret-password","displayName":"Admin"}`

func authedClient(t *testing.T, ts *httptest.Server) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if code := postStatus(t, client, ts.URL+"/api/v1/setup", testCreds); code != http.StatusCreated {
		t.Fatalf("setup: want 201, got %d", code)
	}
	return client
}

func seedEnvironment(t *testing.T, srv *Server) {
	t.Helper()
	if _, err := srv.queries.CreateEnvironment(context.Background(), db.CreateEnvironmentParams{
		Name: "local",
		Kind: "local",
		Url:  "unix:///var/run/docker.sock",
	}); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
}

func TestEnvironments(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{
		up:   true,
		info: docker.SystemInfo{ServerVersion: "28.5.2", Containers: 3, ContainersRunning: 2, Images: 5},
	}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/environments"); code != http.StatusUnauthorized {
		t.Fatalf("environments before auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var envs []environmentResponse
	getJSON(t, client, ts.URL+"/api/v1/environments", &envs)
	if len(envs) != 1 {
		t.Fatalf("environments: want 1, got %d", len(envs))
	}
	if envs[0].Name != "local" || envs[0].Kind != "local" || envs[0].Status != "up" {
		t.Fatalf("environment: got %+v", envs[0])
	}

	var detail environmentDetailResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1", &detail)
	if detail.Status != "up" || detail.System == nil {
		t.Fatalf("detail: got %+v", detail)
	}
	if detail.System.ServerVersion != "28.5.2" || detail.System.Containers != 3 {
		t.Fatalf("detail system: got %+v", detail.System)
	}

	if code := getStatus(t, client, ts.URL+"/api/v1/environments/999"); code != http.StatusNotFound {
		t.Fatalf("missing environment: want 404, got %d", code)
	}
}

func TestEnvironmentDaemonDown(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{up: false, infoErr: errors.New("cannot connect to the docker daemon")}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var envs []environmentResponse
	getJSON(t, client, ts.URL+"/api/v1/environments", &envs)
	if len(envs) != 1 || envs[0].Status != "down" {
		t.Fatalf("environments when daemon down: got %+v", envs)
	}

	var detail environmentDetailResponse
	getJSON(t, client, ts.URL+"/api/v1/environments/1", &detail)
	if detail.Status != "down" || detail.System != nil {
		t.Fatalf("detail when daemon down: got %+v", detail)
	}
}
