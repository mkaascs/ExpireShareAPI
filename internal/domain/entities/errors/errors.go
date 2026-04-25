package errors

import "errors"

var (
	ErrAliasTaken          = errors.New("current alias is already taken")
	ErrFileNotFound        = errors.New("file does not exist")
	ErrNoDownloadsLeft     = errors.New("there is no downloads left")
	ErrFileSizeTooBig      = errors.New("file size too big")
	ErrUploadLimitExceeded = errors.New("upload limit exceeded")

	ErrForbidden           = errors.New("forbidden")
	ErrUserNotFound        = errors.New("user not found")
	ErrRoleNotExist        = errors.New("role does not exist")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrAccessTokenExpired  = errors.New("access token expired")
	ErrAccessTokenRevoked  = errors.New("access token revoked")
	ErrInvalidAccessToken  = errors.New("invalid access token")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidCredentials  = errors.New("invalid login or password")
	ErrInvalidArgument     = errors.New("invalid argument")

	ErrFilePasswordRequired = errors.New("file password required for access")
	ErrFilePasswordInvalid  = errors.New("invalid file password")
)
