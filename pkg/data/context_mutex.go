package data

import (
	"context"
	"time"
)

type tryLocker interface {
	TryLock() bool
}

// lockMutexContext acquires mu without leaving callers queued after their context expires.
func lockMutexContext(ctx context.Context, mu tryLocker) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if mu.TryLock() {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Millisecond):
		}
	}
}
