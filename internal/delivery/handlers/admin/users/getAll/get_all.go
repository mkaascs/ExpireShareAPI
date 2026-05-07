package getAll

import (
	"context"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/users/commands"
	"expire-share/internal/domain/dto/users/results"
	"expire-share/internal/domain/entities"
	"expire-share/internal/lib/log/sl"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

// Response represents paginated users list response
//
//	@Description	Response with paginated users list
type Response struct {
	response.Response
	Page  int             `json:"page,omitempty"`
	Limit int             `json:"limit,omitempty"`
	Total int             `json:"total,omitempty"`
	Users []entities.User `json:"users,omitempty"`
}

type AllUsersGetter interface {
	GetAllUsers(ctx context.Context, command commands.GetAllUsers) (*results.GetAllUsers, error)
}

// New @Summary Get all users
//
//	@Description	Get paginated list of all users. Optionally filter by role. Requires admin secret authorization.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int					false	"Page number (default: 1)"
//	@Param			limit	query		int					false	"Items per page (default: 10, max: 100)"
//	@Param			role	query		string				false	"Filter by role (user, vip, admin)"
//	@Success		200		{object}	Response			"Users list"
//	@Failure		401		{object}	response.Response	"Unauthorized"
//	@Failure		500		{object}	response.Response	"Internal server error"
//	@Router			/api/admin/users [get]
func New(getter AllUsersGetter, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.admin.users.getAll.New"
		log := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		var page, limit int
		util.ScanPaginationArgs(r, &page, &limit)

		var role *entities.UserRole
		if roleQuery := r.URL.Query().Get("role"); roleQuery != "" {
			r := entities.UserRole(roleQuery)
			role = &r
		}

		result, err := getter.GetAllUsers(r.Context(), commands.GetAllUsers{
			Page:  page,
			Limit: limit,
			Role:  role,
		})

		if err != nil {
			const msg = "failed to get users info"
			if util.IsCtxError(err) {
				log.Info(msg, sl.Error(err))
				return
			}

			log.Error(msg, sl.Error(err), slog.Int("page", page), slog.Int("limit", limit))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("users info was sent", slog.Int("page", page), slog.Int("limit", limit))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			Page:  page,
			Limit: limit,
			Total: result.Total,
			Users: result.Users,
		})
	}
}
