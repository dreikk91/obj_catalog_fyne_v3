package viewmodels

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
	"testing"
	"time"
)

func TestVodafoneAuthViewModel_BuildStatusText(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneAuthViewModel()
	got := vm.BuildStatusText(contracts.VodafoneAuthState{
		Phone:          "380501234567",
		Authorized:     true,
		TokenExpiresAt: time.Date(2026, time.April, 2, 13, 0, 0, 0, time.UTC),
	})
	if !strings.Contains(got, "380501234567") {
		t.Fatalf("status must contain phone, got %q", got)
	}
}

func TestVodafoneSIMViewModel_BuildMetadata(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneSIMViewModel()
	name, comment, err := vm.BuildMetadata("380501234567", "1001", "Ломбард", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "1001" || comment != "Ломбард" {
		t.Fatalf("unexpected metadata: name=%q comment=%q", name, comment)
	}
}

func TestVodafoneSIMViewModel_BuildBlockingMetadata(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneSIMViewModel()
	now := time.Date(2026, time.April, 3, 9, 30, 0, 0, time.UTC)

	name, comment, err := vm.BuildBlockingMetadata("1001", "Нема угоди", "", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "1001" {
		t.Fatalf("unexpected name: %q", name)
	}
	if comment != "Нема угоди (03.04.2026)" {
		t.Fatalf("unexpected comment: %q", comment)
	}
}

func TestVodafoneSIMViewModel_BuildBlockingMetadata_ManualReason(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneSIMViewModel()
	now := time.Date(2026, time.April, 3, 9, 30, 0, 0, time.UTC)

	_, comment, err := vm.BuildBlockingMetadata("1001", "Інша причина", "Потрібна перевидача", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment != "Потрібна перевидача (03.04.2026)" {
		t.Fatalf("unexpected comment: %q", comment)
	}
}

func TestVodafoneSIMViewModel_BuildStatusText_IncludesBlockingStatus(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneSIMViewModel()
	got := vm.BuildStatusText(contracts.VodafoneSIMStatus{
		MSISDN:         "380501234567",
		Available:      true,
		SubscriberName: "Obj 1001",
		Blocking: contracts.VodafoneSIMBlockingStatus{
			Status:          "FullBlocked",
			BlockingDateRaw: "2026-04-03T09:00:00Z",
		},
		Connectivity: contracts.VodafoneConnectivityStatus{
			SIMStatus: "active",
		},
	})

	if !strings.Contains(got, "блокування повне") {
		t.Fatalf("status must contain blocking info, got %q", got)
	}
}

func TestVodafoneSIMViewModel_BuildOverviewText(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneSIMViewModel()
	got := vm.BuildOverviewText(contracts.VodafoneSIMStatus{
		MSISDN:    "380501234567",
		Available: true,
		Connectivity: contracts.VodafoneConnectivityStatus{
			SIMStatus: "active",
		},
		LastEvent: contracts.VodafoneLastEvent{
			CallType:     "DATA",
			EventTimeRaw: "2026-04-03 09:00",
		},
	})

	if !strings.Contains(got, "380501234567") {
		t.Fatalf("overview must contain msisdn, got %q", got)
	}
	if !strings.Contains(got, "SIM active") {
		t.Fatalf("overview must contain sim status, got %q", got)
	}
}

func TestVodafoneSIMViewModel_BuildBlockingText(t *testing.T) {
	t.Parallel()

	vm := NewVodafoneSIMViewModel()
	got := vm.BuildBlockingText(contracts.VodafoneSIMStatus{
		Available: true,
		Blocking: contracts.VodafoneSIMBlockingStatus{
			Status:                 "FullBlocked",
			BlockingDateRaw:        "2026-04-03",
			BlockingRequestDateRaw: "2026-04-02",
		},
	})

	if !strings.Contains(got, "Стан блокування: повне") {
		t.Fatalf("blocking text must contain humanized state, got %q", got)
	}
	if !strings.Contains(got, "Дата блокування: 2026-04-03") {
		t.Fatalf("blocking text must contain date, got %q", got)
	}
}
