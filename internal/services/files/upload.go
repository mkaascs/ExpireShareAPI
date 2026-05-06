package files

import (
	"context"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/lib/alias"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

func (fs *Service) UploadFile(ctx context.Context, command commands.UploadFile) (string, error) {
	const fn = "services.file.Service.UploadFile"
	log := fs.log.With(slog.String("fn", fn))

	filesStat, err := fs.fileRepo.GetFilesStatByUserID(ctx, command.UserID)
	if err != nil {
		const msg = "failed to get files stat by user id"
		if isCtxError(err) {
			log.Info(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
			return "", err
		}

		log.Info(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
		return "", fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	err = fs.checkUploadQuote(*filesStat, command.Filesize, command.Roles)
	if err != nil {
		log.Info("access denied", sl.Error(err), slog.Int64("user_id", command.UserID))
		return "", fmt.Errorf("%s: failed to upload quote: %w", fn, err)
	}

	genAlias := alias.Gen(fs.cfg.AliasLength)

	var hashedBytes []byte
	if len(command.Password) > 0 {
		hashedBytes, err = bcrypt.GenerateFromPassword([]byte(command.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Error("failed to hash password", sl.Error(err))
			return "", fmt.Errorf("%s: failed to hash password: %w", fn, err)
		}
	}

	tx, err := fs.fileRepo.BeginTx(ctx)
	if err != nil {
		log.Error("failed to begin tx", sl.Error(err))
		return "", fmt.Errorf("%s: failed to upload file: %w", fn, err)
	}

	success := false
	defer func() {
		if !success {
			if err := tx.Rollback(); err != nil {
				log.Error("failed to rollback tx", sl.Error(err))
			}
		}
	}()

	_, err = fs.fileRepo.AddFileTx(ctx, tx, commands.AddFile{
		Filename:     command.Filename,
		Alias:        genAlias,
		MaxDownloads: command.MaxDownloads,
		TTL:          command.TTL,
		PasswordHash: string(hashedBytes),
		UserID:       command.UserID,
	})

	if err != nil {
		const msg = "failed to add file info"
		if isCtxError(err) {
			log.Info(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
			return "", err
		}

		log.Error(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
		return "", fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	if err := fs.fileStorage.Upload(ctx, command.File, genAlias, command.Filename); err != nil {
		const msg = "failed to upload file"
		if isCtxError(err) {
			log.Info(msg, sl.Error(err))
			return "", err
		}

		log.Error(msg, sl.Error(err))
		return "", fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit tx", sl.Error(err))
		return "", fmt.Errorf("%s: failed to upload file to storage: %w", fn, err)
	}

	success = true
	return genAlias, nil
}
