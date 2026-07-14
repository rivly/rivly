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

type networkAttachmentResponse struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type mountResponse struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Name        string `json:"name"`
	RW          bool   `json:"rw"`
}

type containerDetailResponse struct {
	ID            string                      `json:"id"`
	Name          string                      `json:"name"`
	Image         string                      `json:"image"`
	State         string                      `json:"state"`
	Created       int64                       `json:"created"`
	StartedAt     string                      `json:"startedAt"`
	Command       string                      `json:"command"`
	RestartPolicy string                      `json:"restartPolicy"`
	Ports         []portResponse              `json:"ports"`
	Networks      []networkAttachmentResponse `json:"networks"`
	Mounts        []mountResponse             `json:"mounts"`
	Env           []string                    `json:"env"`
	Labels        map[string]string           `json:"labels"`
}

func (s *Server) handleContainerDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	containerID := chi.URLParam(r, "containerID")

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	detail, err := s.docker.ContainerDetail(r.Context(), env.ID, env.Url, containerID)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not inspect container")
		return
	}

	ports := make([]portResponse, 0, len(detail.Ports))
	for _, p := range detail.Ports {
		ports = append(ports, portResponse{PrivatePort: p.PrivatePort, PublicPort: p.PublicPort, Type: p.Type, IP: p.IP})
	}
	networks := make([]networkAttachmentResponse, 0, len(detail.Networks))
	for _, n := range detail.Networks {
		networks = append(networks, networkAttachmentResponse{Name: n.Name, IP: n.IP})
	}
	mounts := make([]mountResponse, 0, len(detail.Mounts))
	for _, mnt := range detail.Mounts {
		mounts = append(mounts, mountResponse{Type: mnt.Type, Source: mnt.Source, Destination: mnt.Destination, Name: mnt.Name, RW: mnt.RW})
	}
	labels := detail.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	env2 := detail.Env
	if env2 == nil {
		env2 = []string{}
	}

	s.writeJSON(w, http.StatusOK, containerDetailResponse{
		ID:            detail.ID,
		Name:          detail.Name,
		Image:         detail.Image,
		State:         detail.State,
		Created:       detail.Created,
		StartedAt:     detail.StartedAt,
		Command:       detail.Command,
		RestartPolicy: detail.RestartPolicy,
		Ports:         ports,
		Networks:      networks,
		Mounts:        mounts,
		Env:           env2,
		Labels:        labels,
	})
}
