package server

import (
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rivly/rivly/internal/auth"
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

type setupInput struct {
	credentialsInput
	Token string `json:"token"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.queries.CountUsers(r.Context())
	if err != nil {
		s.serverError(w, r, "could not read setup status", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"needsSetup": count == 0})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	count, err := s.queries.CountUsers(r.Context())
	if err != nil {
		s.serverError(w, r, "could not read setup status", err)
		return
	}
	if count > 0 {
		s.writeError(w, http.StatusConflict, "setup has already been completed")
		return
	}

	var in setupInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.badRequest(w, err)
		return
	}
	if !auth.SetupTokenMatches(s.cfg.SetupToken, in.Token) {
		s.logger.Warn("setup rejected: invalid token", "ip", middleware.GetClientIP(r.Context()))
		s.writeError(w, http.StatusForbidden, "invalid setup token")
		return
	}
	email, msg, ok := validateSetup(in.credentialsInput)
	if !ok {
		s.writeError(w, http.StatusBadRequest, msg)
		return
	}

	user, err := s.local.Register(r.Context(), email, in.Password, strings.TrimSpace(in.DisplayName), "admin")
	if err != nil {
		s.serverError(w, r, "could not create the account", err)
		return
	}

	if err := s.startSession(r, user.ID); err != nil {
		s.serverError(w, r, "could not start a session", err)
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
		s.serverError(w, r, "could not start a session", err)
		return
	}
	s.writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.sessions.Destroy(r.Context()); err != nil {
		s.serverError(w, r, "could not log out", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := s.sessions.GetInt64(r.Context(), sessionUserID)
	user, err := s.queries.GetUserByID(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load the account", err)
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

func validateSetup(in credentialsInput) (email string, message string, ok bool) {
	addr, err := mail.ParseAddress(in.Email)
	if err != nil {
		return "", "a valid email address is required", false
	}
	if msg, ok := validateDisplayName(strings.TrimSpace(in.DisplayName)); !ok {
		return "", msg, false
	}
	if msg, ok := validatePassword(in.Password); !ok {
		return "", msg, false
	}
	return strings.ToLower(addr.Address), "", true
}

func validateDisplayName(name string) (message string, ok bool) {
	if len(name) > maxDisplayName {
		return "display name is too long", false
	}
	return "", true
}

func validatePassword(password string) (message string, ok bool) {
	if len(password) < 8 {
		return "password must be at least 8 characters", false
	}
	if len(password) > 128 {
		return "password must be at most 128 characters", false
	}
	return "", true
}
