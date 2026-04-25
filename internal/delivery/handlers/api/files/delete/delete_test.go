package delete

import (
	"context"
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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Delete(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims := &middlewares.UserClaims{UserID: 1, Roles: []entities.UserRole{entities.RoleUser}}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDeleter := mocks.NewMockFileDeleter(ctrl)
		mockDeleter.EXPECT().
			DeleteFile(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.DeleteFile) error {
				require.Equal(t, "abc123", cmd.Alias)
				require.Equal(t, int64(1), cmd.RequestingUserInfo.UserID)
				return nil
			})

		handler := New(mockDeleter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newDeleteRequest("abc123", claims))

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("missing user claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDeleter := mocks.NewMockFileDeleter(ctrl)
		handler := New(mockDeleter, logger)

		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("alias", "abc123")
		r := httptest.NewRequest(http.MethodDelete, "/file/abc123", nil)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("file not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDeleter := mocks.NewMockFileDeleter(ctrl)
		mockDeleter.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).
			Return(domainErrors.ErrFileNotFound)

		handler := New(mockDeleter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newDeleteRequest("not-exist", claims))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDeleter := mocks.NewMockFileDeleter(ctrl)
		mockDeleter.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).
			Return(context.Canceled)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("alias", "abc123")
		ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
		ctx = context.WithValue(ctx, "user_id", int64(1))
		ctx = context.WithValue(ctx, "roles", []entities.UserRole{entities.RoleUser})

		r := httptest.NewRequest(http.MethodDelete, "/file/abc123", nil)
		r = r.WithContext(ctx)

		handler := New(mockDeleter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDeleter := mocks.NewMockFileDeleter(ctrl)
		mockDeleter.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("db error"))

		handler := New(mockDeleter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newDeleteRequest("abc123", claims))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newDeleteRequest(alias string, claims *middlewares.UserClaims) *http.Request {
	r := httptest.NewRequest(http.MethodDelete, "/file/"+alias, nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", alias)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)

	if claims != nil {
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
	}

	return r.WithContext(ctx)
}
