package middlewares

import (
	"crypto/subtle"
	"encoding/base64"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"
	"os"
)

func NewAdminAuth(rateLimiter RateLimiter, log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(slog.String("component", "middleware/admin"))

		adminSecret, err := base64.StdEncoding.DecodeString(os.Getenv("ADMIN_SECRET_BASE64"))
		if err != nil || len(adminSecret) == 0 {
			log.Error("failed to decode ADMIN_SECRET_BASE64", sl.Error(err))
			os.Exit(1)
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := util.ExtractIP(r.RemoteAddr)
			if ip == "" {
				log.Error("remote addr is empty. use middleware.RealIP at first")
				response.RenderError(w, r,
					http.StatusInternalServerError,
					"internal server error")
				return
			}

			allowed, err := rateLimiter.Allow(r.Context(), ip)
			if err != nil {
				log.Warn("rate limiter is disabled", sl.Error(err))
			} else if !allowed {
				log.Info("too many requests", slog.String("remote_addr", ip))
				response.RenderError(w, r,
					http.StatusTooManyRequests,
					"too many requests. try again later")
				return
			}

			secret := extractBearerToken(r.Header.Get("Authorization"))
			if secret == "" {
				log.Info("unauthorized request")
				response.RenderError(w, r,
					http.StatusUnauthorized,
					"unauthorized request")
				return
			}

			decodedSecret, err := base64.StdEncoding.DecodeString(secret)
			if err != nil {
				log.Info("failed to decode secret", sl.Error(err))
				response.RenderError(w, r,
					http.StatusUnauthorized,
					"unauthorized request")
				return
			}

			if subtle.ConstantTimeCompare(decodedSecret, adminSecret) == 1 {
				err := rateLimiter.Reset(r.Context(), ip)
				if err != nil {
					log.Warn("failed to reset rate limit", sl.Error(err), slog.String("remote_addr", ip))
				}

				next.ServeHTTP(w, r)
				return
			}

			log.Info("unauthorized request")
			response.RenderError(w, r,
				http.StatusUnauthorized,
				"unauthorized request")
		})
	}
}
