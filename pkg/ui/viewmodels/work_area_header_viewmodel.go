package viewmodels

import (
	"fmt"

	"fyne.io/fyne/v2/data/binding"

	"obj_catalog_fyne_v3/pkg/models"
)

// WorkAreaHeaderViewModel керує текстом шапки правої панелі через Fyne Data Binding.
type WorkAreaHeaderViewModel struct {
	headerName    binding.String
	headerAddress binding.String
}

func NewWorkAreaHeaderViewModel() *WorkAreaHeaderViewModel {
	vm := &WorkAreaHeaderViewModel{
		headerName:    binding.NewString(),
		headerAddress: binding.NewString(),
	}
	vm.Reset()
	return vm
}

func (vm *WorkAreaHeaderViewModel) HeaderNameBinding() binding.String {
	return vm.headerName
}

func (vm *WorkAreaHeaderViewModel) HeaderAddressBinding() binding.String {
	return vm.headerAddress
}

func (vm *WorkAreaHeaderViewModel) Reset() {
	_ = vm.headerName.Set("← Оберіть об'єкт зі списку")
	_ = vm.headerAddress.Set("")
}

func (vm *WorkAreaHeaderViewModel) ApplyObject(object models.Object) {
	_ = vm.headerName.Set(fmt.Sprintf("%s (№%d)", object.Name, object.ID))
	_ = vm.headerAddress.Set(fmt.Sprintf("📌 %s | 📄 %s", object.Address, object.ContractNum))
}
