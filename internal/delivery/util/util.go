package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

func IsCtxError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func TimeString(duration time.Duration) string {
	return fmt.Sprintf("%02dh%02dm%02ds",
		int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
}

func ExtractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return host
}

func ScanPaginationArgs(r *http.Request, page *int, limit *int) {
	var err error
	*page, err = strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || *page < 1 {
		*page = 1
	}

	*limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || *limit < 1 {
		*limit = 10
	} else if *limit > 100 {
		*limit = 100
	}
}
