package viewmodels

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestWorkAreaDeviceViewModel_BuildObjectPresentation(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()

	presentation := vm.BuildObjectPresentation(models.Object{
		DeviceType:    "Tiras",
		PanelMark:     "TM-1",
		PowerSource:   models.PowerBattery,
		SIM1:          "111",
		SIM2:          "222",
		AutoTestHours: 12,
		ObjChan:       5,
		AkbState:      1,
		TestControl:   1,
		TestTime:      9,
		Phones1:       "380001",
		Notes1:        "note",
		Location1:     "location",
		IsUnderGuard:  true,
		Groups: []models.ObjectGroup{
			{Number: 1, Name: "Room A", Armed: true, StateText: "ПІД ОХОРОНОЮ", RoomName: "Room A"},
			{Number: 2, Name: "Room B", Armed: false, StateText: "ЗНЯТО"},
		},
	})

	if presentation.DeviceTypeText != "🔧 Тип: Tiras" {
		t.Fatalf("unexpected device type text: %q", presentation.DeviceTypeText)
	}
	if presentation.SIMText != "📱 SIM1: 111 | SIM2: 222" {
		t.Fatalf("unexpected sim text: %q", presentation.SIMText)
	}
	if presentation.SIM1Text != "📱 SIM1: 111" || presentation.SIM2Text != "📱 SIM2: 222" {
		t.Fatalf("unexpected split sim texts: sim1=%q sim2=%q", presentation.SIM1Text, presentation.SIM2Text)
	}
	if presentation.SIM1Value != "111" || presentation.SIM2Value != "222" {
		t.Fatalf("unexpected split sim values: sim1=%q sim2=%q", presentation.SIM1Value, presentation.SIM2Value)
	}
	if presentation.SIMCopyText != "111 / 222" {
		t.Fatalf("unexpected sim copy text: %q", presentation.SIMCopyText)
	}
	if presentation.ChannelText != "📡 Канал: GPRS" {
		t.Fatalf("unexpected channel text: %q", presentation.ChannelText)
	}
	if presentation.AkbText != "🔋 АКБ: ТРИВОГА (Розряд/Відсутній)" {
		t.Fatalf("unexpected akb text: %q", presentation.AkbText)
	}
	if presentation.GuardText != "🔒 ПІД ОХОРОНОЮ" {
		t.Fatalf("unexpected guard text: %q", presentation.GuardText)
	}
	if presentation.GroupsText != "🔐 Групи:\nГрупа 1 | Room A | ПІД ОХОРОНОЮ\nГрупа 2 | Room B | ЗНЯТО" {
		t.Fatalf("unexpected groups text: %q", presentation.GroupsText)
	}
}

func TestWorkAreaDeviceViewModel_BuildObjectPresentation_NoGuard(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()

	presentation := vm.BuildObjectPresentation(models.Object{
		ObjChan:      1,
		IsUnderGuard: false,
	})

	if presentation.ChannelText != "📡 Канал: Автододзвон" {
		t.Fatalf("unexpected channel text: %q", presentation.ChannelText)
	}
	if presentation.GuardText != "🔓 ЗНЯТО З ОХОРОНИ" {
		t.Fatalf("unexpected guard text: %q", presentation.GuardText)
	}
}

func TestWorkAreaDeviceViewModel_BuildObjectPresentation_PhoenixBlocked(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()

	presentation := vm.BuildObjectPresentation(models.Object{
		ID:                phoenixObjectIDNamespaceStart + 77,
		BlockedArmedOnOff: 1,
		IsUnderGuard:      false,
		Groups: []models.ObjectGroup{
			{Number: 1, Name: "Група 1", Armed: false, StateText: "ЗАБЛОКОВАНО"},
		},
	})

	if presentation.GuardText != "⛔ ЗАБЛОКОВАНО" {
		t.Fatalf("unexpected phoenix guard text: %q", presentation.GuardText)
	}
	if presentation.GroupsText != "🔐 Групи:\nГрупа 1 | Група 1 | ЗАБЛОКОВАНО" {
		t.Fatalf("unexpected phoenix groups text: %q", presentation.GroupsText)
	}
}

func TestWorkAreaDeviceViewModel_BuildObjectPresentation_PhoenixDisarmed(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()

	presentation := vm.BuildObjectPresentation(models.Object{
		ID:           phoenixObjectIDNamespaceStart + 78,
		IsUnderGuard: false,
		Groups: []models.ObjectGroup{
			{Number: 1, Name: "Група 1", Armed: false, StateText: "БЕЗ ОХОРОНИ"},
		},
	})

	if presentation.GuardText != "🔓 БЕЗ ОХОРОНИ" {
		t.Fatalf("unexpected phoenix disarmed text: %q", presentation.GuardText)
	}
}

func TestWorkAreaDeviceViewModel_BuildObjectPresentation_CASLFallbacks(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()

	presentation := vm.BuildObjectPresentation(models.Object{
		ID:         caslObjectIDNamespaceStart + 24,
		DeviceType: "",
		ObjChan:    5,
		Phones1:    "",
	})

	if presentation.DeviceTypeText != "🔧 Тип: —" {
		t.Fatalf("unexpected device type text: %q", presentation.DeviceTypeText)
	}
	if presentation.SIMText != "📱 SIM1: —" {
		t.Fatalf("unexpected sim text: %q", presentation.SIMText)
	}
	if presentation.SIM1Text != "📱 SIM1: —" || presentation.SIM2Text != "📱 SIM2: —" {
		t.Fatalf("unexpected split sim defaults: sim1=%q sim2=%q", presentation.SIM1Text, presentation.SIM2Text)
	}
	if presentation.PhoneText != "☎️ Тел. об'єкта: —" {
		t.Fatalf("unexpected phone text: %q", presentation.PhoneText)
	}
	if presentation.ChannelText != "📡 Канал: GPRS" {
		t.Fatalf("unexpected channel text: %q", presentation.ChannelText)
	}
	if presentation.GroupsText != "🔐 Групи: —" {
		t.Fatalf("unexpected groups fallback: %q", presentation.GroupsText)
	}
}

func TestWorkAreaDeviceViewModel_BuildExternalPresentation(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()
	lastTest := time.Date(2026, 3, 28, 10, 0, 0, 0, time.Local)
	lastMsg := time.Date(2026, 3, 28, 11, 0, 0, 0, time.Local)

	presentation := vm.BuildExternalPresentation("85%", "Тест ОК", lastTest, lastMsg)
	if presentation.SignalText != "📶 Рівень: 85%" {
		t.Fatalf("unexpected signal text: %q", presentation.SignalText)
	}
	if presentation.LastTestText != "📝 Тест: Тест ОК" {
		t.Fatalf("unexpected test text: %q", presentation.LastTestText)
	}
	if presentation.LastTestTimeText != "📅 Ост. тест: 28.03.2026 10:00:00" {
		t.Fatalf("unexpected test time text: %q", presentation.LastTestTimeText)
	}
	if presentation.LastMessageTimeText != "📅 Ост. подія: 28.03.2026 11:00:00" {
		t.Fatalf("unexpected last message time text: %q", presentation.LastMessageTimeText)
	}
}

func TestWorkAreaDeviceViewModel_BuildExternalPresentation_ZeroTimes(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()
	presentation := vm.BuildExternalPresentation("—", "—", time.Time{}, time.Time{})

	if presentation.LastTestTimeText != "📅 Ост. тест: —" {
		t.Fatalf("unexpected zero test time text: %q", presentation.LastTestTimeText)
	}
	if presentation.LastMessageTimeText != "📅 Ост. подія: —" {
		t.Fatalf("unexpected zero message time text: %q", presentation.LastMessageTimeText)
	}
}

func TestWorkAreaDeviceViewModel_BuildLoadingExternalPresentation(t *testing.T) {
	vm := NewWorkAreaDeviceViewModel()
	presentation := vm.BuildLoadingExternalPresentation()

	if presentation.SignalText != "📶 Рівень: ..." {
		t.Fatalf("unexpected loading signal text: %q", presentation.SignalText)
	}
	if presentation.LastTestText != "📝 Тест: ..." {
		t.Fatalf("unexpected loading test text: %q", presentation.LastTestText)
	}
}
