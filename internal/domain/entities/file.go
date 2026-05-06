package entities

import "time"

type File struct {
	Name          string
	Size          int64
	Alias         string
	DownloadsLeft int16
	PasswordHash  string
	LoadedAt      time.Time
	ExpiresAt     time.Time
	UserID        int64
}

type FilesStat struct {
	UserID int64
	Count  int
	Size   int64
}
