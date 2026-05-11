package app

import (
	"context"
	_ "expire-share/docs"
	"expire-share/internal/app/auth"
	httpApp "expire-share/internal/app/http"
	"expire-share/internal/app/mysql"
	redisApp "expire-share/internal/app/redis"
	"expire-share/internal/config"
	userGet "expire-share/internal/delivery/handlers/admin/users/get"
	userGetAll "expire-share/internal/delivery/handlers/admin/users/getAll"
	"expire-share/internal/delivery/handlers/admin/users/roles/assign"
	"expire-share/internal/delivery/handlers/admin/users/roles/revoke"
	"expire-share/internal/delivery/handlers/api/auth/login"
	"expire-share/internal/delivery/handlers/api/auth/logout"
	"expire-share/internal/delivery/handlers/api/auth/refresh"
	"expire-share/internal/delivery/handlers/api/auth/register"
	"expire-share/internal/delivery/handlers/api/files/delete"
	"expire-share/internal/delivery/handlers/api/files/get"
	"expire-share/internal/delivery/handlers/api/files/getAll"
	"expire-share/internal/delivery/handlers/api/files/stat"
	"expire-share/internal/delivery/handlers/api/upload"
	"expire-share/internal/delivery/handlers/download"
	myMiddleware "expire-share/internal/delivery/middlewares"
	"expire-share/internal/infrastructure/grpc"
	repo "expire-share/internal/infrastructure/mysql"
	rateLimiter "expire-share/internal/infrastructure/redis"
	"expire-share/internal/infrastructure/storage/local"
	"expire-share/internal/services/files"
	"expire-share/internal/services/worker"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

type App struct {
	HTTP  *httpApp.App
	MySql *mysql.App
	Auth  *auth.App
	Redis *redisApp.App

	config config.Config
	logger *slog.Logger
}

func New(config config.Config, logger *slog.Logger) *App {
	httpServer := httpApp.New(logger, config.HttpServer)
	redis := redisApp.New(logger, config.Redis)
	authApp := auth.New(logger, config.AuthService)

	mysql.MustMigrate(logger, config.DbConnectionString)
	mysqlApp, _ := mysql.New(logger, config.DbConnectionString)

	return &App{
		HTTP:   httpServer,
		MySql:  mysqlApp,
		Auth:   authApp,
		Redis:  redis,
		config: config,
		logger: logger,
	}
}

func (a *App) MustMountMiddlewares() {
	a.HTTP.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   a.config.CORS.AllowedOrigins,
		AllowCredentials: a.config.CORS.AllowedCredentials,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
			"X-Resource-Password",
			"X-Request-ID",
		},

		ExposedHeaders: []string{"Link", "Content-Disposition", "Content-Length"},
		MaxAge:         a.config.CORS.MaxAge,
	}))

	a.HTTP.Router.Use(middleware.RequestID)
	a.HTTP.Router.Use(middleware.RealIP)
	a.HTTP.Router.Use(middleware.Recoverer)
	a.HTTP.Router.Use(middleware.URLFormat)
	a.HTTP.Router.Use(myMiddleware.NewLogger(a.logger))
}

func (a *App) MustMountHandlers() {
	fileRepo := repo.NewFileRepo(a.MySql.DB, a.logger)
	fileStorage := local.NewFileStorage(a.config.Storage, a.logger)
	authClient := grpc.NewAuthClient(a.Auth.GRPCConn)
	userClient := grpc.NewUserClient(a.Auth.GRPCConn)

	fileService := files.New(fileRepo, fileStorage, a.logger, a.config)

	if a.config.Env == config.EnvLocal {
		a.HTTP.Router.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "docs/swagger.json")
		})

		a.HTTP.Router.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	a.HTTP.Router.With(myMiddleware.NewRateLimiter(
		rateLimiter.NewRateLimiter(a.Redis.Client, a.config.RateLimiter.Files), a.logger)).
		Get("/download/{alias}", download.New(fileService, a.logger))

	a.HTTP.Router.Route("/api", func(r chi.Router) {
		r.Route("/", func(r chi.Router) {
			r.Use(myMiddleware.NewAuth(authClient, a.logger))
			r.With(myMiddleware.NewTimeoutLimiter(myMiddleware.TimeoutLimiterParams{
				ReadTimeout:  10 * time.Minute,
				WriteTimeout: time.Minute,
			}, a.logger)).
				Post("/upload", upload.New(fileService, a.logger, a.config))

			r.Route("/file", func(r chi.Router) {
				r.Get("/", getAll.New(fileService, a.logger))

				r.Get("/stat", stat.New(fileService, a.logger))

				r.Route("/{alias}", func(r chi.Router) {
					r.Get("/", get.New(fileService, a.logger))
					r.Delete("/", delete.New(fileService, a.logger))
				})
			})
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(myMiddleware.NewAdminAuth(
				rateLimiter.NewRateLimiter(a.Redis.Client, a.config.RateLimiter.Admin),
				a.logger))

			r.Route("/users", func(r chi.Router) {
				r.Get("/{id}", userGet.New(userClient, a.logger))
				r.Get("/", userGetAll.New(userClient, a.logger))

				r.Route("/{id}/roles", func(r chi.Router) {
					r.With(myMiddleware.NewBodyParser[assign.Request](a.config.Service, a.logger)).
						Post("/assign", assign.New(userClient, a.logger))

					r.With(myMiddleware.NewBodyParser[revoke.Request](a.config.Service, a.logger)).
						Post("/revoke", revoke.New(userClient, a.logger))
				})
			})
		})

		r.Route("/auth", func(r chi.Router) {
			r.With(myMiddleware.NewBodyParser[login.Request](a.config.Service, a.logger),
				myMiddleware.NewValidator[login.Request](a.logger)).
				Post("/login", login.New(authClient, a.logger))

			r.With(myMiddleware.NewRateLimiter(rateLimiter.NewRateLimiter(a.Redis.Client, a.config.RateLimiter.Register), a.logger)).
				With(myMiddleware.NewBodyParser[register.Request](a.config.Service, a.logger),
					myMiddleware.NewValidator[register.Request](a.logger)).
				Post("/register", register.New(authClient, a.logger))

			r.With(myMiddleware.NewBodyParser[refresh.Request](a.config.Service, a.logger),
				myMiddleware.NewValidator[refresh.Request](a.logger)).
				Post("/refresh", refresh.New(authClient, a.logger))

			r.With(myMiddleware.NewBodyParser[logout.Request](a.config.Service, a.logger),
				myMiddleware.NewValidator[logout.Request](a.logger)).
				Post("/logout", logout.New(authClient, a.logger))
		})
	})
}

func (a *App) StartFileWorker(ctx context.Context) {
	fileRepo := repo.NewFileRepo(a.MySql.DB, a.logger)
	fileStorage := local.NewFileStorage(a.config.Storage, a.logger)

	fileWorker := worker.NewFileWorker(fileRepo, fileStorage, a.logger, a.config)
	fileWorker.Start(ctx)
}
