package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"

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

func (s *Server) handleImageActions(w http.ResponseWriter, r *http.Request) {
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
