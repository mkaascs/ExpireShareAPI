package files

import (
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"

	"golang.org/x/crypto/bcrypt"
)

type userLimits struct {
	MaxUploadedFiles int
	MaxSize          int64
}

func (fs *Service) checkAccess(fileInfo entities.File, userID int64, roles []entities.UserRole) error {
	if hasRole(roles, entities.RoleAdmin) {
		return nil
	}

	return fs.checkOwner(fileInfo, userID)
}

func (fs *Service) checkUploadQuote(stat entities.FilesStat, filesize int64, roles []entities.UserRole) error {
	if hasRole(roles, entities.RoleAdmin) {
		return nil
	}

	if stat.Count >= fs.cfg.MaxUploadedFiles {
		return domainErrors.ErrUploadLimitExceeded
	}

	if hasRole(roles, entities.RoleVip) && stat.Size+filesize < fs.cfg.MaxFilesSizeForVipInBytes {
		return nil
	}

	if hasRole(roles, entities.RoleUser) && stat.Size+filesize < fs.cfg.MaxFilesSizeForUserInBytes {
		return nil
	}

	return domainErrors.ErrFileSizeTooBig
}

func (fs *Service) checkOwner(fileInfo entities.File, userID int64) error {
	if fileInfo.UserID != userID {
		return domainErrors.ErrForbidden
	}

	return nil
}

func (fs *Service) checkPassword(fileInfo entities.File, password string) error {
	if fileInfo.PasswordHash != "" && password == "" {
		return domainErrors.ErrFilePasswordRequired
	}

	err := bcrypt.CompareHashAndPassword([]byte(fileInfo.PasswordHash), []byte(password))
	if err != nil && fileInfo.PasswordHash != "" {
		return domainErrors.ErrFilePasswordInvalid
	}

	return nil
}

func (fs *Service) getUserLimits(roles []entities.UserRole) userLimits {
	if hasRole(roles, entities.RoleAdmin) || hasRole(roles, entities.RoleVip) {
		return userLimits{
			MaxUploadedFiles: fs.cfg.Permissions.MaxUploadedFiles,
			MaxSize:          fs.cfg.Permissions.MaxFilesSizeForVipInBytes,
		}
	}

	return userLimits{
		MaxUploadedFiles: fs.cfg.Permissions.MaxUploadedFiles,
		MaxSize:          fs.cfg.Permissions.MaxFilesSizeForUserInBytes,
	}
}

func hasRole(roles []entities.UserRole, role entities.UserRole) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}

	return false
}
