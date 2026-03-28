package viewmodels

import "fyne.io/fyne/v2/data/binding"

// WorkAreaExternalStateViewModel зберігає динамічні зовнішні поля вкладки "Стан" через binding.
type WorkAreaExternalStateViewModel struct {
	signal          binding.String
	lastTest        binding.String
	lastTestTime    binding.String
	lastMessageTime binding.String
}

func NewWorkAreaExternalStateViewModel() *WorkAreaExternalStateViewModel {
	vm := &WorkAreaExternalStateViewModel{
		signal:          binding.NewString(),
		lastTest:        binding.NewString(),
		lastTestTime:    binding.NewString(),
		lastMessageTime: binding.NewString(),
	}
	vm.Apply(WorkAreaExternalPresentation{
		SignalText:          "📶 Рівень: ...",
		LastTestText:        "📝 Тест: ...",
		LastTestTimeText:    "📅 Ост. тест: ...",
		LastMessageTimeText: "📅 Ост. подія: ...",
	})
	return vm
}

func (vm *WorkAreaExternalStateViewModel) SignalBinding() binding.String {
	return vm.signal
}

func (vm *WorkAreaExternalStateViewModel) LastTestBinding() binding.String {
	return vm.lastTest
}

func (vm *WorkAreaExternalStateViewModel) LastTestTimeBinding() binding.String {
	return vm.lastTestTime
}

func (vm *WorkAreaExternalStateViewModel) LastMessageTimeBinding() binding.String {
	return vm.lastMessageTime
}

func (vm *WorkAreaExternalStateViewModel) Apply(presentation WorkAreaExternalPresentation) {
	_ = vm.signal.Set(presentation.SignalText)
	_ = vm.lastTest.Set(presentation.LastTestText)
	_ = vm.lastTestTime.Set(presentation.LastTestTimeText)
	_ = vm.lastMessageTime.Set(presentation.LastMessageTimeText)
}
