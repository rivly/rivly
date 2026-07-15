package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	defaultLogTail = 200
	maxLogTail     = 5000
)

type logLineResponse struct {
	Stream  string `json:"stream"`
	Message string `json:"message"`
}

func (s *Server) handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	containerID := chi.URLParam(r, "containerID")

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

	lines, err := s.docker.ContainerLogs(ctx, env.ID, env.Url, containerID, parseLogTail(r), true)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not stream container logs")
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
		case line, ok := <-lines:
			if !ok {
				if _, err := fmt.Fprint(w, "event: end\ndata: {}\n\n"); err != nil {
					return
				}
				flusher.Flush()
				return
			}
			data, err := json.Marshal(logLineResponse{Stream: line.Stream, Message: line.Message})
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

func parseLogTail(r *http.Request) int {
	raw := r.URL.Query().Get("tail")
	if raw == "" {
		return defaultLogTail
	}
	tail, err := strconv.Atoi(raw)
	if err != nil || tail <= 0 {
		return defaultLogTail
	}
	if tail > maxLogTail {
		return maxLogTail
	}
	return tail
}
