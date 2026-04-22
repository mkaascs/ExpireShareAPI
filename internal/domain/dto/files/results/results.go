package results

import (
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
	DownloadsLeft int16
	ExpiresIn     time.Duration
}
