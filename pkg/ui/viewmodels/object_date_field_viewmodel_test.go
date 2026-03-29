package viewmodels

import (
	"testing"
	"time"
)

func TestObjectDateFieldViewModel_Parse(t *testing.T) {
	vm := NewObjectDateFieldViewModel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "display format", raw: "28.03.2026"},
		{name: "iso date", raw: "2026-03-28"},
		{name: "timestamp", raw: "2026-03-28 10:15:00"},
		{name: "rfc3339", raw: "2026-03-28T10:15:00+02:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := vm.Parse(tt.raw); !ok {
				t.Fatalf("expected parse success for %q", tt.raw)
			}
		})
	}
}

func TestObjectDateFieldViewModel_Parse_Invalid(t *testing.T) {
	vm := NewObjectDateFieldViewModel()
	if _, ok := vm.Parse("not-a-date"); ok {
		t.Fatalf("expected parse failure")
	}
}

func TestObjectDateFieldViewModel_ResolvePickerInitial(t *testing.T) {
	vm := NewObjectDateFieldViewModel()
	fallback := time.Date(2026, 3, 28, 11, 0, 0, 0, time.Local)

	got := vm.ResolvePickerInitial("29.03.2026", fallback)
	if got.Day() != 29 || got.Month() != 3 || got.Year() != 2026 {
		t.Fatalf("unexpected resolved date: %v", got)
	}

	got = vm.ResolvePickerInitial("invalid", fallback)
	if !got.Equal(fallback) {
		t.Fatalf("expected fallback date")
	}
}

func TestObjectDateFieldViewModel_FormatForDisplay(t *testing.T) {
	vm := NewObjectDateFieldViewModel()
	d := time.Date(2026, 3, 28, 11, 0, 0, 0, time.Local)

	if got := vm.FormatForDisplay(d); got != "28.03.2026" {
		t.Fatalf("unexpected display format: %q", got)
	}
}
