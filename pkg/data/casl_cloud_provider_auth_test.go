package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestLockCASLMutexWithContext_RespectsDeadline(t *testing.T) {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	start := time.Now()
	unlock, err := lockCASLMutexWithContext(ctx, &mu)
	if err == nil {
		if unlock != nil {
			unlock()
		}
		t.Fatal("expected deadline error while mutex is locked")
	}
	if time.Since(start) > 250*time.Millisecond {
		t.Fatalf("lockCASLMutexWithContext waited too long: %v", time.Since(start))
	}
}

func TestCASLProviderGetEvents_ReturnsCachedEventsWhenAuthLockBusy(t *testing.T) {
	provider := NewCASLCloudProvider("http://127.0.0.1:59999", "", 1, "operator@example.com", "secret")
	defer provider.Shutdown()

	expected := []models.Event{
		{
			ID:       1,
			ObjectID: 100,
			Details:  "cached event",
			Time:     time.Now(),
		},
	}

	provider.mu.Lock()
	provider.cachedEvents = append([]models.Event(nil), expected...)
	provider.mu.Unlock()

	provider.authMu.Lock()
	defer provider.authMu.Unlock()

	start := time.Now()
	got := provider.GetEvents()
	duration := time.Since(start)

	if duration > 3*time.Second {
		t.Fatalf("GetEvents blocked for too long: %v", duration)
	}
	if len(got) != 1 || got[0].Details != expected[0].Details {
		t.Fatalf("GetEvents() = %+v, want cached %+v", got, expected)
	}
}
