package viewmodels

import (
	"errors"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type simPhoneUsageLookupStub struct {
	usages       []contracts.AdminSIMPhoneUsage
	err          error
	lastPhone    string
	lastExclude  *int64
	calledLookup bool
}

func (s *simPhoneUsageLookupStub) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	s.calledLookup = true
	s.lastPhone = phone
	s.lastExclude = excludeObjN
	if s.err != nil {
		return nil, s.err
	}
	return s.usages, nil
}

func TestSIMPhoneUsageViewModel_ResolveUsageText_EmptyPhone(t *testing.T) {
	vm := NewSIMPhoneUsageViewModel()
	stub := &simPhoneUsageLookupStub{}

	text := vm.ResolveUsageText(stub, "   ", nil)
	if text != "" {
		t.Fatalf("expected empty text, got %q", text)
	}
	if stub.calledLookup {
		t.Fatalf("lookup must not be called for empty phone")
	}
}

func TestSIMPhoneUsageViewModel_ResolveUsageText_Error(t *testing.T) {
	vm := NewSIMPhoneUsageViewModel()
	stub := &simPhoneUsageLookupStub{err: errors.New("db error")}

	text := vm.ResolveUsageText(stub, " 0501234567 ", nil)
	if text != "Не вдалося перевірити номер у базі" {
		t.Fatalf("unexpected text: %q", text)
	}
	if !stub.calledLookup {
		t.Fatalf("expected lookup to be called")
	}
	if stub.lastPhone != "0501234567" {
		t.Fatalf("phone must be trimmed, got %q", stub.lastPhone)
	}
}

func TestSIMPhoneUsageViewModel_ResolveUsageText_Success(t *testing.T) {
	vm := NewSIMPhoneUsageViewModel()
	exclude := int64(77)
	stub := &simPhoneUsageLookupStub{
		usages: []contracts.AdminSIMPhoneUsage{
			{ObjN: 1001, Name: "  Об'єкт 1 ", Slot: "SIM1"},
			{ObjN: 1002, Name: "", Slot: "SIM2"},
		},
	}

	text := vm.ResolveUsageText(stub, "0670000000", &exclude)
	expected := "Номер вже використовується: #1001 (Об'єкт 1, SIM1); #1002 (SIM2)"
	if text != expected {
		t.Fatalf("unexpected text: %q", text)
	}
	if stub.lastExclude == nil || *stub.lastExclude != exclude {
		t.Fatalf("exclude pointer must be passed through")
	}
}

func TestSIMPhoneUsageViewModel_FormatUsageList_Empty(t *testing.T) {
	vm := NewSIMPhoneUsageViewModel()
	if got := vm.FormatUsageList(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
