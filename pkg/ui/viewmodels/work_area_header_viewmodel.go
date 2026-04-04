package viewmodels

import (
	"fmt"
	"strings"

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
	_ = vm.headerName.Set(fmt.Sprintf("%s (№%s)", object.Name, ObjectDisplayNumber(object)))
	parts := make([]string, 0, 4)
	if address := strings.TrimSpace(object.Address); address != "" {
		parts = append(parts, "📌 "+address)
	}
	phone := strings.TrimSpace(object.Phones1)
	if phone == "" {
		phone = strings.TrimSpace(object.Phone)
	}
	if phone != "" {
		parts = append(parts, "☎️ "+phone)
	}
	if contract := strings.TrimSpace(object.ContractNum); contract != "" {
		parts = append(parts, "📄 "+contract)
	}
	parts = append(parts, "🧭 "+ObjectSourceByID(object.ID))
	_ = vm.headerAddress.Set(strings.Join(parts, " | "))
}
