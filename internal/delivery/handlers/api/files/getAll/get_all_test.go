package getAll

import (
	"context"
	"encoding/json"
	"errors"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/delivery/util"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/domain/entities"
	"expire-share/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_GetAll(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims := &middlewares.UserClaims{UserID: 1, Roles: []entities.UserRole{entities.RoleUser}}
	files := []results.GetFile{
		{
			DownloadsLeft: 3,
			ExpiresIn:     time.Hour,
			Filesize:      int64(512),
		},
		{
			DownloadsLeft: 5,
			ExpiresIn:     time.Minute,
			Filesize:      int64(1 << 10),
		},
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) (*results.GetAllFiles, error) {
				require.Equal(t, claims.UserID, cmd.UserID)
				require.Equal(t, 1, cmd.Page)
				require.Equal(t, 10, cmd.Limit)
				return &results.GetAllFiles{Files: files, Total: len(files)}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, ""))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, len(files), resp.Total)
		require.Len(t, resp.Files, len(files))
		for index := range files {
			require.Equal(t, files[index].DownloadsLeft, resp.Files[index].DownloadsLeft)
			require.Equal(t, util.TimeString(files[index].ExpiresIn), resp.Files[index].ExpiresIn)
		}
	})

	t.Run("success with custom pagination", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) (*results.GetAllFiles, error) {
				require.Equal(t, 3, cmd.Page)
				require.Equal(t, 25, cmd.Limit)
				return &results.GetAllFiles{Files: files, Total: len(files)}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, "?page=3&limit=25"))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid page", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) (*results.GetAllFiles, error) {
				require.Equal(t, 1, cmd.Page)
				return &results.GetAllFiles{}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, "?page=abc"))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("page zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) (*results.GetAllFiles, error) {
				require.Equal(t, 1, cmd.Page)
				return &results.GetAllFiles{}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, "?page=0"))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("limit over 100", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) (*results.GetAllFiles, error) {
				require.Equal(t, 100, cmd.Limit)
				return &results.GetAllFiles{}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, "?limit=9999"))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			Return(&results.GetAllFiles{Files: []results.GetFile{}, Total: 0}, nil)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, ""))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, 0, resp.Total)
		require.Empty(t, resp.Files)
	})

	t.Run("missing user claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().GetAllFiles(gomock.Any(), gomock.Any()).Times(0)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(nil, ""))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().GetAllFiles(gomock.Any(), gomock.Any()).
			Return(&results.GetAllFiles{}, context.Canceled)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, ""))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().GetAllFiles(gomock.Any(), gomock.Any()).
			Return(&results.GetAllFiles{}, errors.New("db error"))

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetAllRequest(claims, ""))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newGetAllRequest(claims *middlewares.UserClaims, query string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/file"+query, nil)

	if claims != nil {
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		return r.WithContext(ctx)
	}

	return r
}
