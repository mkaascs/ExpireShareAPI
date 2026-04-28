package middlewares

import (
	"context"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"
)

type RateLimiter interface {
	Allow(ctx context.Context, field, value string) (bool, error)
	Reset(ctx context.Context, field, value string) error
}

type RateLimiterParams struct {
	Field string
	Value string
}

const fieldName = "ip"

func NewRateLimiter(limiter RateLimiter, log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(slog.String("component", "middleware/rate-limiter"))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.RemoteAddr == "" {
				log.Error("remote addr is empty. use middleware.RealIP at first")
				response.RenderError(w, r,
					http.StatusInternalServerError,
					"internal server error")
				return
			}

			allowed, err := limiter.Allow(r.Context(), fieldName, r.RemoteAddr)

			if err != nil {
				log.Warn("rate limiter is disabled", sl.Error(err))
			} else if !allowed {
				log.Info("too many requests", slog.String("remote_addr", r.RemoteAddr))
				response.RenderError(w, r,
					http.StatusTooManyRequests,
					"too many requests. try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
