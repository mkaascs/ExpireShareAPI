package getAll

import (
	"context"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/lib/log/sl"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type GetFile struct {
	DownloadsLeft int16  `json:"downloads_left"`
	ExpiresIn     string `json:"expires_at"`
}

type Response struct {
	response.Response
	Files []GetFile `json:"files,omitempty"`
}

type AllFilesGetter interface {
	GetAllFiles(ctx context.Context, command commands.GetAllFiles) ([]results.GetFile, error)
}

func New(getter AllFilesGetter, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.file.getAll.New"
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

		files, err := getter.GetAllFiles(r.Context(), commands.GetAllFiles{
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: claims.UserID,
				Roles:  claims.Roles,
			},
		})

		if err != nil {
			if util.IsCtxError(err) {
				log.Info("failed to get all files", sl.Error(err))
				return
			}

			log.Error("failed to get all files", sl.Error(err), slog.Int64("user_id", claims.UserID))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		result := Response{
			Files: make([]GetFile, 0, len(files)),
		}

		for _, file := range files {
			result.Files = append(result.Files, GetFile{
				DownloadsLeft: file.DownloadsLeft,
				ExpiresIn:     util.TimeString(file.ExpiresIn),
			})
		}

		log.Info("files info was sent", slog.Int64("user_id", claims.UserID), slog.Int("count", len(files)))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, result)
	}
}
