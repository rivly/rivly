package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type imageResponse struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
	InUse   bool     `json:"inUse"`
}

var validImageActions = map[string]bool{"remove": true}

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
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

	images, err := s.docker.Images(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	out := make([]imageResponse, 0, len(images))
	for _, img := range images {
		out = append(out, imageResponse{
			ID:      img.ID,
			Tags:    img.Tags,
			Size:    img.Size,
			Created: img.Created,
			InUse:   img.InUse,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

type imageContainerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type imageDetailResponse struct {
	ID           string                   `json:"id"`
	Tags         []string                 `json:"tags"`
	Digests      []string                 `json:"digests"`
	Size         int64                    `json:"size"`
	Created      int64                    `json:"created"`
	Architecture string                   `json:"architecture"`
	Os           string                   `json:"os"`
	Author       string                   `json:"author"`
	WorkingDir   string                   `json:"workingDir"`
	Command      []string                 `json:"command"`
	Entrypoint   []string                 `json:"entrypoint"`
	Env          []string                 `json:"env"`
	ExposedPorts []string                 `json:"exposedPorts"`
	Labels       map[string]string        `json:"labels"`
	Containers   []imageContainerResponse `json:"containers"`
}

func (s *Server) handleImageDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	imageID := chi.URLParam(r, "imageID")

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	detail, err := s.docker.ImageDetail(r.Context(), env.ID, env.Url, imageID)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not inspect image")
		return
	}

	containers := make([]imageContainerResponse, 0, len(detail.Containers))
	for _, c := range detail.Containers {
		containers = append(containers, imageContainerResponse{ID: c.ID, Name: c.Name})
	}
	s.writeJSON(w, http.StatusOK, imageDetailResponse{
		ID:           detail.ID,
		Tags:         detail.Tags,
		Digests:      detail.Digests,
		Size:         detail.Size,
		Created:      detail.Created,
		Architecture: detail.Architecture,
		Os:           detail.Os,
		Author:       detail.Author,
		WorkingDir:   detail.WorkingDir,
		Command:      detail.Command,
		Entrypoint:   detail.Entrypoint,
		Env:          detail.Env,
		ExposedPorts: detail.ExposedPorts,
		Labels:       detail.Labels,
		Containers:   containers,
	})
}

func (s *Server) handleImageActions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req bulkActionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	if !validImageActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid image selection")
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
	for i, imageID := range req.IDs {
		wg.Add(1)
		go func(i int, imageID string) {
			defer wg.Done()
			if err := s.docker.ImageAction(r.Context(), env.ID, env.Url, imageID, req.Action); err != nil {
				s.logger.Warn("image action failed",
					"action", req.Action, "image", imageID, "err", err)
				results[i] = actionResult{ID: imageID, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: imageID, OK: true}
		}(i, imageID)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

type pullProgressResponse struct {
	Status  string `json:"status,omitempty"`
	ID      string `json:"id,omitempty"`
	Current int64  `json:"current,omitempty"`
	Total   int64  `json:"total,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleImagePull(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	ref := strings.TrimSpace(r.URL.Query().Get("ref"))
	if ref == "" {
		s.writeError(w, http.StatusBadRequest, "image reference is required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming is not supported")
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

	ctx, cancel := s.streamContext(r)
	defer cancel()

	stream, err := s.docker.ImagePull(ctx, env.ID, env.Url, ref)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not pull image")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(eventHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case p, ok := <-stream:
			if !ok {
				if _, err := fmt.Fprint(w, "event: end\ndata: {}\n\n"); err != nil {
					return
				}
				flusher.Flush()
				s.publishEnvironment(ctx, env)
				return
			}
			data, err := json.Marshal(pullProgressResponse{
				Status:  p.Status,
				ID:      p.ID,
				Current: p.Current,
				Total:   p.Total,
				Error:   p.Error,
			})
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

type imagePruneRequest struct {
	All bool `json:"all"`
}

func (s *Server) handleImagePrune(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req imagePruneRequest
	if err := decodeJSON(w, r, &req); err != nil && !errors.Is(err, io.EOF) {
		s.badRequest(w, err)
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

	res, err := s.docker.ImagesPrune(r.Context(), env.ID, env.Url, req.All)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not prune images")
		return
	}
	s.publishEnvironment(r.Context(), env)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"imagesDeleted":  res.ImagesDeleted,
		"spaceReclaimed": res.SpaceReclaimed,
	})
}
