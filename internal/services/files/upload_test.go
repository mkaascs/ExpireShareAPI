package files

import (
	"context"
	"errors"
	"expire-share/internal/config"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/domain/interfaces/tx"
	"expire-share/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestService_UploadFile(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		Storage: config.Storage{
			MaxFileSizeInBytes: 10 * 1024 * 1024,
		},

		Service: config.Service{
			AliasLength: 6,
			Permissions: config.Permissions{
				MaxUploadedFileForUser: 1,
				MaxUploadedFileForVip:  10,
			},
		},
	}

	command := commands.UploadFile{
		File:         io.NopCloser(strings.NewReader("file content")),
		Filesize:     12,
		Filename:     "test.txt",
		Password:     "",
		MaxDownloads: 5,
		TTL:          2 * time.Hour,
		RequestingUserInfo: commands.RequestingUserInfo{
			UserID: int64(1),
			Roles:  []entities.UserRole{entities.RoleUser},
		},
	}

	t.Run("success without password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), command.UserID).
			Return(0, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ tx.Tx, cmd commands.AddFile) (*entities.File, error) {
				require.Equal(t, command.Filename, cmd.Filename)
				require.Equal(t, command.MaxDownloads, cmd.MaxDownloads)
				require.Empty(t, cmd.PasswordHash)
				require.NotEmpty(t, cmd.Alias)
				return &entities.File{Alias: cmd.Alias}, nil
			})

		mockFileStorage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), command.Filename).
			Return(nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), command)
		require.NoError(t, err)
		require.NotEmpty(t, alias)
	})

	t.Run("success with password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), command.UserID).
			Return(0, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ tx.Tx, cmd commands.AddFile) (*entities.File, error) {
				require.NotEmpty(t, cmd.PasswordHash)
				require.Contains(t, cmd.PasswordHash, "$2a$")
				return &entities.File{Alias: cmd.Alias}, nil
			})

		mockFileStorage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), commands.UploadFile{
			File:         io.NopCloser(strings.NewReader("content")),
			Filename:     "secret.txt",
			Password:     "file-password",
			MaxDownloads: 1,
			TTL:          time.Hour,
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: int64(1),
				Roles:  []entities.UserRole{entities.RoleUser},
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, alias)
	})

	t.Run("success vip user exceeds regular limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), gomock.Any()).
			Return(5, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			Return(int64(1), nil)

		mockFileStorage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), commands.UploadFile{
			File:         io.NopCloser(strings.NewReader("content")),
			Filename:     "file.txt",
			MaxDownloads: 1,
			TTL:          time.Hour,
			RequestingUserInfo: commands.RequestingUserInfo{
				UserID: int64(1),
				Roles:  []entities.UserRole{entities.RoleVip},
			},
		})

		require.NoError(t, err)
		require.NotEmpty(t, alias)
	})

	t.Run("upload limit exceeded for regular user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), command.UserID).
			Return(1, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), command)
		require.Empty(t, alias)
		require.ErrorIs(t, err, domainErrors.ErrUploadLimitExceeded)
	})

	t.Run("storage upload error rollback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), command.UserID).
			Return(0, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			Return(int64(1), nil)

		mockFileStorage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("internal error"))

		mockTx.EXPECT().Rollback().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), command)
		require.Empty(t, alias)
		require.Error(t, err)
	})

	t.Run("context canceled on add file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		ctx, cancel := context.WithCancel(context.Background())

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), command.UserID).
			Return(0, nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ tx.Tx, _ commands.AddFile) (*entities.File, error) {
				cancel()
				return nil, context.Canceled
			})

		mockTx.EXPECT().Rollback().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(ctx, command)
		require.Empty(t, alias)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("internal error on count", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().CountByUserID(gomock.Any(), command.UserID).
			Return(0, errors.New("internal error"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), command)
		require.Empty(t, alias)
		require.Error(t, err)
	})
}
