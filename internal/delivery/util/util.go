package util

import (
	"context"
	"errors"
	"fmt"
	"net"
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
