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
)

func (fs *Service) GetFilesStat(ctx context.Context, command commands.GetFilesStat) (*results.GetFilesStat, error) {
	const fn = "services.file.Service.GetFilesStat"
	log := fs.log.With(slog.String("fn", fn))

	filesStat, err := fs.fileRepo.GetFilesStatByUserID(ctx, command.UserID)
	if err != nil {
		const msg = "failed to get file by alias"
		if errors.Is(err, domainErrors.ErrUserNotFound) || isCtxError(err) {
			log.Info(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
			return nil, err
		}

		log.Error(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
		return nil, fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	limits := fs.getUserLimits(command.Roles)

	return &results.GetFilesStat{
		Stat:     *filesStat,
		MaxCount: limits.MaxUploadedFiles,
		MaxSize:  limits.MaxSize,
	}, nil
}
