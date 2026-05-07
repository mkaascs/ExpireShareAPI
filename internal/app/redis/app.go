package redis

import (
	"context"
	"expire-share/internal/config"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

type App struct {
	Client *redis.Client
	logger *slog.Logger
}

func New(logger *slog.Logger, cfg config.Redis) *App {
	return &App{
		Client: redis.NewClient(&redis.Options{
			Addr:        cfg.Addr,
			Password:    cfg.Password,
			DB:          cfg.DB,
			ReadTimeout: cfg.Timeout,
			DialTimeout: cfg.DialTimeout,
			MaxRetries:  cfg.MaxRetries,
		}),

		logger: logger,
	}
}

func (a *App) MustConnect() {
	if err := a.Connect(); err != nil {
		os.Exit(1)
	}
}

func (a *App) Connect() error {
	const fn = "app.redis.App.Connect"
	log := a.logger.With(slog.String("fn", fn), slog.String("driver", "redis"))

	if err := a.Client.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to ping redis db", sl.Error(err))
		return fmt.Errorf("%s: failed to ping redis db: %w", fn, err)
	}

	log.Info("successfully connected to redis db")
	return nil
}

func (a *App) Close() error {
	const fn = "app.redis.App.Close"
	log := a.logger.With(slog.String("fn", fn), slog.String("driver", "redis"))

	if err := a.Client.Close(); err != nil {
		log.Error("failed to close redis db", sl.Error(err))
		return fmt.Errorf("%s: failed to close redis db: %w", fn, err)
	}

	log.Info("redis db successfully closed")
	return nil
}
