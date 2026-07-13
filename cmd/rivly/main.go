package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rivly/rivly/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbPath := os.Getenv("RIVLY_DATABASE")
	if dbPath == "" {
		dbPath = "rivly.db"
	}
	db, err := database.Open(dbPath)
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(db); err != nil {
		logger.Error("run migrations", "err", err)
		os.Exit(1)
	}
	logger.Info("database ready", "path", dbPath)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	logger.Info("Rivly listening", "addr", ":8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
