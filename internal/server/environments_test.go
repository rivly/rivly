package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type fakeDocker struct {
	info               docker.SystemInfo
	infoErr            error
	containers         []docker.Container
	containersErr      error
	detail             docker.ContainerDetail
	detailErr          error
	createdContainerID string
	createContainerErr error
	statsData          []docker.Stats
	statsErr           error
	images             []docker.Image
	imagesErr          error
	imageActionErr     error
	imageDetail        docker.ImageDetail
	imageDetailErr     error
	pullData           []docker.PullProgress
	pullErr            error
	pruneResult        docker.PruneResult
	pruneErr           error
	volumes            []docker.Volume
	volumesErr         error
	volumeActionErr    error
	volumeCreated      docker.Volume
	volumeCreateErr    error
	volumeDetail       docker.VolumeDetail
	volumeDetailErr    error
	networks           []docker.Network
	networksErr        error
	networkActionErr   error
	networkCreated     docker.CreatedNetwork
	networkCreateErr   error
	networkDetail      docker.NetworkDetail
	networkDetailErr   error
	stacks             []docker.Stack
	stacksErr          error
	stackActionErr     error
	logLines           []docker.LogLine
	logErr             error
	execErr            error
	actionErr          error
	registryLoginErr   error
}

func (f fakeDocker) Info(_ context.Context, _ int64, _ string) (docker.SystemInfo, error) {
	return f.info, f.infoErr
}

func (f fakeDocker) Containers(_ context.Context, _ int64, _ string) ([]docker.Container, error) {
	return f.containers, f.containersErr
}

func (f fakeDocker) ContainerDetail(_ context.Context, _ int64, _, _ string) (docker.ContainerDetail, error) {
	return f.detail, f.detailErr
}

func (f fakeDocker) ContainerCreate(_ context.Context, _ int64, _ string, _ docker.ContainerCreateInput) (string, error) {
	return f.createdContainerID, f.createContainerErr
}

func (f fakeDocker) ContainerStats(_ context.Context, _ int64, _, _ string) (<-chan docker.Stats, error) {
	if f.statsErr != nil {
		return nil, f.statsErr
	}
	out := make(chan docker.Stats, len(f.statsData))
	for _, st := range f.statsData {
		out <- st
	}
	close(out)
	return out, nil
}

func (f fakeDocker) Images(_ context.Context, _ int64, _ string) ([]docker.Image, error) {
	return f.images, f.imagesErr
}

func (f fakeDocker) ImageAction(_ context.Context, _ int64, _, _, _ string) error {
	return f.imageActionErr
}

func (f fakeDocker) ImageDetail(_ context.Context, _ int64, _, _ string) (docker.ImageDetail, error) {
	return f.imageDetail, f.imageDetailErr
}

func (f fakeDocker) ImagePull(_ context.Context, _ int64, _, _ string) (<-chan docker.PullProgress, error) {
	if f.pullErr != nil {
		return nil, f.pullErr
	}
	out := make(chan docker.PullProgress, len(f.pullData))
	for _, p := range f.pullData {
		out <- p
	}
	close(out)
	return out, nil
}

func (f fakeDocker) ImagesPrune(_ context.Context, _ int64, _ string, _ bool) (docker.PruneResult, error) {
	return f.pruneResult, f.pruneErr
}

func (f fakeDocker) Volumes(_ context.Context, _ int64, _ string) ([]docker.Volume, error) {
	return f.volumes, f.volumesErr
}

func (f fakeDocker) VolumeAction(_ context.Context, _ int64, _, _, _ string) error {
	return f.volumeActionErr
}

func (f fakeDocker) VolumeCreate(_ context.Context, _ int64, _ string, _ docker.VolumeCreateInput) (docker.Volume, error) {
	return f.volumeCreated, f.volumeCreateErr
}

func (f fakeDocker) VolumeDetail(_ context.Context, _ int64, _, _ string) (docker.VolumeDetail, error) {
	return f.volumeDetail, f.volumeDetailErr
}

func (f fakeDocker) Networks(_ context.Context, _ int64, _ string) ([]docker.Network, error) {
	return f.networks, f.networksErr
}

func (f fakeDocker) NetworkAction(_ context.Context, _ int64, _, _, _ string) error {
	return f.networkActionErr
}

func (f fakeDocker) NetworkCreate(_ context.Context, _ int64, _ string, _ docker.NetworkCreateInput) (docker.CreatedNetwork, error) {
	return f.networkCreated, f.networkCreateErr
}

func (f fakeDocker) NetworkDetail(_ context.Context, _ int64, _, _ string) (docker.NetworkDetail, error) {
	return f.networkDetail, f.networkDetailErr
}

func (f fakeDocker) Stacks(_ context.Context, _ int64, _ string) ([]docker.Stack, error) {
	return f.stacks, f.stacksErr
}

func (f fakeDocker) StackAction(_ context.Context, _ int64, _, _, _ string) error {
	return f.stackActionErr
}

func (f fakeDocker) ContainerLogs(_ context.Context, _ int64, _, _ string, _ int, _ bool) (<-chan docker.LogLine, error) {
	if f.logErr != nil {
		return nil, f.logErr
	}
	out := make(chan docker.LogLine, len(f.logLines))
	for _, line := range f.logLines {
		out <- line
	}
	close(out)
	return out, nil
}

func (f fakeDocker) ContainerExec(_ context.Context, _ int64, _, _ string) (*docker.ExecSession, error) {
	return nil, f.execErr
}

func (f fakeDocker) ContainerAction(_ context.Context, _ int64, _, _, _ string) error {
	return f.actionErr
}

func (f fakeDocker) WatchEvents(_ context.Context, _ int64, _ string) (<-chan struct{}, <-chan error) {
	return nil, nil
}

func (f fakeDocker) RegistryLogin(_ context.Context, _ int64, _, _, _, _ string) error {
	return f.registryLoginErr
}

type fakeCompose struct {
	deployOut string
	deployErr error
	removeErr error
	repoDir   string
}

func (f fakeCompose) Deploy(_ context.Context, _ string, _ int64, _, _, _ string) (string, error) {
	return f.deployOut, f.deployErr
}

func (f fakeCompose) Remove(_ context.Context, _ string, _ int64, _, _, _ string) (string, error) {
	return "", f.removeErr
}

func (f fakeCompose) Discard(_ context.Context, _ string, _ int64, _ string) {}

func (f fakeCompose) RepoDir(_ int64, project string) string {
	return filepath.Join(f.repoDir, project)
}

func (f fakeCompose) DeployRepo(_ context.Context, _ string, _ int64, _, _, _ string) (string, error) {
	return f.deployOut, f.deployErr
}

func (f fakeCompose) RemoveRepo(_ context.Context, _ string, _ int64, _, _, _ string) (string, error) {
	return "", f.removeErr
}

func (f fakeCompose) DiscardRepo(_ context.Context, _ string, _ int64, _, _ string) {}

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
	srv.docker = fakeDocker{infoErr: errors.New("cannot connect to the docker daemon")}
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

func TestEnvironmentSnapshotWhenDown(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{infoErr: errors.New("cannot connect")}
	seedEnvironment(t, srv)

	snap, err := json.Marshal(docker.SystemInfo{ServerVersion: "1.2.3", Containers: 9, Images: 4})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if err := srv.queries.UpdateEnvironmentSnapshot(context.Background(), db.UpdateEnvironmentSnapshotParams{
		Snapshot: sql.NullString{String: string(snap), Valid: true},
		ID:       1,
	}); err != nil {
		t.Fatalf("UpdateEnvironmentSnapshot: %v", err)
	}

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	var envs []environmentDetailResponse
	getJSON(t, client, ts.URL+"/api/v1/environments", &envs)
	if len(envs) != 1 || envs[0].Status != "down" {
		t.Fatalf("snapshot env: got %+v", envs)
	}
	if envs[0].System == nil || envs[0].System.ServerVersion != "1.2.3" || envs[0].System.Containers != 9 {
		t.Fatalf("expected snapshot system on down env: got %+v", envs[0].System)
	}
	if envs[0].LastSeen == nil {
		t.Fatalf("expected lastSeen to be set")
	}
}
