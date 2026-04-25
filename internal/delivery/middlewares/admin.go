package middlewares

import (
	"crypto/subtle"
	"encoding/base64"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"
	"os"
)

func NewAdminAuth(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(slog.String("component", "middleware/admin"))

		adminSecret, err := base64.StdEncoding.DecodeString(os.Getenv("ADMIN_SECRET_BASE64"))
		if err != nil || len(adminSecret) == 0 {
			log.Error("failed to decode ADMIN_SECRET_BASE64", sl.Error(err))
			os.Exit(1)
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			secret := extractBearerToken(r.Header.Get("Authorization"))
			if secret == "" {
				log.Info("unauthorized request")
				response.RenderError(w, r,
					http.StatusUnauthorized,
					"unauthorized request")
				return
			}

			if subtle.ConstantTimeCompare([]byte(secret), adminSecret) == 1 {
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
