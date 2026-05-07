package results

import (
	"expire-share/internal/domain/entities"
	"io"
	"os"
	"time"
)

type DownloadFile struct {
	File     io.Reader
	FileInfo os.FileInfo
	Close    func() error
}

type GetFile struct {
	Alias         string
	Filename      string
	Filesize      int64
	DownloadsLeft int16
	ExpiresIn     time.Duration
}

type GetAllFiles struct {
	Total int
	Files []GetFile
}

type GetFilesStat struct {
	Stat     entities.FilesStat
	MaxSize  int64
	MaxCount int
}
