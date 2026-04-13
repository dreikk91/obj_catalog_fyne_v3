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
	groups, _ := vm.GroupsBinding().Get()
	guard, _ := vm.GuardBinding().Get()
	connection, _ := vm.ConnectionBinding().Get()
	sim1, _ := vm.SIM1Binding().Get()
	sim2, _ := vm.SIM2Binding().Get()
	notes, _ := vm.NotesBinding().Get()

	if deviceType != "🔧 Тип: —" {
		t.Fatalf("unexpected default device type: %q", deviceType)
	}
	if guard != "—" {
		t.Fatalf("unexpected default guard: %q", guard)
	}
	if connection != "—" {
		t.Fatalf("unexpected default connection: %q", connection)
	}
	if groups != "🔐 Групи: —" {
		t.Fatalf("unexpected default groups: %q", groups)
	}
	if sim1 != "📱 SIM1: —" || sim2 != "📱 SIM2: —" {
		t.Fatalf("unexpected default sims: sim1=%q sim2=%q", sim1, sim2)
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
		DeviceTypeText:   "🔧 Тип: Tiras",
		PanelMarkText:    "🏷️ Марка: TM-1",
		GroupsText:       "🔐 Групи:\nГрупа 1: ПІД ОХОРОНОЮ",
		PowerText:        "🔌 🔋 АКБ (резерв)",
		SIMText:          "📱 SIM1: 111",
		SIM1Text:         "📱 SIM1: 111",
		SIM2Text:         "📱 SIM2: 222",
		AutoTestText:     "⏱️ Автотест: кожні 12 год",
		GuardText:        "Знято з охорони",
		ConnectionText:   "На зв'язку",
		SummaryPowerText: "220В в нормі",
		ChannelText:      "📡 Канал: GPRS",
		PhoneText:        "☎️ Тел. об'єкта: 380001",
		AkbText:          "🔋 АКБ: Норма",
		TestControlText:  "⏲️ Контроль тесту: Виключено",
		NotesText:        "note",
		LocationText:     "location",
	})

	gotGuard, _ := vm.GuardBinding().Get()
	if gotGuard != "Знято з охорони" {
		t.Fatalf("unexpected applied guard: %q", gotGuard)
	}
	gotConnection, _ := vm.ConnectionBinding().Get()
	if gotConnection != "На зв'язку" {
		t.Fatalf("unexpected applied connection: %q", gotConnection)
	}
	gotNotes, _ := vm.NotesBinding().Get()
	if gotNotes != "note" {
		t.Fatalf("unexpected applied notes: %q", gotNotes)
	}
	gotSIM2, _ := vm.SIM2Binding().Get()
	if gotSIM2 != "📱 SIM2: 222" {
		t.Fatalf("unexpected applied sim2: %q", gotSIM2)
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
