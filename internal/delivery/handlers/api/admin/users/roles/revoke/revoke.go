package revoke

import (
	"context"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/users/commands"
	"expire-share/internal/domain/entities"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	response.Response
}

type Request struct {
	Role string `json:"role" example:"vip"`
}

type RoleRevoker interface {
	RevokeRole(ctx context.Context, command commands.RevokeRole) error
}

func New(revoker RoleRevoker, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.api.admin.users.roles.revoke.New"
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

		userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Info("failed to parse user id")
			response.RenderError(w, r,
				http.StatusBadRequest,
				"failed to parse user id")
			return
		}

		err = revoker.RevokeRole(r.Context(), commands.RevokeRole{
			UserID: userID,
			Role:   entities.UserRole(request.Role),
		})

		if err != nil {
			const msg = "failed to revoke role"
			if response.RenderAuthServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info(msg, sl.Error(err), slog.String("role", request.Role))
				return
			}

			log.Error(msg, sl.Error(err), slog.String("role", request.Role))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("role revoked successfully", slog.String("role", request.Role), slog.Int64("user_id", userID))
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, Response{})
	}
}
