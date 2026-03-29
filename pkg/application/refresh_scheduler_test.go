package application

import "testing"

func TestShouldRefreshForLatestEventID_InitialSeed(t *testing.T) {
	refresh, nextID, hasNext := shouldRefreshForLatestEventID(100, 0, false)
	if refresh {
		t.Fatalf("expected no refresh on initial seed")
	}
	if !hasNext || nextID != 100 {
		t.Fatalf("unexpected seed state: hasNext=%v nextID=%d", hasNext, nextID)
	}
}

func TestShouldRefreshForLatestEventID_NoChange(t *testing.T) {
	refresh, nextID, hasNext := shouldRefreshForLatestEventID(100, 100, true)
	if refresh {
		t.Fatalf("expected no refresh when IDs are equal")
	}
	if !hasNext || nextID != 100 {
		t.Fatalf("unexpected state: hasNext=%v nextID=%d", hasNext, nextID)
	}
}

func TestShouldRefreshForLatestEventID_NewEvent(t *testing.T) {
	refresh, nextID, hasNext := shouldRefreshForLatestEventID(101, 100, true)
	if !refresh {
		t.Fatalf("expected refresh when latest ID increases")
	}
	if !hasNext || nextID != 101 {
		t.Fatalf("unexpected state: hasNext=%v nextID=%d", hasNext, nextID)
	}
}

func TestShouldRefreshForLatestEventID_ResetOrReconnect(t *testing.T) {
	refresh, nextID, hasNext := shouldRefreshForLatestEventID(7, 100, true)
	if !refresh {
		t.Fatalf("expected refresh when latest ID decreases (reconnect/reset)")
	}
	if !hasNext || nextID != 7 {
		t.Fatalf("unexpected state: hasNext=%v nextID=%d", hasNext, nextID)
	}
}
