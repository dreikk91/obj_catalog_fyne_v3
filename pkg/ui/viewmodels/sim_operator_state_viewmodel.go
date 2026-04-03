package viewmodels

import "fyne.io/fyne/v2/data/binding"

type SIMOperatorStateViewModel struct {
	status binding.String
}

func NewSIMOperatorStateViewModel(initialText string) *SIMOperatorStateViewModel {
	vm := &SIMOperatorStateViewModel{
		status: binding.NewString(),
	}
	_ = vm.status.Set(initialText)
	return vm
}

func (vm *SIMOperatorStateViewModel) StatusBinding() binding.String {
	return vm.status
}

func (vm *SIMOperatorStateViewModel) SetStatus(text string) {
	_ = vm.status.Set(text)
}
