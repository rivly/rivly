package server

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type environmentResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

type systemInfoResponse struct {
	ServerVersion     string `json:"serverVersion"`
	OSType            string `json:"osType"`
	Architecture      string `json:"architecture"`
	KernelVersion     string `json:"kernelVersion"`
	OperatingSystem   string `json:"operatingSystem"`
	Name              string `json:"name"`
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
	return environmentResponse{
		ID:     e.ID,
		Name:   e.Name,
		Kind:   e.Kind,
		URL:    e.Url,
		Status: status,
	}
}

func statusLabel(up bool) string {
	if up {
		return "up"
	}
	return "down"
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.queries.ListEnvironments(r.Context())
	if err != nil {
		s.serverError(w, r, "could not list environments", err)
		return
	}

	out := make([]environmentResponse, 0, len(envs))
	for _, e := range envs {
		status := s.docker.Ping(r.Context(), e.ID, e.Url)
		out = append(out, toEnvironmentResponse(e, statusLabel(status.Up)))
	}
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

	detail := environmentDetailResponse{
		environmentResponse: toEnvironmentResponse(env, statusLabel(false)),
	}
	if info, err := s.docker.Info(r.Context(), env.ID, env.Url); err == nil {
		detail.Status = statusLabel(true)
		detail.System = toSystemInfoResponse(info)
	}
	s.writeJSON(w, http.StatusOK, detail)
}

func toSystemInfoResponse(i docker.SystemInfo) *systemInfoResponse {
	return &systemInfoResponse{
		ServerVersion:     i.ServerVersion,
		OSType:            i.OSType,
		Architecture:      i.Architecture,
		KernelVersion:     i.KernelVersion,
		OperatingSystem:   i.OperatingSystem,
		Name:              i.Name,
		NCPU:              i.NCPU,
		MemTotal:          i.MemTotal,
		Containers:        i.Containers,
		ContainersRunning: i.ContainersRunning,
		ContainersPaused:  i.ContainersPaused,
		ContainersStopped: i.ContainersStopped,
		Images:            i.Images,
	}
}
