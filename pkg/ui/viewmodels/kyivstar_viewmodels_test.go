package viewmodels

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
	"testing"
	"time"
)

func TestKyivstarAuthViewModel_BuildStatusText(t *testing.T) {
	t.Parallel()

	vm := NewKyivstarAuthViewModel()
	got := vm.BuildStatusText(contracts.KyivstarAuthState{
		ClientID:       "client-1",
		Configured:     true,
		Authorized:     true,
		TokenExpiresAt: time.Now().Add(30 * time.Minute),
	})
	if !strings.Contains(got, "Kyivstar: токен активний") {
		t.Fatalf("unexpected status text: %q", got)
	}
}

func TestKyivstarSIMViewModel_BuildMetadata(t *testing.T) {
	t.Parallel()

	vm := NewKyivstarSIMViewModel()
	deviceName, deviceID, err := vm.BuildMetadata("380671234567", "1001", "Obj 1001", "")
	if err != nil {
		t.Fatalf("BuildMetadata() error = %v", err)
	}
	if deviceName != "Obj 1001" || deviceID != "1001" {
		t.Fatalf("BuildMetadata() = %q, %q", deviceName, deviceID)
	}
}

func TestKyivstarSIMViewModel_BuildStatusText(t *testing.T) {
	t.Parallel()

	vm := NewKyivstarSIMViewModel()
	got := vm.BuildStatusText(contracts.KyivstarSIMStatus{
		MSISDN:       "380671234567",
		Available:    true,
		NumberStatus: "ACTIVE",
		DeviceName:   "Obj 1001",
		DeviceID:     "1001",
		IsOnline:     true,
		Services: []contracts.KyivstarSIMServiceStatus{
			{ServiceID: "10", Name: "DATA", Status: "ACTIVE"},
		},
	})
	if !strings.Contains(got, "Kyivstar: 380671234567") {
		t.Fatalf("unexpected status text: %q", got)
	}
	if !strings.Contains(got, "DATA=ACTIVE") {
		t.Fatalf("expected services in status text: %q", got)
	}
}
