package viewmodels

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestWorkAreaDeviceStateViewModel_DefaultState(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaDeviceStateViewModel()

	deviceType, _ := vm.DeviceTypeBinding().Get()
	guard, _ := vm.GuardBinding().Get()
	notes, _ := vm.NotesBinding().Get()

	if deviceType != "🔧 Тип: —" {
		t.Fatalf("unexpected default device type: %q", deviceType)
	}
	if guard != "🔒 Стан: —" {
		t.Fatalf("unexpected default guard: %q", guard)
	}
	if notes != "" {
		t.Fatalf("unexpected default notes: %q", notes)
	}
}

func TestWorkAreaDeviceStateViewModel_ApplyAndReset(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaDeviceStateViewModel()
	vm.Apply(WorkAreaDevicePresentation{
		DeviceTypeText:  "🔧 Тип: Tiras",
		PanelMarkText:   "🏷️ Марка: TM-1",
		PowerText:       "🔌 🔋 АКБ (резерв)",
		SIMText:         "📱 SIM1: 111",
		AutoTestText:    "⏱️ Автотест: кожні 12 год",
		GuardText:       "🔓 ЗНЯТО З ОХОРОНИ",
		ChannelText:     "📡 Канал: GPRS",
		PhoneText:       "☎️ Тел. об'єкта: 380001",
		AkbText:         "🔋 АКБ: Норма",
		TestControlText: "⏲️ Контроль тесту: Виключено",
		NotesText:       "note",
		LocationText:    "location",
	})

	gotGuard, _ := vm.GuardBinding().Get()
	if gotGuard != "🔓 ЗНЯТО З ОХОРОНИ" {
		t.Fatalf("unexpected applied guard: %q", gotGuard)
	}
	gotNotes, _ := vm.NotesBinding().Get()
	if gotNotes != "note" {
		t.Fatalf("unexpected applied notes: %q", gotNotes)
	}

	vm.Reset()
	gotDeviceType, _ := vm.DeviceTypeBinding().Get()
	gotLocation, _ := vm.LocationBinding().Get()
	if gotDeviceType != "🔧 Тип: —" {
		t.Fatalf("unexpected reset device type: %q", gotDeviceType)
	}
	if gotLocation != "" {
		t.Fatalf("unexpected reset location: %q", gotLocation)
	}
}
