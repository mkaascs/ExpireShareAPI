package assign

import (
	"context"
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

const requestField = "request"

func TestHandler_AssignRole(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAssigning := mocks.NewMockRoleAssigning(ctrl)
		mockAssigning.EXPECT().
			AssignRole(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd interface{}) error {
				return nil
			})

		handler := New(mockAssigning, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newAssignRequest("1", "admin"))

		require.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missing parsed body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAssigning := mocks.NewMockRoleAssigning(ctrl)
		mockAssigning.EXPECT().AssignRole(gomock.Any(), gomock.Any()).Times(0)

		handler := New(mockAssigning, logger)
		w := httptest.NewRecorder()

		r := httptest.NewRequest(http.MethodPost, "/admin/users/1/roles", nil)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("id", "1")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))

		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid user id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAssigning := mocks.NewMockRoleAssigning(ctrl)
		mockAssigning.EXPECT().AssignRole(gomock.Any(), gomock.Any()).Times(0)

		handler := New(mockAssigning, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newAssignRequest("abc", "admin"))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAssigning := mocks.NewMockRoleAssigning(ctrl)
		mockAssigning.EXPECT().
			AssignRole(gomock.Any(), gomock.Any()).
			Return(domainErrors.ErrUserNotFound)

		handler := New(mockAssigning, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newAssignRequest("1", "admin"))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAssigning := mocks.NewMockRoleAssigning(ctrl)
		mockAssigning.EXPECT().
			AssignRole(gomock.Any(), gomock.Any()).
			Return(context.Canceled)

		handler := New(mockAssigning, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newAssignRequest("1", "admin"))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAssigning := mocks.NewMockRoleAssigning(ctrl)
		mockAssigning.EXPECT().
			AssignRole(gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("db error"))

		handler := New(mockAssigning, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newAssignRequest("1", "admin"))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newAssignRequest(userID string, role string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/admin/users/"+userID+"/roles", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", userID)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, requestField, Request{Role: role})

	return r.WithContext(ctx)
}
