package main

import (
	"avito-test-assignment-backend/internal/config"
	"avito-test-assignment-backend/internal/http-server/handlers/pr/create"
	"avito-test-assignment-backend/internal/http-server/handlers/teams/add"
	"avito-test-assignment-backend/internal/http-server/handlers/teams/get"
	"avito-test-assignment-backend/internal/http-server/handlers/users/set"
	"avito-test-assignment-backend/internal/service"
	"avito-test-assignment-backend/internal/storage/postgres"
	slogpretty "avito-test-assignment-backend/pkg/handlers/slogPretty"
	"avito-test-assignment-backend/pkg/middleware/mwLogger"
	"avito-test-assignment-backend/pkg/sl"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {

	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("Starting API", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	storage, err := postgres.New(cfg.StoragePath)
	if err != nil {
		log.Error("Failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	service := service.NewService(storage)

	_, _ = storage, service

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/team/add", add.New(log, service))
	router.Get("/team/get/{team_name}", get.New(log, service))
	router.Get("/users/setIsActive", set.New(log, service))
	router.Post("/pullRequest/create",  create.New(log, service))

	log.Info("Starting HTTP server", slog.String("addr", cfg.Address))

	serv := &http.Server{
		Addr: cfg.Address,
		Handler: router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	if errServ := serv.ListenAndServe(); errServ != nil {
		log.Error("Failed to start HTTP server")
	}

	log.Error("HTTP server stopped")
	
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
