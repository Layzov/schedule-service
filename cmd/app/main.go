package main

import (
	"avito-test-assignment-backend/internal/config"
	"avito-test-assignment-backend/internal/http-server/handlers/pr/create"
	"avito-test-assignment-backend/internal/http-server/handlers/pr/merge"
	"avito-test-assignment-backend/internal/http-server/handlers/pr/reassign"
	"avito-test-assignment-backend/internal/http-server/handlers/teams/add"
	"avito-test-assignment-backend/internal/http-server/handlers/teams/get"
	"avito-test-assignment-backend/internal/http-server/handlers/users/reviews"
	"avito-test-assignment-backend/internal/http-server/handlers/users/set"
	svc "avito-test-assignment-backend/internal/service"
	"avito-test-assignment-backend/internal/storage/postgres"
	slogpretty "avito-test-assignment-backend/pkg/handlers/slogPretty"
	"avito-test-assignment-backend/pkg/middleware/mwLogger"
	"avito-test-assignment-backend/pkg/sl"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func CORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}


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

	service := svc.NewService(storage)

	// _, _ = storage, service

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(CORS)

	router.Post("/team/add", add.New(log, service))
	router.Get("/team/get", get.New(log, service))
	router.Post("/users/setIsActive", set.New(log, service))
	router.Get("/users/getReview", reviews.New(log, service))
	router.Post("/pullRequest/create",  create.New(log, service))
	router.Post("/pullRequest/merge", merge.New(log, service))
	router.Post("/pullRequest/reassign", reassign.New(log, service))


	serv := &http.Server{
		Addr: cfg.Address,
		Handler: router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	serverErrCh := make(chan error, 1)

	go func() {
		log.Info("Starting HTTP server", slog.String("addr", cfg.Address))
		if err := serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		} else {
			serverErrCh <- nil
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("Received shutdown signal", slog.String("signal", sig.String()))
	case err := <-serverErrCh:
		if err != nil {
			log.Error("HTTP server stopped unexpectedly", sl.Err(err))
		} else {
			log.Info("HTTP server stopped gracefully")
		}
	}

	shutdownTimeout := cfg.HTTPServer.ShutdownTimeout
	
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Info("Shutting down HTTP server", slog.String("timeout", shutdownTimeout.String()))

	if err := serv.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", sl.Err(err))
	} else {
		log.Info("Server shutdown complete")
	}

	if storage != nil {
		if err := storage.Close(); err != nil {
			log.Error("Failed to close storage", sl.Err(err))
		} else {
			log.Info("Storage closed")
		}
	} else {
		log.Debug("Storage is nil, nothing to close")
	}

	log.Info("Shutdown finished, server stopped")

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

