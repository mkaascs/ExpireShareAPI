package stat

import (
	"context"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

// Response represents file storage statistics response
//
//	@Description	Response with user file storage stats
type Response struct {
	response.Response
	OccupiedSize int64 `json:"occupied_size,omitempty"`
	MaxSize      int64 `json:"max_size,omitempty"`
	Count        int   `json:"count,omitempty"`
	MaxCount     int   `json:"max_count,omitempty"`
}

type FilesStatGetter interface {
	GetFilesStat(ctx context.Context, command commands.GetFilesStat) (*results.GetFilesStat, error)
}

// New @Summary Get file storage stats
//
//	@Description	Get current user's file storage statistics: occupied size, total count and their limits.
//	@Tags			file
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	Response			"Storage stats"
//	@Failure		401	{object}	response.Response	"Unauthorized"
//	@Failure		500	{object}	response.Response	"Internal server error"
//	@Router			/api/file/stat [get]
func New(getter FilesStatGetter, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.file.stat.New"
		log := log.With(slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, err := middlewares.GetUserClaims(r)
		if err != nil {
			log.Error("failed to get user claims", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		result, err := getter.GetFilesStat(r.Context(), commands.GetFilesStat{
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: claims.UserID,
				Roles:  claims.Roles,
			},
		})

		if err != nil {
			if util.IsCtxError(err) {
				log.Info("failed to get files stat", sl.Error(err))
				return
			}

			log.Error("failed to get files stat", sl.Error(err), slog.Int64("user_id", claims.UserID))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("files stat was sent", slog.Int64("user_id", claims.UserID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, &Response{
			OccupiedSize: result.Stat.Size,
			Count:        result.Stat.Count,
			MaxSize:      result.MaxSize,
			MaxCount:     result.MaxCount,
		})
	}
}
