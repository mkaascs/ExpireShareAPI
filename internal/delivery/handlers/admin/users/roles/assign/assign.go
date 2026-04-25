package assign

import (
	"context"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/users/commands"
	"expire-share/internal/domain/entities"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"strconv"
)

type Response struct {
	response.Response
}

type Request struct {
	Role string `json:"role"`
}

type RoleAssigning interface {
	AssignRole(ctx context.Context, command commands.AssignRole) error
}

func New(assigning RoleAssigning, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.admin.users.roles.assign.New"
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

		err = assigning.AssignRole(r.Context(), commands.AssignRole{
			UserID: userID,
			Role:   entities.UserRole(request.Role),
		})

		if err != nil {
			const msg = "failed to assign role"
			if response.RenderAuthServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info(msg, slog.Int64("user_id", userID))
				return
			}

			log.Error(msg, slog.Int64("user_id", userID))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("role assigned successfully", slog.Int64("user_id", userID), slog.String("role", request.Role))
		render.Status(r, http.StatusCreated)
	}
}
