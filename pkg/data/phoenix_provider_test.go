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
