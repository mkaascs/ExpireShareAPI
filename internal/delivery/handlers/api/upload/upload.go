package upload

import (
	"context"
	"errors"
	"expire-share/internal/config"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/delivery/util/response"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/lib/log/sl"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	MaxDownloads int16         `json:"max_downloads,omitempty" validate:"min=1;max=10000" example:"5"`
	TTL          time.Duration `json:"ttl,omitempty" example:"2h30m"`
	Password     string        `json:"password,omitempty" example:"1234"`
}

// Response represents file upload response
//
//	@Description	Response after successful file upload
type Response struct {
	response.Response
	Alias string `json:"alias,omitempty"`
}

type FileUploader interface {
	UploadFile(ctx context.Context, command commands.UploadFile) (string, error)
}

// New @Summary Upload file
//
//	@Description	Uploads file to server with optional password protection, download limit, and expiration time. Requires authentication.
//	@Tags			file
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file			formData	file				true	"File to upload"
//	@Param			max_downloads	formData	int16				false	"Maximum number of downloads (max: 10000)"
//	@Param			ttl				formData	string				false	"Time to live (e.g., '1h', '2h30m', '7d')"
//	@Param			password		formData	string				false	"File password (optional, required for download if set)"
//	@Success		201				{object}	Response			"File uploaded successfully"
//	@Failure		400				{object}	response.Response	"Invalid request"
//	@Failure		401				{object}	response.Response	"Unauthorized"
//	@Failure		403				{object}	response.Response	"Forbidden (upload limit exceeded)"
//	@Failure		413				{object}	response.Response	"File too large"
//	@Failure		422				{object}	response.Response	"Unprocessable entity"
//	@Failure		500				{object}	response.Response	"Internal server error"
//	@Router			/api/upload [post]
func New(uploader FileUploader, log *slog.Logger, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "http.upload.api.New"
		log := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, err := middlewares.GetUserClaims(r)
		if err != nil {
			log.Error("failed to get user claims", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		err = r.ParseMultipartForm(cfg.MaxFileSizeInBytes)
		if err != nil {
			log.Info("failed to parse form", sl.Error(err))
			response.RenderError(w, r,
				http.StatusBadRequest,
				"failed to parse multipart/form")
			return
		}

		request, err := getRequestFromForm(cfg.Service, r)
		if err != nil {
			log.Info("failed to parse form", sl.Error(err))
			response.RenderError(w, r,
				http.StatusBadRequest,
				err.Error())
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			log.Info("file is required", sl.Error(err))
			response.RenderError(w, r,
				http.StatusBadRequest,
				"file is required")
			return
		}

		defer func(file multipart.File) {
			if err := file.Close(); err != nil {
				log.Error("failed to close file", sl.Error(err))
			}
		}(file)

		alias, err := uploader.UploadFile(r.Context(), commands.UploadFile{
			File:         file,
			Filesize:     header.Size,
			Filename:     header.Filename,
			Password:     request.Password,
			MaxDownloads: request.MaxDownloads,
			TTL:          request.TTL,
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: claims.UserID,
				Roles:  claims.Roles,
			},
		})

		if err != nil {
			if response.RenderFileServiceError(w, r, err) || util.IsCtxError(err) {
				log.Info("failed to upload file", sl.Error(err))
				return
			}

			log.Error("failed to upload file", sl.Error(err))
			response.RenderError(w, r,
				http.StatusInternalServerError,
				"internal server error")
			return
		}

		log.Info("file was successfully uploaded", slog.String("alias", alias))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{
			Alias: alias,
		})
	}
}

func getRequestFromForm(cfg config.Service, r *http.Request) (Request, error) {
	var maxDownloads int16
	maxDownloadsStr := r.FormValue("max_downloads")

	if maxDownloadsStr != "" {
		parsedDownloads, err := strconv.ParseInt(r.FormValue("max_downloads"), 10, 16)
		if err != nil {
			return Request{}, errors.New("max_downloads must be a number")
		}

		if parsedDownloads <= 0 || parsedDownloads > 10000 {
			return Request{}, errors.New("max_downloads must be between 1 and 10000")
		}

		maxDownloads = int16(parsedDownloads)

	} else {
		maxDownloads = cfg.MaxDownloads
	}

	var ttl time.Duration
	ttlStr := r.FormValue("ttl")

	if ttlStr != "" {
		var err error
		ttl, err = time.ParseDuration(ttlStr)
		if err != nil {
			return Request{}, errors.New("ttl must be like '1h30m'")
		}

	} else {
		ttl = cfg.DefaultTtl
	}

	return Request{
		MaxDownloads: maxDownloads,
		TTL:          ttl,
		Password:     r.FormValue("password"),
	}, nil
}
