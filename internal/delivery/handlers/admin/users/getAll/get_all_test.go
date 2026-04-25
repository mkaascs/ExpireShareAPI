package getAll

import (
	"context"
	"encoding/json"
	"expire-share/internal/domain/dto/users/commands"
	"expire-share/internal/domain/dto/users/results"
	"expire-share/internal/domain/entities"
	"expire-share/internal/mocks"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_GetAllUsers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success with default pagination", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedUsers := []entities.User{
			{ID: 1, Email: "a@example.com", Login: "user_a"},
			{ID: 2, Email: "b@example.com", Login: "user_b"},
		}

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), commands.GetAllUsers{Page: 1, Limit: 10, Role: nil}).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllUsers) (results.GetAllUsers, error) {
				require.Equal(t, 1, cmd.Page)
				require.Equal(t, 10, cmd.Limit)
				require.Nil(t, cmd.Role)
				return results.GetAllUsers{Users: expectedUsers, Total: 2}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, 1, resp.Page)
		require.Equal(t, 10, resp.Limit)
		require.Equal(t, 2, resp.Total)
		require.Len(t, resp.Users, 2)
		require.Equal(t, expectedUsers[0].ID, resp.Users[0].ID)
	})

	t.Run("success with custom pagination", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), commands.GetAllUsers{Page: 3, Limit: 20, Role: nil}).
			Return(results.GetAllUsers{Users: []entities.User{}, Total: 0}, nil)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users?page=3&limit=20", nil))

		require.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Equal(t, 3, resp.Page)
		require.Equal(t, 20, resp.Limit)
	})

	t.Run("success with filter by role", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		role := entities.UserRole("admin")

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllUsers) (results.GetAllUsers, error) {
				require.NotNil(t, cmd.Role)
				require.Equal(t, role, *cmd.Role)
				return results.GetAllUsers{Users: []entities.User{}, Total: 0}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users?role=admin", nil))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid page", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllUsers) (results.GetAllUsers, error) {
				require.Equal(t, 1, cmd.Page)
				return results.GetAllUsers{}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users?page=abc", nil))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("page zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllUsers) (results.GetAllUsers, error) {
				require.Equal(t, 1, cmd.Page)
				return results.GetAllUsers{}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users?page=0", nil))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("limit over 100", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllUsers) (results.GetAllUsers, error) {
				require.Equal(t, 100, cmd.Limit)
				return results.GetAllUsers{}, nil
			})

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users?limit=999", nil))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), gomock.Any()).
			Return(results.GetAllUsers{}, context.Canceled)

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

		require.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockGetter := mocks.NewMockAllUsersGetter(ctrl)
		mockGetter.EXPECT().
			GetUsers(gomock.Any(), gomock.Any()).
			Return(results.GetAllUsers{}, fmt.Errorf("db error"))

		handler := New(mockGetter, logger)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
