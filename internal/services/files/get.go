package files

import (
	"context"
	"errors"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	domainErrors "expire-share/internal/domain/entities/errors"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"
	"time"
)

func (fs *Service) GetFileByAlias(ctx context.Context, command commands.GetFile) (*results.GetFile, error) {
	const fn = "services.file.Service.GetFileByAlias"
	log := fs.log.With(slog.String("fn", fn))

	fileInfo, err := fs.fileRepo.GetFileByAlias(ctx, command.Alias)
	if err != nil {
		const msg = "failed to get file by alias"
		if errors.Is(err, domainErrors.ErrFileNotFound) || isCtxError(err) {
			log.Info(msg, sl.Error(err), slog.String("alias", command.Alias))
			return nil, err
		}

		log.Error(msg, sl.Error(err), slog.String("alias", command.Alias))
		return nil, fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	err = fs.checkOwner(*fileInfo, command.UserID)
	if err != nil {
		log.Info("access denied", sl.Error(err), slog.String("alias", command.Alias))
		return nil, fmt.Errorf("%s: access denied: %w", fn, err)
	}

	return &results.GetFile{
		Alias:         fileInfo.Alias,
		Filename:      fileInfo.Name,
		Filesize:      fileInfo.Size,
		DownloadsLeft: fileInfo.DownloadsLeft,
		ExpiresIn:     time.Until(fileInfo.ExpiresAt),
	}, nil
}
