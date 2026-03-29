package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestObjectSourceHelpers(t *testing.T) {
	if got := ObjectSourceByID(1); got != ObjectSourceBridge {
		t.Fatalf("expected bridge source, got %q", got)
	}
	if got := ObjectSourceByID(caslObjectIDNamespaceStart + 1); got != ObjectSourceCASL {
		t.Fatalf("expected casl source, got %q", got)
	}
	if !IsCASLObjectID(caslObjectIDNamespaceStart + 5) {
		t.Fatalf("expected CASL ID to be detected")
	}
	if IsCASLObjectID(42) {
		t.Fatalf("expected non-CASL ID")
	}
}

func TestNormalizeObjectSourceFilter(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ObjectSourceAll},
		{in: "Всі джерела (12)", want: ObjectSourceAll},
		{in: "CASL", want: ObjectSourceCASL},
		{in: "CASL Cloud (3)", want: ObjectSourceCASL},
		{in: "МІСТ", want: ObjectSourceBridge},
		{in: "БД/МІСТ (4)", want: ObjectSourceBridge},
	}

	for _, tt := range tests {
		if got := NormalizeObjectSourceFilter(tt.in); got != tt.want {
			t.Fatalf("NormalizeObjectSourceFilter(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildObjectSourceOptions(t *testing.T) {
	options := BuildObjectSourceOptions(10, 7, 3)
	if len(options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(options))
	}
	if options[0] != "Всі джерела (10)" {
		t.Fatalf("unexpected option[0]: %q", options[0])
	}
	if options[1] != "БД/МІСТ (7)" {
		t.Fatalf("unexpected option[1]: %q", options[1])
	}
	if options[2] != "CASL Cloud (3)" {
		t.Fatalf("unexpected option[2]: %q", options[2])
	}
}

func TestObjectDisplayNumber(t *testing.T) {
	casl := models.Object{
		ID:        caslObjectIDNamespaceStart + 123,
		Name:      "1003 Офіс",
		PanelMark: "CASL #1003",
	}
	if got := ObjectDisplayNumber(casl); got != "1003" {
		t.Fatalf("CASL ObjectDisplayNumber = %q, want %q", got, "1003")
	}

	bridge := models.Object{ID: 42, Name: "Bridge"}
	if got := ObjectDisplayNumber(bridge); got != "42" {
		t.Fatalf("Bridge ObjectDisplayNumber = %q, want %q", got, "42")
	}
}
