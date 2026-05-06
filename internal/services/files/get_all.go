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

func (fs *Service) GetAllFiles(ctx context.Context, command commands.GetAllFiles) ([]results.GetFile, error) {
	const fn = "services.files.Service.GetAllFiles"
	log := fs.log.With(slog.String("fn", fn))

	filesInfo, err := fs.fileRepo.GetFilesByUserID(ctx, command.UserID)
	if err != nil {
		const msg = "failed to get user files"
		if isCtxError(err) {
			log.Info(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
			return nil, err
		}

		log.Error(msg, sl.Error(err), slog.Int64("user_id", command.UserID))
		return nil, fmt.Errorf("%s: %s: %w", fn, msg, err)
	}

	result := make([]results.GetFile, 0, len(filesInfo))
	for _, fileInfo := range filesInfo {
		result = append(result, results.GetFile{
			Alias:         fileInfo.Alias,
			Filename:      fileInfo.Name,
			Filesize:      fileInfo.Size,
			DownloadsLeft: fileInfo.DownloadsLeft,
			ExpiresIn:     time.Until(fileInfo.ExpiresAt),
		})
	}

	return result, nil
}
