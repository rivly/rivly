package environment

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type fakeDocker struct {
	info         docker.SystemInfo
	infoErr      error
	watchStarted chan int64
}

func (f fakeDocker) Info(_ context.Context) (docker.SystemInfo, error) {
	return f.info, f.infoErr
}

func (f fakeDocker) factory() Docker {
	return func(id int64, _ string) (DockerClient, error) { return environmentClient{f, id}, nil }
}

type environmentClient struct {
	fakeDocker
	id int64
}

func (c environmentClient) WatchEvents(ctx context.Context) (<-chan struct{}, <-chan error) {
	return c.watch(c.id)
}

func (f fakeDocker) watch(id int64) (<-chan struct{}, <-chan error) {
	if f.watchStarted != nil {
		select {
		case f.watchStarted <- id:
		default:
		}
	}
	return nil, nil
}

type watcherHarness struct {
	*Service
	queries  *db.Queries
	observed chan Detail
	running  sync.WaitGroup
}

func newWatcherHarness(t *testing.T, dockerClient Docker) *watcherHarness {
	t.Helper()

	sqlDB, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	h := &watcherHarness{
		queries:  db.New(sqlDB),
		observed: make(chan Detail, 16),
	}
	h.Service = New(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		h.queries,
		dockerClient,
		0,
		func(_ context.Context, d Detail) {
			select {
			case h.observed <- d:
			default:
			}
		},
		nil,
		func(fn func()) {
			h.running.Add(1)
			go func() {
				defer h.running.Done()
				fn()
			}()
		},
	)
	return h
}

func (h *watcherHarness) background(ctx context.Context, loop func(context.Context)) {
	h.running.Add(1)
	go func() {
		defer h.running.Done()
		loop(ctx)
	}()
}

func (h *watcherHarness) wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		h.running.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *watcherHarness) seedEnvironment(t *testing.T) {
	t.Helper()
	if _, err := h.queries.CreateEnvironment(context.Background(), db.CreateEnvironmentParams{
		Name: "local", Kind: "local", Url: "unix:///var/run/docker.sock",
	}); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
}
