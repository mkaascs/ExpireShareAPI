package files

import (
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"

	"golang.org/x/crypto/bcrypt"
)

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

	if hasRole(roles, entities.RoleVip) && stat.Size+filesize > fs.cfg.MaxFilesSizeForVipInBytes {
		return domainErrors.ErrFileSizeTooBig
	}

	if hasRole(roles, entities.RoleUser) && stat.Size+filesize > fs.cfg.MaxFilesSizeForUserInBytes {
		return domainErrors.ErrFileSizeTooBig
	}

	if stat.Count >= fs.cfg.MaxUploadedFiles {
		return domainErrors.ErrUploadLimitExceeded
	}

	return domainErrors.ErrUploadLimitExceeded
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

func hasRole(roles []entities.UserRole, role entities.UserRole) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}

	return false
}
