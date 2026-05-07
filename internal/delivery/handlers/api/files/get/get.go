package get

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

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

// Response represents file information response
//
//	@Description	Response with file info
type Response struct {
	response.Response
	Filename      string `json:"filename"`
	Filesize      int64  `json:"filesize"`
	DownloadsLeft int16  `json:"downloads_left,omitempty"`
	ExpiresIn     string `json:"expires_in,omitempty"`
}

type FileGetter interface {
	GetFileByAlias(ctx context.Context, command commands.GetFile) (*results.GetFile, error)
}

// New @Summary Get file info
//
//	@Description	Get info about uploaded file by its alias. Requires authentication and file ownership.
//	@Tags			file
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			alias	path		string	true	"File alias"
//	@Success		200		{object}	Response
//	@Failure		401		{object}	response.Response	"Unauthorized"
//	@Failure		403		{object}	response.Response	"Forbidden (not file owner)"
//	@Failure		404		{object}	response.Response	"File not found"
//	@Failure		500		{object}	response.Response	"Internal server error"
//	@Router			/api/file/{alias} [get]
func New(getter FileGetter, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.file.api.get.New"
		log := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		alias := chi.URLParam(r, "alias")

		claims, err := middlewares.GetUserClaims(r)
		if err != nil {
			log.Error("failed to get user claims", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		file, err := getter.GetFileByAlias(r.Context(), commands.GetFile{
			Alias:  alias,
			UserID: claims.UserID,
		})

		if err != nil {
			const msg = "failed to get file info by alias"
			if response.RenderFileServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info(msg, sl.Error(err), slog.String("alias", alias))
				return
			}

			log.Error(msg, sl.Error(err), slog.String("alias", alias))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("file info was sent", slog.String("alias", alias))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			Filename:      file.Filename,
			Filesize:      file.Filesize,
			DownloadsLeft: file.DownloadsLeft,
			ExpiresIn:     util.TimeString(file.ExpiresIn),
		})
	}
}
