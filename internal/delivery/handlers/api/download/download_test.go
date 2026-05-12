package download

import (
	"bytes"
	"context"
	"errors"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/mocks"
	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHandler_Download(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)

		fileContent := "hello world"
		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, command commands.DownloadFile) (*results.DownloadFile, error) {
				require.Equal(t, "abc123", command.Alias)
				require.Empty(t, command.Password)
				return newFileResult(fileContent, "test.txt"), nil
			})

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest("abc123", ""))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, fileContent, w.Body.String())
		require.Contains(t, w.Header().Get("Content-Disposition"), "test.txt")
		require.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("success with password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)

		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, command commands.DownloadFile) (*results.DownloadFile, error) {
				require.Equal(t, "secret123", command.Password)
				return newFileResult("data", "file.bin"), nil
			})

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest("xyz", "secret123"))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("file not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)
		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			Return(nil, domainErrors.ErrFileNotFound)

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest("not_exist", ""))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("password required", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)
		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			Return(nil, domainErrors.ErrFilePasswordRequired)

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest("protected", ""))

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)
		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			Return(nil, domainErrors.ErrFilePasswordInvalid)

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest("protected", "wrongpass"))

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)
		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			Return(nil, context.Canceled)

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("alias", "abc")
		r := httptest.NewRequest(http.MethodGet, "/download/abc", nil)
		r = r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, routeCtx))

		handler.ServeHTTP(w, r)

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDownloader := mocks.NewMockFileDownloader(ctrl)
		mockDownloader.EXPECT().
			DownloadFile(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("internal error"))

		handler := New(mockDownloader, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest("abc", ""))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newRequest(alias, password string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/download/"+alias, nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", alias)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))

	if password != "" {
		r.Header.Set("X-Resource-Password", password)
	}

	return r
}

func newFileResult(content string, filename string) *results.DownloadFile {
	file := io.NopCloser(bytes.NewBufferString(content))
	return &results.DownloadFile{
		File:     file,
		FileInfo: mockFileInfo{name: filename, size: int64(len(content))},
		Close:    func() error { return file.Close() },
	}
}

type mockFileInfo struct {
	name string
	size int64
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() interface{}   { return nil }
