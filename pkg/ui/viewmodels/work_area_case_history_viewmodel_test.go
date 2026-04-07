package viewmodels

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestWorkAreaCaseHistoryViewModel_BuildGroups_ForCASLObject(t *testing.T) {
	vm := NewWorkAreaCaseHistoryViewModel()
	object := &models.Object{ID: ids.CASLObjectIDNamespaceStart + 51, Name: "CASL object"}
	base := time.Date(2026, 4, 6, 10, 0, 0, 0, time.Local)

	events := []models.Event{
		{ID: 8, Time: base.Add(8 * time.Minute), Type: models.EventDisarm, Details: "Зняття групи № 1"},
		{ID: 7, Time: base.Add(7 * time.Minute), Type: models.EventOperatorAction, Details: "Взяття в роботу об'єкта"},
		{ID: 6, Time: base.Add(6 * time.Minute), Type: models.EventRestore, ZoneNumber: 4, Details: "Норма в зоні № 4"},
		{ID: 5, Time: base.Add(5 * time.Minute), Type: models.EventBurglary, ZoneNumber: 4, Details: "Тривога в зоні № 4"},
		{ID: 4, Time: base.Add(4 * time.Minute), Type: models.EventArm, Details: "Взяття групи № 1"},
		{ID: 3, Time: base.Add(3 * time.Minute), Type: models.EventRestore, ZoneNumber: 2, Details: "Норма в зоні № 2"},
		{ID: 2, Time: base.Add(2 * time.Minute), Type: models.EventFault, ZoneNumber: 2, Details: "Проблема в зоні № 2"},
		{ID: 1, Time: base.Add(1 * time.Minute), Type: models.EventArm, Details: "Стартова подія до кейсу"},
	}

	groups := vm.BuildGroups(object, events)
	if len(groups) != 2 {
		t.Fatalf("expected 2 case groups, got %d", len(groups))
	}

	if groups[0].Root.Type != models.EventBurglary {
		t.Fatalf("expected latest case root to be burglary, got %s", groups[0].Root.Type)
	}
	if len(groups[0].Events) != 4 {
		t.Fatalf("expected 4 events in latest case, got %d", len(groups[0].Events))
	}
	if groups[1].Root.Type != models.EventFault {
		t.Fatalf("expected older case root to be fault, got %s", groups[1].Root.Type)
	}
	if len(groups[1].Events) != 3 {
		t.Fatalf("expected 3 events in older case, got %d", len(groups[1].Events))
	}
}

func TestWorkAreaCaseHistoryViewModel_BuildGroups_IgnoresNonCASLObject(t *testing.T) {
	vm := NewWorkAreaCaseHistoryViewModel()
	object := &models.Object{ID: 42, Name: "Bridge object"}
	events := []models.Event{
		{ID: 1, Time: time.Now(), Type: models.EventBurglary, Details: "Alarm"},
	}

	groups := vm.BuildGroups(object, events)
	if len(groups) != 0 {
		t.Fatalf("expected no groups for non-CASL object, got %d", len(groups))
	}
}

func TestWorkAreaCaseHistoryViewModel_FindGroupForAlarm_PrefersMatchingZoneAndType(t *testing.T) {
	vm := NewWorkAreaCaseHistoryViewModel()
	object := &models.Object{ID: ids.CASLObjectIDNamespaceStart + 77, Name: "CASL object"}
	base := time.Date(2026, 4, 6, 12, 0, 0, 0, time.Local)

	events := []models.Event{
		{ID: 1, Time: base.Add(1 * time.Minute), Type: models.EventBurglary, ZoneNumber: 2, Details: "Тривога в зоні 2"},
		{ID: 2, Time: base.Add(2 * time.Minute), Type: models.EventRestore, ZoneNumber: 2, Details: "Норма в зоні 2"},
		{ID: 3, Time: base.Add(3 * time.Minute), Type: models.EventBurglary, ZoneNumber: 4, Details: "Тривога в зоні 4"},
		{ID: 4, Time: base.Add(4 * time.Minute), Type: models.EventOperatorAction, Details: "Оператор взяв об'єкт"},
	}

	group, ok := vm.FindGroupForAlarm(object, models.Alarm{
		ObjectID:   object.ID,
		Time:       base.Add(3 * time.Minute),
		Type:       models.AlarmBurglary,
		ZoneNumber: 4,
	}, events)
	if !ok {
		t.Fatalf("expected matching case group")
	}
	if group.Root.ZoneNumber != 4 {
		t.Fatalf("expected zone 4 case, got zone %d", group.Root.ZoneNumber)
	}
	if group.Root.Type != models.EventBurglary {
		t.Fatalf("expected burglary root, got %s", group.Root.Type)
	}
}

func TestWorkAreaCaseHistoryViewModel_BuildSections_GroupsByStage(t *testing.T) {
	vm := NewWorkAreaCaseHistoryViewModel()
	base := time.Date(2026, 4, 6, 13, 0, 0, 0, time.Local)

	group := WorkAreaCaseHistoryGroup{
		Root: models.Event{ID: 1, Time: base, Type: models.EventBurglary, ZoneNumber: 4, Details: "Тривога"},
		Events: []models.Event{
			{ID: 1, Time: base, Type: models.EventBurglary, ZoneNumber: 4, Details: "Тривога"},
			{ID: 2, Time: base.Add(1 * time.Minute), Type: models.EventOperatorAction, Details: "Оператор відкрив картку"},
			{ID: 3, Time: base.Add(2 * time.Minute), Type: models.EventDisarm, Details: "Зняття групи"},
			{ID: 4, Time: base.Add(3 * time.Minute), Type: models.EventRestore, ZoneNumber: 4, Details: "Норма"},
		},
	}

	sections := vm.BuildSections(group)
	if len(sections) != 4 {
		t.Fatalf("expected 4 timeline sections, got %d", len(sections))
	}
	if sections[0].Key != caseHistorySectionRoot {
		t.Fatalf("expected first section to be root, got %s", sections[0].Key)
	}
	if sections[1].Key != caseHistorySectionOperator {
		t.Fatalf("expected second section to be operator, got %s", sections[1].Key)
	}
	if sections[2].Key != caseHistorySectionGuard {
		t.Fatalf("expected third section to be guard, got %s", sections[2].Key)
	}
	if sections[3].Key != caseHistorySectionRestore {
		t.Fatalf("expected fourth section to be restore, got %s", sections[3].Key)
	}
}
