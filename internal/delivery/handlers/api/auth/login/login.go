package login

import (
	"context"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/auth/commands"
	"expire-share/internal/domain/dto/auth/results"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

// Request represents login request body
//
//	@Description	Login credentials for authentication
type Request struct {
	Login    string `json:"login" validate:"required,min=3,max=64" example:"user"`
	Password string `json:"password" validate:"required,min=6,max=128" example:"expire123"`
}

// Response represents login response
//
//	@Description	Authentication response with tokens
type Response struct {
	response.Response
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`

	// Token expiration time in seconds
	//	@example	900
	ExpiresIn int64 `json:"expires_in,omitempty"`
}

type UserLogin interface {
	Login(ctx context.Context, command commands.Login) (*results.Login, error)
}

// New @Summary User login
//
//	@Description	Authenticate user with login and password. Returns access and refresh tokens.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		Request				true	"Login credentials"
//	@Success		200		{object}	Response			"Login successful"
//	@Failure		400		{object}	response.Response	"Invalid request body"
//	@Failure		401		{object}	response.Response	"Invalid login or password"
//	@Failure		422		{object}	response.Response	"Validation error"
//	@Failure		500		{object}	response.Response	"Internal server error"
//	@Router			/api/auth/login [post]
func New(login UserLogin, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.api.auth.login.New"
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

		result, err := login.Login(r.Context(), commands.Login{
			Login:    request.Login,
			Password: request.Password,
		})

		if err != nil {
			if response.RenderAuthServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info("failed to login", sl.Error(err))
				return
			}

			log.Error("failed to login", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("user login successfully", slog.Int64("user_id", result.User.ID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresIn:    result.ExpiresIn,
		})
	}
}
