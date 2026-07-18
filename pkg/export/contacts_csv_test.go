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
			Contacts:     []models.Contact{{Name: "CASL user", Phone: "+38 (050) 000-00-03"}},
		},
		{
			Source:       contracts.FrontendSourceBridge,
			ObjectNumber: "1001",
			Object:       models.Object{Name: "Bridge object", Address: "Bridge address"},
			Contacts: []models.Contact{
				{Name: "Second", Position: "Owner", Phone: "050-000-00-02", Priority: 2},
				{Name: "First", Position: "Manager", Phone: "050 000 00 01 / (067) 000-00-01", Priority: 1},
			},
		},
		{
			Source:       contracts.FrontendSourcePhoenix,
			ObjectNumber: "L00028",
			Object:       models.Object{Name: "Phoenix object", Address: "Phoenix address"},
			Contacts:     []models.Contact{{Name: "Phoenix user", Phone: "+38 (050) 000-00-04"}},
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
	if first[9] != "+380500000001" || first[12] != "*210011" || first[14] != "+380670000001" {
		t.Fatalf("first contact phones/speed dial = %#v", first)
	}
	if records[2][2] != "Second" || records[2][12] != "*210012" {
		t.Fatalf("second Bridge contact = %#v", records[2])
	}
	if records[3][0] != "Phoenix" || records[3][9] != "+380500000004" || records[3][12] != "*3000281" {
		t.Fatalf("Phoenix contact = %#v", records[3])
	}
	if records[4][0] != "CASL" || records[4][9] != "+380500000003" || records[4][12] != "*4771" {
		t.Fatalf("CASL contact = %#v", records[4])
	}
}

func TestNormalizeContactPhone(t *testing.T) {
	tests := map[string]string{
		"+38 (098) 985-25-98": "+380989852598",
		"067-674-94-48":       "+380676749448",
		"380 67 674 94 48":    "+380676749448",
		"0038 067 674 94 48":  "+380676749448",
		"67 674 94 48":        "+380676749448",
		"+48 123 456 789":     "+48123456789",
		"12345":               "12345",
	}

	for input, want := range tests {
		if got := normalizeContactPhone(input); got != want {
			t.Errorf("normalizeContactPhone(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestContactSpeedDialRequiresNumericObjectNumber(t *testing.T) {
	if got := contactSpeedDial(contracts.FrontendSourcePhoenix, "panel", 1); got != "" {
		t.Fatalf("contactSpeedDial() = %q, want empty", got)
	}
}

func TestWriteContactsCSVCreatesHeaderBeforeContactsAreLoaded(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "contacts.csv")
	count, err := WriteContactsCSV(filePath, nil)
	if err != nil {
		t.Fatalf("WriteContactsCSV() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("WriteContactsCSV() count = %d, want 0", count)
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
	if len(records) != 1 || len(records[0]) != len(contactsCSVHeader) {
		t.Fatalf("header-only CSV = %#v", records)
	}
}
