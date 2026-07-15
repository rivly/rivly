package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/rivly/rivly/internal/auth"
	"github.com/rivly/rivly/internal/database/db"
)

const maxDisplayName = 100

type updateProfileInput struct {
	DisplayName string `json:"displayName"`
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var in updateProfileInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(in.DisplayName)
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "display name is required")
		return
	}
	if len(name) > maxDisplayName {
		s.writeError(w, http.StatusBadRequest, "display name is too long")
		return
	}

	userID := s.sessions.GetInt64(r.Context(), sessionUserID)
	user, err := s.queries.UpdateUserProfile(r.Context(), db.UpdateUserProfileParams{
		DisplayName: name,
		ID:          userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not update the account", err)
		return
	}
	s.writeJSON(w, http.StatusOK, toUserResponse(user))
}

type changePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var in changePasswordInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg, ok := validatePassword(in.NewPassword); !ok {
		s.writeError(w, http.StatusBadRequest, msg)
		return
	}

	userID := s.sessions.GetInt64(r.Context(), sessionUserID)
	cred, err := s.queries.GetPasswordCredential(r.Context(), userID)
	if err != nil || !cred.Secret.Valid {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	match, err := auth.VerifyPassword(in.CurrentPassword, cred.Secret.String)
	if err != nil || !match {
		s.writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		s.serverError(w, r, "could not update the password", err)
		return
	}
	if err := s.queries.UpdatePasswordCredential(r.Context(), db.UpdatePasswordCredentialParams{
		Secret: sql.NullString{String: hash, Valid: true},
		UserID: userID,
	}); err != nil {
		s.serverError(w, r, "could not update the password", err)
		return
	}

	s.destroyOtherSessions(r.Context(), userID)
	if err := s.sessions.RenewToken(r.Context()); err != nil {
		s.serverError(w, r, "could not refresh the session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) destroyOtherSessions(ctx context.Context, userID int64) {
	current := s.sessions.Token(ctx)
	err := s.sessions.Iterate(ctx, func(ctx context.Context) error {
		if s.sessions.Token(ctx) == current {
			return nil
		}
		if s.sessions.GetInt64(ctx, sessionUserID) != userID {
			return nil
		}
		if derr := s.sessions.Destroy(ctx); derr != nil {
			s.logger.Warn("could not destroy session", "err", derr)
		}
		return nil
	})
	if err != nil {
		s.logger.Error("could not iterate sessions", "err", err)
	}
}
