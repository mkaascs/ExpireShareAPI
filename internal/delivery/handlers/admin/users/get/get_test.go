package get

import (
	"context"
	"encoding/json"
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

func TestHandler_GetUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedUser := &entities.User{
			ID:    1,
			Email: "user@example.com",
			Login: "user",
		}

		mockGetter := mocks.NewMockUserGetter(ctrl)
		mockGetter.EXPECT().
			GetUserByID(gomock.Any(), int64(1)).
			DoAndReturn(func(ctx context.Context, userID int64) (*entities.User, error) {
				require.Equal(t, int64(1), userID)
				return expectedUser, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newUserRequest("1"))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, expectedUser.ID, resp.User.ID)
		require.Equal(t, expectedUser.Email, resp.User.Email)
		require.Equal(t, expectedUser.Login, resp.User.Login)
	})

	t.Run("user id is not a number", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockUserGetter(ctrl)
		mockGetter.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).Times(0)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newUserRequest("abc"))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty user id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockUserGetter(ctrl)
		mockGetter.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).Times(0)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newUserRequest(""))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockUserGetter(ctrl)
		mockGetter.EXPECT().
			GetUserByID(gomock.Any(), int64(999)).
			Return(nil, domainErrors.ErrUserNotFound)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newUserRequest("999"))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockUserGetter(ctrl)
		mockGetter.EXPECT().
			GetUserByID(gomock.Any(), int64(1)).
			Return(nil, context.Canceled)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newUserRequest("1"))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockUserGetter(ctrl)
		mockGetter.EXPECT().
			GetUserByID(gomock.Any(), int64(1)).
			Return(nil, fmt.Errorf("db connection lost"))

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newUserRequest("1"))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newUserRequest(id string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/admin/users/"+id, nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", id)

	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}
