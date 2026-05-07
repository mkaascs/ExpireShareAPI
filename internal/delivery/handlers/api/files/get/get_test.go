package get

import (
	"context"
	"encoding/json"
	"expire-share/internal/delivery/middlewares"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/mocks"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_Get(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims := &middlewares.UserClaims{UserID: 1, Roles: []entities.UserRole{entities.RoleUser}}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFileGetter(ctrl)
		mockGetter.EXPECT().
			GetFileByAlias(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetFile) (*results.GetFile, error) {
				require.Equal(t, "abc123", cmd.Alias)
				require.Equal(t, int64(1), cmd.RequestingUserInfo.UserID)
				require.Equal(t, claims.Roles, cmd.RequestingUserInfo.Roles)
				return &results.GetFile{
					DownloadsLeft: 3,
					ExpiresIn:     2 * time.Hour,
					Filesize:      512,
				}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest("abc123", claims))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, int16(3), resp.DownloadsLeft)
		require.Equal(t, int64(512), resp.Filesize)
		require.NotEmpty(t, resp.ExpiresIn)
	})

	t.Run("missing user claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFileGetter(ctrl)
		handler := New(mockGetter, logger)

		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("alias", "abc123")
		r := httptest.NewRequest(http.MethodGet, "/file/abc123", nil)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("file not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFileGetter(ctrl)
		mockGetter.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(nil, domainErrors.ErrFileNotFound)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest("not-exist", claims))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFileGetter(ctrl)
		mockGetter.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(nil, context.Canceled)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest("abc123", claims))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockFileGetter(ctrl)
		mockGetter.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("db error"))

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newGetRequest("abc123", claims))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newGetRequest(alias string, claims *middlewares.UserClaims) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/file/"+alias, nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", alias)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)

	if claims != nil {
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
	}

	return r.WithContext(ctx)
}
