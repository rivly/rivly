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
	"github.com/rivly/rivly/internal/events"
)

type dockerService interface {
	Info(ctx context.Context, id int64, host string) (docker.SystemInfo, error)
	Containers(ctx context.Context, id int64, host string) ([]docker.Container, error)
	Images(ctx context.Context, id int64, host string) ([]docker.Image, error)
	ImageAction(ctx context.Context, id int64, host, imageID, action string) error
	Volumes(ctx context.Context, id int64, host string) ([]docker.Volume, error)
	VolumeAction(ctx context.Context, id int64, host, volumeName, action string) error
	Networks(ctx context.Context, id int64, host string) ([]docker.Network, error)
	NetworkAction(ctx context.Context, id int64, host, networkID, action string) error
	Stacks(ctx context.Context, id int64, host string) ([]docker.Stack, error)
	StackAction(ctx context.Context, id int64, host, project, action string) error
	ContainerLogs(ctx context.Context, id int64, host, containerID string, tail int, follow bool) (<-chan docker.LogLine, error)
	ContainerExec(ctx context.Context, id int64, host, containerID string) (*docker.ExecSession, error)
	ContainerAction(ctx context.Context, id int64, host, containerID, action string) error
	WatchEvents(ctx context.Context, id int64, host string) (<-chan struct{}, <-chan error)
}

type Server struct {
	logger       *slog.Logger
	queries      *db.Queries
	sessions     *scs.SessionManager
	local        *auth.Local
	docker       dockerService
	events       *events.Hub
	cfg          config.Config
	setupMu      sync.Mutex
	envStateMu   sync.Mutex
	lastEnvState map[int64]string
}

func New(
	logger *slog.Logger,
	queries *db.Queries,
	sessions *scs.SessionManager,
	local *auth.Local,
	docker dockerService,
	eventsHub *events.Hub,
	cfg config.Config,
) *Server {
	return &Server{
		logger:       logger,
		queries:      queries,
		sessions:     sessions,
		local:        local,
		docker:       docker,
		events:       eventsHub,
		cfg:          cfg,
		lastEnvState: make(map[int64]string),
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

	authLimit := httprate.LimitBy(10, time.Minute, func(r *http.Request) (string, error) {
		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
	})

	r.Get("/api/health", s.handleHealth)
	r.Route("/api/v1", func(r chi.Router) {
		r.With(s.requireEventAuth).Get("/events", s.handleEvents)
		r.With(s.requireEventAuth).Get("/environments/{id}/containers/{containerID}/logs", s.handleContainerLogs)
		r.With(s.requireEventAuth).Get("/environments/{id}/containers/{containerID}/exec", s.handleContainerExec)

		r.Group(func(r chi.Router) {
			r.Use(s.sessions.LoadAndSave)

			r.Get("/setup", s.handleSetupStatus)
			r.With(authLimit).Post("/setup", s.handleSetup)
			r.With(authLimit).Post("/login", s.handleLogin)
			r.Post("/logout", s.handleLogout)
			r.With(s.requireAuth).Get("/me", s.handleMe)

			r.Route("/environments", func(r chi.Router) {
				r.Use(s.requireAuth)
				r.Get("/", s.handleListEnvironments)
				r.Get("/{id}", s.handleGetEnvironment)
				r.Get("/{id}/stacks", s.handleListStacks)
				r.Post("/{id}/stacks/actions", s.handleStackActions)
				r.Get("/{id}/containers", s.handleListContainers)
				r.Post("/{id}/containers/actions", s.handleContainerActions)
				r.Get("/{id}/images", s.handleListImages)
				r.Post("/{id}/images/actions", s.handleImageActions)
				r.Get("/{id}/volumes", s.handleListVolumes)
				r.Post("/{id}/volumes/actions", s.handleVolumeActions)
				r.Get("/{id}/networks", s.handleListNetworks)
				r.Post("/{id}/networks/actions", s.handleNetworkActions)
			})
		})
	})

	return r
}
