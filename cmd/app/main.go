package main

import (
	"rasp-service/internal/config"
	availCreate "rasp-service/internal/http-server/handlers/availability_templates/create"
	availGet "rasp-service/internal/http-server/handlers/availability_templates/get"
	availUpdate "rasp-service/internal/http-server/handlers/availability_templates/update"
	availDelete "rasp-service/internal/http-server/handlers/availability_templates/delete"
	timeBlockCreate "rasp-service/internal/http-server/handlers/time_blocks/create"
	timeBlockGet "rasp-service/internal/http-server/handlers/time_blocks/get"
	timeBlockUpdate "rasp-service/internal/http-server/handlers/time_blocks/update"
	timeBlockDelete "rasp-service/internal/http-server/handlers/time_blocks/delete"
	slotGet "rasp-service/internal/http-server/handlers/slots/get"
	slotGenerate "rasp-service/internal/http-server/handlers/slots/generate"
	bookingCreate "rasp-service/internal/http-server/handlers/bookings/create"
	bookingGet "rasp-service/internal/http-server/handlers/bookings/get"
	bookingCancel "rasp-service/internal/http-server/handlers/bookings/cancel"
	bookingReschedule "rasp-service/internal/http-server/handlers/bookings/reschedule"
	bookingConfirm "rasp-service/internal/http-server/handlers/bookings/confirm"
	bookingDelete "rasp-service/internal/http-server/handlers/bookings/delete"
	attendanceCreate "rasp-service/internal/http-server/handlers/attendance/create"
	attendanceGet "rasp-service/internal/http-server/handlers/attendance/get"
	svc "rasp-service/internal/service"
	"rasp-service/internal/storage/postgres"
	"rasp-service/internal/lock"
	slogpretty "rasp-service/pkg/handlers/slogPretty"
	"rasp-service/pkg/middleware/mwLogger"
	"rasp-service/pkg/sl"
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

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

	locker, err := lock.NewRedisLock(cfg.RedisAddr)
	if err != nil {
		log.Error("Failed to init redis lock", sl.Err(err))
		os.Exit(1)
	}

	service := svc.NewService(storage, locker)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(CORS)

	// Availability Templates
	router.Post("/availability_templates", availCreate.New(log, service))
	router.Get("/availability_templates/{id}", availGet.New(log, service))
	router.Put("/availability_templates/{id}", availUpdate.New(log, service))
	router.Delete("/availability_templates/{id}", availDelete.New(log, service))

	// Time Blocks
	router.Post("/time_blocks", timeBlockCreate.New(log, service))
	router.Get("/time_blocks/{id}", timeBlockGet.New(log, service))
	router.Put("/time_blocks/{id}", timeBlockUpdate.New(log, service))
	router.Delete("/time_blocks/{id}", timeBlockDelete.New(log, service))

	// Slots
	// router.Get("/slots", slotGet.New(log, service))
	router.Get("/slots/{id}", slotGet.New(log, service))
	// router.Get("/slots/batch", slotGet.New(log, service))
	router.Post("/slots/generate", slotGenerate.New(log, service))

	// Bookings
	router.Post("/bookings", bookingCreate.New(log, service))
	// router.Get("/bookings", bookingGet.New(log, service))
	router.Get("/bookings/{id}", bookingGet.New(log, service))
	router.Put("/bookings/{id}/cancel", bookingCancel.New(log, service))
	router.Post("/bookings/reschedule", bookingReschedule.New(log, service))
	router.Post("/bookings/{id}/confirm", bookingConfirm.New(log, service))
	router.Delete("/bookings/{id}", bookingDelete.New(log, service))

	// Attendance
	router.Post("/attendance", attendanceCreate.New(log, service))
	router.Get("/attendance", attendanceGet.New(log, service))
	router.Get("/attendance/{id}", attendanceGet.New(log, service))

	serv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
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

	if locker != nil {
		if err := locker.Close(); err != nil {
			log.Error("Failed to close locker", sl.Err(err))
		} else {
			log.Info("Locker closed")
		}
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
