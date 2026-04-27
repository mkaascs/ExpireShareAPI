package auth

import (
	"context"
	"expire-share/internal/config"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

type App struct {
	GRPCConn *grpc.ClientConn
	logger   *slog.Logger
	config   config.AuthService
}

func New(logger *slog.Logger, config config.AuthService) *App {
	return &App{
		logger: logger,
		config: config,
	}
}

func (a *App) MustConnect() {
	if err := a.Connect(); err != nil {
		os.Exit(1)
	}
}

func (a *App) Connect() error {
	const fn = "app.auth.App.Connect"
	log := a.logger.
		With(slog.String("fn", fn)).
		With(slog.String("addr", a.config.Addr))

	conn, err := grpc.NewClient(
		a.config.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Error("failed to dial grpc connection", sl.Error(err))
		return fmt.Errorf("%s: failed to dial grpc connection: %w", fn, err)
	}

	conn.Connect()
	if !tryToConnect(context.Background(), conn) {
		log.Error("failed to connect to auth service")
		return fmt.Errorf("%s: failed to connect to auth service", fn)
	}

	log.Info("connected to grpc server successfully")

	a.GRPCConn = conn
	return nil
}

func (a *App) Close() error {
	const fn = "app.auth.App.Close"
	log := a.logger.With(slog.String("fn", fn))

	if err := a.GRPCConn.Close(); err != nil {
		log.Error("failed to close grpc connection", sl.Error(err))
		return fmt.Errorf("failed to close grpc connection: %w", err)
	}

	log.Info("grpc connection closed successfully", slog.String("addr", a.config.Addr))

	return nil
}

func tryToConnect(ctx context.Context, grpcConn *grpc.ClientConn) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	timer := time.NewTicker(time.Second / 2)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			info := grpcConn.GetState()
			if info == connectivity.Ready {
				return true
			}

			if info == connectivity.TransientFailure || info == connectivity.Shutdown {
				return false
			}
		}
	}
}
