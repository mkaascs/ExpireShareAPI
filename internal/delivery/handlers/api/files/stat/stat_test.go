package stat

import (
	"context"
	"encoding/json"
	"errors"
	"expire-share/internal/delivery/middlewares"
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
)

func TestHandler_Stat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims := &middlewares.UserClaims{UserID: 1, Roles: []entities.UserRole{entities.RoleUser}}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFilesStatGetter(ctrl)
		mockGetter.EXPECT().
			GetFilesStat(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetFilesStat) (*results.GetFilesStat, error) {
				require.Equal(t, claims.UserID, cmd.UserID)
				require.Equal(t, claims.Roles, cmd.Roles)
				return &results.GetFilesStat{
					Stat:     entities.FilesStat{Count: 3, Size: 1024},
					MaxSize:  500 * 1024 * 1024,
					MaxCount: 10,
				}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newStatRequest(claims))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, int64(1024), resp.OccupiedSize)
		require.Equal(t, int64(500*1024*1024), resp.MaxSize)
		require.Equal(t, 3, resp.Count)
		require.Equal(t, 10, resp.MaxCount)
	})

	t.Run("missing user claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFilesStatGetter(ctrl)
		mockGetter.EXPECT().GetFilesStat(gomock.Any(), gomock.Any()).Times(0)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newStatRequest(nil))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFilesStatGetter(ctrl)
		mockGetter.EXPECT().GetFilesStat(gomock.Any(), gomock.Any()).
			Return(nil, context.Canceled)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newStatRequest(claims))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFilesStatGetter(ctrl)
		mockGetter.EXPECT().GetFilesStat(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newStatRequest(claims))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newStatRequest(claims *middlewares.UserClaims) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/file/stat", nil)

	if claims != nil {
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		return r.WithContext(ctx)
	}

	return r
}
