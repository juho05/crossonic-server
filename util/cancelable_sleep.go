package util

import (
	"context"
	"time"
)

// CancelableSleep waits until duration passes or ctx is canceled.
// Returns ctx.Err() if it is the reason for the return.
func CancelableSleep(ctx context.Context, duration time.Duration) error {
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
