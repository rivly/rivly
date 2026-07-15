package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rivly/rivly/internal/auth"
	"github.com/rivly/rivly/internal/compose"
	"github.com/rivly/rivly/internal/config"
	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
	"github.com/rivly/rivly/internal/events"
	"github.com/rivly/rivly/internal/gitcred"
	"github.com/rivly/rivly/internal/registry"
	"github.com/rivly/rivly/internal/secret"
	"github.com/rivly/rivly/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	sqlDB, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	if err := database.Migrate(sqlDB); err != nil {
		return err
	}
	logger.Info("database ready", "path", cfg.DatabasePath)

	queries := db.New(sqlDB)
	if err := seedLocalEnvironment(context.Background(), queries, cfg.DockerHost); err != nil {
		return err
	}

	cfg.SetupToken, err = resolveSetupToken(context.Background(), queries, cfg.SetupToken, logger)
	if err != nil {
		return err
	}

	cipher, err := secret.LoadOrCreate(cfg.DataDir)
	if err != nil {
		return err
	}
	registries := registry.NewStore(queries, cipher)
	gitCredentials := gitcred.NewStore(queries, cipher)

	dockerManager := docker.NewManager()
	dockerManager.SetAuthResolver(registries.AuthFor)
	defer dockerManager.Close()

	composeRunner := compose.NewRunner(cfg.ComposeBin, cfg.DataDir)
	eventsHub := events.NewHub()
	sessions := auth.NewSessionManager(sqlDB)
	local := auth.NewLocal(queries, sqlDB)
	srv := server.New(logger, queries, sessions, local, dockerManager, composeRunner, eventsHub, registries, gitCredentials, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go srv.RunPoller(ctx)
	go srv.RunWatchers(ctx)
	go srv.RunGitPoller(ctx)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	httpServer.RegisterOnShutdown(srv.CloseStreams)

	go func() {
		logger.Info("Rivly listening", "addr", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

func resolveSetupToken(ctx context.Context, queries *db.Queries, pinned string, logger *slog.Logger) (string, error) {
	count, err := queries.CountUsers(ctx)
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}
	if pinned != "" {
		logger.Info("setup is open, using the token from RIVLY_SETUP_TOKEN")
		return pinned, nil
	}
	token, err := auth.NewSetupToken()
	if err != nil {
		return "", err
	}
	logger.Info("setup is open, use this token to create the first account", "setup_token", token)
	return token, nil
}

func seedLocalEnvironment(ctx context.Context, queries *db.Queries, host string) error {
	count, err := queries.CountEnvironments(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = queries.CreateEnvironment(ctx, db.CreateEnvironmentParams{
		Name: "local",
		Kind: "local",
		Url:  host,
	})
	return err
}
