package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestWorkAreaOverviewPrioritizesOperationalProblems(t *testing.T) {
	vm := NewWorkAreaOverviewViewModel()
	object := models.Object{
		ID:               ids.PhoenixObjectIDNamespaceStart + 35,
		ConnectionStatus: models.ConnectionStatusOffline,
		DeviceType:       "Лунь-9С",
		PanelMark:        "SIM800C",
	}
	device := NewWorkAreaDeviceViewModel().BuildObjectPresentation(object)
	overview := vm.Build(object, []models.Zone{
		{Number: 2, Name: "Вікно", Status: models.ZoneAlarm},
		{Number: 1, Name: "Коридор", Status: models.ZoneNormal},
	}, nil, device)

	if overview.Source != ObjectSourcePhoenix {
		t.Fatalf("Source = %q, want %q", overview.Source, ObjectSourcePhoenix)
	}
	if overview.SummaryTone != WorkAreaOverviewCritical {
		t.Fatalf("SummaryTone = %q, want critical", overview.SummaryTone)
	}
	if overview.ProblemZoneCount != 1 || overview.ProblemZones[0].Number != 2 {
		t.Fatalf("ProblemZones = %#v", overview.ProblemZones)
	}
	if overview.Device != "Лунь-9С / SIM800C" {
		t.Fatalf("Device = %q", overview.Device)
	}
}

func TestWorkAreaOverviewKeepsPriorityContactsAndCounts(t *testing.T) {
	vm := NewWorkAreaOverviewViewModel()
	object := models.Object{
		ID:                         ids.CASLObjectIDNamespaceStart + 10,
		PreferredResponseGroupName: "ГМР 3",
		Location1:                  "Другий поверх, праве крило",
		Notes1:                     "Ключі у чергового",
		ConnectionStatus:           models.ConnectionStatusOnline,
		MonitoringStatus:           models.MonitoringStatusActive,
		PowerFault:                 0,
		AkbState:                   0,
	}
	contacts := []models.Contact{
		{Name: "Другий", Priority: 2},
		{Name: "Перший", Priority: 1},
		{Name: "Без пріоритету"},
	}
	zones := []models.Zone{
		{Number: 1, GroupID: "a", Status: models.ZoneNormal},
		{Number: 2, GroupID: "b", Status: models.ZoneNormal},
	}
	device := NewWorkAreaDeviceViewModel().BuildObjectPresentation(object)
	overview := vm.Build(object, zones, contacts, device)

	if overview.SummaryTone != WorkAreaOverviewNormal {
		t.Fatalf("SummaryTone = %q, want normal", overview.SummaryTone)
	}
	if overview.GroupCount != 2 || overview.ZoneCount != 2 || overview.ContactCount != 3 {
		t.Fatalf("unexpected counts: %#v", overview)
	}
	if overview.PriorityContacts[0].Name != "Перший" {
		t.Fatalf("PriorityContacts = %#v", overview.PriorityContacts)
	}
	if overview.ResponseGroup != "ГМР 3" {
		t.Fatalf("ResponseGroup = %q", overview.ResponseGroup)
	}
	if overview.Location != "Другий поверх, праве крило" {
		t.Fatalf("Location = %q", overview.Location)
	}
	if overview.AdditionalInfo != "Ключі у чергового" {
		t.Fatalf("AdditionalInfo = %q", overview.AdditionalInfo)
	}
}

func TestWorkAreaOverviewDoesNotReportUnknownCASLPowerAsNormal(t *testing.T) {
	vm := NewWorkAreaOverviewViewModel()
	object := models.Object{
		ID:               ids.CASLObjectIDNamespaceStart + 11,
		ConnectionStatus: models.ConnectionStatusOnline,
		PowerFault:       -1,
		AkbState:         -1,
	}
	device := NewWorkAreaDeviceViewModel().BuildObjectPresentation(object)
	overview := vm.Build(object, nil, nil, device)

	if overview.SummaryTone != WorkAreaOverviewWarning {
		t.Fatalf("SummaryTone = %q, want warning", overview.SummaryTone)
	}
	if overview.Summary != "СТАН ЖИВЛЕННЯ НЕ ВИЗНАЧЕНО" {
		t.Fatalf("Summary = %q", overview.Summary)
	}
}

func TestWorkAreaOverviewDoesNotTreatUnknownZoneAsTechnicalProblem(t *testing.T) {
	vm := NewWorkAreaOverviewViewModel()
	object := models.Object{
		ConnectionStatus: models.ConnectionStatusOnline,
		PowerFault:       0,
		AkbState:         0,
	}
	device := NewWorkAreaDeviceViewModel().BuildObjectPresentation(object)
	overview := vm.Build(object, []models.Zone{{Number: 1, Name: "Невідомий стан"}}, nil, device)

	if overview.ProblemZoneCount != 0 {
		t.Fatalf("ProblemZoneCount = %d, want 0", overview.ProblemZoneCount)
	}
	if overview.SummaryTone != WorkAreaOverviewNormal {
		t.Fatalf("SummaryTone = %q, want normal", overview.SummaryTone)
	}
}
