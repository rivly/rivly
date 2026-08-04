package server

import (
	"context"
	"net/http"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
	"github.com/rivly/rivly/internal/environment"
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

func toEnvironmentDetailResponse(d environment.Detail) environmentDetailResponse {
	return environmentDetailResponse{
		environmentResponse: environmentResponse{
			ID:       d.ID,
			Name:     d.Name,
			Kind:     d.Kind,
			URL:      d.URL,
			Status:   d.Status,
			LastSeen: d.LastSeen,
		},
		System: toSystemInfoResponse(d.System),
	}
}

func toSystemInfoResponse(i *docker.SystemInfo) *systemInfoResponse {
	if i == nil {
		return nil
	}
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

func (s *Server) emitEnvironment(_ context.Context, detail environment.Detail) {
	s.events.Publish("environment.updated", toEnvironmentDetailResponse(detail))
}

func (s *Server) publishEnvironment(ctx context.Context, e db.Environment) {
	s.environments.Publish(ctx, e)
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	details, err := s.environments.BuildAll(r.Context())
	if err != nil {
		s.serverError(w, r, "could not list environments", err)
		return
	}

	out := make([]environmentDetailResponse, 0, len(details))
	for _, d := range details {
		out = append(out, toEnvironmentDetailResponse(d))
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	detail := s.environments.Build(r.Context(), environmentFrom(r))
	s.writeJSON(w, http.StatusOK, toEnvironmentDetailResponse(detail))
}
