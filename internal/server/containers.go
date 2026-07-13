package server

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type containerResponse struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Image   string         `json:"image"`
	State   string         `json:"state"`
	Status  string         `json:"status"`
	Stack   string         `json:"stack"`
	Created int64          `json:"created"`
	IP      string         `json:"ip"`
	Ports   []portResponse `json:"ports"`
}

type portResponse struct {
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort"`
	Type        string `json:"type"`
	IP          string `json:"ip,omitempty"`
}

func (s *Server) handleListContainers(w http.ResponseWriter, r *http.Request) {
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

	containers, err := s.docker.Containers(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	out := make([]containerResponse, 0, len(containers))
	for _, c := range containers {
		ports := make([]portResponse, 0, len(c.Ports))
		for _, p := range c.Ports {
			ports = append(ports, portResponse{
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
				IP:          p.IP,
			})
		}
		out = append(out, containerResponse{
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Stack:   c.Stack,
			Created: c.Created,
			IP:      c.IP,
			Ports:   ports,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}
