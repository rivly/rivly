package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const eventHeartbeat = 25 * time.Second

func (s *Server) requireEventAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.sessions.Cookie.Name)
		if err != nil {
			s.writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx, err := s.sessions.Load(r.Context(), cookie.Value)
		if err != nil {
			s.serverError(w, r, "could not load session", err)
			return
		}
		if s.sessions.GetInt64(ctx, sessionUserID) == 0 {
			s.writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub, unsubscribe := s.events.Subscribe()
	defer unsubscribe()

	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(eventHeartbeat)
	defer heartbeat.Stop()

	ctx, cancel := s.streamContext(r)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case evt := <-sub:
			data, err := json.Marshal(evt)
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
