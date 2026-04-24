package util

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func IsCtxError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func TimeString(duration time.Duration) string {
	return fmt.Sprintf("%02dh%02dm%02ds",
		int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
}
