package download

import (
	"context"
	"errors"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// Response represents standard API error response
//
//	@Description	Standard error response structure
type Response struct {
	response.Response
}

type FileDownloader interface {
	DownloadFile(ctx context.Context, command commands.DownloadFile) (*results.DownloadFile, error)
}

// New @Summary Download file
//
//	@Description	Downloads uploaded file by its alias. If file is password-protected, provide password in X-Resource-Password header.
//	@Tags			file
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			alias				path		string				true	"File alias"
//	@Param			X-Resource-Password	header		string				false	"File password (required for password-protected files)"
//	@Success		200					{file}		binary				"File content"
//	@Failure		403					{object}	response.Response	"File password required or invalid password"
//	@Failure		404					{object}	response.Response	"File not found or has expired"
//	@Failure		500					{object}	response.Response	"Internal server error"
//	@Router			/download/{alias} [get]
func New(downloader FileDownloader, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.api.download.New"
		log := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		alias := chi.URLParam(r, "alias")
		password := r.Header.Get("X-Resource-Password")

		file, err := downloader.DownloadFile(r.Context(), commands.DownloadFile{
			Alias:    alias,
			Password: password,
		})

		if err != nil {
			const msg = "failed to get file info"
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

		defer func() {
			if err := file.Close(); err != nil {
				log.Error("failed to close file", sl.Error(err))
			}
		}()

		contentType := mime.TypeByExtension(filepath.Ext(file.FileInfo.Name()))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.FileInfo.Name()))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", file.FileInfo.Size()))

		if _, err := io.Copy(w, file.File); err != nil {
			if errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe) {
				log.Info("client disconnected during download")
				return
			}

			log.Error("failed to write response", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("file was successfully downloaded", slog.String("alias", alias))
	}
}
