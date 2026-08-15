package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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
	detached := context.WithoutCancel(ctx)
	s.spawn(func() { s.environments.Publish(detached, e) })
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

const maxEnvironmentName = 64

var dockerHostSchemes = map[string]bool{
	"unix":  true,
	"npipe": true,
	"tcp":   true,
	"ssh":   true,
	"http":  true,
	"https": true,
}

type environmentInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func validateEnvironment(in environmentInput) (name, endpoint, kind, message string, ok bool) {
	name = strings.TrimSpace(in.Name)
	if name == "" {
		return "", "", "", "a name is required", false
	}
	if len(name) > maxEnvironmentName {
		return "", "", "", "name is too long", false
	}

	endpoint = strings.TrimSpace(in.URL)
	parsed, err := url.Parse(endpoint)
	if err != nil || !dockerHostSchemes[parsed.Scheme] {
		return "", "", "", "endpoint must start with unix://, tcp://, ssh:// or npipe://", false
	}
	if parsed.Scheme == "unix" || parsed.Scheme == "npipe" {
		if parsed.Path == "" {
			return "", "", "", "a socket path is required", false
		}
		kind = "local"
	} else {
		if parsed.Host == "" {
			return "", "", "", "a host is required", false
		}
		kind = "remote"
	}
	return name, endpoint, kind, "", true
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var in environmentInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.badRequest(w, err)
		return
	}
	name, endpoint, kind, message, ok := validateEnvironment(in)
	if !ok {
		s.writeError(w, http.StatusBadRequest, message)
		return
	}

	created, err := s.queries.CreateEnvironment(r.Context(), db.CreateEnvironmentParams{
		Name: name,
		Kind: kind,
		Url:  endpoint,
	})
	if err != nil {
		s.serverError(w, r, "could not create the environment", err)
		return
	}

	s.publishEnvironment(r.Context(), created)
	s.writeJSON(w, http.StatusCreated, toEnvironmentDetailResponse(s.environments.Build(r.Context(), created)))
}

func (s *Server) handleUpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var in environmentInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.badRequest(w, err)
		return
	}
	name, endpoint, kind, message, ok := validateEnvironment(in)
	if !ok {
		s.writeError(w, http.StatusBadRequest, message)
		return
	}

	updated, err := s.queries.UpdateEnvironment(r.Context(), db.UpdateEnvironmentParams{
		Name: name,
		Kind: kind,
		Url:  endpoint,
		ID:   id,
	})
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not update the environment", err)
		return
	}

	s.publishEnvironment(r.Context(), updated)
	s.writeJSON(w, http.StatusOK, toEnvironmentDetailResponse(s.environments.Build(r.Context(), updated)))
}

func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	removed, err := s.queries.DeleteEnvironment(r.Context(), id)
	if err != nil {
		s.serverError(w, r, "could not remove the environment", err)
		return
	}
	if removed == 0 {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}

	s.events.Publish("environment.removed", map[string]int64{"id": id})
	w.WriteHeader(http.StatusNoContent)
}
