package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/gitcred"
)

type gitCredentialResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	CreatedBy string `json:"createdBy"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

func toGitCredentialResponse(c gitcred.Credential) gitCredentialResponse {
	return gitCredentialResponse{
		ID:        c.ID,
		Name:      c.Name,
		Username:  c.Username,
		CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *Server) handleListGitCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := s.gitcreds.List(r.Context())
	if err != nil {
		s.serverError(w, r, "could not load git credentials", err)
		return
	}
	out := make([]gitCredentialResponse, 0, len(creds))
	for _, c := range creds {
		out = append(out, toGitCredentialResponse(c))
	}
	s.writeJSON(w, http.StatusOK, out)
}

type gitCredentialRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (s *Server) handleCreateGitCredential(w http.ResponseWriter, r *http.Request) {
	var req gitCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	if req.Name == "" || req.Username == "" || req.Token == "" {
		s.writeError(w, http.StatusBadRequest, "name, username and token are required")
		return
	}

	cred, err := s.gitcreds.Create(r.Context(), req.Name, req.Username, req.Token, s.currentUserName(r))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.writeError(w, http.StatusConflict, "a credential with this name already exists")
			return
		}
		s.serverError(w, r, "could not save git credential", err)
		return
	}
	s.writeJSON(w, http.StatusCreated, toGitCredentialResponse(cred))
}

func (s *Server) handleUpdateGitCredential(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid credential id")
		return
	}
	var req gitCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	if req.Name == "" || req.Username == "" {
		s.writeError(w, http.StatusBadRequest, "name and username are required")
		return
	}

	cred, err := s.gitcreds.Update(r.Context(), id, req.Name, req.Username, req.Token)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "credential not found")
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.writeError(w, http.StatusConflict, "a credential with this name already exists")
			return
		}
		s.serverError(w, r, "could not update git credential", err)
		return
	}
	s.writeJSON(w, http.StatusOK, toGitCredentialResponse(cred))
}

func (s *Server) handleDeleteGitCredential(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid credential id")
		return
	}
	if err := s.gitcreds.Delete(r.Context(), id); err != nil {
		s.serverError(w, r, "could not delete git credential", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
