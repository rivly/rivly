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

type networkResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	Scope   string `json:"scope"`
	Stack   string `json:"stack"`
	Created int64  `json:"created"`
	InUse   bool   `json:"inUse"`
}

var validNetworkActions = map[string]bool{"remove": true}

func (s *Server) handleListNetworks(w http.ResponseWriter, r *http.Request) {
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

	networks, err := s.docker.Networks(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	out := make([]networkResponse, 0, len(networks))
	for _, n := range networks {
		out = append(out, networkResponse{
			ID:      n.ID,
			Name:    n.Name,
			Driver:  n.Driver,
			Scope:   n.Scope,
			Stack:   n.Stack,
			Created: n.Created,
			InUse:   n.InUse,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleNetworkActions(w http.ResponseWriter, r *http.Request) {
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
	if !validNetworkActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid network selection")
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
	for i, networkID := range req.IDs {
		wg.Add(1)
		go func(i int, networkID string) {
			defer wg.Done()
			if err := s.docker.NetworkAction(r.Context(), env.ID, env.Url, networkID, req.Action); err != nil {
				s.logger.Warn("network action failed",
					"action", req.Action, "network", networkID, "err", err)
				results[i] = actionResult{ID: networkID, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: networkID, OK: true}
		}(i, networkID)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
