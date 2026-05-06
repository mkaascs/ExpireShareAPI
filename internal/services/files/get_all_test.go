package files

import (
	"context"
	"errors"
	"expire-share/internal/config"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	"expire-share/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestService_GetAllFiles(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{}

	command := commands.GetAllFiles{
		RequestingUserInfo: commands.RequestingUserInfo{
			UserID: 1,
			Roles:  []entities.UserRole{entities.RoleUser},
		},
	}

	files := []entities.File{
		{
			Name:          "file.txt",
			Alias:         "file-alias1",
			DownloadsLeft: 5,
			ExpiresAt:     time.Now().Add(time.Hour),
			Size:          int64(2 << 10),
		},
		{
			Name:          "file.pdf",
			Alias:         "file-alias2",
			DownloadsLeft: 1,
			ExpiresAt:     time.Now().Add(time.Minute),
			Size:          int64(2 << 42),
		},
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesByUserID(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, userID int64) ([]entities.File, error) {
				require.Equal(t, command.UserID, userID)
				return files, nil
			})

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.NoError(t, err)
		require.Len(t, result, len(files))
		for index := range files {
			require.Equal(t, files[index].Alias, result[index].Alias)
			require.Equal(t, files[index].Name, result[index].Filename)
			require.Equal(t, files[index].DownloadsLeft, result[index].DownloadsLeft)
			require.WithinDuration(t, files[index].ExpiresAt, time.Now().Add(result[index].ExpiresIn), time.Second)
		}
	})

	t.Run("success with empty file list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return([]entities.File{}, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return(nil, context.Canceled)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.Equal(t, context.Canceled, err)
		require.Empty(t, result)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("internal error"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.Error(t, err)
		require.Empty(t, result)
	})
}
