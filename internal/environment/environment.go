package environment

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

const (
	StatusUp   = "up"
	StatusDown = "down"
)

const maxParallelProbes = 8

type DockerClient interface {
	Info(ctx context.Context) (docker.SystemInfo, error)
	WatchEvents(ctx context.Context) (<-chan struct{}, <-chan error)
}

type Docker func(envID int64, host string) (DockerClient, error)

type Detail struct {
	ID       int64
	Name     string
	Kind     string
	URL      string
	Status   string
	LastSeen *int64
	System   *docker.SystemInfo
}

type Observer func(ctx context.Context, detail Detail)

type Signer func(ctx context.Context, envID int64) string

type Spawner func(fn func())

type Service struct {
	logger       *slog.Logger
	queries      *db.Queries
	docker       Docker
	observe      Observer
	sign         Signer
	spawn        Spawner
	pollInterval time.Duration

	mu        sync.Mutex
	lastState map[int64]string
}

func New(
	logger *slog.Logger,
	queries *db.Queries,
	dockerClient Docker,
	pollInterval time.Duration,
	observe Observer,
	sign Signer,
	spawn Spawner,
) *Service {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &Service{
		logger:       logger,
		queries:      queries,
		docker:       dockerClient,
		observe:      observe,
		sign:         sign,
		spawn:        spawn,
		pollInterval: pollInterval,
		lastState:    make(map[int64]string),
	}
}

func (s *Service) Build(ctx context.Context, e db.Environment) Detail {
	detail := Detail{
		ID:     e.ID,
		Name:   e.Name,
		Kind:   e.Kind,
		URL:    e.Url,
		Status: StatusDown,
	}
	if e.SnapshotAt.Valid {
		seen := e.SnapshotAt.Int64
		detail.LastSeen = &seen
	}

	if client, cerr := s.docker(e.ID, e.Url); cerr == nil {
		if info, err := client.Info(ctx); err == nil {
			detail.Status = StatusUp
			detail.System = &info
			s.saveSnapshot(ctx, e.ID, info)
			return detail
		}
	}

	if e.Snapshot.Valid {
		var snapshot docker.SystemInfo
		if err := json.Unmarshal([]byte(e.Snapshot.String), &snapshot); err == nil {
			detail.System = &snapshot
		}
	}
	return detail
}

func (s *Service) BuildAll(ctx context.Context) ([]Detail, error) {
	envs, err := s.queries.ListEnvironments(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]Detail, len(envs))
	sem := make(chan struct{}, maxParallelProbes)
	var wg sync.WaitGroup
	for i, e := range envs {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out[i] = s.Build(ctx, e)
		}()
	}
	wg.Wait()
	return out, nil
}

func (s *Service) saveSnapshot(ctx context.Context, id int64, info docker.SystemInfo) {
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	if err := s.queries.UpdateEnvironmentSnapshot(ctx, db.UpdateEnvironmentSnapshotParams{
		Snapshot: sql.NullString{String: string(data), Valid: true},
		ID:       id,
	}); err != nil {
		s.logger.Error("could not save environment snapshot", "err", err, "env", id)
	}
}
