package logout

import (
	"context"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/auth/commands"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

// Request represents logout request body
//
//	@Description	Tokens to invalidate on logout
type Request struct {
	AccessToken  string `json:"access_token" validate:"required"`
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Response represents logout response
//
//	@Description	Empty response on successful logout
type Response struct {
	response.Response
}

type UserLogout interface {
	Logout(ctx context.Context, command commands.Logout) error
}

// New @Summary User logout
//
//	@Description	Invalidate access and refresh tokens. User will need to login again to obtain new tokens.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	Request	true	"Tokens to invalidate"
//	@Success		200		"Logout successful"
//	@Failure		400		{object}	response.Response	"Invalid request body"
//	@Failure		401		{object}	response.Response	"Invalid tokens"
//	@Failure		422		{object}	response.Response	"Validation error"
//	@Failure		500		{object}	response.Response	"Internal server error"
//	@Router			/api/auth/logout [post]
func New(logout UserLogout, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.api.auth.logout.New"
		log := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		request, ok := middlewares.GetParsedBodyRequest[Request](r)
		if !ok {
			log.Error("failed to parse request")
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		err := logout.Logout(r.Context(), commands.Logout{
			AccessToken:  request.AccessToken,
			RefreshToken: request.RefreshToken,
		})

		if err != nil {
			if response.RenderAuthServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info("failed to logout", sl.Error(err))
				return
			}

			log.Error("failed to logout", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("user logout successfully")
		render.Status(r, http.StatusOK)
	}
}
