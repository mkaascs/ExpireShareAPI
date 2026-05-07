package files

import (
	"context"
	"errors"
	"expire-share/internal/config"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	"expire-share/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
		Page:   1,
		Limit:  10,
		UserID: 1,
	}

	repoResult := []entities.File{
		{
			Alias:         "file-alias1",
			Name:          "file.txt",
			Size:          int64(2 << 10),
			DownloadsLeft: 5,
			ExpiresAt:     time.Now().Add(time.Hour),
		},
		{
			Alias:         "file-alias2",
			Name:          "file.pdf",
			Size:          int64(2 << 42),
			DownloadsLeft: 1,
			ExpiresAt:     time.Now().Add(time.Minute),
		},
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesByUserID(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) ([]entities.File, int, error) {
				assert.Equal(t, command.UserID, cmd.UserID)
				assert.Equal(t, command.Page, cmd.Page)
				assert.Equal(t, command.Limit, cmd.Limit)
				return repoResult, len(repoResult), nil
			})

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Files, len(repoResult))
		require.Equal(t, len(repoResult), result.Total)
		for i := range repoResult {
			require.Equal(t, repoResult[i].Alias, result.Files[i].Alias)
			require.Equal(t, repoResult[i].Name, result.Files[i].Filename)
			require.Equal(t, repoResult[i].DownloadsLeft, result.Files[i].DownloadsLeft)
		}
	})

	t.Run("success with custom pagination passed to repo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		customCmd := commands.GetAllFiles{
			Page:   3,
			Limit:  25,
			UserID: 1,
		}

		mockFileRepo.EXPECT().
			GetFilesByUserID(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, cmd commands.GetAllFiles) ([]entities.File, int, error) {
				assert.Equal(t, 3, cmd.Page)
				assert.Equal(t, 25, cmd.Limit)
				return repoResult, len(repoResult), nil
			})

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), customCmd)
		require.NoError(t, err)
		require.Equal(t, len(repoResult), result.Total)
		require.NotNil(t, result)
	})

	t.Run("success with empty file list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return([]entities.File{}, 0, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Empty(t, result.Files)
		require.Equal(t, 0, result.Total)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return(nil, 0, context.Canceled)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, result)
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return(nil, 0, context.DeadlineExceeded)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Nil(t, result)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().
			GetFilesByUserID(gomock.Any(), gomock.Any()).
			Return(nil, 0, errors.New("db connection lost"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetAllFiles(context.Background(), command)
		require.Error(t, err)
		require.Nil(t, result)
	})
}
