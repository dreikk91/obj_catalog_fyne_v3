//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestObjectListClipboardTextUsesVisibleFields(t *testing.T) {
	object := models.Object{
		ID:            42,
		DisplayNumber: "A-42",
		Name:          "  Office  ",
		Address:       " Main st. 1 ",
	}

	got := objectListClipboardText(object)
	want := "№A-42 | Office | Main st. 1"
	if got != want {
		t.Fatalf("objectListClipboardText() = %q, want %q", got, want)
	}
}

func TestObjectListClipboardTextFallsBackToNumericDisplayNumber(t *testing.T) {
	got := objectListClipboardText(models.Object{ID: 77})

	if got != "№77" {
		t.Fatalf("objectListClipboardText() = %q, want numeric display number", got)
	}
}
