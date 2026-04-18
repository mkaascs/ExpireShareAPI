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
		},
		{
			DownloadsLeft: 5,
			ExpiresIn:     time.Minute,
		},
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().
			GetAllFiles(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) ([]results.GetFile, error) {
				require.Equal(t, claims.UserID, cmd.RequestingUserInfo.UserID)
				require.Equal(t, claims.Roles, cmd.RequestingUserInfo.Roles)
				return files, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest(claims))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Len(t, resp.Files, len(files))
		for index := range files {
			require.Equal(t, files[index].DownloadsLeft, resp.Files[index].DownloadsLeft)
			require.Equal(t, util.TimeString(files[index].ExpiresIn), resp.Files[index].ExpiresIn)
		}
	})

	t.Run("missing user claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		handler := New(mockGetter, logger)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest(nil))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().GetAllFiles(gomock.Any(), gomock.Any()).
			Return(nil, context.Canceled)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest(claims))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllFilesGetter(ctrl)
		mockGetter.EXPECT().GetAllFiles(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest(claims))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newGetRequest(claims *middlewares.UserClaims) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/file", nil)

	if claims != nil {
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		return r.WithContext(ctx)
	}

	return r
}
