package viewmodels

import "fyne.io/fyne/v2/data/binding"

type VodafoneSIMStateViewModel struct {
	status binding.String
}

func NewVodafoneSIMStateViewModel() *VodafoneSIMStateViewModel {
	vm := &VodafoneSIMStateViewModel{
		status: binding.NewString(),
	}
	_ = vm.status.Set("Vodafone: перевірка за запитом")
	return vm
}

func (vm *VodafoneSIMStateViewModel) StatusBinding() binding.String {
	return vm.status
}

func (vm *VodafoneSIMStateViewModel) SetStatus(text string) {
	_ = vm.status.Set(text)
}
