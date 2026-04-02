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
