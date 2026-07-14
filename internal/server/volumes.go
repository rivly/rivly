package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/docker"
)

var resourceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req createVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if !resourceNamePattern.MatchString(req.Name) {
		s.writeError(w, http.StatusBadRequest, "invalid volume name")
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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	name := chi.URLParam(r, "name")

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req bulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validVolumeActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid volume selection")
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

	results := make([]actionResult, len(req.IDs))
	var wg sync.WaitGroup
	for i, name := range req.IDs {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			if err := s.docker.VolumeAction(r.Context(), env.ID, env.Url, name, req.Action); err != nil {
				s.logger.Warn("volume action failed",
					"action", req.Action, "volume", name, "err", err)
				results[i] = actionResult{ID: name, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: name, OK: true}
		}(i, name)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
