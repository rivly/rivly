package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type volumeResponse struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	Stack      string `json:"stack"`
	Created    int64  `json:"created"`
	InUse      bool   `json:"inUse"`
}

var validVolumeActions = map[string]bool{"remove": true}

func (s *Server) handleListVolumes(w http.ResponseWriter, r *http.Request) {
	env := environmentFrom(r)

	volumes, err := s.docker.Volumes(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	out := make([]volumeResponse, 0, len(volumes))
	for _, v := range volumes {
		out = append(out, volumeResponse{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Stack:      v.Stack,
			Created:    v.Created,
			InUse:      v.InUse,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

type createVolumeRequest struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
}

func (s *Server) handleCreateVolume(w http.ResponseWriter, r *http.Request) {
	var req createVolumeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if !resourceNamePattern.MatchString(req.Name) {
		s.writeError(w, http.StatusBadRequest, "invalid volume name")
		return
	}

	env := environmentFrom(r)

	vol, err := s.docker.VolumeCreate(r.Context(), env.ID, env.Url, docker.VolumeCreateInput{
		Name:   req.Name,
		Driver: strings.TrimSpace(req.Driver),
	})
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not create volume")
		return
	}
	s.publishEnvironment(r.Context(), env)
	s.writeJSON(w, http.StatusCreated, volumeResponse{
		Name:       vol.Name,
		Driver:     vol.Driver,
		Mountpoint: vol.Mountpoint,
		Stack:      vol.Stack,
		Created:    vol.Created,
		InUse:      vol.InUse,
	})
}

type volumeContainerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type volumeDetailResponse struct {
	Name       string                    `json:"name"`
	Driver     string                    `json:"driver"`
	Mountpoint string                    `json:"mountpoint"`
	Scope      string                    `json:"scope"`
	Created    int64                     `json:"created"`
	Labels     map[string]string         `json:"labels"`
	Options    map[string]string         `json:"options"`
	Containers []volumeContainerResponse `json:"containers"`
}

func (s *Server) handleVolumeDetail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	env := environmentFrom(r)

	detail, err := s.docker.VolumeDetail(r.Context(), env.ID, env.Url, name)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not inspect volume")
		return
	}

	containers := make([]volumeContainerResponse, 0, len(detail.Containers))
	for _, c := range detail.Containers {
		containers = append(containers, volumeContainerResponse{ID: c.ID, Name: c.Name})
	}
	s.writeJSON(w, http.StatusOK, volumeDetailResponse{
		Name:       detail.Name,
		Driver:     detail.Driver,
		Mountpoint: detail.Mountpoint,
		Scope:      detail.Scope,
		Created:    detail.Created,
		Labels:     detail.Labels,
		Options:    detail.Options,
		Containers: containers,
	})
}

func (s *Server) handleVolumeActions(w http.ResponseWriter, r *http.Request) {
	s.handleBulkAction(w, r, bulkAction{
		noun:    "volume",
		allowed: validVolumeActions,
		apply: func(ctx context.Context, env db.Environment, id, action string) error {
			return s.docker.VolumeAction(ctx, env.ID, env.Url, id, action)
		},
	})
}
