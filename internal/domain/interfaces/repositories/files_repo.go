package repositories

import (
	"context"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/entities"
	"expire-share/internal/domain/interfaces/tx"
)

type FileRepo interface {
	tx.Beginner

	GetFileByAlias(ctx context.Context, alias string) (*entities.File, error)
	GetFilesByUserID(ctx context.Context, userID int64) ([]entities.File, error)
	GetFilesStatByUserID(ctx context.Context, userID int64) (*entities.FilesStat, error)

	AddFileTx(ctx context.Context, tx tx.Tx, command commands.AddFile) (int64, error)
	DecrementDownloadsByAliasTx(ctx context.Context, tx tx.Tx, alias string) (int16, error)
	DeleteFileTx(ctx context.Context, tx tx.Tx, alias string) error
	DeleteExpiredFilesTx(ctx context.Context, tx tx.Tx, limit int) ([]string, error)
}
