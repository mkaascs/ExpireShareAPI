package log

import (
	"expire-share/internal/config"
	"fmt"
	"log"
	"log/slog"
	"os"
)

func MustLoad(env string) *slog.Logger {
	lg, err := Load(env)
	if err != nil {
		log.Fatal(err)
	}

	return lg
}

func Load(environment string) (*slog.Logger, error) {
	var lg *slog.Logger

	switch environment {
	case config.EnvLocal:
		lg = slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug}))
	case config.EnvDev:
		lg = slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}))
	case config.EnvProd:
		lg = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		return nil, fmt.Errorf("unknown environment: %s", environment)
	}

	return lg, nil
}
