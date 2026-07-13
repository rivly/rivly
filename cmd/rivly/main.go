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
	"github.com/rivly/rivly/internal/config"
	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
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
	cfg := config.Load()

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
	sessions := auth.NewSessionManager(sqlDB)
	local := auth.NewLocal(queries)
	srv := server.New(logger, queries, sessions, local, cfg)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
