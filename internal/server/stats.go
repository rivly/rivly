package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type statsResponse struct {
	CPUPercent float64 `json:"cpuPercent"`
	MemUsage   uint64  `json:"memUsage"`
	MemLimit   uint64  `json:"memLimit"`
	MemPercent float64 `json:"memPercent"`
	NetRx      uint64  `json:"netRx"`
	NetTx      uint64  `json:"netTx"`
	BlockRead  uint64  `json:"blockRead"`
	BlockWrite uint64  `json:"blockWrite"`
	Pids       uint64  `json:"pids"`
}

func (s *Server) handleContainerStats(w http.ResponseWriter, r *http.Request) {
	containerID := chi.URLParam(r, "containerID")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	ctx, cancel := s.streamContext(r)
	defer cancel()

	stream, err := dockerFrom(r).ContainerStats(ctx, containerID)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not stream stats")
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
		case st, ok := <-stream:
			if !ok {
				if _, err := fmt.Fprint(w, "event: end\ndata: {}\n\n"); err != nil {
					return
				}
				flusher.Flush()
				return
			}
			data, err := json.Marshal(statsResponse{
				CPUPercent: st.CPUPercent,
				MemUsage:   st.MemUsage,
				MemLimit:   st.MemLimit,
				MemPercent: st.MemPercent,
				NetRx:      st.NetRx,
				NetTx:      st.NetTx,
				BlockRead:  st.BlockRead,
				BlockWrite: st.BlockWrite,
				Pids:       st.Pids,
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
