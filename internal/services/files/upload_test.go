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
	"github.com/stretchr/testify/assert"
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
		Service: config.Service{
			AliasLength: 6,
			Permissions: config.Permissions{
				MaxFilesSizeForVipInBytes:  2 << 31,
				MaxFilesSizeForUserInBytes: 2 << 29,
				MaxUploadedFiles:           50,
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

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			DoAndReturn(func(_ context.Context, userID int64) (*entities.FilesStat, error) {
				assert.Equal(t, command.UserID, userID)
				return &entities.FilesStat{
					UserID: command.UserID,
					Size:   2 << 12,
					Count:  12,
				}, nil
			})

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ tx.Tx, cmd commands.AddFile) (*entities.File, error) {
				assert.Equal(t, command.Filename, cmd.Filename)
				assert.Equal(t, command.MaxDownloads, cmd.MaxDownloads)
				assert.Empty(t, cmd.PasswordHash)
				assert.NotEmpty(t, cmd.Alias)
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

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			DoAndReturn(func(_ context.Context, userID int64) (*entities.FilesStat, error) {
				assert.Equal(t, command.UserID, userID)
				return &entities.FilesStat{
					UserID: command.UserID,
					Size:   2 << 12,
					Count:  12,
				}, nil
			})

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().AddFileTx(gomock.Any(), mockTx, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ tx.Tx, cmd commands.AddFile) (*entities.File, error) {
				assert.NotEmpty(t, cmd.PasswordHash)
				assert.Contains(t, cmd.PasswordHash, "$2a$")
				return &entities.File{Alias: cmd.Alias}, nil
			})

		mockFileStorage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)

		commandWithPsw := command
		commandWithPsw.Password = "123456"

		alias, err := fileService.UploadFile(context.Background(), commandWithPsw)
		require.NoError(t, err)
		require.NotEmpty(t, alias)
	})

	t.Run("upload filesize limit exceeded for vip", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), gomock.Any()).
			Return(&entities.FilesStat{
				UserID: command.UserID,
				Size:   2 << 30,
			}, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)

		vipCommand := command
		vipCommand.Filesize = 2 << 31
		vipCommand.Roles = []entities.UserRole{entities.RoleVip}

		alias, err := fileService.UploadFile(context.Background(), vipCommand)

		require.Empty(t, alias)
		require.ErrorIs(t, err, domainErrors.ErrFileSizeTooBig)
	})

	t.Run("upload filesize limit exceeded for regular user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			Return(&entities.FilesStat{
				UserID: command.UserID,
				Size:   2 << 28,
			}, nil)

		userCommand := command
		userCommand.Filesize = 2 << 29
		userCommand.Roles = []entities.UserRole{entities.RoleUser}

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), userCommand)
		require.Empty(t, alias)
		require.ErrorIs(t, err, domainErrors.ErrFileSizeTooBig)
	})

	t.Run("upload limit exceeded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			Return(&entities.FilesStat{
				UserID: command.UserID,
				Size:   2 << 12,
				Count:  cfg.MaxUploadedFiles,
			}, nil)

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

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			Return(&entities.FilesStat{}, nil)

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

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			Return(&entities.FilesStat{}, nil)

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

	t.Run("internal error on get files stat", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFilesStatByUserID(gomock.Any(), command.UserID).
			Return(nil, errors.New("internal error"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		alias, err := fileService.UploadFile(context.Background(), command)
		require.Empty(t, alias)
		require.Error(t, err)
	})
}
