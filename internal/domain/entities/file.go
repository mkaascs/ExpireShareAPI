package entities

import "time"

type File struct {
	Filename      string
	Filesize      int64
	Alias         string
	DownloadsLeft int16
	PasswordHash  string
	LoadedAt      time.Time
	ExpiresAt     time.Time
	UserID        int64
}
