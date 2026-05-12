package get

import (
	"context"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/entities"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

// Response represents user information response
//
//	@Description	Response with user info
type Response struct {
	response.Response
	entities.User `json:"user,omitempty"`
}

type UserGetter interface {
	GetUserByID(ctx context.Context, userID int64) (*entities.User, error)
}

// New @Summary Get user by ID
//
//	@Description	Get user info by their ID. Requires admin secret authorization.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int					true	"User ID"
//	@Success		200	{object}	Response			"User info"
//	@Failure		400	{object}	response.Response	"Invalid user ID"
//	@Failure		401	{object}	response.Response	"Unauthorized"
//	@Failure		404	{object}	response.Response	"User not found"
//	@Failure		500	{object}	response.Response	"Internal server error"
//	@Router			/api/admin/users/{id} [get]
func New(getter UserGetter, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.api.admin.users.get.New"
		log := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Info("failed to parse user id")
			response.RenderError(w, r,
				http.StatusBadRequest,
				"failed to parse user id")
			return
		}

		user, err := getter.GetUserByID(r.Context(), userID)
		if err != nil {
			const msg = "failed to get user by id"
			if response.RenderAuthServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info(msg, sl.Error(err), slog.Int64("user_id", userID))
				return
			}

			log.Error(msg, sl.Error(err), slog.Int64("user_id", userID))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("user info was sent", slog.Int64("user_id", userID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			User: *user,
		})
	}
}
