package commands

import (
	"expire-share/internal/domain/entities"
	"io"
	"time"
)

type RequestingUserInfo struct {
	UserID int64
	Roles  []entities.UserRole
}

type UploadFile struct {
	File         io.Reader
	Filesize     int64
	Filename     string
	MaxDownloads int16
	Password     string
	TTL          time.Duration
	RequestingUserInfo
}

type DownloadFile struct {
	Alias    string
	Password string
}

type GetFile struct {
	Alias string
	RequestingUserInfo
}

type GetAllFiles struct {
	RequestingUserInfo
}

type DeleteFile struct {
	Alias string
	RequestingUserInfo
}

type AddFile struct {
	Filename     string
	Filesize     int64
	Alias        string
	MaxDownloads int16
	PasswordHash string
	TTL          time.Duration
	UserID       int64
}
