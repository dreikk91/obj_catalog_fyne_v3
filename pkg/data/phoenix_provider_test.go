package data

import (
	"database/sql"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestPhoenixGroupStateText(t *testing.T) {
	tests := []struct {
		name  string
		open  sql.NullBool
		off   sql.NullBool
		test  sql.NullBool
		state sql.NullInt64
		want  string
	}{
		{name: "alarm", state: sql.NullInt64{Int64: 2, Valid: true}, want: "ТРИВОГА"},
		{name: "disabled", off: sql.NullBool{Bool: true, Valid: true}, want: "ЗАБЛОКОВАНО"},
		{name: "test", test: sql.NullBool{Bool: true, Valid: true}, want: "СТЕНДИ"},
		{name: "open", open: sql.NullBool{Bool: true, Valid: true}, want: "БЕЗ ОХОРОНИ"},
		{name: "armed", want: "ПІД ОХОРОНОЮ"},
	}

	for _, tt := range tests {
		if got := phoenixGroupStateText(tt.open, tt.off, tt.test, tt.state); got != tt.want {
			t.Fatalf("%s: phoenixGroupStateText() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPhoenixEventType(t *testing.T) {
	tests := []struct {
		name    string
		code    sql.NullString
		typeID  sql.NullInt64
		details string
		want    models.EventType
	}{
		{
			name:   "intrusion by type code",
			code:   sql.NullString{String: "E130", Valid: true},
			typeID: sql.NullInt64{Int64: 1, Valid: true},
			want:   models.EventBurglary,
		},
		{
			name:   "fire by type code",
			code:   sql.NullString{String: "E110", Valid: true},
			typeID: sql.NullInt64{Int64: 14, Valid: true},
			want:   models.EventFire,
		},
		{
			name:   "panic by type code",
			code:   sql.NullString{String: "E120", Valid: true},
			typeID: sql.NullInt64{Int64: 128, Valid: true},
			want:   models.EventPanic,
		},
		{
			name:   "tamper by type code",
			code:   sql.NullString{String: "E383", Valid: true},
			typeID: sql.NullInt64{Int64: 28, Valid: true},
			want:   models.EventTamper,
		},
		{
			name:   "medical by type code",
			code:   sql.NullString{String: "E100", Valid: true},
			typeID: sql.NullInt64{Int64: 85, Valid: true},
			want:   models.EventMedical,
		},
		{
			name:   "gas by type code",
			code:   sql.NullString{String: "E151", Valid: true},
			typeID: sql.NullInt64{Int64: 156, Valid: true},
			want:   models.EventGas,
		},
		{
			name:   "fault by type code",
			code:   sql.NullString{String: "E300", Valid: true},
			typeID: sql.NullInt64{Int64: 15, Valid: true},
			want:   models.EventFault,
		},
		{
			name:   "intrusion cid overrides faulty type code",
			code:   sql.NullString{String: "E130", Valid: true},
			typeID: sql.NullInt64{Int64: 15, Valid: true},
			want:   models.EventBurglary,
		},
		{
			name:   "power fail cid overrides faulty type code",
			code:   sql.NullString{String: "E301", Valid: true},
			typeID: sql.NullInt64{Int64: 15, Valid: true},
			want:   models.EventPowerFail,
		},
		{
			name:   "power fail by type code",
			code:   sql.NullString{String: "E301", Valid: true},
			typeID: sql.NullInt64{Int64: 6, Valid: true},
			want:   models.EventPowerFail,
		},
		{
			name:   "power restore by type code",
			code:   sql.NullString{String: "R301", Valid: true},
			typeID: sql.NullInt64{Int64: 7, Valid: true},
			want:   models.EventPowerOK,
		},
		{
			name:   "battery low by type code",
			code:   sql.NullString{String: "E302", Valid: true},
			typeID: sql.NullInt64{Int64: 8, Valid: true},
			want:   models.EventBatteryLow,
		},
		{
			name:   "offline by type code",
			code:   sql.NullString{String: "E350", Valid: true},
			typeID: sql.NullInt64{Int64: 18, Valid: true},
			want:   models.EventOffline,
		},
		{
			name:   "online by type code",
			code:   sql.NullString{String: "R350", Valid: true},
			typeID: sql.NullInt64{Int64: 19, Valid: true},
			want:   models.EventOnline,
		},
		{
			name:   "arm by type code",
			code:   sql.NullString{String: "E401", Valid: true},
			typeID: sql.NullInt64{Int64: 3, Valid: true},
			want:   models.EventArm,
		},
		{
			name:   "disarm by type code",
			code:   sql.NullString{String: "R401", Valid: true},
			typeID: sql.NullInt64{Int64: 4, Valid: true},
			want:   models.EventDisarm,
		},
		{
			name:   "test by type code",
			code:   sql.NullString{String: "T100", Valid: true},
			typeID: sql.NullInt64{Int64: 5, Valid: true},
			want:   models.EventTest,
		},
		{
			name:   "system by type code",
			code:   sql.NullString{String: "SYS", Valid: true},
			typeID: sql.NullInt64{Int64: 20, Valid: true},
			want:   models.SystemEvent,
		},
		{
			name:   "operator action by type code",
			code:   sql.NullString{String: "ACC", Valid: true},
			typeID: sql.NullInt64{Int64: 36, Valid: true},
			want:   models.EventOperatorAction,
		},
		{
			name:   "notification by type code",
			code:   sql.NullString{String: "REP", Valid: true},
			typeID: sql.NullInt64{Int64: 67, Valid: true},
			want:   models.EventNotification,
		},
		{
			name:    "restore by text fallback",
			code:    sql.NullString{String: "R110", Valid: true},
			details: "Відновлення",
			want:    models.EventRestore,
		},
		{
			name:    "arm by text fallback",
			code:    sql.NullString{String: "E401", Valid: true},
			details: "Постановка групи",
			want:    models.EventArm,
		},
	}

	for _, tt := range tests {
		if got := phoenixEventType(tt.code, tt.typeID, tt.details); got != tt.want {
			t.Fatalf("%s: phoenixEventType() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPhoenixBuildObjects_AllGroupsDisarmed(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	rows := []phoenixObjectGroupRow{
		{
			PanelID:   "L00028",
			GroupNo:   1,
			GroupName: sql.NullString{String: "Офіс", Valid: true},
			IsOpen:    sql.NullBool{Bool: true, Valid: true},
		},
		{
			PanelID:   "L00028",
			GroupNo:   2,
			GroupName: sql.NullString{String: "Склад", Valid: true},
			IsOpen:    sql.NullBool{Bool: true, Valid: true},
		},
	}

	objects := provider.buildObjects(rows)
	if len(objects) != 1 {
		t.Fatalf("buildObjects() returned %d objects, want 1", len(objects))
	}

	obj := objects[0]
	if obj.GuardState != 0 {
		t.Fatalf("GuardState = %d, want 0 for fully disarmed phoenix object", obj.GuardState)
	}
	if obj.IsUnderGuard {
		t.Fatalf("IsUnderGuard = true, want false for fully disarmed phoenix object")
	}
	if obj.BlockedArmedOnOff != 0 {
		t.Fatalf("BlockedArmedOnOff = %d, want 0 for fully disarmed phoenix object", obj.BlockedArmedOnOff)
	}
	if obj.StatusText != "БЕЗ ОХОРОНИ" {
		t.Fatalf("StatusText = %q, want %q", obj.StatusText, "БЕЗ ОХОРОНИ")
	}
}

func TestPhoenixBuildObjects_MixedGroupStateStaysGuarded(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	rows := []phoenixObjectGroupRow{
		{
			PanelID:   "L00029",
			GroupNo:   1,
			GroupName: sql.NullString{String: "Офіс", Valid: true},
			IsOpen:    sql.NullBool{Bool: false, Valid: true},
		},
		{
			PanelID:   "L00029",
			GroupNo:   2,
			GroupName: sql.NullString{String: "Склад", Valid: true},
			IsOpen:    sql.NullBool{Bool: true, Valid: true},
		},
	}

	objects := provider.buildObjects(rows)
	if len(objects) != 1 {
		t.Fatalf("buildObjects() returned %d objects, want 1", len(objects))
	}

	obj := objects[0]
	if obj.GuardState == 0 {
		t.Fatalf("GuardState = %d, want guarded state for mixed-group object", obj.GuardState)
	}
	if !obj.IsUnderGuard {
		t.Fatalf("IsUnderGuard = false, want true for mixed-group object")
	}
	if obj.BlockedArmedOnOff != 0 {
		t.Fatalf("BlockedArmedOnOff = %d, want 0 for mixed phoenix object", obj.BlockedArmedOnOff)
	}
	if obj.StatusText != "ЧАСТКОВО БЕЗ ОХОРОНИ" {
		t.Fatalf("StatusText = %q, want %q", obj.StatusText, "ЧАСТКОВО БЕЗ ОХОРОНИ")
	}
}

func TestPhoenixBuildObjects_DisabledObjectMarkedBlocked(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	rows := []phoenixObjectGroupRow{
		{
			PanelID:       "L00030",
			GroupNo:       1,
			GroupName:     sql.NullString{String: "Офіс", Valid: true},
			IsOpen:        sql.NullBool{Bool: false, Valid: true},
			GroupDisabled: sql.NullBool{Bool: true, Valid: true},
		},
	}

	objects := provider.buildObjects(rows)
	if len(objects) != 1 {
		t.Fatalf("buildObjects() returned %d objects, want 1", len(objects))
	}

	obj := objects[0]
	if obj.BlockedArmedOnOff != 1 {
		t.Fatalf("BlockedArmedOnOff = %d, want 1 for disabled phoenix object", obj.BlockedArmedOnOff)
	}
	if obj.StatusText != "ЗАБЛОКОВАНО" {
		t.Fatalf("StatusText = %q, want %q", obj.StatusText, "ЗАБЛОКОВАНО")
	}
}

func TestPhoenixBuildObjects_TestPanelMarkedStand(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	rows := []phoenixObjectGroupRow{
		{
			PanelID:   "L00031",
			GroupNo:   1,
			GroupName: sql.NullString{String: "Стенд", Valid: true},
			TestPanel: sql.NullBool{Bool: true, Valid: true},
		},
	}

	objects := provider.buildObjects(rows)
	if len(objects) != 1 {
		t.Fatalf("buildObjects() returned %d objects, want 1", len(objects))
	}

	obj := objects[0]
	if obj.BlockedArmedOnOff != 2 {
		t.Fatalf("BlockedArmedOnOff = %d, want 2 for test panel", obj.BlockedArmedOnOff)
	}
	if obj.StatusText != "СТЕНДИ" {
		t.Fatalf("StatusText = %q, want %q", obj.StatusText, "СТЕНДИ")
	}
}

func TestPhoenixBuildPhoenixAlarms_ReturnsOnlyActiveAlarmRows(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	rows := []phoenixObjectGroupRow{
		{
			PanelID:     "L00031",
			GroupNo:     1,
			GroupName:   sql.NullString{String: "Офіс", Valid: true},
			CompanyName: sql.NullString{String: "Компанія 31", Valid: true},
			Address:     sql.NullString{String: "Адреса 31", Valid: true},
			GroupTime:   sql.NullTime{Time: time.Date(2026, time.April, 6, 14, 0, 0, 0, time.UTC), Valid: true},
			StateEvent:  sql.NullInt64{Int64: 2, Valid: true},
		},
		{
			PanelID:     "L00031",
			GroupNo:     2,
			GroupName:   sql.NullString{String: "Стенд", Valid: true},
			TestPanel:   sql.NullBool{Bool: true, Valid: true},
			StateEvent:  sql.NullInt64{Int64: 2, Valid: true},
			CompanyName: sql.NullString{String: "Компанія 31", Valid: true},
			GroupTime:   sql.NullTime{Time: time.Date(2026, time.April, 6, 14, 1, 0, 0, time.UTC), Valid: true},
		},
		{
			PanelID:     "L00032",
			GroupNo:     1,
			GroupName:   sql.NullString{String: "Склад", Valid: true},
			CompanyName: sql.NullString{String: "Компанія 32", Valid: true},
			StateEvent:  sql.NullInt64{Int64: 1, Valid: true},
		},
	}

	alarms := provider.buildPhoenixAlarms(rows)
	if len(alarms) != 1 {
		t.Fatalf("buildPhoenixAlarms() returned %d alarms, want 1", len(alarms))
	}

	alarm := alarms[0]
	if alarm.ObjectNumber != "L00031" {
		t.Fatalf("ObjectNumber = %q, want L00031", alarm.ObjectNumber)
	}
	if alarm.ObjectName != "Компанія 31" {
		t.Fatalf("ObjectName = %q, want Компанія 31", alarm.ObjectName)
	}
	if alarm.Details != "Офіс" {
		t.Fatalf("Details = %q, want Офіс", alarm.Details)
	}
	if alarm.Type != models.AlarmFire {
		t.Fatalf("Type = %q, want %q", alarm.Type, models.AlarmFire)
	}
	if alarm.SC1 != 1 {
		t.Fatalf("SC1 = %d, want 1", alarm.SC1)
	}
}

func TestPhoenixBuildActiveAlarms_UsesTempRowsAndMapsAlarmType(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	rows := []phoenixActiveAlarmRow{
		{
			EventID:      sql.NullInt64{Int64: 501, Valid: true},
			PanelID:      "L00041",
			GroupNo:      2,
			ZoneNo:       sql.NullInt64{Int64: 7, Valid: true},
			TimeEvent:    sql.NullTime{Time: time.Date(2026, time.April, 8, 9, 30, 0, 0, time.UTC), Valid: true},
			EventCode:    sql.NullString{String: "E130", Valid: true},
			CodeMessage:  sql.NullString{String: "Проникнення", Valid: true},
			TypeCodeID:   sql.NullInt64{Int64: 1, Valid: true},
			GroupMessage: sql.NullString{String: "Склад", Valid: true},
			GroupName:    sql.NullString{String: "Склад", Valid: true},
			ZoneName:     sql.NullString{String: "Двері", Valid: true},
			CompanyName:  sql.NullString{String: "Компанія 41", Valid: true},
			Address:      sql.NullString{String: "Адреса 41", Valid: true},
		},
		{
			EventID:      sql.NullInt64{Int64: 502, Valid: true},
			PanelID:      "L00042",
			GroupNo:      3,
			TimeEvent:    sql.NullTime{Time: time.Date(2026, time.April, 8, 9, 31, 0, 0, time.UTC), Valid: true},
			EventCode:    sql.NullString{String: "E110", Valid: true},
			CodeMessage:  sql.NullString{String: "Пожежа", Valid: true},
			TypeCodeID:   sql.NullInt64{Int64: 14, Valid: true},
			GroupMessage: sql.NullString{String: "Тест", Valid: true},
			GroupName:    sql.NullString{String: "Тест", Valid: true},
			CompanyName:  sql.NullString{String: "Компанія 41", Valid: true},
		},
	}

	alarms := provider.buildPhoenixActiveAlarms(rows)
	if len(alarms) != 2 {
		t.Fatalf("buildPhoenixActiveAlarms() returned %d alarms, want 2", len(alarms))
	}

	alarm := alarms[1]
	if alarm.ObjectNumber != "L00041" {
		t.Fatalf("ObjectNumber = %q, want L00041", alarm.ObjectNumber)
	}
	if alarm.Type != models.AlarmBurglary {
		t.Fatalf("Type = %q, want %q", alarm.Type, models.AlarmBurglary)
	}
	if alarm.ZoneNumber != 7 {
		t.Fatalf("ZoneNumber = %d, want 7", alarm.ZoneNumber)
	}
	if alarm.Details != "Проникнення [Двері] | Склад" {
		t.Fatalf("Details = %q, want %q", alarm.Details, "Проникнення [Двері] | Склад")
	}
	if alarm.SC1 != 1 {
		t.Fatalf("SC1 = %d, want 1", alarm.SC1)
	}
	if alarms[0].Type != models.AlarmFire {
		t.Fatalf("latest alarm type = %q, want %q", alarms[0].Type, models.AlarmFire)
	}
	if alarms[0].ObjectNumber != "L00042" {
		t.Fatalf("newest object number = %q, want %q", alarms[0].ObjectNumber, "L00042")
	}
}

func TestPhoenixBuildActiveAlarms_GroupKeepsLatestAlarmEvenAfterRestore(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")

	alarmTime := time.Date(2026, time.April, 8, 12, 0, 0, 0, time.UTC)
	restoreTime := alarmTime.Add(2 * time.Minute)

	rows := []phoenixActiveAlarmRow{
		{
			EventID:       sql.NullInt64{Int64: 7001, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9001, Valid: true},
			PanelID:       "L00050",
			GroupNo:       1,
			ZoneNo:        sql.NullInt64{Int64: 4, Valid: true},
			TimeEvent:     sql.NullTime{Time: alarmTime, Valid: true},
			EventCode:     sql.NullString{String: "E130", Valid: true},
			CodeMessage:   sql.NullString{String: "Проникнення", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 1, Valid: true},
			GroupMessage:  sql.NullString{String: "Офіс", Valid: true},
			GroupName:     sql.NullString{String: "Офіс", Valid: true},
			ZoneName:      sql.NullString{String: "Двері", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 50", Valid: true},
		},
		{
			EventID:       sql.NullInt64{Int64: 7002, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9001, Valid: true},
			PanelID:       "L00050",
			GroupNo:       1,
			ZoneNo:        sql.NullInt64{Int64: 4, Valid: true},
			TimeEvent:     sql.NullTime{Time: restoreTime, Valid: true},
			EventCode:     sql.NullString{String: "R130", Valid: true},
			CodeMessage:   sql.NullString{String: "Відновлення", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 2, Valid: true},
			GroupMessage:  sql.NullString{String: "Офіс", Valid: true},
			GroupName:     sql.NullString{String: "Офіс", Valid: true},
			ZoneName:      sql.NullString{String: "Двері", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 50", Valid: true},
		},
	}

	alarms := provider.buildPhoenixActiveAlarms(rows)
	if len(alarms) != 1 {
		t.Fatalf("buildPhoenixActiveAlarms() returned %d alarms, want 1 grouped alarm", len(alarms))
	}

	alarm := alarms[0]
	if alarm.Type != models.AlarmBurglary {
		t.Fatalf("Type = %q, want %q", alarm.Type, models.AlarmBurglary)
	}
	if got, want := alarm.Details, "Проникнення [Двері] | Офіс"; got != want {
		t.Fatalf("Details = %q, want %q", got, want)
	}
	if !alarm.Time.Equal(normalizePhoenixEventTime(alarmTime)) {
		t.Fatalf("Time = %v, want %v", alarm.Time, normalizePhoenixEventTime(alarmTime))
	}
	if alarm.SC1 != 5 {
		t.Fatalf("SC1 = %d, want 5 (latest restore color)", alarm.SC1)
	}
	if len(alarm.SourceMsgs) != 2 {
		t.Fatalf("expected 2 source messages, got %d", len(alarm.SourceMsgs))
	}
	if alarm.SourceMsgs[0].IsAlarm {
		t.Fatalf("newest source message must be restore/non-alarm, got %+v", alarm.SourceMsgs[0])
	}
	if alarm.SourceMsgs[0].SC1 != 5 || alarm.SourceMsgs[1].SC1 != 1 {
		t.Fatalf("unexpected source message SC1 sequence: %+v", alarm.SourceMsgs)
	}
	if !alarm.SourceMsgs[1].IsAlarm {
		t.Fatalf("older source message must be alarm, got %+v", alarm.SourceMsgs[1])
	}
}

func TestPhoenixBuildActiveAlarms_GroupUsesNewestAlarmWhenSeveralAlarmsExist(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")

	firstAlarm := time.Date(2026, time.April, 8, 13, 0, 0, 0, time.UTC)
	restore := firstAlarm.Add(1 * time.Minute)
	secondAlarm := firstAlarm.Add(2 * time.Minute)

	rows := []phoenixActiveAlarmRow{
		{
			EventID:       sql.NullInt64{Int64: 7101, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9101, Valid: true},
			PanelID:       "L00051",
			GroupNo:       2,
			TimeEvent:     sql.NullTime{Time: firstAlarm, Valid: true},
			EventCode:     sql.NullString{String: "E130", Valid: true},
			CodeMessage:   sql.NullString{String: "Перша тривога", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 1, Valid: true},
			GroupMessage:  sql.NullString{String: "Склад", Valid: true},
			GroupName:     sql.NullString{String: "Склад", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 51", Valid: true},
		},
		{
			EventID:       sql.NullInt64{Int64: 7102, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9101, Valid: true},
			PanelID:       "L00051",
			GroupNo:       2,
			TimeEvent:     sql.NullTime{Time: restore, Valid: true},
			EventCode:     sql.NullString{String: "R130", Valid: true},
			CodeMessage:   sql.NullString{String: "Відновлення", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 2, Valid: true},
			GroupMessage:  sql.NullString{String: "Склад", Valid: true},
			GroupName:     sql.NullString{String: "Склад", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 51", Valid: true},
		},
		{
			EventID:       sql.NullInt64{Int64: 7103, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9101, Valid: true},
			PanelID:       "L00051",
			GroupNo:       2,
			TimeEvent:     sql.NullTime{Time: secondAlarm, Valid: true},
			EventCode:     sql.NullString{String: "E110", Valid: true},
			CodeMessage:   sql.NullString{String: "Пожежа", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 14, Valid: true},
			GroupMessage:  sql.NullString{String: "Склад", Valid: true},
			GroupName:     sql.NullString{String: "Склад", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 51", Valid: true},
		},
	}

	alarms := provider.buildPhoenixActiveAlarms(rows)
	if len(alarms) != 1 {
		t.Fatalf("buildPhoenixActiveAlarms() returned %d alarms, want 1 grouped alarm", len(alarms))
	}

	alarm := alarms[0]
	if alarm.Type != models.AlarmFire {
		t.Fatalf("Type = %q, want %q", alarm.Type, models.AlarmFire)
	}
	if got, want := alarm.Details, "Пожежа | Склад"; got != want {
		t.Fatalf("Details = %q, want %q", got, want)
	}
	if !alarm.Time.Equal(normalizePhoenixEventTime(secondAlarm)) {
		t.Fatalf("Time = %v, want %v", alarm.Time, normalizePhoenixEventTime(secondAlarm))
	}
	if len(alarm.SourceMsgs) != 3 {
		t.Fatalf("expected 3 source messages, got %d", len(alarm.SourceMsgs))
	}
	if !alarm.SourceMsgs[0].IsAlarm {
		t.Fatalf("newest source message should be alarm, got %+v", alarm.SourceMsgs[0])
	}
}

func TestPhoenixBuildActiveAlarms_GroupKeepsFireColorWhenLatestIsFault(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")

	alarmTime := time.Date(2026, time.April, 8, 14, 0, 0, 0, time.UTC)
	faultTime := alarmTime.Add(2 * time.Minute)

	rows := []phoenixActiveAlarmRow{
		{
			EventID:       sql.NullInt64{Int64: 7201, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9201, Valid: true},
			PanelID:       "L00052",
			GroupNo:       1,
			TimeEvent:     sql.NullTime{Time: alarmTime, Valid: true},
			EventCode:     sql.NullString{String: "E110", Valid: true},
			CodeMessage:   sql.NullString{String: "Пожежа", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 14, Valid: true},
			GroupMessage:  sql.NullString{String: "Офіс", Valid: true},
			GroupName:     sql.NullString{String: "Офіс", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 52", Valid: true},
		},
		{
			EventID:       sql.NullInt64{Int64: 7202, Valid: true},
			EventParentID: sql.NullInt64{Int64: 9201, Valid: true},
			PanelID:       "L00052",
			GroupNo:       1,
			TimeEvent:     sql.NullTime{Time: faultTime, Valid: true},
			EventCode:     sql.NullString{String: "E300", Valid: true},
			CodeMessage:   sql.NullString{String: "Несправність лінії", Valid: true},
			TypeCodeID:    sql.NullInt64{Int64: 15, Valid: true},
			GroupMessage:  sql.NullString{String: "Офіс", Valid: true},
			GroupName:     sql.NullString{String: "Офіс", Valid: true},
			CompanyName:   sql.NullString{String: "Компанія 52", Valid: true},
		},
	}

	alarms := provider.buildPhoenixActiveAlarms(rows)
	if len(alarms) != 1 {
		t.Fatalf("buildPhoenixActiveAlarms() returned %d alarms, want 1 grouped alarm", len(alarms))
	}

	alarm := alarms[0]
	if alarm.Type != models.AlarmFire {
		t.Fatalf("Type = %q, want %q", alarm.Type, models.AlarmFire)
	}
	if alarm.SC1 != 1 {
		t.Fatalf("SC1 = %d, want 1 (fire color while latest is fault)", alarm.SC1)
	}
}

func TestPhoenixTimeoutMinutes(t *testing.T) {
	tests := []struct {
		name  string
		value sql.NullTime
		want  int64
	}{
		{
			name:  "zero",
			value: sql.NullTime{},
			want:  0,
		},
		{
			name: "ten minutes",
			value: sql.NullTime{
				Time:  time.Date(1900, time.January, 1, 0, 10, 0, 0, time.UTC),
				Valid: true,
			},
			want: 10,
		},
		{
			name: "two hours",
			value: sql.NullTime{
				Time:  time.Date(1900, time.January, 1, 2, 0, 0, 0, time.UTC),
				Valid: true,
			},
			want: 120,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := phoenixTimeoutMinutes(tt.value); got != tt.want {
				t.Fatalf("phoenixTimeoutMinutes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPhoenixTestControlText(t *testing.T) {
	tests := []struct {
		name  string
		value sql.NullTime
		want  string
	}{
		{
			name:  "empty",
			value: sql.NullTime{},
			want:  "",
		},
		{
			name: "minutes",
			value: sql.NullTime{
				Time:  time.Date(1900, time.January, 1, 0, 10, 0, 0, time.UTC),
				Valid: true,
			},
			want: "кожні 10 хв",
		},
		{
			name: "hours",
			value: sql.NullTime{
				Time:  time.Date(1900, time.January, 1, 2, 0, 0, 0, time.UTC),
				Valid: true,
			},
			want: "кожні 2 год",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := phoenixTestControlText(tt.value); got != tt.want {
				t.Fatalf("phoenixTestControlText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPhoenixZoneTypeText(t *testing.T) {
	if got := phoenixZoneTypeText(sql.NullBool{Bool: false, Valid: true}); got != "Охоронна" {
		t.Fatalf("expected default zone type to be охоронна, got %q", got)
	}

	if got := phoenixZoneTypeText(sql.NullBool{Bool: true, Valid: true}); got != "Тривожна кнопка" {
		t.Fatalf("expected alarm button zone type, got %q", got)
	}

	if got := phoenixZoneTypeText(sql.NullBool{}); got != "Охоронна" {
		t.Fatalf("expected invalid alarm flag to fallback to охоронна, got %q", got)
	}
}

func TestPhoenixZoneStatus(t *testing.T) {
	if got := phoenixZoneStatus(sql.NullInt64{Int64: 1, Valid: true}); got != models.ZoneNormal {
		t.Fatalf("expected status 1 to map to normal, got %q", got)
	}

	if got := phoenixZoneStatus(sql.NullInt64{Int64: 2, Valid: true}); got != models.ZoneAlarm {
		t.Fatalf("expected status 2 to map to alarm, got %q", got)
	}

	if got := phoenixZoneStatus(sql.NullInt64{}); got != models.ZoneNormal {
		t.Fatalf("expected invalid status to map to normal, got %q", got)
	}
}

func TestReversePhoenixEvents(t *testing.T) {
	t.Parallel()

	events := []models.Event{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	reversePhoenixEvents(events)

	if events[0].ID != 3 || events[1].ID != 2 || events[2].ID != 1 {
		t.Fatalf("unexpected reverse order: %+v", events)
	}
}

func TestMaxPhoenixEventID(t *testing.T) {
	t.Parallel()

	rows := []phoenixEventRow{
		{EventID: 101},
		{EventID: 150},
		{EventID: 120},
	}

	if got := maxPhoenixEventID(rows, 99); got != 150 {
		t.Fatalf("maxPhoenixEventID() = %d, want 150", got)
	}
}

func TestMapPhoenixEventRowsPreservesInputOrder(t *testing.T) {
	t.Parallel()

	rows := []phoenixEventRow{
		{EventID: 20},
		{EventID: 10},
	}

	events := mapPhoenixEventRows(rows, func(row phoenixEventRow) models.Event {
		return models.Event{ID: int(row.EventID)}
	})

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ID != 20 || events[1].ID != 10 {
		t.Fatalf("mapPhoenixEventRows() changed order: %+v", events)
	}
}

func TestPhoenixMapEventRow_NormalizesEventTimeToLocalWallClock(t *testing.T) {
	t.Parallel()

	provider := NewPhoenixDataProvider(nil, "")
	src := time.Date(2026, time.April, 6, 12, 34, 56, 123000000, time.UTC)

	event := provider.mapEventRow(phoenixEventRow{
		EventID:   77,
		PanelID:   "L00028",
		TimeEvent: src,
	})

	if event.Time.IsZero() {
		t.Fatal("expected mapped event time")
	}
	if event.Time.Location() != time.Local {
		t.Fatalf("event time location = %v, want %v", event.Time.Location(), time.Local)
	}
	if event.Time.Year() != 2026 || event.Time.Month() != time.April || event.Time.Day() != 6 {
		t.Fatalf("unexpected event date: %v", event.Time)
	}
	if event.Time.Hour() != 12 || event.Time.Minute() != 34 || event.Time.Second() != 56 {
		t.Fatalf("wall clock time must be preserved, got %v", event.Time)
	}
}
