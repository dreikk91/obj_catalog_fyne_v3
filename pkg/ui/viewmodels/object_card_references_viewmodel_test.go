package viewmodels

import (
	"errors"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type objectCardReferenceProviderStub struct {
	typeItems      []contracts.DictionaryItem
	regionItems    []contracts.DictionaryItem
	ppkItems       []contracts.PPKConstructorItem
	subServerItems []contracts.AdminSubServer

	typeErr      error
	regionErr    error
	ppkErr       error
	subServerErr error
}

func (s *objectCardReferenceProviderStub) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	if s.typeErr != nil {
		return nil, s.typeErr
	}
	return s.typeItems, nil
}

func (s *objectCardReferenceProviderStub) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	if s.regionErr != nil {
		return nil, s.regionErr
	}
	return s.regionItems, nil
}

func (s *objectCardReferenceProviderStub) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	if s.ppkErr != nil {
		return nil, s.ppkErr
	}
	return s.ppkItems, nil
}

func (s *objectCardReferenceProviderStub) ListSubServers() ([]contracts.AdminSubServer, error) {
	if s.subServerErr != nil {
		return nil, s.subServerErr
	}
	return s.subServerItems, nil
}

func TestObjectCardReferencesViewModel_LoadAndLookup(t *testing.T) {
	vm := NewObjectCardReferencesViewModel()
	vm.Load(
		[]contracts.DictionaryItem{{ID: 2, Name: "Пультовий"}},
		[]contracts.DictionaryItem{{ID: 1, Name: "Галицький"}},
		[]contracts.PPKConstructorItem{{ID: 11, Name: "ППК-А"}},
		[]contracts.AdminSubServer{{ID: 3, Info: "SBS", Bind: "sbs-a", Type: 2}},
	)

	if got := vm.ObjectTypeID(vm.ObjectTypeOptions()[0]); got != 2 {
		t.Fatalf("unexpected object type id: %d", got)
	}
	if got := vm.RegionID(vm.RegionLabelByID(1)); got != 1 {
		t.Fatalf("unexpected region id: %d", got)
	}
	if got := vm.SubServerBind(vm.SubServerLabelByBind("sbs-a")); got != "sbs-a" {
		t.Fatalf("unexpected subserver bind: %q", got)
	}
}

func TestObjectCardReferencesViewModel_RefreshPPKOptions_AcceptsShiftedPreferredID(t *testing.T) {
	vm := NewObjectCardReferencesViewModel()
	vm.Load(
		nil,
		nil,
		[]contracts.PPKConstructorItem{
			{ID: 15, Name: "ППК-15"},
			{ID: 21, Name: "ППК-21"},
		},
		nil,
	)

	selected := vm.RefreshPPKOptions(115)
	if vm.PPKID(selected) != 15 {
		t.Fatalf("expected shifted preferred id to resolve to 15, got %d", vm.PPKID(selected))
	}
}

func TestObjectCardReferencesViewModel_LoadFromProvider(t *testing.T) {
	vm := NewObjectCardReferencesViewModel()
	provider := &objectCardReferenceProviderStub{
		typeItems:      []contracts.DictionaryItem{{ID: 2, Name: "Пультовий"}},
		regionItems:    []contracts.DictionaryItem{{ID: 1, Name: "Галицький"}},
		ppkItems:       []contracts.PPKConstructorItem{{ID: 11, Name: "ППК-А"}},
		subServerItems: []contracts.AdminSubServer{{ID: 3, Info: "SBS", Bind: "sbs-a", Type: 2}},
	}

	if err := vm.LoadFromProvider(provider); err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(vm.ObjectTypeOptions()) == 0 {
		t.Fatalf("expected object type options to be loaded")
	}
	if got := vm.RegionID(vm.RegionLabelByID(1)); got != 1 {
		t.Fatalf("unexpected region id after load: %d", got)
	}
}

func TestObjectCardReferencesViewModel_LoadFromProvider_Error(t *testing.T) {
	vm := NewObjectCardReferencesViewModel()
	provider := &objectCardReferenceProviderStub{
		typeErr: errors.New("db unavailable"),
	}

	err := vm.LoadFromProvider(provider)
	if err == nil {
		t.Fatalf("expected load error")
	}
	if err.Error() != "не вдалося завантажити типи об'єктів: db unavailable" {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
}
