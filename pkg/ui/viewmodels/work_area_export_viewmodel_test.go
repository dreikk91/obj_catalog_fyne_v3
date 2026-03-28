package viewmodels

import (
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestWorkAreaExportViewModel_BuildObjectExportData(t *testing.T) {
	vm := NewWorkAreaExportViewModel()
	lastTest := time.Date(2026, 3, 28, 10, 0, 0, 0, time.Local)
	eventTime := time.Date(2026, 3, 28, 9, 30, 0, 0, time.Local)

	exportData := vm.BuildObjectExportData(
		models.Object{
			ID:            101,
			Name:          "Obj",
			Address:       "Addr",
			ContractNum:   "C-1",
			LaunchDate:    "01.01.2026",
			SIM1:          "111",
			SIM2:          "222",
			DeviceType:    "Panel",
			AutoTestHours: 12,
			ObjChan:       5,
			Phones1:       "380001",
			Location1:     "Loc",
			Notes1:        "Note",
		},
		[]models.Zone{{Number: 1, Name: "Zone 1", SensorType: "Smoke", Status: models.ZoneNormal}},
		[]models.Contact{{Name: "John", Phone: "123", Position: "Head"}},
		[]models.Event{{Time: eventTime, Type: models.EventFire, ZoneNumber: 1, Details: "Alarm"}},
		WorkAreaExternalData{
			Signal:      "85%",
			TestMessage: "OK",
			LastTest:    lastTest,
		},
	)

	if exportData.Number != 101 {
		t.Fatalf("unexpected export number: %d", exportData.Number)
	}
	if exportData.SimCard != "111 / 222" {
		t.Fatalf("unexpected sim text: %q", exportData.SimCard)
	}
	if exportData.TestPeriod != "Кожні 12 год" {
		t.Fatalf("unexpected test period: %q", exportData.TestPeriod)
	}
	if exportData.Channel != "GPRS" {
		t.Fatalf("unexpected channel: %q", exportData.Channel)
	}
	if !strings.Contains(exportData.LastEvent, "ПОЖЕЖА") {
		t.Fatalf("unexpected last event: %q", exportData.LastEvent)
	}
	if !strings.Contains(exportData.LastTest, "28.03.2026 10:00:00") {
		t.Fatalf("unexpected last test: %q", exportData.LastTest)
	}
	if !strings.Contains(exportData.LastTest, "OK") {
		t.Fatalf("unexpected last test details: %q", exportData.LastTest)
	}
	if len(exportData.Zones) != 1 || len(exportData.Responsibles) != 1 {
		t.Fatalf("unexpected related rows count: zones=%d contacts=%d", len(exportData.Zones), len(exportData.Responsibles))
	}
}

func TestWorkAreaExportViewModel_BuildExcelRowTSV(t *testing.T) {
	vm := NewWorkAreaExportViewModel()
	row := vm.BuildExcelRowTSV(
		models.Object{
			ID:          202,
			LaunchDate:  " 02.02.2026 ",
			ContractNum: "C-2",
			Name:        "Obj\tName",
			Address:     "Addr\nLine",
			DeviceType:  "Type",
			PanelMark:   "Mark",
			SIM1:        "111",
			SIM2:        "222",
			Notes1:      "Note",
		},
		[]models.Contact{{Name: "Manager", Phone: "380001"}},
	)

	parts := strings.Split(row, "\t")
	if len(parts) != 16 {
		t.Fatalf("unexpected TSV columns count: %d", len(parts))
	}
	if parts[0] != "202" {
		t.Fatalf("unexpected object id column: %q", parts[0])
	}
	if parts[5] != "Obj Name" {
		t.Fatalf("unexpected sanitized name column: %q", parts[5])
	}
	if parts[6] != "Addr Line" {
		t.Fatalf("unexpected sanitized address column: %q", parts[6])
	}
	if parts[13] != "Manager" || parts[14] != "380001" {
		t.Fatalf("unexpected manager columns: %q / %q", parts[13], parts[14])
	}
}
