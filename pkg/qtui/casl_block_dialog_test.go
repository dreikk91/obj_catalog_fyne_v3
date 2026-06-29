//go:build qt

package qtui

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestBuildCASLDeviceBlockRequest(t *testing.T) {
	now := time.Date(2026, time.June, 28, 12, 0, 0, 0, time.UTC)
	device := contracts.CASLDeviceDetails{DeviceID: "42", Number: 1001}

	temporary, err := buildCASLDeviceBlockRequest(device, 1, 30, "Технічні роботи", false, now)
	if err != nil {
		t.Fatalf("temporary request: %v", err)
	}
	if temporary.TimeUnblock != now.Add(90*time.Minute).Unix() {
		t.Fatalf("temporary TimeUnblock = %d", temporary.TimeUnblock)
	}

	permanent, err := buildCASLDeviceBlockRequest(device, 0, 0, "Постійне блокування", true, now)
	if err != nil {
		t.Fatalf("permanent request: %v", err)
	}
	if permanent.TimeUnblock != caslPermanentBlockUnix {
		t.Fatalf("permanent TimeUnblock = %d", permanent.TimeUnblock)
	}
}

func TestBuildCASLDeviceBlockRequestRejectsInvalidDuration(t *testing.T) {
	device := contracts.CASLDeviceDetails{DeviceID: "42", Number: 1001}
	if _, err := buildCASLDeviceBlockRequest(device, 0, 0, "Причина", false, time.Now()); err == nil {
		t.Fatal("zero temporary duration must fail")
	}
	if _, err := buildCASLDeviceBlockRequest(device, 24, 1, "Причина", false, time.Now()); err == nil {
		t.Fatal("duration over 24 hours must fail")
	}
}
