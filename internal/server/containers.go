package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/docker"
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

type portMapping struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Proto         string `json:"proto"`
}

type mountInput struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

type runContainerRequest struct {
	Name          string        `json:"name"`
	Image         string        `json:"image"`
	Command       string        `json:"command"`
	Env           []envVar      `json:"env"`
	Ports         []portMapping `json:"ports"`
	Mounts        []mountInput  `json:"mounts"`
	Network       string        `json:"network"`
	RestartPolicy string        `json:"restartPolicy"`
	Start         bool          `json:"start"`
}

var validRestartPolicies = map[string]bool{
	"no":             true,
	"always":         true,
	"unless-stopped": true,
	"on-failure":     true,
}

func (s *Server) handleCreateContainer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req runContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		s.writeError(w, http.StatusBadRequest, "image is required")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name != "" && !resourceNamePattern.MatchString(req.Name) {
		s.writeError(w, http.StatusBadRequest, "invalid container name")
		return
	}
	if req.RestartPolicy == "" {
		req.RestartPolicy = "no"
	}
	if !validRestartPolicies[req.RestartPolicy] {
		s.writeError(w, http.StatusBadRequest, "invalid restart policy")
		return
	}

	input := docker.ContainerCreateInput{
		Name:          req.Name,
		Image:         req.Image,
		Command:       strings.Fields(req.Command),
		Network:       strings.TrimSpace(req.Network),
		RestartPolicy: req.RestartPolicy,
		Start:         req.Start,
	}
	for _, e := range req.Env {
		key := strings.TrimSpace(e.Key)
		if key == "" {
			continue
		}
		input.Env = append(input.Env, key+"="+e.Value)
	}
	for _, p := range req.Ports {
		containerPort := strings.TrimSpace(p.ContainerPort)
		if containerPort == "" {
			continue
		}
		input.Ports = append(input.Ports, docker.PortMapping{
			HostPort:      strings.TrimSpace(p.HostPort),
			ContainerPort: containerPort,
			Proto:         strings.TrimSpace(p.Proto),
		})
	}
	for _, m := range req.Mounts {
		source := strings.TrimSpace(m.Source)
		target := strings.TrimSpace(m.Target)
		if source == "" || target == "" {
			continue
		}
		input.Mounts = append(input.Mounts, docker.MountInput{
			Source:   source,
			Target:   target,
			ReadOnly: m.ReadOnly,
		})
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

	containerID, err := s.docker.ContainerCreate(r.Context(), env.ID, env.Url, input)
	if err != nil {
		s.logger.Warn("container create failed", "image", req.Image, "err", err)
		if containerID != "" {
			s.publishEnvironment(r.Context(), env)
			s.writeError(w, http.StatusBadGateway, "container created but could not start")
			return
		}
		if strings.Contains(err.Error(), "pull image") {
			s.writeError(w, http.StatusBadGateway, "could not pull the image")
			return
		}
		s.writeError(w, http.StatusBadGateway, "could not create container")
		return
	}
	s.publishEnvironment(r.Context(), env)
	s.writeJSON(w, http.StatusCreated, map[string]any{"id": containerID})
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
