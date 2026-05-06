package files

import (
	"context"
	"expire-share/internal/config"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/domain/interfaces/tx"
	"expire-share/internal/mocks"
	"expire-share/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestService_DownloadFile(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{}

	command := commands.DownloadFile{
		Alias:    "file-alias",
		Password: "",
	}

	fileContent := "file content"
	file := io.NopCloser(strings.NewReader(fileContent))
	validStorageResult := &results.DownloadFile{
		File:  file,
		Close: file.Close,
	}

	t.Run("success downloads left > 0", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{
				Alias:        command.Alias,
				PasswordHash: "",
			}, nil)

		mockFileStorage.EXPECT().Download(gomock.Any(), command.Alias).
			Return(validStorageResult, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().DecrementDownloadsByAliasTx(gomock.Any(), mockTx, command.Alias).
			Return(int16(2), nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), command)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("success last download deletes file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{Alias: command.Alias}, nil)

		mockFileStorage.EXPECT().Download(gomock.Any(), command.Alias).
			Return(validStorageResult, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().DecrementDownloadsByAliasTx(gomock.Any(), mockTx, command.Alias).
			Return(int16(0), nil)

		mockFileRepo.EXPECT().DeleteFileTx(gomock.Any(), mockTx, command.Alias).
			Return(nil)

		mockFileStorage.EXPECT().Delete(gomock.Any(), command.Alias).
			Return(nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), command)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("success with password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{
				Alias:        command.Alias,
				PasswordHash: testutil.HashPassword(t, "correct-password"),
			}, nil)

		mockFileStorage.EXPECT().Download(gomock.Any(), command.Alias).
			Return(validStorageResult, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().DecrementDownloadsByAliasTx(gomock.Any(), mockTx, command.Alias).
			Return(int16(1), nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), commands.DownloadFile{
			Alias:    command.Alias,
			Password: "correct-password",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("invalid password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{
				Alias:        command.Alias,
				PasswordHash: testutil.HashPassword(t, "correct-password"),
			}, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), commands.DownloadFile{
			Alias:    command.Alias,
			Password: "wrong-password",
		})
		require.Nil(t, result)
		require.ErrorIs(t, err, domainErrors.ErrFilePasswordInvalid)
	})

	t.Run("file not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(nil, domainErrors.ErrFileNotFound)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), command)
		require.Nil(t, result)
		require.ErrorIs(t, err, domainErrors.ErrFileNotFound)
	})

	t.Run("storage download error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{Alias: command.Alias}, nil)

		mockFileStorage.EXPECT().Download(gomock.Any(), command.Alias).
			Return(nil, domainErrors.ErrFileNotFound)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), command)
		require.Nil(t, result)
		require.Error(t, err)
	})

	t.Run("decrement downloads error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{Alias: command.Alias}, nil)

		mockFileStorage.EXPECT().Download(gomock.Any(), command.Alias).
			Return(validStorageResult, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().DecrementDownloadsByAliasTx(gomock.Any(), mockTx, command.Alias).
			Return(int16(0), domainErrors.ErrFileNotFound)

		mockTx.EXPECT().Rollback().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(context.Background(), command)
		require.Nil(t, result)
		require.Error(t, err)
	})

	t.Run("context canceled on decrement", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		ctx, cancel := context.WithCancel(context.Background())

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), command.Alias).
			Return(&entities.File{Alias: command.Alias}, nil)

		mockFileStorage.EXPECT().Download(gomock.Any(), command.Alias).
			Return(validStorageResult, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().DecrementDownloadsByAliasTx(gomock.Any(), mockTx, command.Alias).
			DoAndReturn(func(_ context.Context, _ tx.Tx, _ string) (int16, error) {
				cancel()
				return 0, context.Canceled
			})

		mockTx.EXPECT().Rollback().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		result, err := fileService.DownloadFile(ctx, command)
		require.Nil(t, result)
		require.ErrorIs(t, err, context.Canceled)
	})
}
