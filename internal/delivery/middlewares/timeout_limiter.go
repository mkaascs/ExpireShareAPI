package middlewares

import (
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"
	"time"
)

type TimeoutLimiterParams struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func NewTimeoutLimiter(params TimeoutLimiterParams, log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(slog.String("component", "middleware/timeout-limiter"))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			if err := rc.SetReadDeadline(time.Now().Add(params.ReadTimeout)); err != nil {
				log.Warn("failed to set read deadline", sl.Error(err))
			}

			if err := rc.SetWriteDeadline(time.Now().Add(params.WriteTimeout)); err != nil {
				log.Warn("failed to set write deadline", sl.Error(err))
			}

			next.ServeHTTP(w, r)
		})
	}
}
