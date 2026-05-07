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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestService_GetFileByAlias(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{}

	command := commands.GetFile{
		Alias:  "file-alias",
		UserID: 1,
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			DoAndReturn(func(_ context.Context, alias string) (*entities.File, error) {
				assert.Equal(t, command.Alias, alias)
				return &entities.File{
					Name:          "file.txt",
					Alias:         command.Alias,
					PasswordHash:  "",
					UserID:        command.UserID,
					DownloadsLeft: 5,
					ExpiresAt:     time.Now().Add(2 * time.Hour),
					Size:          int64(2 << 10),
				}, nil
			})

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFileByAlias(context.Background(), command)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "file.txt", result.Filename)
		require.Equal(t, command.Alias, result.Alias)
		require.Equal(t, int16(5), result.DownloadsLeft)
		require.Equal(t, int64(2<<10), result.Filesize)
		require.Positive(t, result.ExpiresIn)
	})

	t.Run("file not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(nil, domainErrors.ErrFileNotFound)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFileByAlias(context.Background(), command)
		require.Nil(t, result)
		require.ErrorIs(t, err, domainErrors.ErrFileNotFound)
	})

	t.Run("forbidden another user file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{
				Alias:  command.Alias,
				UserID: int64(99),
			}, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFileByAlias(context.Background(), command)
		require.Nil(t, result)
		require.ErrorIs(t, err, domainErrors.ErrForbidden)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(nil, context.Canceled)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFileByAlias(ctx, command)
		require.Nil(t, result)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(nil, errors.New("internal error"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.GetFileByAlias(context.Background(), command)
		require.Nil(t, result)
		require.Error(t, err)
	})
}
