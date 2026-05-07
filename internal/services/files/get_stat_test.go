package files

import (
	"context"
	"errors"
	"expire-share/internal/config"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
)

func TestService_GetFilesStat(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		Service: config.Service{
			Permissions: config.Permissions{
				MaxUploadedFiles:          50,
				MaxFilesSizeForUserInBytes: 500 * 1024 * 1024,
				MaxFilesSizeForVipInBytes:  2 * 1024 * 1024 * 1024,
			},
		},
	}

	t.Run("success for regular user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		stat := &entities.FilesStat{UserID: 1, Count: 3, Size: 1024}
		mockFileRepo.EXPECT().
			GetFilesStatByUserID(gomock.Any(), int64(1)).
			Return(stat, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFilesStat(context.Background(), commands.GetFilesStat{
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: 1,
				Roles:  []entities.UserRole{entities.RoleUser},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 3, result.Stat.Count)
		require.Equal(t, int64(1024), result.Stat.Size)
		require.Equal(t, cfg.Permissions.MaxUploadedFiles, result.MaxCount)
		require.Equal(t, cfg.Permissions.MaxFilesSizeForUserInBytes, result.MaxSize)
	})

	t.Run("success for vip user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		stat := &entities.FilesStat{UserID: 2, Count: 10, Size: 512 * 1024}
		mockFileRepo.EXPECT().
			GetFilesStatByUserID(gomock.Any(), int64(2)).
			Return(stat, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFilesStat(context.Background(), commands.GetFilesStat{
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: 2,
				Roles:  []entities.UserRole{entities.RoleVip},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, cfg.Permissions.MaxUploadedFiles, result.MaxCount)
		require.Equal(t, cfg.Permissions.MaxFilesSizeForVipInBytes, result.MaxSize)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesStatByUserID(gomock.Any(), int64(1)).
			Return(nil, domainErrors.ErrUserNotFound)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFilesStat(context.Background(), commands.GetFilesStat{
			RequestingUserInfo: commands.RequestingUserInfo{UserID: 1},
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, domainErrors.ErrUserNotFound)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockFileRepo.EXPECT().
			GetFilesStatByUserID(gomock.Any(), int64(1)).
			Return(nil, context.Canceled)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFilesStat(ctx, commands.GetFilesStat{
			RequestingUserInfo: commands.RequestingUserInfo{UserID: 1},
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesStatByUserID(gomock.Any(), int64(1)).
			Return(nil, errors.New("db error"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFilesStat(context.Background(), commands.GetFilesStat{
			RequestingUserInfo: commands.RequestingUserInfo{UserID: 1},
		})

		require.Nil(t, result)
		require.Error(t, err)
	})
}
