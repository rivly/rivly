package server

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/rivly/rivly/internal/auth"
	"github.com/rivly/rivly/internal/config"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

type dockerService interface {
	Ping(ctx context.Context, id int64, host string) docker.Status
	Info(ctx context.Context, id int64, host string) (docker.SystemInfo, error)
}

type Server struct {
	logger   *slog.Logger
	queries  *db.Queries
	sessions *scs.SessionManager
	local    *auth.Local
	docker   dockerService
	cfg      config.Config
	setupMu  sync.Mutex
}

func New(
	logger *slog.Logger,
	queries *db.Queries,
	sessions *scs.SessionManager,
	local *auth.Local,
	docker dockerService,
	cfg config.Config,
) *Server {
	return &Server{
		logger:   logger,
		queries:  queries,
		sessions: sessions,
		local:    local,
		docker:   docker,
		cfg:      cfg,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(s.requestLogger)
	r.Use(s.recoverer)

	crossOrigin := http.NewCrossOriginProtection()
	for _, origin := range s.cfg.TrustedOrigins {
		if err := crossOrigin.AddTrustedOrigin(origin); err != nil {
			s.logger.Error("invalid trusted origin", "origin", origin, "err", err)
		}
	}
	r.Use(crossOrigin.Handler)
	r.Use(secureCookies)
	r.Use(s.sessions.LoadAndSave)

	authLimit := httprate.LimitBy(10, time.Minute, func(r *http.Request) (string, error) {
		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
	})

	r.Get("/api/health", s.handleHealth)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/setup", s.handleSetupStatus)
		r.With(authLimit).Post("/setup", s.handleSetup)
		r.With(authLimit).Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)
		r.With(s.requireAuth).Get("/me", s.handleMe)

		r.Route("/environments", func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/", s.handleListEnvironments)
			r.Get("/{id}", s.handleGetEnvironment)
		})
	})

	return r
}
