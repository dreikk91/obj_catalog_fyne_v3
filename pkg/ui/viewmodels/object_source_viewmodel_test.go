package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestObjectSourceHelpers(t *testing.T) {
	if got := ObjectSourceByID(1); got != ObjectSourceBridge {
		t.Fatalf("expected bridge source, got %q", got)
	}
	if got := ObjectSourceByID(ids.PhoenixObjectIDNamespaceStart + 1); got != ObjectSourcePhoenix {
		t.Fatalf("expected phoenix source, got %q", got)
	}
	if got := ObjectSourceByID(ids.CASLObjectIDNamespaceStart + 1); got != ObjectSourceCASL {
		t.Fatalf("expected casl source, got %q", got)
	}
	if !ids.IsPhoenixObjectID(ids.PhoenixObjectIDNamespaceStart + 5) {
		t.Fatalf("expected Phoenix ID to be detected")
	}
	if !ids.IsCASLObjectID(ids.CASLObjectIDNamespaceStart + 5) {
		t.Fatalf("expected CASL ID to be detected")
	}
	if ids.IsPhoenixObjectID(42) {
		t.Fatalf("expected non-Phoenix ID")
	}
	if ids.IsCASLObjectID(42) {
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
		{in: "Phoenix", want: ObjectSourcePhoenix},
		{in: "Phoenix (2)", want: ObjectSourcePhoenix},
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
	options := BuildObjectSourceOptions(10, 5, 2, 3)
	if len(options) != 4 {
		t.Fatalf("expected 4 options, got %d", len(options))
	}
	if options[0] != "Всі джерела (10)" {
		t.Fatalf("unexpected option[0]: %q", options[0])
	}
	if options[1] != "БД/МІСТ (5)" {
		t.Fatalf("unexpected option[1]: %q", options[1])
	}
	if options[2] != "Phoenix (2)" {
		t.Fatalf("unexpected option[2]: %q", options[2])
	}
	if options[3] != "CASL Cloud (3)" {
		t.Fatalf("unexpected option[3]: %q", options[3])
	}
}

func TestObjectDisplayNumber(t *testing.T) {
	phoenix := models.Object{
		ID:            ids.PhoenixObjectIDNamespaceStart + 55,
		DisplayNumber: "L00028",
		Name:          "Phoenix Object",
	}
	if got := ObjectDisplayNumber(phoenix); got != "L00028" {
		t.Fatalf("Phoenix ObjectDisplayNumber = %q, want %q", got, "L00028")
	}

	casl := models.Object{
		ID:        ids.CASLObjectIDNamespaceStart + 123,
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
