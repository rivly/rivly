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
	"github.com/rivly/rivly/internal/environment"
	"github.com/rivly/rivly/internal/events"
	"github.com/rivly/rivly/internal/gitcred"
	"github.com/rivly/rivly/internal/registry"
	"github.com/rivly/rivly/internal/stack"
	"github.com/rivly/rivly/web"
)

type dockerService interface {
	Info(ctx context.Context, id int64, host string) (docker.SystemInfo, error)
	Containers(ctx context.Context, id int64, host string) ([]docker.Container, error)
	ContainerDetail(ctx context.Context, id int64, host, containerID string) (docker.ContainerDetail, error)
	ContainerCreate(ctx context.Context, id int64, host string, in docker.ContainerCreateInput) (string, error)
	ContainerStats(ctx context.Context, id int64, host, containerID string) (<-chan docker.Stats, error)
	Images(ctx context.Context, id int64, host string) ([]docker.Image, error)
	ImageAction(ctx context.Context, id int64, host, imageID, action string) error
	ImageDetail(ctx context.Context, id int64, host, imageID string) (docker.ImageDetail, error)
	ImagePull(ctx context.Context, id int64, host, ref string) (<-chan docker.PullProgress, error)
	ImagesPrune(ctx context.Context, id int64, host string, all bool) (docker.PruneResult, error)
	Volumes(ctx context.Context, id int64, host string) ([]docker.Volume, error)
	VolumeAction(ctx context.Context, id int64, host, volumeName, action string) error
	VolumeCreate(ctx context.Context, id int64, host string, in docker.VolumeCreateInput) (docker.Volume, error)
	VolumeDetail(ctx context.Context, id int64, host, name string) (docker.VolumeDetail, error)
	Networks(ctx context.Context, id int64, host string) ([]docker.Network, error)
	NetworkAction(ctx context.Context, id int64, host, networkID, action string) error
	NetworkCreate(ctx context.Context, id int64, host string, in docker.NetworkCreateInput) (docker.CreatedNetwork, error)
	NetworkDetail(ctx context.Context, id int64, host, networkID string) (docker.NetworkDetail, error)
	Stacks(ctx context.Context, id int64, host string) ([]docker.Stack, error)
	StackAction(ctx context.Context, id int64, host, project, action string) error
	ContainerLogs(ctx context.Context, id int64, host, containerID string, tail int, follow bool) (<-chan docker.LogLine, error)
	ContainerExec(ctx context.Context, id int64, host, containerID string) (*docker.ExecSession, error)
	ContainerAction(ctx context.Context, id int64, host, containerID, action string) error
	WatchEvents(ctx context.Context, id int64, host string) (<-chan struct{}, <-chan error)
	RegistryLogin(ctx context.Context, id int64, host, server, username, password string) error
}

type composeRunner interface {
	Deploy(ctx context.Context, dockerHost string, envID int64, project, content, env string) (string, error)
	Remove(ctx context.Context, dockerHost string, envID int64, project, content, env string) (string, error)
	Discard(ctx context.Context, dockerHost string, envID int64, project string)
	RepoDir(envID int64, project string) string
	DeployRepo(ctx context.Context, dockerHost string, envID int64, project, file, env string) (string, error)
	RemoveRepo(ctx context.Context, dockerHost string, envID int64, project, file, env string) (string, error)
	DiscardRepo(ctx context.Context, dockerHost string, envID int64, project, file string)
}

type Server struct {
	logger         *slog.Logger
	queries        *db.Queries
	sessions       *scs.SessionManager
	local          *auth.Local
	docker         dockerService
	compose        composeRunner
	events         *events.Hub
	registries     *registry.Store
	gitcreds       *gitcred.Store
	cfg            config.Config
	setupMu        sync.Mutex
	stacks         *stack.Service
	environments   *environment.Service
	streamsClosing context.Context
	closeStreams   context.CancelFunc
	running        sync.WaitGroup
}

func (s *Server) spawn(fn func()) {
	s.running.Add(1)
	go func() {
		defer s.running.Done()
		fn()
	}()
}

func (s *Server) Background(ctx context.Context, loop func(context.Context)) {
	s.spawn(func() { loop(ctx) })
}

func (s *Server) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.running.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func New(
	logger *slog.Logger,
	queries *db.Queries,
	sessions *scs.SessionManager,
	local *auth.Local,
	docker dockerService,
	compose composeRunner,
	eventsHub *events.Hub,
	registries *registry.Store,
	gitcreds *gitcred.Store,
	cfg config.Config,
) *Server {
	streamsClosing, closeStreams := context.WithCancel(context.Background())
	s := &Server{
		logger:         logger,
		queries:        queries,
		sessions:       sessions,
		local:          local,
		docker:         docker,
		compose:        compose,
		events:         eventsHub,
		registries:     registries,
		gitcreds:       gitcreds,
		cfg:            cfg,
		streamsClosing: streamsClosing,
		closeStreams:   closeStreams,
	}
	s.environments = environment.New(
		logger, queries, docker, cfg.PollInterval,
		s.emitEnvironment,
		func(ctx context.Context, envID int64) string { return s.stacks.Signature(ctx, envID) },
		s.spawn,
	)
	s.stacks = stack.New(logger, queries, docker, compose, gitcreds, s.environments.Publish, s.spawn)
	return s
}

func (s *Server) RunStackSync(ctx context.Context) {
	s.stacks.RunSync(ctx)
}

func (s *Server) RunPoller(ctx context.Context) {
	s.environments.RunPoller(ctx)
}

func (s *Server) RunWatchers(ctx context.Context) {
	s.environments.RunWatchers(ctx)
}

func (s *Server) CloseStreams() {
	s.closeStreams()
}

func (s *Server) streamContext(r *http.Request) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(s.streamsClosing, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(s.requestLogger)
	r.Use(s.recoverer)
	r.Use(securityHeaders)

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

	if assets, ok := web.Dashboard(); ok {
		r.NotFound(s.dashboardHandler(assets))
	} else {
		s.logger.Warn("dashboard is not embedded, serving the API only",
			"hint", "run bun run build in web/ before building the binary")
	}

	r.Get("/api/health", s.handleHealth)
	r.Route("/api/v1", func(r chi.Router) {
		r.With(s.requireEventAuth).Get("/events", s.handleEvents)

		stream := r.With(s.requireEventAuth, s.withEnvironment)
		stream.Get("/environments/{id}/containers/{containerID}/logs", s.handleContainerLogs)
		stream.Get("/environments/{id}/containers/{containerID}/stats", s.handleContainerStats)
		stream.Get("/environments/{id}/images/pull", s.handleImagePull)
		stream.Get("/environments/{id}/containers/{containerID}/exec", s.handleContainerExec)

		r.Group(func(r chi.Router) {
			r.Use(s.sessions.LoadAndSave)

			r.With(s.requireAuth).Get("/version", s.handleVersion)
			r.Get("/setup", s.handleSetupStatus)
			r.With(authLimit).Post("/setup", s.handleSetup)
			r.With(authLimit).Post("/login", s.handleLogin)
			r.Post("/logout", s.handleLogout)
			r.With(s.requireAuth).Get("/me", s.handleMe)
			r.With(s.requireAuth).Put("/me", s.handleUpdateProfile)
			r.With(s.requireAuth, authLimit).Post("/me/password", s.handleChangePassword)

			r.Route("/environments", func(r chi.Router) {
				r.Use(s.requireAuth)
				r.Get("/", s.handleListEnvironments)

				r.Route("/{id}", func(r chi.Router) {
					r.Use(s.withEnvironment)
					r.Get("/", s.handleGetEnvironment)
					r.Get("/stacks", s.handleListStacks)
					r.Post("/stacks", s.handleDeployStack)
					r.Get("/stacks/{name}", s.handleGetStack)
					r.Post("/stacks/actions", s.handleStackActions)
					r.Get("/containers", s.handleListContainers)
					r.Post("/containers", s.handleCreateContainer)
					r.Get("/containers/{containerID}", s.handleContainerDetail)
					r.Post("/containers/actions", s.handleContainerActions)
					r.Get("/images", s.handleListImages)
					r.Post("/images/actions", s.handleImageActions)
					r.Post("/images/prune", s.handleImagePrune)
					r.Get("/images/{imageID}", s.handleImageDetail)
					r.Get("/volumes", s.handleListVolumes)
					r.Post("/volumes", s.handleCreateVolume)
					r.Post("/volumes/actions", s.handleVolumeActions)
					r.Get("/volumes/{name}", s.handleVolumeDetail)
					r.Get("/networks", s.handleListNetworks)
					r.Post("/networks", s.handleCreateNetwork)
					r.Post("/networks/actions", s.handleNetworkActions)
					r.Get("/networks/{networkID}", s.handleNetworkDetail)
				})
			})

			r.Route("/registries", func(r chi.Router) {
				r.Use(s.requireAuth)
				r.Get("/", s.handleListRegistries)
				r.Post("/", s.handleCreateRegistry)
				r.Post("/test", s.handleTestRegistry)
				r.Put("/{id}", s.handleUpdateRegistry)
				r.Delete("/{id}", s.handleDeleteRegistry)
			})

			r.Route("/git-credentials", func(r chi.Router) {
				r.Use(s.requireAuth)
				r.Get("/", s.handleListGitCredentials)
				r.Post("/", s.handleCreateGitCredential)
				r.Put("/{id}", s.handleUpdateGitCredential)
				r.Delete("/{id}", s.handleDeleteGitCredential)
			})
		})
	})

	return r
}
