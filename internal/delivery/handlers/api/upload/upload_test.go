package upload_test

import (
	"bytes"
	"context"
	"encoding/json"
	"expire-share/internal/config"
	"expire-share/internal/delivery/handlers/api/upload"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/mocks"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_Upload(t *testing.T) {
	testCfg := config.Config{
		Service: config.Service{
			Permissions: config.Permissions{
				MaxFilesSizeForVipInBytes: 2 << 32,
			},
			MaxDownloads: 5,
			DefaultTtl:   2 * time.Hour,
		},
	}

	claims := &middlewares.UserClaims{
		UserID: 1,
		Roles:  []entities.UserRole{entities.RoleUser},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		mockUploader.EXPECT().
			UploadFile(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.UploadFile) (string, error) {
				require.Equal(t, "test.txt", cmd.Filename)
				require.Equal(t, int64(1), cmd.RequestingUserInfo.UserID)
				require.Equal(t, 2*time.Hour, cmd.TTL)
				require.Equal(t, int16(3), cmd.MaxDownloads)
				return "abc123", nil
			})

		r := buildMultipartRequest(t, "test.txt", "hello world", map[string]string{
			"ttl":           "2h",
			"max_downloads": "3",
		})

		r = withClaims(r, claims)

		handler := upload.New(mockUploader, logger, testCfg)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusCreated, w.Code)
		resp := parseResponse(t, w)
		require.Equal(t, "abc123", resp.Alias)
	})

	t.Run("success with password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		mockUploader.EXPECT().
			UploadFile(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.UploadFile) (string, error) {
				require.Equal(t, "secret", cmd.Password)
				return "xyz789", nil
			})

		r := buildMultipartRequest(t, "doc.pdf", "pdf content", map[string]string{
			"ttl":           "1h",
			"max_downloads": "3",
			"password":      "secret",
		})

		r = withClaims(r, claims)

		handler := upload.New(mockUploader, logger, testCfg)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success with defaults", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		mockUploader.EXPECT().
			UploadFile(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.UploadFile) (string, error) {
				require.Equal(t, testCfg.Service.DefaultTtl, cmd.TTL)
				require.Equal(t, testCfg.Service.MaxDownloads, cmd.MaxDownloads)
				return "def456", nil
			})

		r := buildMultipartRequest(t, "file.txt", "content", nil)
		r = withClaims(r, claims)

		handler := upload.New(mockUploader, logger, testCfg)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missing user claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		handler := upload.New(mockUploader, logger, testCfg)

		r := buildMultipartRequest(t, "test.txt", "data", map[string]string{"ttl": "1h"})

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid ttl format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		handler := upload.New(mockUploader, logger, testCfg)

		r := buildMultipartRequest(t, "test.txt", "data", map[string]string{
			"ttl": "not-a-duration",
		})

		r = withClaims(r, claims)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid max_downloads not a number", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		handler := upload.New(mockUploader, logger, testCfg)

		r := buildMultipartRequest(t, "test.txt", "data", map[string]string{
			"ttl":           "1h",
			"max_downloads": "not-a-number",
		})

		r = withClaims(r, claims)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("max_downloads out of range", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		handler := upload.New(mockUploader, logger, testCfg)

		r := buildMultipartRequest(t, "test.txt", "data", map[string]string{
			"ttl":           "1h",
			"max_downloads": "99999",
		})

		r = withClaims(r, claims)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing file in form", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		handler := upload.New(mockUploader, logger, testCfg)

		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		require.NoError(t, mw.WriteField("ttl", "1h"))
		require.NoError(t, mw.WriteField("max_downloads", "5"))
		require.NoError(t, mw.Close())

		r := httptest.NewRequest(http.MethodPost, "/upload", &buf)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		r = withClaims(r, claims)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("file size too big", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		mockUploader.EXPECT().
			UploadFile(gomock.Any(), gomock.Any()).
			Return("", domainErrors.ErrFileSizeTooBig)

		r := buildMultipartRequest(t, "big.bin", "lots of data", map[string]string{
			"ttl": "1h",
		})

		r = withClaims(r, claims)

		handler := upload.New(mockUploader, logger, testCfg)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		mockUploader.EXPECT().
			UploadFile(gomock.Any(), gomock.Any()).
			Return("", context.Canceled)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		r := buildMultipartRequest(t, "test.txt", "data", map[string]string{"ttl": "1h"})
		r = r.WithContext(ctx)
		r = withClaims(r, claims)

		handler := upload.New(mockUploader, logger, testCfg)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUploader := mocks.NewMockFileUploader(ctrl)
		mockUploader.EXPECT().
			UploadFile(gomock.Any(), gomock.Any()).
			Return("", fmt.Errorf("storage unavailable"))

		r := buildMultipartRequest(t, "test.txt", "data", map[string]string{"ttl": "1h"})
		r = withClaims(r, claims)

		handler := upload.New(mockUploader, logger, testCfg)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func buildMultipartRequest(t *testing.T, filename, content string, opts map[string]string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for key, val := range opts {
		require.NoError(t, w.WriteField(key, val))
	}

	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())

	routeCtx := chi.NewRouteContext()
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}

func withClaims(r *http.Request, claims *middlewares.UserClaims) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, "user_id", claims.UserID)
	ctx = context.WithValue(ctx, "roles", claims.Roles)
	return r.WithContext(ctx)
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) upload.Response {
	t.Helper()
	var resp upload.Response
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}
