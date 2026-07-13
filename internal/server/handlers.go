package server

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/rivly/rivly/internal/database/db"
)

type userResponse struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

func toUserResponse(u db.User) userResponse {
	return userResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
}

type credentialsInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.queries.CountUsers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not read setup status")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"needsSetup": count == 0})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, err := s.queries.CountUsers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not read setup status")
		return
	}
	if count > 0 {
		s.writeError(w, http.StatusConflict, "setup has already been completed")
		return
	}

	var in credentialsInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg, ok := validateCredentials(in); !ok {
		s.writeError(w, http.StatusBadRequest, msg)
		return
	}

	user, err := s.local.Register(r.Context(), in.Email, in.Password, strings.TrimSpace(in.DisplayName), "admin")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not create the account")
		return
	}

	if err := s.startSession(r, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not start a session")
		return
	}
	s.writeJSON(w, http.StatusCreated, toUserResponse(user))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in credentialsInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.local.Authenticate(r.Context(), in.Email, in.Password)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := s.startSession(r, user.ID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not start a session")
		return
	}
	s.writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.sessions.Destroy(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not log out")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := s.sessions.GetInt64(r.Context(), sessionUserID)
	user, err := s.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	s.writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (s *Server) startSession(r *http.Request, userID int64) error {
	if err := s.sessions.RenewToken(r.Context()); err != nil {
		return err
	}
	s.sessions.Put(r.Context(), sessionUserID, userID)
	return nil
}

func validateCredentials(in credentialsInput) (string, bool) {
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return "a valid email address is required", false
	}
	if len(in.Password) < 8 {
		return "password must be at least 8 characters", false
	}
	return "", true
}
