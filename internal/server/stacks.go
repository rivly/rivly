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

type stackResponse struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Services   int    `json:"services"`
	Running    int    `json:"running"`
	Total      int    `json:"total"`
	State      string `json:"state"`
	WorkingDir string `json:"workingDir"`
}

var validStackActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"remove":  true,
}

func (s *Server) handleListStacks(w http.ResponseWriter, r *http.Request) {
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

	stacks, err := s.docker.Stacks(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	out := make([]stackResponse, 0, len(stacks))
	for _, st := range stacks {
		out = append(out, stackResponse{
			Name:       st.Name,
			Type:       st.Type,
			Services:   st.Services,
			Running:    st.Running,
			Total:      st.Total,
			State:      st.State,
			WorkingDir: st.WorkingDir,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleStackActions(w http.ResponseWriter, r *http.Request) {
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
	if !validStackActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid stack selection")
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
	for i, project := range req.IDs {
		wg.Add(1)
		go func(i int, project string) {
			defer wg.Done()
			if err := s.docker.StackAction(r.Context(), env.ID, env.Url, project, req.Action); err != nil {
				s.logger.Warn("stack action failed",
					"action", req.Action, "stack", project, "err", err)
				results[i] = actionResult{ID: project, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: project, OK: true}
		}(i, project)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
