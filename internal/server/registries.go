package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/registry"
)

type registryResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Server    string `json:"server"`
	Username  string `json:"username"`
	CreatedBy string `json:"createdBy"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

func toRegistryResponse(r registry.Registry) registryResponse {
	return registryResponse{
		ID:        r.ID,
		Name:      r.Name,
		Server:    r.Server,
		Username:  r.Username,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func normalizeServer(server string) string {
	server = strings.TrimSpace(strings.ToLower(server))
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")
	return strings.TrimSuffix(server, "/")
}

func (s *Server) currentUserName(r *http.Request) string {
	if user, err := s.queries.GetUserByID(r.Context(), s.sessions.GetInt64(r.Context(), sessionUserID)); err == nil {
		return user.DisplayName
	}
	return ""
}

func (s *Server) localEnvironment(ctx context.Context) (db.Environment, error) {
	envs, err := s.queries.ListEnvironments(ctx)
	if err != nil {
		return db.Environment{}, err
	}
	for _, e := range envs {
		if e.Kind == "local" {
			return e, nil
		}
	}
	if len(envs) > 0 {
		return envs[0], nil
	}
	return db.Environment{}, sql.ErrNoRows
}

func (s *Server) handleListRegistries(w http.ResponseWriter, r *http.Request) {
	regs, err := s.registries.List(r.Context())
	if err != nil {
		s.serverError(w, r, "could not load registries", err)
		return
	}
	out := make([]registryResponse, 0, len(regs))
	for _, reg := range regs {
		out = append(out, toRegistryResponse(reg))
	}
	s.writeJSON(w, http.StatusOK, out)
}

type registryRequest struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleCreateRegistry(w http.ResponseWriter, r *http.Request) {
	var req registryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Server = normalizeServer(req.Server)
	req.Username = strings.TrimSpace(req.Username)
	if req.Name == "" || req.Server == "" || req.Username == "" || req.Password == "" {
		s.writeError(w, http.StatusBadRequest, "name, registry url, username and password are required")
		return
	}

	reg, err := s.registries.Create(r.Context(), req.Name, req.Server, req.Username, req.Password, s.currentUserName(r))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.writeError(w, http.StatusConflict, "a registry for this server already exists")
			return
		}
		s.serverError(w, r, "could not save registry", err)
		return
	}
	s.writeJSON(w, http.StatusCreated, toRegistryResponse(reg))
}

func (s *Server) handleUpdateRegistry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid registry id")
		return
	}
	var req registryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Server = normalizeServer(req.Server)
	req.Username = strings.TrimSpace(req.Username)
	if req.Name == "" || req.Server == "" || req.Username == "" {
		s.writeError(w, http.StatusBadRequest, "name, registry url and username are required")
		return
	}

	reg, err := s.registries.Update(r.Context(), id, req.Name, req.Server, req.Username, req.Password)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "registry not found")
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.writeError(w, http.StatusConflict, "a registry for this server already exists")
			return
		}
		s.serverError(w, r, "could not update registry", err)
		return
	}
	s.writeJSON(w, http.StatusOK, toRegistryResponse(reg))
}

func (s *Server) handleDeleteRegistry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid registry id")
		return
	}
	if err := s.registries.Delete(r.Context(), id); err != nil {
		s.serverError(w, r, "could not delete registry", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type testRegistryRequest struct {
	ID       int64  `json:"id"`
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleTestRegistry(w http.ResponseWriter, r *http.Request) {
	var req testRegistryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	server := normalizeServer(req.Server)
	username := strings.TrimSpace(req.Username)
	password := req.Password

	if req.ID != 0 && password == "" {
		storedServer, storedUser, storedPass, err := s.registries.Credentials(r.Context(), req.ID)
		if err != nil {
			s.writeError(w, http.StatusNotFound, "registry not found")
			return
		}
		password = storedPass
		if server == "" {
			server = storedServer
		}
		if username == "" {
			username = storedUser
		}
	}

	if server == "" || username == "" || password == "" {
		s.writeError(w, http.StatusBadRequest, "server, username and password are required")
		return
	}

	env, err := s.localEnvironment(r.Context())
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "no local environment to test against")
		return
	}
	if err := s.docker.RegistryLogin(r.Context(), env.ID, env.Url, server, username, password); err != nil {
		s.logger.Warn("registry login failed", "server", server, "err", err)
		s.writeError(w, http.StatusBadGateway, "could not authenticate with the registry")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
