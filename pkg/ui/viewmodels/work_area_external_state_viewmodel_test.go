package viewmodels

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestWorkAreaExternalStateViewModel_DefaultState(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaExternalStateViewModel()

	signal, _ := vm.SignalBinding().Get()
	lastTest, _ := vm.LastTestBinding().Get()
	lastTestTime, _ := vm.LastTestTimeBinding().Get()
	lastMessageTime, _ := vm.LastMessageTimeBinding().Get()

	if signal != "📶 Рівень: ..." {
		t.Fatalf("unexpected default signal: %q", signal)
	}
	if lastTest != "📝 Тест: ..." {
		t.Fatalf("unexpected default last test: %q", lastTest)
	}
	if lastTestTime != "📅 Ост. тест: ..." {
		t.Fatalf("unexpected default last test time: %q", lastTestTime)
	}
	if lastMessageTime != "📅 Ост. подія: ..." {
		t.Fatalf("unexpected default last message time: %q", lastMessageTime)
	}
}

func TestWorkAreaExternalStateViewModel_Apply(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaExternalStateViewModel()
	vm.Apply(WorkAreaExternalPresentation{
		SignalText:          "📶 Рівень: 85%",
		LastTestText:        "📝 Тест: OK",
		LastTestTimeText:    "📅 Ост. тест: 28.03.2026 10:00:00",
		LastMessageTimeText: "📅 Ост. подія: 28.03.2026 10:05:00",
	})

	signal, _ := vm.SignalBinding().Get()
	lastTest, _ := vm.LastTestBinding().Get()
	lastTestTime, _ := vm.LastTestTimeBinding().Get()
	lastMessageTime, _ := vm.LastMessageTimeBinding().Get()

	if signal != "📶 Рівень: 85%" {
		t.Fatalf("unexpected signal: %q", signal)
	}
	if lastTest != "📝 Тест: OK" {
		t.Fatalf("unexpected last test: %q", lastTest)
	}
	if lastTestTime != "📅 Ост. тест: 28.03.2026 10:00:00" {
		t.Fatalf("unexpected last test time: %q", lastTestTime)
	}
	if lastMessageTime != "📅 Ост. подія: 28.03.2026 10:05:00" {
		t.Fatalf("unexpected last message time: %q", lastMessageTime)
	}
}
