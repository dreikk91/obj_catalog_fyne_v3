package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
)

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

func TestShouldAutoResetSIM1ByOperator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sim1         string
		wantVodafone bool
		wantKyivstar bool
	}{
		{name: "vodafone local", sim1: "0501234567", wantVodafone: true},
		{name: "vodafone international", sim1: "+380991234567", wantVodafone: true},
		{name: "kyivstar local", sim1: "0671234567", wantKyivstar: true},
		{name: "kyivstar international", sim1: "+380971234567", wantKyivstar: true},
		{name: "empty", sim1: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldAutoResetVodafoneSIM1(tt.sim1); got != tt.wantVodafone {
				t.Fatalf("shouldAutoResetVodafoneSIM1(%q) = %v, want %v", tt.sim1, got, tt.wantVodafone)
			}
			if got := shouldAutoResetKyivstarSIM1(tt.sim1); got != tt.wantKyivstar {
				t.Fatalf("shouldAutoResetKyivstarSIM1(%q) = %v, want %v", tt.sim1, got, tt.wantKyivstar)
			}
		})
	}
}

func TestSIMAutoResetJournalAppendf(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "vodafone_auto_reset.log")
	journal := newSimAutoResetJournal(path)

	journal.Appendf("об'єкт %d reset sim результат: orderID=%s state=%s", 1001, "42", "accepted")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "об'єкт 1001 reset sim результат: orderID=42 state=accepted") {
		t.Fatalf("journal = %q, want reset result line", got)
	}
}

func TestSIMAutoResetThrottleWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	window := 24 * time.Hour
	history := simAutoResetHistory{
		simAutoResetThrottleKey(1001, "0501234567"): {
			now.Add(-23 * time.Hour).Format(time.RFC3339Nano),
			now.Add(-1 * time.Hour).Format(time.RFC3339Nano),
			now.Add(-25 * time.Hour).Format(time.RFC3339Nano),
		},
	}

	attempts := pruneSimAutoResetAttempts(history[simAutoResetThrottleKey(1001, "0501234567")], window, now)
	if len(attempts) != 2 {
		t.Fatalf("pruned attempts = %d, want 2", len(attempts))
	}
	nextAllowed := nextSimAutoResetAllowedAt(attempts, window)
	want := now.Add(time.Hour)
	if !nextAllowed.Equal(want) {
		t.Fatalf("next allowed = %s, want %s", nextAllowed, want)
	}
}

func TestSIMAutoResetStatisticsFilter(t *testing.T) {
	t.Parallel()

	filter := simAutoResetStatisticsFilter()
	if filter.ConnectionMode != contracts.StatsConnectionOffline {
		t.Fatalf("ConnectionMode = %v, want offline", filter.ConnectionMode)
	}
	if filter.ProtocolFilter != contracts.StatsProtocolMost {
		t.Fatalf("ProtocolFilter = %q, want most", filter.ProtocolFilter)
	}
	if filter.ChannelCode == nil || *filter.ChannelCode != 5 {
		t.Fatalf("ChannelCode = %v, want 5", filter.ChannelCode)
	}
	if filter.GuardState == nil || *filter.GuardState != 1 {
		t.Fatalf("GuardState = %v, want 1", filter.GuardState)
	}
	if filter.BlockMode == nil || *filter.BlockMode != contracts.DisplayBlockNone {
		t.Fatalf("BlockMode = %v, want none", filter.BlockMode)
	}
}
