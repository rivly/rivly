package server

import (
	"net/http"
	"sync"
)

const maxBulkActions = 200

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

func (s *Server) handleContainerActions(w http.ResponseWriter, r *http.Request) {
	var req bulkActionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	if !validActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid container selection")
		return
	}

	env := environmentFrom(r)

	results := make([]actionResult, len(req.IDs))
	var wg sync.WaitGroup
	for i, containerID := range req.IDs {
		wg.Add(1)
		go func(i int, containerID string) {
			defer wg.Done()
			if err := s.docker.ContainerAction(r.Context(), env.ID, env.Url, containerID, req.Action); err != nil {
				s.logger.Warn("container action failed",
					"action", req.Action, "container", containerID, "err", err)
				results[i] = actionResult{ID: containerID, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: containerID, OK: true}
		}(i, containerID)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
