package files

import (
	"context"
	"expire-share/internal/domain/dto/files/commands"
	"expire-share/internal/domain/dto/files/results"
	"expire-share/internal/lib/log/sl"
	"fmt"
	"log/slog"
	"time"
)

func (fs *Service) GetAllFiles(ctx context.Context, command commands.GetAllFiles) (*results.GetAllFiles, error) {
	const fn = "services.files.Service.GetFilesByUserID"
	log := fs.log.With(slog.String("fn", fn))

	filesInfo, total, err := fs.fileRepo.GetFilesByUserID(ctx, command)
	if err != nil {
		const msg = "failed to get user files"
		if isCtxError(err) {
			log.Info(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
			return nil, err
		}

		log.Error(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
		return nil, fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	result := &results.GetAllFiles{
		Total: total,
		Files: make([]results.GetFile, 0, len(filesInfo)),
	}

	for _, file := range filesInfo {
		result.Files = append(result.Files, results.GetFile{
			Alias:         file.Alias,
			Filename:      file.Name,
			Filesize:      file.Size,
			DownloadsLeft: file.DownloadsLeft,
			ExpiresIn:     time.Until(file.ExpiresAt),
		})
	}

	return result, nil
}
