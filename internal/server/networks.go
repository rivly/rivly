package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/docker"
)

type networkResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	Scope   string `json:"scope"`
	Stack   string `json:"stack"`
	Created int64  `json:"created"`
	InUse   bool   `json:"inUse"`
}

var validNetworkActions = map[string]bool{"remove": true}

func (s *Server) handleListNetworks(w http.ResponseWriter, r *http.Request) {
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

	networks, err := s.docker.Networks(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	out := make([]networkResponse, 0, len(networks))
	for _, n := range networks {
		out = append(out, networkResponse{
			ID:      n.ID,
			Name:    n.Name,
			Driver:  n.Driver,
			Scope:   n.Scope,
			Stack:   n.Stack,
			Created: n.Created,
			InUse:   n.InUse,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

type createNetworkRequest struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Subnet string `json:"subnet"`
}

func (s *Server) handleCreateNetwork(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req createNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if !resourceNamePattern.MatchString(req.Name) {
		s.writeError(w, http.StatusBadRequest, "invalid network name")
		return
	}
	req.Subnet = strings.TrimSpace(req.Subnet)
	if req.Subnet != "" {
		if _, perr := netip.ParsePrefix(req.Subnet); perr != nil {
			s.writeError(w, http.StatusBadRequest, "invalid subnet (expected CIDR, e.g. 172.20.0.0/16)")
			return
		}
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

	created, err := s.docker.NetworkCreate(r.Context(), env.ID, env.Url, docker.NetworkCreateInput{
		Name:   req.Name,
		Driver: strings.TrimSpace(req.Driver),
		Subnet: req.Subnet,
	})
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not create network")
		return
	}
	s.publishEnvironment(r.Context(), env)
	s.writeJSON(w, http.StatusCreated, map[string]any{
		"id":      created.ID,
		"name":    req.Name,
		"warning": created.Warning,
	})
}

type networkSubnetResponse struct {
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
}

type networkContainerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IPv4 string `json:"ipv4"`
}

type networkDetailResponse struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Driver     string                     `json:"driver"`
	Scope      string                     `json:"scope"`
	Internal   bool                       `json:"internal"`
	Attachable bool                       `json:"attachable"`
	Created    int64                      `json:"created"`
	Subnets    []networkSubnetResponse    `json:"subnets"`
	Labels     map[string]string          `json:"labels"`
	Containers []networkContainerResponse `json:"containers"`
}

func (s *Server) handleNetworkDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	networkID := chi.URLParam(r, "networkID")

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	detail, err := s.docker.NetworkDetail(r.Context(), env.ID, env.Url, networkID)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "could not inspect network")
		return
	}

	subnets := make([]networkSubnetResponse, 0, len(detail.Subnets))
	for _, sn := range detail.Subnets {
		subnets = append(subnets, networkSubnetResponse{Subnet: sn.Subnet, Gateway: sn.Gateway})
	}
	containers := make([]networkContainerResponse, 0, len(detail.Containers))
	for _, c := range detail.Containers {
		containers = append(containers, networkContainerResponse{ID: c.ID, Name: c.Name, IPv4: c.IPv4})
	}
	s.writeJSON(w, http.StatusOK, networkDetailResponse{
		ID:         detail.ID,
		Name:       detail.Name,
		Driver:     detail.Driver,
		Scope:      detail.Scope,
		Internal:   detail.Internal,
		Attachable: detail.Attachable,
		Created:    detail.Created,
		Subnets:    subnets,
		Labels:     detail.Labels,
		Containers: containers,
	})
}

func (s *Server) handleNetworkActions(w http.ResponseWriter, r *http.Request) {
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
	if !validNetworkActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid network selection")
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
	for i, networkID := range req.IDs {
		wg.Add(1)
		go func(i int, networkID string) {
			defer wg.Done()
			if err := s.docker.NetworkAction(r.Context(), env.ID, env.Url, networkID, req.Action); err != nil {
				s.logger.Warn("network action failed",
					"action", req.Action, "network", networkID, "err", err)
				results[i] = actionResult{ID: networkID, OK: false, Error: "action failed"}
				return
			}
			results[i] = actionResult{ID: networkID, OK: true}
		}(i, networkID)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
