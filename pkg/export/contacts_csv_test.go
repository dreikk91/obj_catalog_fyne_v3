package export

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestWriteContactsCSVGroupsSourcesAndBuildsSpeedDial(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "contacts.csv")
	objects := []ContactExportObject{
		{
			Source:       contracts.FrontendSourceCASL,
			ObjectNumber: "C-77",
			Object:       models.Object{Name: "CASL object", Address: "CASL address"},
			Contacts:     []models.Contact{{Name: "CASL user", Phone: "0500000003"}},
		},
		{
			Source:       contracts.FrontendSourceBridge,
			ObjectNumber: "1001",
			Object:       models.Object{Name: "Bridge object", Address: "Bridge address"},
			Contacts: []models.Contact{
				{Name: "Second", Position: "Owner", Phone: "0500000002", Priority: 2},
				{Name: "First", Position: "Manager", Phone: "0500000001 / 0670000001", Priority: 1},
			},
		},
		{
			Source:       contracts.FrontendSourcePhoenix,
			ObjectNumber: "L00028",
			Object:       models.Object{Name: "Phoenix object", Address: "Phoenix address"},
			Contacts:     []models.Contact{{Name: "Phoenix user", Phone: "0500000004"}},
		},
	}

	count, err := WriteContactsCSV(filePath, objects)
	if err != nil {
		t.Fatalf("WriteContactsCSV() error = %v", err)
	}
	if count != 4 {
		t.Fatalf("WriteContactsCSV() count = %d, want 4", count)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open CSV: %v", err)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("CSV records = %d, want 5", len(records))
	}
	if len(records[0]) != len(contactsCSVHeader) {
		t.Fatalf("CSV columns = %d, want %d", len(records[0]), len(contactsCSVHeader))
	}

	first := records[1]
	if first[0] != "МІСТ" || first[2] != "First" || first[6] != "Bridge object" {
		t.Fatalf("first contact = %#v", first)
	}
	if first[9] != "0500000001" || first[12] != "*210011" || first[14] != "0670000001" {
		t.Fatalf("first contact phones/speed dial = %#v", first)
	}
	if records[2][2] != "Second" || records[2][12] != "*210012" {
		t.Fatalf("second Bridge contact = %#v", records[2])
	}
	if records[3][0] != "Phoenix" || records[3][12] != "*3000281" {
		t.Fatalf("Phoenix contact = %#v", records[3])
	}
	if records[4][0] != "CASL" || records[4][12] != "*4771" {
		t.Fatalf("CASL contact = %#v", records[4])
	}
}

func TestContactSpeedDialRequiresNumericObjectNumber(t *testing.T) {
	if got := contactSpeedDial(contracts.FrontendSourcePhoenix, "panel", 1); got != "" {
		t.Fatalf("contactSpeedDial() = %q, want empty", got)
	}
}
