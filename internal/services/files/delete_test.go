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
	"testing"
)

func TestService_DeleteFile(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{}

	command := commands.DeleteFile{
		Alias: "file-alias",
		RequestingUserInfo: commands.RequestingUserInfo{
			UserID: int64(1),
			Roles:  []entities.UserRole{entities.RoleUser},
		},
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, alias string) (*entities.File, error) {
				require.Equal(t, command.Alias, alias)
				return &entities.File{
					Alias:        command.Alias,
					PasswordHash: "",
					UserID:       command.UserID,
				}, nil
			})

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockFileRepo.EXPECT().DeleteFileTx(gomock.Any(), mockTx, gomock.Any()).
			DoAndReturn(func(_ context.Context, tx tx.Tx, alias string) error {
				require.Equal(t, command.Alias, alias)
				return nil
			})

		mockFileStorage.EXPECT().Delete(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, alias string) error {
				require.Equal(t, command.Alias, alias)
				return nil
			})

		mockTx.EXPECT().Commit().Return(nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		err := fileService.DeleteFile(context.Background(), command)
		require.NoError(t, err)
	})

	t.Run("delete another user file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(&entities.File{
				Alias:        command.Alias,
				PasswordHash: "",
				UserID:       int64(2),
			}, nil)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		err := fileService.DeleteFile(context.Background(), command)
		require.ErrorIs(t, err, domainErrors.ErrForbidden)
	})

	t.Run("file not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(nil, domainErrors.ErrFileNotFound)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		err := fileService.DeleteFile(context.Background(), command)
		require.ErrorIs(t, err, domainErrors.ErrFileNotFound)
	})

	t.Run("internal storage error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(&entities.File{
				Alias:  command.Alias,
				UserID: command.UserID,
			}, nil)

		mockFileRepo.EXPECT().DeleteFileTx(gomock.Any(), mockTx, gomock.Any()).
			Return(nil)

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockTx.EXPECT().Rollback().Return(nil)

		mockFileStorage.EXPECT().Delete(gomock.Any(), gomock.Any()).
			Return(errors.New("internal error"))

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		err := fileService.DeleteFile(context.Background(), command)
		require.Error(t, err)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := mocks.NewMockTx(ctrl)
		mockFileRepo := mocks.NewMockFileRepo(ctrl)
		mockFileStorage := mocks.NewMockFile(ctrl)

		ctx, cancel := context.WithCancel(context.Background())

		mockFileRepo.EXPECT().GetFileByAlias(gomock.Any(), gomock.Any()).
			Return(&entities.File{
				Alias:  command.Alias,
				UserID: command.UserID,
			}, nil)

		mockFileRepo.EXPECT().DeleteFileTx(ctx, mockTx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, tx tx.Tx, alias string) error {
				cancel()
				return nil
			})

		mockFileRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockTx.EXPECT().Rollback().Return(nil)

		mockFileStorage.EXPECT().Delete(gomock.Any(), gomock.Any()).
			Return(context.Canceled)

		fileService := New(mockFileRepo, mockFileStorage, log, cfg)
		err := fileService.DeleteFile(ctx, command)
		require.ErrorIs(t, err, context.Canceled)
	})
}
