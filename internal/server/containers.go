package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/docker"
	"github.com/rivly/rivly/internal/stack"
)

type containerAPI interface {
	Containers(ctx context.Context) ([]docker.Container, error)
	ContainerDetail(ctx context.Context, containerID string) (docker.ContainerDetail, error)
	ContainerCreate(ctx context.Context, in docker.ContainerCreateInput) (string, error)
	ContainerStats(ctx context.Context, containerID string) (<-chan docker.Stats, error)
	ContainerLogs(ctx context.Context, containerID string, tail int, follow bool) (<-chan docker.LogLine, error)
	ContainerExec(ctx context.Context, containerID string) (*docker.ExecSession, error)
	ContainerAction(ctx context.Context, containerID, action string) error
}

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
	containers, err := dockerFrom(r).Containers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	s.writeJSON(w, http.StatusOK, mapSlice(containers, toContainerResponse))
}

func toPortResponse(p docker.Port) portResponse {
	return portResponse{
		PrivatePort: p.PrivatePort,
		PublicPort:  p.PublicPort,
		Type:        p.Type,
		IP:          p.IP,
	}
}

func toContainerResponse(c docker.Container) containerResponse {
	return containerResponse{
		ID:      c.ID,
		Name:    c.Name,
		Image:   c.Image,
		State:   c.State,
		Status:  c.Status,
		Stack:   c.Stack,
		Created: c.Created,
		IP:      c.IP,
		Ports:   mapSlice(c.Ports, toPortResponse),
	}
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
	Name          string         `json:"name"`
	Image         string         `json:"image"`
	Command       string         `json:"command"`
	Env           []stack.EnvVar `json:"env"`
	Ports         []portMapping  `json:"ports"`
	Mounts        []mountInput   `json:"mounts"`
	Network       string         `json:"network"`
	RestartPolicy string         `json:"restartPolicy"`
	Start         bool           `json:"start"`
}

var validRestartPolicies = map[string]bool{
	"no":             true,
	"always":         true,
	"unless-stopped": true,
	"on-failure":     true,
}

func (s *Server) handleCreateContainer(w http.ResponseWriter, r *http.Request) {
	var req runContainerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
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

	env := environmentFrom(r)

	containerID, err := dockerFrom(r).ContainerCreate(r.Context(), input)
	if err != nil {
		s.logger.Warn("container create failed", "image", req.Image, "err", err)
		if containerID != "" {
			s.publishEnvironment(r.Context(), env)
			s.writeError(w, http.StatusBadGateway, "container created but could not start")
			return
		}
		if errors.Is(err, docker.ErrImagePull) {
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
	containerID := chi.URLParam(r, "containerID")

	detail, err := dockerFrom(r).ContainerDetail(r.Context(), containerID)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not inspect container")
		return
	}

	labels := detail.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	variables := detail.Env
	if variables == nil {
		variables = []string{}
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
		Ports:         mapSlice(detail.Ports, toPortResponse),
		Networks: mapSlice(detail.Networks, func(n docker.NetworkAttachment) networkAttachmentResponse {
			return networkAttachmentResponse{Name: n.Name, IP: n.IP}
		}),
		Mounts: mapSlice(detail.Mounts, func(m docker.Mount) mountResponse {
			return mountResponse{Type: m.Type, Source: m.Source, Destination: m.Destination, Name: m.Name, RW: m.RW}
		}),
		Env:    variables,
		Labels: labels,
	})
}
