package viewmodels

import "fyne.io/fyne/v2/data/binding"

// SIMUsageStateViewModel зберігає тексти підказок використання SIM.
type SIMUsageStateViewModel struct {
	sim1 binding.String
	sim2 binding.String
}

func NewSIMUsageStateViewModel() *SIMUsageStateViewModel {
	vm := &SIMUsageStateViewModel{
		sim1: binding.NewString(),
		sim2: binding.NewString(),
	}
	vm.Clear()
	return vm
}

func (vm *SIMUsageStateViewModel) SIM1Binding() binding.String { return vm.sim1 }
func (vm *SIMUsageStateViewModel) SIM2Binding() binding.String { return vm.sim2 }

func (vm *SIMUsageStateViewModel) SetSIM1(text string) {
	_ = vm.sim1.Set(text)
}

func (vm *SIMUsageStateViewModel) SetSIM2(text string) {
	_ = vm.sim2.Set(text)
}

func (vm *SIMUsageStateViewModel) Clear() {
	_ = vm.sim1.Set("")
	_ = vm.sim2.Set("")
}
