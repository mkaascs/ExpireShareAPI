package register

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

// Request represents registration request body
//
//	@Description	Registration data for new user account
type Request struct {
	Login    string `json:"login" validate:"required,min=3,max=64" example:"user"`
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required,min=6,max=128" example:"expire123"`
}

// Response represents registration response
//
//	@Description	Registration response with user ID
type Response struct {
	response.Response
	UserID int64 `json:"user_id,omitempty"`
}

type UserRegister interface {
	Register(ctx context.Context, command commands.Register) (*results.Register, error)
}

// New @Summary User registration
//
//	@Description	Create new user account with login, email and password.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		Request				true	"Registration data"
//	@Success		200		{object}	Response			"Registration successful"
//	@Failure		400		{object}	response.Response	"Invalid request body"
//	@Failure		409		{object}	response.Response	"User with this login or email already exists"
//	@Failure		422		{object}	response.Response	"Validation error"
//	@Failure		500		{object}	response.Response	"Internal server error"
//	@Router			/api/auth/register [post]
func New(register UserRegister, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.api.auth.register.New"
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

		result, err := register.Register(r.Context(), commands.Register{
			Login:    request.Login,
			Email:    request.Email,
			Password: request.Password,
		})

		if err != nil {
			if response.RenderAuthServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info("failed to register user", sl.Error(err))
				return
			}

			log.Error("failed to register user", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("user register successfully", slog.Int64("user_id", result.UserID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			UserID: result.UserID,
		})
	}
}
