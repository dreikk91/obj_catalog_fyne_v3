package viewmodels

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestWorkAreaHeaderViewModel_DefaultState(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaHeaderViewModel()

	name, err := vm.HeaderNameBinding().Get()
	if err != nil {
		t.Fatalf("unexpected get name error: %v", err)
	}
	if name != "← Оберіть об'єкт зі списку" {
		t.Fatalf("unexpected default header name: %q", name)
	}

	address, err := vm.HeaderAddressBinding().Get()
	if err != nil {
		t.Fatalf("unexpected get address error: %v", err)
	}
	if address != "" {
		t.Fatalf("unexpected default header address: %q", address)
	}
}

func TestWorkAreaHeaderViewModel_ApplyObject(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaHeaderViewModel()
	vm.ApplyObject(models.Object{
		ID:          123,
		Name:        "Object",
		Address:     "Main St",
		ContractNum: "C-001",
	})

	name, _ := vm.HeaderNameBinding().Get()
	if name != "Object (№123)" {
		t.Fatalf("unexpected formatted name: %q", name)
	}

	address, _ := vm.HeaderAddressBinding().Get()
	if address != "📌 Main St | 📄 C-001 | 🧭 БД/МІСТ" {
		t.Fatalf("unexpected formatted address: %q", address)
	}
}

func TestWorkAreaHeaderViewModel_Reset(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaHeaderViewModel()
	vm.ApplyObject(models.Object{ID: 1, Name: "A", Address: "B", ContractNum: "C"})
	vm.Reset()

	name, _ := vm.HeaderNameBinding().Get()
	address, _ := vm.HeaderAddressBinding().Get()
	if name != "← Оберіть об'єкт зі списку" || address != "" {
		t.Fatalf("unexpected reset state: %q / %q", name, address)
	}
}

func TestWorkAreaHeaderViewModel_ApplyObjectCASLNumber(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewWorkAreaHeaderViewModel()
	vm.ApplyObject(models.Object{
		ID:          caslObjectIDNamespaceStart + 24,
		Name:        "Офіс",
		Address:     "Border 1",
		ContractNum: "C-003",
		PanelMark:   "CASL #1003",
	})

	name, _ := vm.HeaderNameBinding().Get()
	if name != "Офіс (№1003)" {
		t.Fatalf("unexpected formatted CASL name: %q", name)
	}
}
