package http

import (
	"context"
	"errors"
	"expire-share/internal/config"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

type App struct {
	Router chi.Router
	server *http.Server
	logger *slog.Logger
	port   int
}

func New(logger *slog.Logger, cfg config.HttpServer) *App {
	router := chi.NewRouter()

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &App{
		Router: router,
		server: &server,
		logger: logger,
		port:   cfg.Port,
	}
}

func (a *App) MustRun() {
	const fn = "app.http.App.MustConnect"
	if err := a.Run(); err != nil {
		a.logger.Error("failed to run http server", sl.Error(err), slog.String("fn", fn))
		os.Exit(1)
	}
}

func (a *App) Run() error {
	const fn = "app.http.App.Connect"
	log := a.logger.With(slog.String("fn", fn))

	log.Info("http server is running", slog.Int("port", a.port))

	if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("internal server error", sl.Error(err))
		return fmt.Errorf("internal server error: %w", err)
	}

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	const fn = "app.http.App.Shutdown"
	log := a.logger.With(slog.String("fn", fn))

	log.Info("http server is shutting down", slog.Int("port", a.port))

	if err := a.server.Shutdown(ctx); err != nil {
		log.Error("failed to shutdown http server", sl.Error(err))
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}

	return nil
}
