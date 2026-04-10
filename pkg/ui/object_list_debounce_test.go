package ui

import (
	"testing"
	"time"
)

func TestObjectListPanelScheduleFilterApplyDebouncesSearch(t *testing.T) {
	t.Parallel()

	panel := &ObjectListPanel{
		searchDebounceDelay: 20 * time.Millisecond,
	}

	calls := make(chan uint64, 4)
	panel.runFilterRequestFn = func(version uint64) {
		calls <- version
	}

	panel.scheduleFilterApply(panel.searchDebounceDelay)
	panel.scheduleFilterApply(panel.searchDebounceDelay)
	panel.scheduleFilterApply(panel.searchDebounceDelay)

	select {
	case version := <-calls:
		if version != 3 {
			t.Fatalf("debounced version = %d, want 3", version)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected debounced filter request")
	}

	select {
	case version := <-calls:
		t.Fatalf("unexpected extra filter request: %d", version)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestObjectListPanelScheduleFilterApplyImmediateCancelsPendingDebounce(t *testing.T) {
	t.Parallel()

	panel := &ObjectListPanel{
		searchDebounceDelay: 40 * time.Millisecond,
	}

	calls := make(chan uint64, 4)
	panel.runFilterRequestFn = func(version uint64) {
		calls <- version
	}

	panel.scheduleFilterApply(panel.searchDebounceDelay)
	panel.scheduleFilterApply(0)

	select {
	case version := <-calls:
		if version != 2 {
			t.Fatalf("immediate version = %d, want 2", version)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected immediate filter request")
	}

	select {
	case version := <-calls:
		t.Fatalf("unexpected stale debounced request: %d", version)
	case <-time.After(100 * time.Millisecond):
	}
}
