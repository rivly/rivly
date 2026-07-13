package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type environmentResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
	Status   string `json:"status"`
	LastSeen *int64 `json:"lastSeen,omitempty"`
}

type systemInfoResponse struct {
	ServerVersion     string `json:"serverVersion"`
	OSType            string `json:"osType"`
	Architecture      string `json:"architecture"`
	KernelVersion     string `json:"kernelVersion"`
	OperatingSystem   string `json:"operatingSystem"`
	Name              string `json:"name"`
	Swarm             bool   `json:"swarm"`
	Nodes             int    `json:"nodes"`
	NCPU              int    `json:"ncpu"`
	MemTotal          int64  `json:"memTotal"`
	Containers        int    `json:"containers"`
	ContainersRunning int    `json:"containersRunning"`
	ContainersPaused  int    `json:"containersPaused"`
	ContainersStopped int    `json:"containersStopped"`
	Images            int    `json:"images"`
}

type environmentDetailResponse struct {
	environmentResponse
	System *systemInfoResponse `json:"system,omitempty"`
}

func toEnvironmentResponse(e db.Environment, status string) environmentResponse {
	resp := environmentResponse{
		ID:     e.ID,
		Name:   e.Name,
		Kind:   e.Kind,
		URL:    e.Url,
		Status: status,
	}
	if e.SnapshotAt.Valid {
		seen := e.SnapshotAt.Int64
		resp.LastSeen = &seen
	}
	return resp
}

func statusLabel(up bool) string {
	if up {
		return "up"
	}
	return "down"
}

// buildEnvironment queries the daemon live. On success it returns the fresh
// system info and refreshes the stored snapshot; on failure it falls back to
// the last known snapshot so the environment stays informative while down.
func (s *Server) buildEnvironment(ctx context.Context, e db.Environment) environmentDetailResponse {
	detail := environmentDetailResponse{
		environmentResponse: toEnvironmentResponse(e, statusLabel(false)),
	}

	if info, err := s.docker.Info(ctx, e.ID, e.Url); err == nil {
		detail.Status = statusLabel(true)
		detail.System = toSystemInfoResponse(info)
		s.saveSnapshot(ctx, e.ID, info)
		return detail
	}

	if e.Snapshot.Valid {
		var snap docker.SystemInfo
		if err := json.Unmarshal([]byte(e.Snapshot.String), &snap); err == nil {
			detail.System = toSystemInfoResponse(snap)
		}
	}
	return detail
}

func (s *Server) saveSnapshot(ctx context.Context, id int64, info docker.SystemInfo) {
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

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.queries.ListEnvironments(r.Context())
	if err != nil {
		s.serverError(w, r, "could not list environments", err)
		return
	}

	out := make([]environmentDetailResponse, len(envs))
	var wg sync.WaitGroup
	for i, e := range envs {
		wg.Add(1)
		go func(i int, e db.Environment) {
			defer wg.Done()
			out[i] = s.buildEnvironment(r.Context(), e)
		}(i, e)
	}
	wg.Wait()
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	s.writeJSON(w, http.StatusOK, s.buildEnvironment(r.Context(), env))
}

func toSystemInfoResponse(i docker.SystemInfo) *systemInfoResponse {
	return &systemInfoResponse{
		ServerVersion:     i.ServerVersion,
		OSType:            i.OSType,
		Architecture:      i.Architecture,
		KernelVersion:     i.KernelVersion,
		OperatingSystem:   i.OperatingSystem,
		Name:              i.Name,
		Swarm:             i.Swarm,
		Nodes:             i.Nodes,
		NCPU:              i.NCPU,
		MemTotal:          i.MemTotal,
		Containers:        i.Containers,
		ContainersRunning: i.ContainersRunning,
		ContainersPaused:  i.ContainersPaused,
		ContainersStopped: i.ContainersStopped,
		Images:            i.Images,
	}
}
