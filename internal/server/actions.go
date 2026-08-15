package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/rivly/rivly/internal/database/db"
)

const (
	maxBulkActions     = 200
	maxParallelActions = 8
)

var validActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"pause":   true,
	"unpause": true,
	"kill":    true,
	"remove":  true,
}

type bulkActionRequest struct {
	Action string   `json:"action"`
	IDs    []string `json:"ids"`
}

type actionResult struct {
	ID    string `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type bulkAction struct {
	noun    string
	allowed map[string]bool
	apply   func(ctx context.Context, env db.Environment, id, action string) error
}

func (s *Server) handleBulkAction(w http.ResponseWriter, r *http.Request, spec bulkAction) {
	var req bulkActionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	if !spec.allowed[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid "+spec.noun+" selection")
		return
	}

	ctx := r.Context()
	env := environmentFrom(r)

	results := make([]actionResult, len(req.IDs))
	sem := make(chan struct{}, maxParallelActions)
	var wg sync.WaitGroup
	for i, id := range req.IDs {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := spec.apply(ctx, env, id, req.Action); err != nil {
				s.logger.Warn("bulk action failed",
					"kind", spec.noun, "action", req.Action, "id", id, "err", err)
				results[i] = actionResult{ID: id, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: id, OK: true}
		}()
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleContainerActions(w http.ResponseWriter, r *http.Request) {
	s.handleBulkAction(w, r, bulkAction{
		noun:    "container",
		allowed: validActions,
		apply: func(ctx context.Context, env db.Environment, id, action string) error {
			return dockerFrom(r).ContainerAction(ctx, id, action)
		},
	})
}
