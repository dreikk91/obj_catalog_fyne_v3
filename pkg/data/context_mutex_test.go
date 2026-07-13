package data

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLockMutexContextStopsWaitingAfterCancellation(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if lockMutexContext(ctx, &mu) {
		t.Fatal("lockMutexContext acquired a mutex that remained locked")
	}
}
