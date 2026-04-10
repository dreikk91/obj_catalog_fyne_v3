package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNormalizeCASLBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty uses default",
			input: "",
			want:  caslDefaultBaseURL,
		},
		{
			name:  "adds http scheme",
			input: "10.32.1.221:50003",
			want:  "http://10.32.1.221:50003",
		},
		{
			name:  "trims trailing slash",
			input: "http://10.32.1.221:50003/",
			want:  "http://10.32.1.221:50003",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeCASLBaseURL(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeCASLBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCASLDevice_UnmarshalLinesObject(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"device_id":"23",
		"obj_id":"24",
		"number":1003,
		"type":"TYPE_DEVICE_CASL",
		"lines":{
			"1":{"id":1,"name":"Вхід"},
			"2":"Пожежна зона"
		}
	}`)

	var device caslDevice
	if err := json.Unmarshal(raw, &device); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(device.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(device.Lines))
	}
	if strings.TrimSpace(device.Lines[0].Name.String()) == "" {
		t.Fatalf("line[0] name must not be empty")
	}
	if strings.TrimSpace(device.Lines[1].Name.String()) != "Пожежна зона" {
		t.Fatalf("unexpected line[1] name: %q", device.Lines[1].Name.String())
	}
}

func TestCASLDevice_UnmarshalLinesArrayAliases(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"device_id":"23",
		"obj_id":"24",
		"number":1003,
		"type":"TYPE_DEVICE_CASL",
		"lines":[
			{
				"line_id":173,
				"line_number":5,
				"group_number":1,
				"adapter_type":"SYS",
				"adapter_number":0,
				"description":"Штора вікна тил",
				"line_type":"EMPTY",
				"isBlocked":true
			}
		]
	}`)

	var device caslDevice
	if err := json.Unmarshal(raw, &device); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(device.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(device.Lines))
	}
	line := device.Lines[0]
	if line.ID.Int64() != 173 {
		t.Fatalf("unexpected line id: %d", line.ID.Int64())
	}
	if line.Number.Int64() != 5 {
		t.Fatalf("unexpected line number: %d", line.Number.Int64())
	}
	if line.Name.String() != "Штора вікна тил" {
		t.Fatalf("unexpected line name: %q", line.Name.String())
	}
	if line.Type.String() != "EMPTY" {
		t.Fatalf("unexpected line type: %q", line.Type.String())
	}
	if line.AdapterType.String() != "SYS" {
		t.Fatalf("unexpected adapter type: %q", line.AdapterType.String())
	}
	if !line.IsBlocked {
		t.Fatalf("expected blocked line")
	}
}

func TestCASLProvider_ReadUsersRejectsUserWithoutID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-users","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"last_name":"Broken","phone_numbers":[{"active":true,"number":"+380501112233"}]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	_, err := provider.readUsers(context.Background())
	if err == nil || !strings.Contains(err.Error(), "user_id is required") {
		t.Fatalf("expected validation error for missing user_id, got %v", err)
	}
}

func TestCASLProvider_ReadUsers_IgnoresBrokenActivePhoneWithoutNumber(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-users-broken-phone","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"41","last_name":"Іваненко","first_name":"Іван","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"   "},{"active":true,"number":"+380671112233"}]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	users, err := provider.readUsers(context.Background())
	if err != nil {
		t.Fatalf("expected broken empty phone to be ignored, got err=%v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if len(users[0].PhoneNumbers) != 1 {
		t.Fatalf("expected 1 sanitized phone, got %d", len(users[0].PhoneNumbers))
	}
	if users[0].PhoneNumbers[0].Number != "+380671112233" {
		t.Fatalf("unexpected sanitized phone: %q", users[0].PhoneNumbers[0].Number)
	}
}

func TestCASLProvider_ReadDevicesRejectsDeviceWithoutIdentity(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-devices","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"name":"Broken device","lines":{"1":{"line_id":5,"group_number":1,"adapter_type":"SYS"}}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	_, err := provider.readDevices(context.Background())
	if err == nil || !strings.Contains(err.Error(), "device_id is required") {
		t.Fatalf("expected validation error for missing device identity, got %v", err)
	}
}

func TestCASLProvider_ReadGrdObjectsRejectsObjectWithoutObjID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-objects","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"name":"Broken object","device_id":2,"device_number":1001}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	_, err := provider.readGrdObjects(context.Background())
	if err == nil || !strings.Contains(err.Error(), "obj_id is required") {
		t.Fatalf("expected validation error for missing obj_id, got %v", err)
	}
}

func TestMapCASLPultToObject(t *testing.T) {
	t.Parallel()

	item := caslPult{
		PultID:   "123",
		Name:     "ЛОТ \"Поліс\"",
		Nickname: "CDP_1",
		Lat:      50.450398,
		Lng:      30.523644,
	}

	obj := mapCASLPultToObject(item)
	if wantID := ids.CASLObjectIDNamespaceStart + 123; obj.ID != wantID {
		t.Fatalf("unexpected ID: got %d, want %d", obj.ID, wantID)
	}
	if !ids.IsCASLObjectID(obj.ID) {
		t.Fatalf("expected CASL namespace ID, got %d", obj.ID)
	}
	if obj.Name != "ЛОТ \"Поліс\"" {
		t.Fatalf("unexpected Name: got %q", obj.Name)
	}
	if obj.ContractNum != "CDP_1" {
		t.Fatalf("unexpected ContractNum: got %q", obj.ContractNum)
	}
	if obj.Address == "" {
		t.Fatalf("expected formatted coordinates in Address")
	}
}

func TestPreferredCASLObjectNumber(t *testing.T) {
	t.Parallel()

	if got := preferredCASLObjectNumber("25", "1004 Будинок", 1004); got != "1004" {
		t.Fatalf("expected device number priority, got %q", got)
	}
	if got := preferredCASLObjectNumber("25", "1004 Будинок", 0); got != "1004" {
		t.Fatalf("expected leading number from object name, got %q", got)
	}
	if got := preferredCASLObjectNumber("25", "Будинок", 0); got != "25" {
		t.Fatalf("expected fallback to obj_id, got %q", got)
	}
}

func TestStableCASLID_Deterministic(t *testing.T) {
	t.Parallel()

	id1 := stableCASLID("1", "name", "nick")
	id2 := stableCASLID("1", "name", "nick")
	id3 := stableCASLID("2", "name", "nick")

	if id1 <= 0 {
		t.Fatalf("expected positive ID, got %d", id1)
	}
	if id1 != id2 {
		t.Fatalf("stableCASLID must be deterministic: %d != %d", id1, id2)
	}
	if id1 == id3 {
		t.Fatalf("different inputs should produce different IDs: %d == %d", id1, id3)
	}
}

func TestDecodeCASLDeviceType(t *testing.T) {
	t.Parallel()

	if got := decodeCASLDeviceType("TYPE_DEVICE_Dunay_4L"); got != "Дунай-4L" {
		t.Fatalf("unexpected mapped type: %q", got)
	}
	if got := decodeCASLDeviceType("TYPE_DEVICE_Ajax_SIA"); got != "Ajax(SIA)" {
		t.Fatalf("unexpected mapped type: %q", got)
	}
	if got := decodeCASLDeviceType("UNKNOWN_TYPE"); got != "UNKNOWN_TYPE" {
		t.Fatalf("unexpected fallback type: %q", got)
	}
}

func TestMapCASLObjectStatusState(t *testing.T) {
	t.Parallel()

	offline := mapCASLObjectStatusState("Немає зв'язку", false)
	if offline.Status != models.StatusOffline || offline.IsConnState != 0 {
		t.Fatalf("unexpected offline state: %+v", offline)
	}

	alarm := mapCASLObjectStatusState("Тривога в зоні", false)
	if alarm.Status != models.StatusFire || alarm.AlarmState != 1 {
		t.Fatalf("unexpected alarm state: %+v", alarm)
	}

	disarmed := mapCASLObjectStatusState("Виключено", false)
	if disarmed.IsUnderGuard || disarmed.GuardState != 0 {
		t.Fatalf("unexpected disarmed state: %+v", disarmed)
	}

	blocked := mapCASLObjectStatusState("Включено", true)
	if blocked.StatusText != "ЗАБЛОКОВАНО" || blocked.TechAlarmState != 1 {
		t.Fatalf("unexpected blocked state: %+v", blocked)
	}
}

func TestMapCASLDeviceGroupsToObjectGroups(t *testing.T) {
	t.Parallel()

	rawGroups := map[string]any{
		"1": map[string]any{
			"state":   "GROUP_ON",
			"room_id": "10",
		},
		"2": map[string]any{
			"is_on":     0,
			"room_name": "Склад",
		},
	}

	rooms := []caslRoom{
		{RoomID: "10", Name: "Офіс"},
	}

	groups := mapCASLDeviceGroupsToObjectGroups(rawGroups, rooms)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Number != 1 || !groups[0].Armed || groups[0].RoomName != "Офіс" {
		t.Fatalf("unexpected first group: %+v", groups[0])
	}
	if groups[1].Number != 2 || groups[1].Armed || groups[1].RoomName != "Склад" {
		t.Fatalf("unexpected second group: %+v", groups[1])
	}
}

func TestMapCASLDeviceGroupsToObjectGroups_NestedContainer(t *testing.T) {
	t.Parallel()

	rawGroups := map[string]any{
		"groups": map[string]any{
			"1": map[string]any{
				"state":   "GROUP_ON",
				"room_id": "10",
			},
			"2": map[string]any{
				"state":   "group_off",
				"room_id": "20",
			},
		},
	}

	rooms := []caslRoom{
		{RoomID: "10", Name: "Офіс"},
		{RoomID: "20", Name: "Склад"},
	}

	groups := mapCASLDeviceGroupsToObjectGroups(rawGroups, rooms)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Number != 1 || !groups[0].Armed || groups[0].RoomName != "Офіс" {
		t.Fatalf("unexpected first group: %+v", groups[0])
	}
	if groups[1].Number != 2 || groups[1].Armed || groups[1].RoomName != "Склад" {
		t.Fatalf("unexpected second group: %+v", groups[1])
	}
}

func TestAlignCASLGroupsWithDeviceLines_UsesLineGroupNumberInsteadOfRoomID(t *testing.T) {
	t.Parallel()

	groups := []models.ObjectGroup{
		{
			ID:        "casl:group=25",
			Source:    "casl",
			Number:    25,
			Armed:     true,
			StateText: "ПІД ОХОРОНОЮ",
			RoomID:    "25",
			RoomName:  "Будинок",
		},
	}

	lines := []caslDeviceLine{
		{
			ID:          1,
			Number:      1,
			Name:        "Вхід",
			GroupNumber: 5,
			RoomID:      "25",
		},
	}

	rooms := []caslRoom{{RoomID: "25", Name: "Будинок"}}

	aligned := alignCASLGroupsWithDeviceLines(groups, lines, rooms)
	if len(aligned) != 1 {
		t.Fatalf("expected 1 group after alignment, got %d (%+v)", len(aligned), aligned)
	}
	if aligned[0].Number != 5 {
		t.Fatalf("expected group number 5, got %+v", aligned[0])
	}
	if aligned[0].RoomID != "25" || aligned[0].RoomName != "Будинок" {
		t.Fatalf("expected room context to be preserved, got %+v", aligned[0])
	}
}

func TestAlignCASLGroupsWithDeviceLines_MultiGroupRoomAndSharedGroupAcrossRooms(t *testing.T) {
	t.Parallel()

	groups := []models.ObjectGroup{
		{
			ID:        "casl:group=25",
			Source:    "casl",
			Number:    25,
			Armed:     true,
			StateText: "ПІД ОХОРОНОЮ",
			RoomID:    "25",
			RoomName:  "Будинок",
		},
		{
			ID:        "casl:group=26",
			Source:    "casl",
			Number:    26,
			Armed:     false,
			StateText: "ЗНЯТО",
			RoomID:    "26",
			RoomName:  "Гараж",
		},
	}

	lines := []caslDeviceLine{
		{ID: 1, Number: 1, Name: "Вхід", GroupNumber: 5, RoomID: "25"},
		{ID: 2, Number: 2, Name: "Рух", GroupNumber: 6, RoomID: "25"},
		{ID: 3, Number: 3, Name: "Склад", GroupNumber: 5, RoomID: "26"},
	}

	rooms := []caslRoom{
		{RoomID: "25", Name: "Будинок"},
		{RoomID: "26", Name: "Гараж"},
	}

	aligned := alignCASLGroupsWithDeviceLines(groups, lines, rooms)
	if len(aligned) != 2 {
		t.Fatalf("expected 2 logical groups after alignment, got %d (%+v)", len(aligned), aligned)
	}
	if aligned[0].Number != 5 {
		t.Fatalf("expected first logical group number 5, got %+v", aligned[0])
	}
	if aligned[1].Number != 6 {
		t.Fatalf("expected second logical group number 6, got %+v", aligned[1])
	}
}

func TestNormalizeCASLGroupStatistics(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"groupStatistics": map[string]any{
			"2010": map[string]any{
				"1": float64(0),
				"5": float64(1),
			},
		},
	}

	got := normalizeCASLGroupStatistics(raw)
	if len(got) != 1 {
		t.Fatalf("expected 1 object in stats, got %d", len(got))
	}
	if got["2010"][1] != 0 || got["2010"][5] != 1 {
		t.Fatalf("unexpected normalized stats: %+v", got)
	}
}

func TestMergeCASLGroupsWithStatistics_CreatesGroupsFromStatistic(t *testing.T) {
	t.Parallel()

	groups := mergeCASLGroupsWithStatistics(nil, map[int]int{
		5: 1,
		6: 0,
	})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups from statistics, got %d", len(groups))
	}
	if groups[0].Number != 5 || !groups[0].Armed || groups[0].StateText != "ПІД ОХОРОНОЮ" {
		t.Fatalf("unexpected first group: %+v", groups[0])
	}
	if groups[1].Number != 6 || groups[1].Armed || groups[1].StateText != "ЗНЯТО" {
		t.Fatalf("unexpected second group: %+v", groups[1])
	}
}

func TestMapCASLRealtimeRow_UserActionObjectOnly(t *testing.T) {
	t.Parallel()

	source := map[string]any{
		"type":       "user_action",
		"action":     "GRD_OBJ_NOTIF",
		"obj_id":     "25",
		"obj_name":   "1004 Будинок Хіміч Н.П.",
		"alarm_type": "ALARM_TYPE_OPERATOR",
		"time":       float64(1774788999196),
	}

	row, ok := mapCASLRealtimeRow(source, "")
	if !ok {
		t.Fatal("expected row to be parsed")
	}
	if row.ObjID != "25" {
		t.Fatalf("unexpected obj id: %q", row.ObjID)
	}
	if row.Code != "GRD_OBJ_NOTIF" {
		t.Fatalf("unexpected code: %q", row.Code)
	}
	if row.Type != "user_action" {
		t.Fatalf("unexpected type: %q", row.Type)
	}
	if row.Time <= 0 {
		t.Fatalf("expected unix ms time, got %d", row.Time)
	}
}

func TestMapCASLRealtimeRow_RejectsMissingTime(t *testing.T) {
	t.Parallel()

	source := map[string]any{
		"type":     "user_action",
		"action":   "GRD_OBJ_NOTIF",
		"obj_id":   "25",
		"obj_name": "1004 Будинок Хіміч Н.П.",
	}

	if row, ok := mapCASLRealtimeRow(source, ""); ok {
		t.Fatalf("expected row without time to be rejected, got %+v", row)
	}
}

func TestExtractCASLRealtimeRows_WrappedDataArray(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"tag":"ppk_in",
		"data":[
			{
				"ppk_num":1003,
				"obj_id":"25",
				"action":"GROUP_ON",
				"time":1774788999196,
				"line_number":5
			}
		]
	}`)

	rows := extractCASLRealtimeRows(raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Type != "ppk_in" {
		t.Fatalf("unexpected type: %q", rows[0].Type)
	}
	if rows[0].Number != 5 {
		t.Fatalf("unexpected line number: %d", rows[0].Number)
	}
}

func TestExtractCASLRealtimeRows_IgnoresUnknownNestedMaps(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"tag":"ppk_in",
		"meta":{
			"obj_id":"25",
			"action":"GROUP_ON",
			"time":1774788999196
		}
	}`)

	rows := extractCASLRealtimeRows(raw)
	if len(rows) != 0 {
		t.Fatalf("expected no rows from unknown nested meta payload, got %+v", rows)
	}
}

func TestExtractCASLRealtimeRows_RejectsPrefixedGarbage(t *testing.T) {
	t.Parallel()

	raw := []byte(`debug {"tag":"ppk_in","data":[{"ppk_num":1003,"obj_id":"25","action":"GROUP_ON","time":1774788999196}]}`)

	rows := extractCASLRealtimeRows(raw)
	if len(rows) != 0 {
		t.Fatalf("expected no rows from prefixed garbage payload, got %+v", rows)
	}
}

func TestExtractCASLRealtimeConnID_TextEnvelope(t *testing.T) {
	t.Parallel()

	raw := []byte(`{conn_id: "4cf37e81ec99fafe2ee28d0044f12e9a19d6c539c96", tag: "ppk_in"}`)
	got := extractCASLRealtimeConnID(raw)
	if got != "4cf37e81ec99fafe2ee28d0044f12e9a19d6c539c96" {
		t.Fatalf("unexpected conn_id: %q", got)
	}
}

func TestExtractCASLRealtimeConnID_IgnoresEmbeddedDebugText(t *testing.T) {
	t.Parallel()

	raw := []byte(`debug conn_id=4cf37e81ec99fafe2ee28d0044f12e9a19d6c539c96`)
	got := extractCASLRealtimeConnID(raw)
	if got != "" {
		t.Fatalf("expected empty conn_id for embedded debug text, got %q", got)
	}
}

func TestBuildCASLUserActionDetails(t *testing.T) {
	t.Parallel()

	details := buildCASLUserActionDetails(CASLObjectEvent{
		Action:    "GRD_OBJ_NOTIF",
		ObjID:     "25",
		ObjName:   "1004 Будинок Хіміч Н.П.",
		AlarmType: "ALARM_TYPE_OPERATOR",
	}, nil)
	if details != "Попадання тривоги в стрічку" {
		t.Fatalf("unexpected notif details: %q", details)
	}

	details = buildCASLUserActionDetails(CASLObjectEvent{
		Action:  "GRD_OBJ_PICK",
		UserFIO: "Островська Марина",
	}, nil)
	if !strings.Contains(details, "Взяття в роботу об'єкта") || !strings.Contains(details, "Островська") {
		t.Fatalf("unexpected pick details: %q", details)
	}

	details = buildCASLUserActionDetails(CASLObjectEvent{
		Action: "GRD_OBJ_FINISH",
		Cause:  "CAUSES_FALSE_ALARM",
		Note:   "Хибний виклик",
	}, map[string]string{"CAUSES_FALSE_ALARM": "Хибна тривога"})
	if !strings.Contains(details, "Завершення відпрацювання тривоги") ||
		!strings.Contains(details, "Причина: Хибна тривога") ||
		!strings.Contains(details, "Примітка: Хибний виклик") {
		t.Fatalf("unexpected finish details: %q", details)
	}

	details = buildCASLUserActionDetails(CASLObjectEvent{
		Action:       "DEVICE_BLOCK",
		BlockMessage: "Сервісні роботи",
		TimeUnblock:  time.Now().Add(2 * time.Hour).Unix(),
	}, nil)
	if !strings.Contains(details, "Блокування ППК") ||
		!strings.Contains(details, "Причина: Сервісні роботи") ||
		!strings.Contains(details, "До:") {
		t.Fatalf("unexpected block details: %q", details)
	}
}

func TestNormalizeCASLObjectEvent_UserActionAliases(t *testing.T) {
	t.Parallel()

	got := normalizeCASLObjectEvent(caslObjectEvent{
		PPKNum:     caslInt64(1003),
		ObjID:      caslText("25"),
		DictName:   caslText("GRD_OBJ_PICK"),
		UserAction: caslText("grd_object_action"),
		Type:       "user_action",
		UserFIO:    caslText("Островська Марина"),
		Time:       caslInt64(1774790159852),
	})

	if got.Action != "GRD_OBJ_PICK" {
		t.Fatalf("unexpected action: %q", got.Action)
	}
	if got.Code != "GRD_OBJ_PICK" {
		t.Fatalf("unexpected code: %q", got.Code)
	}
	if got.Type != "user_action" {
		t.Fatalf("unexpected type: %q", got.Type)
	}
	if got.UserActionType != "grd_object_action" {
		t.Fatalf("unexpected user action type: %q", got.UserActionType)
	}
}

func TestMapCASLObjectEvents_UserActionAliasDetails(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceNumber: caslInt64(1003),
	}

	events := provider.mapCASLObjectEvents(context.Background(), record, []caslObjectEvent{
		{
			PPKNum:     caslInt64(1003),
			ObjID:      caslText("25"),
			DictName:   caslText("GRD_OBJ_PICK"),
			UserAction: caslText("grd_object_action"),
			Type:       "user_action",
			UserFIO:    caslText("Островська Марина"),
			Time:       caslInt64(1774790159852),
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !strings.Contains(events[0].Details, "Взяття в роботу об'єкта") {
		t.Fatalf("unexpected details: %q", events[0].Details)
	}
	if strings.Contains(events[0].Details, "src=user_action") {
		t.Fatalf("user_action source noise must not be shown: %q", events[0].Details)
	}
}

func TestMapCASLObjectEvents_PPKDetailsMatchOriginalCASL(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1003),
	}
	device := caslDevice{
		DeviceID: caslText("23"),
		ObjID:    caslText("25"),
		Number:   caslInt64(1003),
		Type:     caslText("TYPE_DEVICE_CASL"),
		Lines: []caslDeviceLine{
			{
				ID:            caslInt64(2),
				Number:        caslInt64(2),
				Name:          caslText("Кнопка в касі"),
				Description:   caslText("Кнопка в касі"),
				LineType:      caslText("ZONE_ALARM_ON_KZ"),
				AdapterType:   caslText("SYS"),
				AdapterNumber: caslInt64(0),
			},
		},
	}

	provider.mu.Lock()
	provider.deviceByDeviceID["23"] = device
	provider.deviceByObjectID["25"] = device
	provider.deviceByNumber[1003] = device
	provider.cachedDevicesAt = time.Now()
	provider.cachedUsers["41"] = caslUser{
		UserID:     "41",
		LastName:   "Іваненко",
		FirstName:  "Іван",
		MiddleName: "Іванович",
	}
	provider.cachedUsersAt = time.Now()
	provider.cachedDictionary = map[string]any{
		"translate": map[string]any{
			"uk": map[string]any{
				"E130":               "Тривога в зоні № {number}",
				"ZONE_ALARM_ON_KZ":   "Тривожний шлейф",
				"CAUSES_FALSE_ALARM": "Хибна тривога",
			},
		},
		"dictionary": map[string]any{
			"translate": map[string]any{
				"uk": map[string]any{
					"E130":             "Тривога в зоні № {number}",
					"ZONE_ALARM_ON_KZ": "Тривожний шлейф",
				},
			},
		},
	}
	provider.cachedDictionaryAt = time.Now()
	provider.mu.Unlock()

	events := provider.mapCASLObjectEvents(context.Background(), record, []caslObjectEvent{
		{
			PPKNum:    caslInt64(1003),
			DeviceID:  caslText("23"),
			ObjID:     caslText("25"),
			Code:      caslText("LINE_BAD"),
			Type:      "ppk_event",
			Number:    caslInt64(2),
			ContactID: caslText("E130"),
			HozUserID: caslText("41"),
			Time:      caslInt64(1774790159852),
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != models.EventPanic {
		t.Fatalf("expected panic event after ATTACK remap, got %s", events[0].Type)
	}
	if !strings.Contains(events[0].Details, "Напад № 2") {
		t.Fatalf("expected ATTACK remap in details, got %q", events[0].Details)
	}
	if !strings.Contains(events[0].Details, "Опис: Кнопка в касі") {
		t.Fatalf("expected line description in details, got %q", events[0].Details)
	}
	if !strings.Contains(events[0].Details, "(E130)") {
		t.Fatalf("expected contact id in details, got %q", events[0].Details)
	}
	if !strings.Contains(events[0].Details, "Користувач: Іваненко Іван Іванович") {
		t.Fatalf("expected hoz user in details, got %q", events[0].Details)
	}
	if !strings.Contains(events[0].Details, "Адаптер: SYS") {
		t.Fatalf("expected adapter type in details, got %q", events[0].Details)
	}
	if !strings.Contains(events[0].Details, "Тип: Тривожний шлейф") {
		t.Fatalf("expected line type in details, got %q", events[0].Details)
	}
}

func TestNormalizeCASLObjectEvent_MgrActionTypeUsesMgrSubtype(t *testing.T) {
	t.Parallel()

	got := normalizeCASLObjectEvent(caslObjectEvent{
		PPKNum:     caslInt64(1003),
		ObjID:      caslText("25"),
		UserAction: caslText("mgr_action"),
		MgrAction:  caslText("GRD_OBJ_MGR_ARRIVE"),
		Type:       "user_action",
		Time:       caslInt64(1774790159852),
	})

	if got.Action != "GRD_OBJ_MGR_ARRIVE" {
		t.Fatalf("unexpected action: %q", got.Action)
	}
	if got.Code != "GRD_OBJ_MGR_ARRIVE" {
		t.Fatalf("unexpected code: %q", got.Code)
	}
	if got.UserActionType != "mgr_action" {
		t.Fatalf("unexpected user action type: %q", got.UserActionType)
	}
	if got.MgrActionType != "GRD_OBJ_MGR_ARRIVE" {
		t.Fatalf("unexpected mgr action type: %q", got.MgrActionType)
	}
}

func TestMapCASLObjectEvents_UserActionUnknownFallback(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceNumber: caslInt64(1003),
	}

	events := provider.mapCASLObjectEvents(context.Background(), record, []caslObjectEvent{
		{
			PPKNum:  caslInt64(1003),
			ObjID:   caslText("25"),
			Code:    caslText("не встановлено"),
			Type:    "user_action",
			Time:    caslInt64(1774790159852),
			UserID:  caslText("483"),
			UserFIO: caslText("Оператор"),
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != models.EventOperatorAction {
		t.Fatalf("unexpected type: %s", events[0].Type)
	}
	if events[0].GetTypeDisplay() == "НЕСПРАВНІСТЬ" {
		t.Fatalf("unexpected fallback type display: %q", events[0].GetTypeDisplay())
	}
	if strings.Contains(strings.ToLower(events[0].Details), "не встановлено") {
		t.Fatalf("unexpected raw unknown details: %q", events[0].Details)
	}
	if strings.Contains(events[0].Details, "src=user_action") {
		t.Fatalf("unexpected source suffix: %q", events[0].Details)
	}
}

func TestClassifyCASLEventTypeWithContext_UserActionFallback(t *testing.T) {
	t.Parallel()

	got := classifyCASLEventTypeWithContext("", "", "user_action", "не встановлено")
	if got != models.EventOperatorAction {
		t.Fatalf("expected EventOperatorAction fallback for user_action, got %s", got)
	}
}

func TestClassifyCASLEventTypeWithContext_UserActionByCode(t *testing.T) {
	t.Parallel()

	got := classifyCASLEventTypeWithContext("GRD_OBJ_NOTIF", "", "user_action", "")
	if got != models.EventAlarmNotification {
		t.Fatalf("expected EventAlarmNotification for GRD_OBJ_NOTIF, got %s", got)
	}
}

func TestClassifyCASLEventType_CASLCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code     string
		expected models.EventType
	}{
		{code: "FIRE_ALARM", expected: models.EventFire},
		{code: "BURGLARY_ALARM", expected: models.EventBurglary},
		{code: "ALM_IO", expected: models.EventBurglary},
		{code: "PANIC_ALARM", expected: models.EventPanic},
		{code: "MEDICAL_ALARM", expected: models.EventMedical},
		{code: "GAS_ALARM", expected: models.EventGas},
		{code: "SABOTAGE_AD", expected: models.EventTamper},
		{code: "NO_220", expected: models.EventPowerFail},
		{code: "OO_OK_220", expected: models.EventPowerOK},
		{code: "GRD_OBJ_PICK", expected: models.EventOperatorAction},
		{code: "GRD_OBJ_ASS_MGR", expected: models.EventManagerAssigned},
		{code: "GRD_OBJ_MGR_ARRIVE", expected: models.EventManagerArrived},
		{code: "GRD_OBJ_FINISH", expected: models.EventAlarmFinished},
		{code: "DEVICE_BLOCK", expected: models.EventDeviceBlocked},
		{code: "DEVICE_UNBLOCK", expected: models.EventDeviceUnblocked},
		{code: "PPK_FW_VERSION", expected: models.EventService},
		{code: "UPD_START", expected: models.SystemEvent},
		{code: "E627", expected: models.SystemEvent},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			got := classifyCASLEventType(tc.code)
			if got != tc.expected {
				t.Fatalf("unexpected type for %s: got=%s want=%s", tc.code, got, tc.expected)
			}
		})
	}
}

func TestClassifyCASLEventType_CASLAlarmRestoreCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected models.EventType
	}{
		{name: "fire finish", code: "FIRE_ALARM_FINISH", expected: models.EventRestore},
		{name: "gas finish", code: "GAS_ALARM_FINISH", expected: models.EventRestore},
		{name: "medical finish", code: "MEDICAL_ALARM_FINISH", expected: models.EventRestore},
		{name: "medical finish new", code: "MEDICAL_ALARM_FINISH_NEW", expected: models.EventRestore},
		{name: "water leak finish", code: "WATER_LEAK_FINISH", expected: models.EventRestore},
		{name: "heat restore", code: "HEAT_ALARM_RESTORE", expected: models.EventRestore},
		{name: "water restore", code: "WATER_ALARM_RES", expected: models.EventRestore},
		{name: "panic button release", code: "ALM_BTN_RLZ", expected: models.EventRestore},
		{name: "co restore", code: "CO_OKEY", expected: models.EventRestore},
		{name: "tamper norm", code: "SENS_TAMP_N", expected: models.EventRestore},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classifyCASLEventType(tc.code)
			if got != tc.expected {
				t.Fatalf("unexpected type for %s: got=%s want=%s", tc.code, got, tc.expected)
			}
		})
	}
}

func TestClassifyCASLEventType_CASLNonAlarmBeforeBroadHeuristics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected models.EventType
	}{
		{name: "fire trouble restore", code: "FIRE_TROUBLE_RESTORE", expected: models.EventRestore},
		{name: "fire test end", code: "FIRE_TEST_END", expected: models.EventTest},
		{name: "gas trouble", code: "GAS_TROUBLE", expected: models.EventFault},
		{name: "medical trouble", code: "MED_TROUBLE", expected: models.EventFault},
		{name: "sprinkler alarm restore", code: "SPRIN_ALARM_RES", expected: models.EventRestore},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classifyCASLEventType(tc.code)
			if got != tc.expected {
				t.Fatalf("unexpected type for %s: got=%s want=%s", tc.code, got, tc.expected)
			}
		})
	}
}

func TestClassifyCASLEventType_PeriodicPollResponseIsTest(t *testing.T) {
	t.Parallel()

	const details = "Відповідь на опитування - норма шлейфа № 7"

	if got := classifyCASLEventType(details); got != models.EventTest {
		t.Fatalf("expected EventTest for periodic poll response details, got %s", got)
	}

	got := classifyCASLEventTypeWithContext("61184", "", "ppk_event", details)
	if got != models.EventTest {
		t.Fatalf("expected EventTest for periodic poll response context, got %s", got)
	}
}

func TestClassifyCASLEventTypeWithContext_E134IOAlarmIsBurglary(t *testing.T) {
	t.Parallel()

	got := classifyCASLEventTypeWithContext("", "E134", "ppk_event", "Тривога IO")
	if got != models.EventBurglary {
		t.Fatalf("expected EventBurglary for E134 IO alarm, got %s", got)
	}
}

func TestClassifyCASLEventTypeWithContext_FaultSourceStillUsesE134Override(t *testing.T) {
	t.Parallel()

	got := classifyCASLEventTypeWithContext("", "E134", "fault", "Тривога IO")
	if got != models.EventBurglary {
		t.Fatalf("expected EventBurglary for fault source with E134, got %s", got)
	}
}

func TestMapEventTypeToAlarmType_CASLCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType models.EventType
		alarmType models.AlarmType
	}{
		{eventType: models.EventBurglary, alarmType: models.AlarmBurglary},
		{eventType: models.EventPanic, alarmType: models.AlarmPanic},
		{eventType: models.EventMedical, alarmType: models.AlarmMedical},
		{eventType: models.EventGas, alarmType: models.AlarmGas},
		{eventType: models.EventTamper, alarmType: models.AlarmTamper},
	}

	for _, tc := range tests {
		alarmType, ok := mapEventTypeToAlarmType(tc.eventType)
		if !ok {
			t.Fatalf("expected alarm mapping for %s", tc.eventType)
		}
		if alarmType != tc.alarmType {
			t.Fatalf("unexpected alarm type for %s: got=%s want=%s", tc.eventType, alarmType, tc.alarmType)
		}
	}
}

func TestMapCASLAlarmType_GRDObjectNotifVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want models.AlarmType
	}{
		{raw: string(models.AlarmOperator), want: models.AlarmOperator},
		{raw: string(models.AlarmDevice), want: models.AlarmDevice},
		{raw: string(models.AlarmMobile), want: models.AlarmMobile},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			got, ok := mapCASLAlarmType(tc.raw)
			if !ok {
				t.Fatalf("expected mapping for %s", tc.raw)
			}
			if got != tc.want {
				t.Fatalf("mapCASLAlarmType(%q) = %s, want %s", tc.raw, got, tc.want)
			}
		})
	}
}

func TestMapCASLEventSC1_ExtendedCASLTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType models.EventType
		want      int
	}{
		{eventType: models.EventAlarmNotification, want: 1},
		{eventType: models.EventOperatorAction, want: 30},
		{eventType: models.EventManagerAssigned, want: 30},
		{eventType: models.EventManagerArrived, want: 28},
		{eventType: models.EventManagerCanceled, want: 30},
		{eventType: models.EventAlarmFinished, want: 5},
		{eventType: models.EventDeviceBlocked, want: 29},
		{eventType: models.EventDeviceUnblocked, want: 28},
		{eventType: models.EventService, want: 30},
	}

	for _, tc := range tests {
		if got := mapCASLEventSC1(tc.eventType); got != tc.want {
			t.Fatalf("unexpected SC1 for %s: got=%d want=%d", tc.eventType, got, tc.want)
		}
	}
}

func TestExtractCASLTranslatorByType_NestedPayload(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"TYPE_DEVICE_Ajax": map[string]any{
			"E130": map[string]any{
				"msg":      "Тривога проникнення",
				"isAlarm":  1,
				"priority": 5,
			},
			"R402": map[string]any{
				"message": "Взяття групи № {number}",
			},
			"events": []any{
				map[string]any{
					"contact_id": "E301",
					"text":       "Втрата живлення 220В",
				},
			},
		},
	}

	got := extractCASLTranslatorByType(raw, "TYPE_DEVICE_Ajax")
	if got == nil {
		t.Fatal("expected non-nil translator map")
	}
	if got["E130"] != "Тривога проникнення" {
		t.Fatalf("unexpected E130 mapping: %q", got["E130"])
	}
	if got["R402"] != "Взяття групи № {number}" {
		t.Fatalf("unexpected R402 mapping: %q", got["R402"])
	}
	if got["E301"] != "Втрата живлення 220В" {
		t.Fatalf("unexpected E301 mapping from list payload: %q", got["E301"])
	}
}

func TestExtractCASLTranslatorByType_CodeAndTypeEventPayload(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"MAKS_PRO": []any{
			map[string]any{
				"code":      130,
				"typeEvent": "E",
				"name":      "ZONE_ALM",
				"isAlarm":   1,
			},
			map[string]any{
				"code":      130,
				"typeEvent": "R",
				"name":      "ZONE_NORM",
				"isAlarm":   0,
			},
			map[string]any{
				"code":      301,
				"typeEvent": "E",
				"name":      "NO_220",
				"isAlarm":   1,
			},
		},
	}

	got := extractCASLTranslatorByType(raw, "MAKS_PRO")
	if got == nil {
		t.Fatal("expected non-nil translator map")
	}
	if got["E130"] != "ZONE_ALM" {
		t.Fatalf("unexpected E130 mapping: %q", got["E130"])
	}
	if got["R130"] != "ZONE_NORM" {
		t.Fatalf("unexpected R130 mapping: %q", got["R130"])
	}
	if got["E301"] != "NO_220" {
		t.Fatalf("unexpected E301 mapping: %q", got["E301"])
	}
}

func TestExtractCASLTranslatorAlarmFlagsByType_CodeAndTypeEventPayload(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"SATEL": []any{
			map[string]any{
				"code":      152,
				"typeEvent": "E",
				"name":      "REFRIGERATION_ALARM",
				"isAlarm":   1,
			},
			map[string]any{
				"code":      152,
				"typeEvent": "R",
				"name":      "REFRIGERATION_RESTORE",
				"isAlarm":   0,
			},
		},
	}

	got := extractCASLTranslatorAlarmFlagsByType(raw, "SATEL")
	if got == nil {
		t.Fatal("expected non-nil alarm flags")
	}
	if !got["E152"] {
		t.Fatalf("expected E152 to be alarm, got %+v", got)
	}
	if got["R152"] {
		t.Fatalf("expected R152 to be non-alarm, got %+v", got)
	}
}

func TestGetMessageTranslatorByDeviceType_FallbackParamName(t *testing.T) {
	t.Parallel()

	requests := make([]map[string]any, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != caslCommandPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		requests = append(requests, payload)

		w.Header().Set("Content-Type", "application/json")
		if strings.TrimSpace(asString(payload["typeDevice"])) != "" {
			_, _ = w.Write([]byte(`{"status":"error","error":"WRONG_FORMAT"}`))
			return
		}
		if strings.TrimSpace(asString(payload["device_type"])) != "MAKS_PRO" {
			t.Fatalf("expected fallback payload with device_type=MAKS_PRO, got: %v", payload)
		}
		_, _ = w.Write([]byte(`{"status":"ok","data":{"R401":"Взяття групи № {number}"}}`))
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := provider.GetMessageTranslatorByDeviceType(ctx, "MAKS_PRO")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", raw)
	}
	if strings.TrimSpace(asString(data["R401"])) != "Взяття групи № {number}" {
		t.Fatalf("unexpected translator value: %+v", data)
	}

	if len(requests) != 2 {
		t.Fatalf("expected 2 requests (typeDevice + fallback), got %d", len(requests))
	}
	if strings.TrimSpace(asString(requests[0]["typeDevice"])) != "MAKS_PRO" {
		t.Fatalf("expected first request to use typeDevice, got: %v", requests[0])
	}
	if strings.TrimSpace(asString(requests[1]["device_type"])) != "MAKS_PRO" {
		t.Fatalf("expected second request to use device_type, got: %v", requests[1])
	}
}

func TestDecodeCASLEventDescription_ContactIDFallback(t *testing.T) {
	t.Parallel()

	if got := decodeCASLEventDescription(nil, nil, "", "E130", 2); got != "Тривога проникнення" {
		t.Fatalf("unexpected E130 fallback: %q", got)
	}
	if got := decodeCASLEventDescription(nil, nil, "", "R402", 4); got != "Взяття групи № 4" {
		t.Fatalf("unexpected R402 fallback with zone substitution: %q", got)
	}
	if got := decodeCASLEventDescription(nil, nil, "", "E627", 0); got != "Старт процесу оновлення чи застосування нових налаштувань" {
		t.Fatalf("unexpected E627 fallback: %q", got)
	}
	if got := decodeCASLEventDescription(nil, nil, "", "R627", 0); got != "Старт процесу оновлення чи застосування нових налаштувань" {
		t.Fatalf("unexpected R627 fallback: %q", got)
	}
	if got := decodeCASLEventDescription(nil, nil, "61184", "", 7); got != "Відповідь на опитування - норма шлейфа № 7" {
		t.Fatalf("unexpected 61184 fallback: %q", got)
	}
	if got := decodeCASLEventDescription(nil, nil, "", "R999", 0); got != "Відновлення ContactID R999" {
		t.Fatalf("unexpected generic restore fallback: %q", got)
	}
}

func TestDecodeCASLEventDescription_PrefersCodeOverContactID(t *testing.T) {
	t.Parallel()

	translator := map[string]string{
		"E400":  "Загальна подія E400",
		"16145": "Тривога тривожна кнопка радіобрелок",
	}

	got := decodeCASLEventDescription(translator, nil, "16145", "E400", 0)
	if got != "Тривога тривожна кнопка радіобрелок" {
		t.Fatalf("expected code-priority translation, got: %q", got)
	}
}

func TestDecodeCASLEventDescription_ProtocolCodesFromJournal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		contact  string
		number   int
		expected string
	}{
		{name: "group on", code: "16400", contact: "E400", number: 1, expected: "Постановка групи 1"},
		{name: "group off", code: "18449", contact: "E400", number: 2, expected: "Зняття групи № 2"},
		{name: "zone alarm", code: "62209", contact: "E130", number: 1, expected: "Тривога в зоні № 1"},
		{name: "zone normal", code: "62977", contact: "R130", number: 1, expected: "Норма в зоні № 1"},
		{name: "inner zone alarm", code: "64770", contact: "E132", number: 2, expected: "Тривога внутрішньої зони № 2"},
		{name: "inner zone normal", code: "65026", contact: "R132", number: 2, expected: "Норма внутрішньої зони № 2"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decodeCASLEventDescription(nil, nil, tc.code, tc.contact, tc.number, "TYPE_DEVICE_Dunay_4L")
			if got != tc.expected {
				t.Fatalf("unexpected decode for code=%s: got=%q want=%q", tc.code, got, tc.expected)
			}
		})
	}
}

func TestDecodeCASLEventDescription_ProtocolCodeOverContactFallback(t *testing.T) {
	t.Parallel()

	got := decodeCASLEventDescription(nil, nil, "16400", "E400", 1, "TYPE_DEVICE_Dunay_4L")
	if got != "Постановка групи 1" {
		t.Fatalf("expected protocol decode over generic contact fallback, got: %q", got)
	}
}

func TestDecodeCASLEventDescription_UsesCanonicalUkrainianMessageKeyTemplate(t *testing.T) {
	t.Parallel()

	dict := map[string]string{
		"GROUP_ON": "Armed group",
	}

	got := decodeCASLEventDescription(nil, dict, "16400", "", 1, "TYPE_DEVICE_Dunay_4L")
	if got != "Постановка групи 1" {
		t.Fatalf("unexpected decode from canonical template: %q", got)
	}
}

func TestCASLBodyForDebugLog_PrettyAndMasked(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"type":"read_grd_object","token":"secret-token","nested":{"pwd":"x"}}`)
	formatted := caslBodyForDebugLog(raw)

	if !strings.Contains(formatted, "\n") {
		t.Fatalf("expected pretty json with new lines, got: %q", formatted)
	}
	if strings.Contains(formatted, "secret-token") {
		t.Fatalf("token must be masked, got: %q", formatted)
	}
	if strings.Contains(formatted, "\"pwd\": \"x\"") {
		t.Fatalf("pwd must be masked, got: %q", formatted)
	}
	if !strings.Contains(formatted, "\"token\": \"***\"") {
		t.Fatalf("expected masked token field, got: %q", formatted)
	}
}

func TestCASLProvider_LoginAndReadObjects(t *testing.T) {
	t.Parallel()

	var loginCalls int
	var commandCalls int
	lastPing := int64(1767706195359)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			loginCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-1","user_id":"u-1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			commandCalls++
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			if strings.TrimSpace(asString(payload["token"])) != "token-1" {
				t.Fatalf("expected token-1, got %v", payload["token"])
			}
			w.Header().Set("Content-Type", "application/json")
			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr","device_id":"23","device_number":1003,"rooms":[{"room_id":"1","name":"Room A","description":"Desc","rtsp":""}]}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"name":"Ломбард","type":"TYPE_DEVICE_CASL","timeout":3600,"lastPingDate":1767706195359,"blocked":true,"sim1":"+380501234567","sim2":"+380671234567","lines":[{"id":1,"name":"Вхідні двері"}]}]}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":0,"accum":0,"online":1,"lastPingDate":1774769732941}}`))
			case "get_objects_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"groupStatistics":{"24":{"1":1}},"countOfRooms":1}}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_CASL":"ППКО CASL"},"translate":{"uk":{"R401":"Взяття групи № {number}"}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"R401":"Взяття групи № {number}"}}`))
			default:
				t.Fatalf("unexpected command type: %v", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].BlockedArmedOnOff != 1 || objects[0].StatusText != "ЗАБЛОКОВАНО" {
		t.Fatalf("expected blocked object from read_device, got mode=%d status=%q", objects[0].BlockedArmedOnOff, objects[0].StatusText)
	}
	if got := objects[0].LastTestTime.UnixMilli(); got != lastPing {
		t.Fatalf("unexpected last test time from read_device: %d", got)
	}
	if loginCalls != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
	if commandCalls != 4 {
		t.Fatalf("expected 4 command calls after GetObjects, got %d", commandCalls)
	}

	gotByID := provider.GetObjectByID(strconv.Itoa(objects[0].ID))
	if gotByID == nil {
		t.Fatalf("expected object by ID")
	}
	if gotByID.Name != "Object 24" {
		t.Fatalf("unexpected object name: %q", gotByID.Name)
	}
	if gotByID.DeviceType != "CASL" {
		t.Fatalf("unexpected object device type: %q", gotByID.DeviceType)
	}
	if gotByID.Notes1 != "Ломбард" {
		t.Fatalf("unexpected object note: %q", gotByID.Notes1)
	}
	if gotByID.SIM1 != "+380501234567" {
		t.Fatalf("unexpected object sim1: %q", gotByID.SIM1)
	}
	if gotByID.SIM2 != "+380671234567" {
		t.Fatalf("unexpected object sim2: %q", gotByID.SIM2)
	}
	if gotByID.BlockedArmedOnOff != 1 || gotByID.StatusText != "ЗАБЛОКОВАНО" {
		t.Fatalf("expected blocked object by id, got mode=%d status=%q", gotByID.BlockedArmedOnOff, gotByID.StatusText)
	}
	if gotByID.TestControl != 1 || gotByID.TestTime != 60 || gotByID.AutoTestHours != 1 {
		t.Fatalf("unexpected test interval: control=%d time=%d autoHours=%d", gotByID.TestControl, gotByID.TestTime, gotByID.AutoTestHours)
	}
	if commandCalls != 7 {
		t.Fatalf("expected 7 command calls after GetObjectByID, got %d", commandCalls)
	}
}

func TestCASLProvider_ReloginOnWrongFormat(t *testing.T) {
	t.Parallel()

	var loginCalls int
	var commandCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			loginCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"fresh-token","user_id":"u-1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			commandCalls++
			w.Header().Set("Content-Type", "application/json")
			if commandCalls == 1 {
				_, _ = w.Write([]byte(`{"status":"error","error":"WRONG_FORMAT"}`))
				return
			}
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			if strings.TrimSpace(asString(payload["token"])) != "fresh-token" {
				t.Fatalf("expected refreshed token, got %v", payload["token"])
			}
			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"Object 25","device_id":"31","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"31","obj_id":"25","number":1004,"type":"TYPE_DEVICE_CASL","sim1":"+380501234567","sim2":""}]}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_CASL":"ППКО CASL"}}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "stale-token", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].TestControl != 1 {
		t.Fatalf("expected CASL test control to be always enabled, got %d", objects[0].TestControl)
	}
	if loginCalls != 1 {
		t.Fatalf("expected 1 relogin call, got %d", loginCalls)
	}
	if commandCalls != 5 {
		t.Fatalf("expected 5 command calls, got %d", commandCalls)
	}
}

func TestCASLProvider_ObjectDetailsEndpoints(t *testing.T) {
	t.Parallel()

	lastPing := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-2","user_id":"u-2","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr 24","description":"Main object","status":"Включено","contract":"C-24","device_id":"23","device_number":1003,"manager_id":"3","in_charge":["3"],"rooms":[{"room_id":"1","name":"Room A","description":"Office","rtsp":""}]}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"type":"TYPE_DEVICE_Dunay_4L","sim1":"+380501112233","sim2":"+380671112233","lines":[{"id":1,"name":"Тривожна кнопка"}]}]}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"3","last_name":"Petrenko","first_name":"Ihor","middle_name":"M","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+380971112233"}]}]}`))
			case "read_events_by_id":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"ppk_num":1003,"time":1774769226380,"code":"GROUP_ON","type":"ppk_event","number":1,"contact_id":"R401"}]}`))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"24":[]}}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":-1,"accum":-1,"door":-1,"online":0,"lastPingDate":` + strconv.FormatInt(lastPing, 10) + `,"lines":{},"groups":{},"adapters":{}}}`))
			case "get_objects_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"groupStatistics":{"24":{"1":0}},"countOfRooms":1}}`))
			case "get_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"device_id":"23","obj_id":"24","responseFrequencies":5,"communicQuality":5,"powerFailure":5,"criminogenicity":0,"customWins":3}}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{"R401":"Взяття групи № {number}"}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"R401":"Взяття групи № {number}"}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	objectID := strconv.Itoa(objects[0].ID)

	zones := provider.GetZones(objectID)
	if len(zones) != 1 || zones[0].Name != "Тривожна кнопка" {
		t.Fatalf("unexpected zones: %+v", zones)
	}

	contacts := provider.GetEmployees(objectID)
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].Phone != "+380971112233" {
		t.Fatalf("unexpected contact phone: %q", contacts[0].Phone)
	}

	events := provider.GetObjectEvents(objectID)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != models.EventArm {
		t.Fatalf("expected EventArm, got %s", events[0].Type)
	}
	if !strings.Contains(events[0].Details, "Постановка групи 1") {
		t.Fatalf("expected translated details, got %q", events[0].Details)
	}
	if strings.Contains(events[0].Details, "Опис:") {
		t.Fatalf("group event must not include zone description, got %q", events[0].Details)
	}

	signal, testMsg, _, lastMsg := provider.GetExternalData(objectID)
	if signal != "н/д" {
		t.Fatalf("unexpected signal payload: %q", signal)
	}
	if !strings.Contains(testMsg, "freq=5") {
		t.Fatalf("unexpected test payload: %q", testMsg)
	}
	if !strings.Contains(testMsg, "alarms=3") {
		t.Fatalf("unexpected alarms payload: %q", testMsg)
	}
	if lastMsg.IsZero() {
		t.Fatalf("expected non-zero last message time")
	}
}

func TestCASLProvider_GetEmployees_EnrichesRoomUsersFromReadUser(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-room-users","user_id":"u-room","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr 24","device_id":"23","device_number":1003,"rooms":[{"room_id":"10","name":"Офіс","description":"Офіс","users":[{"user_id":"41","role":"IN_CHARGE"}]}]}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"41","last_name":"Іваненко","first_name":"Іван","middle_name":"Іванович","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+380671112233"}]}]}`))
			default:
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	objectID := strconv.Itoa(objects[0].ID)
	contacts := provider.GetEmployees(objectID)
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].Name != "Іваненко Іван Іванович" {
		t.Fatalf("unexpected contact name: %q", contacts[0].Name)
	}
	if contacts[0].Phone != "+380671112233" {
		t.Fatalf("unexpected contact phone: %q", contacts[0].Phone)
	}
	if contacts[0].Position != "IN_CHARGE" {
		t.Fatalf("unexpected contact position: %q", contacts[0].Position)
	}
}

func TestCASLProvider_GetObjectEvents_IncludesGeneralTapeItemHistory(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-history","user_id":"u-history","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"51","name":"1004 Будинок","address":"Addr 51","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"51","number":1004,"type":"TYPE_DEVICE_CASL"}]}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_events_by_id":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{"E130":"Тривога в зоні № {number}","R130":"Норма в зоні № {number}"}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"E130":"Тривога в зоні № {number}","R130":"Норма в зоні № {number}"}}`))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"51":[{"dict_name":"GRD_OBJ_PICK","time":1775479716847},{"dict_name":"GRD_OBJ_NOTIF","time":1775479707727},{"code":"ZONE_ALM","time":1775479707370,"number":4,"contact_id":"E130"},{"code":"ZONE_NORM","time":1775479708821,"number":4,"contact_id":"R130"}]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	objectID := strconv.Itoa(objects[0].ID)
	events := provider.GetObjectEvents(objectID)
	if len(events) != 4 {
		t.Fatalf("expected 4 case history events, got %d", len(events))
	}
	if events[0].Type != models.EventOperatorAction {
		t.Fatalf("expected latest history event to be operator action, got %s", events[0].Type)
	}
	if !strings.Contains(events[0].Details, "Взяття в роботу") {
		t.Fatalf("expected GRD_OBJ_PICK details in latest event, got %q", events[0].Details)
	}
	foundAlarm := false
	foundRestore := false
	for _, item := range events {
		if item.Type == models.EventBurglary && strings.Contains(item.Details, "Тривога в зоні") {
			foundAlarm = true
		}
		if item.Type == models.EventRestore && strings.Contains(item.Details, "Норма в зоні") {
			foundRestore = true
		}
	}
	if !foundAlarm {
		t.Fatalf("expected zone alarm in case history, got %+v", events)
	}
	if !foundRestore {
		t.Fatalf("expected zone restore in case history, got %+v", events)
	}
}

func TestCASLProvider_GetObjectByID_FallsBackToObjectsStatisticGroups(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-stat-groups","user_id":"u-stat","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr 24","device_id":"23","device_number":1003}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"type":"TYPE_DEVICE_CASL"}]}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":0,"accum":0,"online":1,"groups":{},"lines":{},"adapters":{}}}`))
			case "get_objects_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"groupStatistics":{"24":{"5":1,"6":0}},"countOfRooms":2}}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_CASL":"ППКО CASL"}}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	objectID := strconv.Itoa(objects[0].ID)
	details := provider.GetObjectByID(objectID)
	if details == nil {
		t.Fatal("expected object details")
	}
	if len(details.Groups) != 2 {
		t.Fatalf("expected 2 groups from objects statistic, got %d (%+v)", len(details.Groups), details.Groups)
	}
	if details.Groups[0].Number != 5 || !details.Groups[0].Armed {
		t.Fatalf("unexpected first group: %+v", details.Groups[0])
	}
	if details.Groups[1].Number != 6 || details.Groups[1].Armed {
		t.Fatalf("unexpected second group: %+v", details.Groups[1])
	}
}

func TestCASLProvider_ReadConnectionsFallback(t *testing.T) {
	t.Parallel()

	var readConnectionsCalls int
	var readDeviceCalls int
	var readUserCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-connections","user_id":"u-c","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"error","error":"WRONG_FORMAT"}`))
			case "read_connections":
				readConnectionsCalls++
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"guardedObject":{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","address":"Львівська обл., с. Раковець","status":"Включено","manager":{"user_id":"3","last_name":"Островський","first_name":"Володимир","middle_name":"Іванович","role":"MANAGER","phone_numbers":[{"active":true,"number":"+38 (067) 700-17-75"}]},"rooms":[{"room_id":"32","name":"Будинок","description":"Будинок","users":[{"user_id":"37","last_name":"Хіміч","first_name":"Надія","middle_name":"Павлівна","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+38 (098) 729-90-91"}]},{"user_id":"38","last_name":"Хіміч","first_name":"Арсен","middle_name":"_","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+38 (063) 790-24-19"}]}]}]},"device":{"device_id":"24","obj_id":"25","number":1004,"name":"MAKS PRO","type":"TYPE_DEVICE_Ajax","sim1":"+38 (098) 162-68-59","lines":{"1":{"line_id":173,"description":"Рух тамбур"}}}}]}`))
			case "read_device":
				readDeviceCalls++
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_user":
				readUserCalls++
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_Ajax":"Ajax"}}}`))
			default:
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].DeviceType != "Ajax" {
		t.Fatalf("unexpected device type: %q", objects[0].DeviceType)
	}
	if objects[0].SIM1 != "+38 (098) 162-68-59" {
		t.Fatalf("unexpected SIM1: %q", objects[0].SIM1)
	}

	objectID := strconv.Itoa(objects[0].ID)
	contacts := provider.GetEmployees(objectID)
	if len(contacts) != 3 {
		t.Fatalf("expected 3 contacts from read_connections payload, got %d", len(contacts))
	}
	if contacts[0].Name != "Хіміч Надія Павлівна" {
		t.Fatalf("unexpected first contact name: %q", contacts[0].Name)
	}
	if contacts[2].Name != "Островський Володимир Іванович" {
		t.Fatalf("unexpected manager contact name: %q", contacts[2].Name)
	}
	if readConnectionsCalls == 0 {
		t.Fatalf("expected read_connections to be used as fallback")
	}
	if readDeviceCalls != 0 {
		t.Fatalf("read_device should not be required, got %d calls", readDeviceCalls)
	}
	if readUserCalls != 0 {
		t.Fatalf("read_user should not be required, got %d calls", readUserCalls)
	}
}

func TestCASLProvider_ReadConnectionsFallback_DeviceMapTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-connections-map","user_id":"u-cm","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"error","error":"WRONG_FORMAT"}`))
			case "read_connections":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"guardedObject":{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","address":"Львівська обл., с. Раковець","status":"Включено","device_number":1004},"devices":{"1":{"device_id":"24","obj_id":"25","number":1004,"name":"MAKS PRO","type":"TYPE_DEVICE_Ajax","timeout":360,"lastPingDate":1767706195359,"blocked":true,"sim1":"+38 (098) 162-68-59","lines":{"1":{"line_id":173,"description":"Рух тамбур"}}}}}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_Ajax":"Ajax"}}}`))
			default:
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	got := objects[0]
	if got.TestControl != 1 {
		t.Fatalf("expected enabled test control, got %d", got.TestControl)
	}
	if got.TestTime != 6 {
		t.Fatalf("expected 6 minutes from 360 seconds, got %d", got.TestTime)
	}
	if got.AutoTestHours != 0 {
		t.Fatalf("expected minute-based interval, got %d hours", got.AutoTestHours)
	}
	if got.BlockedArmedOnOff != 1 || got.StatusText != "ЗАБЛОКОВАНО" {
		t.Fatalf("expected blocked object from devices map, got mode=%d status=%q", got.BlockedArmedOnOff, got.StatusText)
	}
	if got.LastTestTime.UnixMilli() != 1767706195359 {
		t.Fatalf("unexpected last test time from devices map: %d", got.LastTestTime.UnixMilli())
	}
	if got.SIM1 != "+38 (098) 162-68-59" {
		t.Fatalf("unexpected SIM1: %q", got.SIM1)
	}
}

func TestCASLProvider_GetZones_UsesDeviceLines(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-zones","user_id":"u-z","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr 24","device_id":"23","device_number":1003,"rooms":[{"room_id":"1","name":"Офіс","description":"Office","rtsp":""},{"room_id":"2","name":"Склад","description":"Storage","rtsp":""}]}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"lines":[{"id":1,"number":1,"name":"Периметр","type":"Магнітоконтакт"},{"id":2,"number":2,"name":"Рух","type":"PIR"}]}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":0,"accum":0,"online":1,"groups":{"groups":{"1":{"group_number":1,"state":"GROUP_ON","room_id":"1"},"2":{"group_number":2,"state":"GROUP_OFF","room_id":"2"}}}}}`))
			default:
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	objectID := strconv.Itoa(objects[0].ID)
	zones := provider.GetZones(objectID)
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones from device lines, got %d (%+v)", len(zones), zones)
	}

	if zones[0].Number != 1 || zones[0].Name != "Периметр" || zones[0].IsBypassed {
		t.Fatalf("unexpected first zone: %+v", zones[0])
	}
	if zones[0].SensorType != "Магнітоконтакт" {
		t.Fatalf("expected first line type, got: %q", zones[0].SensorType)
	}
	if zones[1].Number != 2 || zones[1].Name != "Рух" || zones[1].IsBypassed {
		t.Fatalf("unexpected second zone: %+v", zones[1])
	}
	if zones[1].SensorType != "PIR" {
		t.Fatalf("expected second line type, got: %q", zones[1].SensorType)
	}
}

func TestCASLProvider_GetZones_PrefersExplicitGroupNumberOverRoomID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-zones-groups","user_id":"u-zg","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr 24","device_id":"23","device_number":1003,"rooms":[{"room_id":"25","name":"Будинок","description":"House","rtsp":""},{"room_id":"26","name":"Гараж","description":"Garage","rtsp":""}]}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"lines":[{"id":1,"number":1,"name":"Вхід","type":"Магнітоконтакт","room_id":"25","group_number":5},{"id":2,"number":2,"name":"Рух","type":"PIR","room_id":"25","group_number":6},{"id":3,"number":3,"name":"Склад","type":"PIR","room_id":"26","group_number":5}]}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":0,"accum":0,"online":1,"groups":{"groups":{"25":{"state":"GROUP_ON","room_id":"25"},"26":{"state":"GROUP_OFF","room_id":"26"}}}}}`))
			default:
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	objectID := strconv.Itoa(objects[0].ID)
	details := provider.GetObjectByID(objectID)
	if details == nil {
		t.Fatal("expected object details")
	}
	if len(details.Groups) != 2 {
		t.Fatalf("expected 2 logical groups, got %d (%+v)", len(details.Groups), details.Groups)
	}
	if details.Groups[0].Number != 5 || details.Groups[1].Number != 6 {
		t.Fatalf("expected logical groups 5 and 6, got %+v", details.Groups)
	}

	zones := provider.GetZones(objectID)
	if len(zones) != 3 {
		t.Fatalf("expected 3 zones from device lines, got %d (%+v)", len(zones), zones)
	}
	if zones[0].GroupNumber != 5 {
		t.Fatalf("expected first zone to belong to group 5, got %+v", zones[0])
	}
	if zones[1].GroupNumber != 6 {
		t.Fatalf("expected second zone to belong to group 6, got %+v", zones[1])
	}
	if zones[2].GroupNumber != 5 {
		t.Fatalf("expected third zone to belong to group 5, got %+v", zones[2])
	}
}

func TestCASLProvider_GetEvents_ReturnsRealtimeCache(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	provider.cachedEvents = []models.Event{
		{ID: 2, ObjectID: mapCASLObjectID("24"), ObjectName: "Object 24", Type: models.EventFire, Details: "Пожежна тривога"},
		{ID: 1, ObjectID: mapCASLObjectID("24"), ObjectName: "Object 24", Type: models.EventArm, Details: "Взяття групи"},
	}
	provider.mu.Unlock()

	events := provider.GetEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != models.EventFire {
		t.Fatalf("expected latest event type fire, got %s", events[0].Type)
	}
	if events[1].Type != models.EventArm {
		t.Fatalf("expected second event type arm, got %s", events[1].Type)
	}
	if events[0].ObjectName != "Object 24" {
		t.Fatalf("unexpected object name: %q", events[0].ObjectName)
	}
	if !strings.Contains(events[0].Details, "Пожежна тривога") {
		t.Fatalf("expected decoded details, got %q", events[0].Details)
	}
}

func TestCASLProvider_GetAlarms_DoesNotUseReadEventsRowsAsActiveAlarmSource(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()
	readEventsPayload := fmt.Sprintf(`{"status":"ok","data":[{"ppk_num":1003,"time":%d,"code":"FIRE_ALARM","contact_id":"E110","number":2,"type":"ppk_event","obj_id":"24","obj_name":"Object 24"},{"ppk_num":1003,"time":%d,"code":"NO_220","contact_id":"E301","number":0,"type":"ppk_event","obj_id":"24","obj_name":"Object 24"}]}`, nowMs, nowMs+1000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(readEventsPayload))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","device_id":"23","device_number":1003}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"type":"TYPE_DEVICE_CASL"}]}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"E110":"Пожежна тривога № {number}","E301":"Втрата живлення 220В"}}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"alarm_reasons":{"12":"Пожежа"}}}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 0 {
		t.Fatalf("expected no active alarms from read_events, got %d", len(alarms))
	}
}

func TestCASLProvider_GetAlarms_DoesNotUseReadEventsRowsWithoutPPKAsActiveAlarmSource(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()
	readEventsPayload := fmt.Sprintf(`{"status":"ok","data":[{"time":%d,"code":"FIRE_ALARM","contact_id":"E110","number":1,"type":"ppk_event","obj_id":"24","obj_name":"Object 24"}]}`, nowMs)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(readEventsPayload))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","device_id":"23","device_number":1003}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"type":"TYPE_DEVICE_CASL"}]}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"E110":"Пожежна тривога № {number}"}}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"alarm_reasons":{"12":"Пожежа"}}}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 0 {
		t.Fatalf("expected no active alarms from read_events row without ppk_num, got %d", len(alarms))
	}
}

func TestCASLProvider_GetAlarms_DoesNotUseGeneralTapeItemAsActiveAlarmSource(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"215","name":"Object 215","device_id":"23","device_number":1003}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"215","number":1003,"type":"MAKS_PRO"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"MAKS_PRO":[{"code":130,"typeEvent":"E","name":"ZONE_ALM","isAlarm":1}]}}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":{"215":[{"code":62221,"time":%d,"contact_id":"E130","number":13}]}}`, nowMs)))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 0 {
		t.Fatalf("expected no active alarms from get_general_tape_item history, got %d", len(alarms))
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_UsesTranslatorIsAlarmForCustomDevice(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"25","number":1004,"type":"SATEL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"user_device_types":["SATEL"],"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"SATEL":[{"code":152,"typeEvent":"E","name":"REFRIGERATION_ALARM","isAlarm":1}]}}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"25","obj_name":"1004 Будинок Хіміч Н.П.","time":%d,"code":"152","type_event":"E","zone":7,"event_type":"ppk_event"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"25":[]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from custom device translator, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmNotification {
		t.Fatalf("alarm type = %s, want %s", alarms[0].Type, models.AlarmNotification)
	}
	if alarms[0].SC1 != mapCASLEventSC1(models.EventAlarmNotification) {
		t.Fatalf("alarm SC1 = %d, want %d", alarms[0].SC1, mapCASLEventSC1(models.EventAlarmNotification))
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_UsesReadAlarmEventsForStandardDevice(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"25","number":1004,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"user_device_types":["SATEL"],"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[{"code":"DOOR_OP","is_alarm_in_start":1,"is_alarm":1}]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"25","obj_name":"1004 Будинок Хіміч Н.П.","time":%d,"code":"DOOR_OP","zone":1,"event_type":"ppk_event"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"25":[]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from read_alarm_events standard device path, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmNotification {
		t.Fatalf("alarm type = %s, want %s", alarms[0].Type, models.AlarmNotification)
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_PreservesObjectNumber(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"25","number":1004,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"E110":"Пожежна тривога № {number}"}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"25","obj_name":"1004 Будинок Хіміч Н.П.","time":%d,"code":"FIRE_ALARM","contact_id":"E110","zone":2,"event_type":"ppk_event"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"25":[]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from get_general_tape_objects, got %d", len(alarms))
	}
	if got := alarms[0].ObjectNumber; got != "1004" {
		t.Fatalf("alarm object number = %q, want 1004", got)
	}
	if got := alarms[0].GetObjectNumberDisplay(); got != "1004" {
		t.Fatalf("alarm display number = %q, want 1004", got)
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_PreservesAlarmTypeAndUsesHistoryCause(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"51","name":"1004 Будинок","address":"Addr 51","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"51","number":1004,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{"E130":"Тривога в зоні № {number}","R130":"Норма в зоні № {number}"}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"E130":"Тривога в зоні № {number}","R130":"Норма в зоні № {number}"}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"51","obj_name":"1004 Будинок","obj_address":"Addr 51","time":%d,"action":"GRD_OBJ_NOTIF","alarm_type":"ALARM_TYPE_DEVICE","event_type":"user_action"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":{"51":[{"dict_name":"GRD_OBJ_PICK","time":%d},{"dict_name":"GRD_OBJ_NOTIF","time":%d},{"code":"ZONE_ALM","time":%d,"number":4,"contact_id":"E130"},{"code":"ZONE_NORM","time":%d,"number":4,"contact_id":"R130"}]}}`, nowMs+1000, nowMs, nowMs-500, nowMs+1500)))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from get_general_tape_objects with history enrichment, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmDevice {
		t.Fatalf("alarm type = %s, want %s", alarms[0].Type, models.AlarmDevice)
	}
	if alarms[0].Address != "Addr 51" {
		t.Fatalf("alarm address = %q, want Addr 51", alarms[0].Address)
	}
	if !strings.Contains(alarms[0].Details, "Тривога в зоні № 4") {
		t.Fatalf("alarm details = %q, want history-derived cause", alarms[0].Details)
	}
	if len(alarms[0].SourceMsgs) == 0 {
		t.Fatalf("expected SourceMsgs from get_general_tape_item history")
	}
	foundAlarmMsg := false
	for _, msg := range alarms[0].SourceMsgs {
		if msg.IsAlarm && strings.Contains(msg.Details, "Тривога в зоні № 4") {
			foundAlarmMsg = true
			break
		}
	}
	if !foundAlarmMsg {
		t.Fatalf("expected alarm message in SourceMsgs, got %+v", alarms[0].SourceMsgs)
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_GenericAlarmTypeDoesNotBecomeFault(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"25","number":1004,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"alarm_reasons":{"12":"Пожежа"}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"25","obj_name":"1004 Будинок Хіміч Н.П.","time":%d,"event_type":"alarm","reasonAlarm":"12","zone":4}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"25":[]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from generic get_general_tape_objects alarm row, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmNotification {
		t.Fatalf("alarm type = %s, want %s", alarms[0].Type, models.AlarmNotification)
	}
	if alarms[0].SC1 != mapCASLEventSC1(models.EventAlarmNotification) {
		t.Fatalf("alarm SC1 = %d, want %d", alarms[0].SC1, mapCASLEventSC1(models.EventAlarmNotification))
	}
	if got := alarms[0].Details; got != "Пожежа" {
		t.Fatalf("alarm details = %q, want Пожежа", got)
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_UsesReasonAlarmJSON(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"26","name":"1005 Склад","device_id":"24","device_number":1005}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"24","obj_id":"26","number":1005,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"FIRE_ALARM":"Пожежна тривога № {number}"}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"26","obj_name":"1005 Склад","time":%d,"alarm_type":"ALARM_TYPE_MOBILE","event_type":"alarm","reasonAlarm":"{\"msg\":\"FIRE_ALARM\",\"num\":5}"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"26":[]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from reasonAlarm JSON, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmMobile {
		t.Fatalf("alarm type = %s, want %s", alarms[0].Type, models.AlarmMobile)
	}
	if got := alarms[0].Details; got != "Пожежна тривога № 5" {
		t.Fatalf("alarm details = %q, want translated JSON reason", got)
	}
	if len(alarms[0].SourceMsgs) != 0 {
		t.Fatalf("expected empty SourceMsgs when history is absent, got %+v", alarms[0].SourceMsgs)
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjects_ALMIODoesNotBecomeFault(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"25","number":1004,"type":"TYPE_DEVICE_Dunay_4L"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"obj_id":"25","obj_name":"1004 Будинок Хіміч Н.П.","time":%d,"code":"ALM_IO","contact_id":"E134","zone":1,"event_type":"ppk_event"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"25":[]}}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from ALM_IO general tape row, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmBurglary {
		t.Fatalf("alarm type = %s, want %s", alarms[0].Type, models.AlarmBurglary)
	}
	if alarms[0].SC1 != mapCASLEventSC1(models.EventBurglary) {
		t.Fatalf("alarm SC1 = %d, want %d", alarms[0].SC1, mapCASLEventSC1(models.EventBurglary))
	}
}

func TestCASLProvider_GetAlarms_FromGeneralTapeObjectsRowWithOnlyDeviceID_UsesDeviceNumber(t *testing.T) {
	t.Parallel()

	nowMs := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"1004 Будинок Хіміч Н.П.","device_id":"23","device_number":1004}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"25","number":1004,"type":"TYPE_DEVICE_CASL"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"translate":{"uk":{}}}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"E110":"Пожежна тривога № {number}"}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","data":[{"device_id":"23","obj_name":"1004 Будинок Хіміч Н.П.","time":%d,"code":"FIRE_ALARM","contact_id":"E110","zone":1,"event_type":"ppk_event"}]}`, nowMs)))
			case "get_general_tape_item":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from general tape row with device_id, got %d", len(alarms))
	}
	if got := alarms[0].ObjectNumber; got != "1004" {
		t.Fatalf("alarm object number = %q, want 1004", got)
	}
	if got := alarms[0].GetObjectNumberDisplay(); got != "1004" {
		t.Fatalf("alarm display number = %q, want 1004", got)
	}
}

func TestCASLProvider_GetAlarms_UsesRealtimeCacheWhenReadEventsFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")
			switch cmdType {
			case "read_events":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"error","error":"INTERNAL"}`))
			case "read_grd_object", "read_connections", "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			default:
				t.Fatalf("unexpected command type: %s (payload: %v)", cmdType, payload)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	nowTS := int64(1774970891649)
	seed := stableCASLAlarmSeed("FIRE_ALARM", "E110", 0)
	provider.mu.Lock()
	provider.realtimeAlarmByObjID["24|z0"] = models.Alarm{
		ID:         stableCASLAlarmID("24", nowTS, seed),
		ObjectID:   mapCASLObjectID("24"),
		ObjectName: "24 | Object 24",
		Time:       time.UnixMilli(nowTS).Local(),
		Details:    "Нова тривога",
		Type:       models.AlarmFire,
	}
	provider.mu.Unlock()

	alarms := provider.GetAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from realtime cache, got %d", len(alarms))
	}
	if alarms[0].ObjectName != "24 | Object 24" {
		t.Fatalf("unexpected object name: %q", alarms[0].ObjectName)
	}
}

func TestCASLProvider_SubscribeRealtimeTags_DoesNotUseTapeTag(t *testing.T) {
	t.Parallel()

	calledTags := make([]string, 0, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != caslSubscribePath {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		calledTags = append(calledTags, strings.TrimSpace(asString(payload["tag"])))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	if err := provider.subscribeRealtimeTags(context.Background(), "conn-1"); err != nil {
		t.Fatalf("subscribeRealtimeTags error: %v", err)
	}

	if len(calledTags) == 0 {
		t.Fatal("expected at least one realtime tag subscription")
	}

	sawPPKIn := false
	sawUserAction := false
	for _, tag := range calledTags {
		switch tag {
		case "ppk_in":
			sawPPKIn = true
		case "user_action":
			sawUserAction = true
		case "tape":
			t.Fatalf("unexpected deprecated realtime tag subscription: %q", tag)
		}
	}

	if !sawPPKIn {
		t.Fatal("expected ppk_in subscription")
	}
	if !sawUserAction {
		t.Fatal("expected user_action subscription")
	}
}

func TestCASLProvider_AppendRealtimeRows_UpdatesAlarmCacheWithoutTapeTag(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1003),
	}
	provider.cachedObjects = []caslGrdObject{record}
	provider.cachedObjectsAt = time.Now()
	provider.objectByInternalID[mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))] = record

	device := caslDevice{
		DeviceID: caslText("23"),
		ObjID:    caslText("25"),
		Number:   caslInt64(1003),
		Type:     caslText("TYPE_DEVICE_CASL"),
	}
	provider.deviceByDeviceID = map[string]caslDevice{"23": device}
	provider.deviceByObjectID = map[string]caslDevice{"25": device}
	provider.deviceByNumber = map[int64]caslDevice{1003: device}
	provider.cachedDevicesAt = time.Now()
	provider.cachedDictionary = map[string]any{"E110": "Пожежна тривога"}
	provider.cachedDictionaryAt = time.Now()
	provider.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	if err := provider.appendRealtimeRows(context.Background(), []CASLObjectEvent{
		{
			Action:    "GRD_OBJ_NOTIF",
			ObjID:     "25",
			ObjName:   "1004 Будинок Хіміч Н.П.",
			AlarmType: "ALARM_TYPE_OPERATOR",
			Time:      nowMs,
			Type:      "user_action",
		},
	}); err != nil {
		t.Fatalf("appendRealtimeRows error: %v", err)
	}

	alarms := provider.snapshotRealtimeAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 realtime alarm, got %d", len(alarms))
	}
	if alarms[0].ObjectName != "1003 | 1004 Будинок Хіміч Н.П." {
		t.Fatalf("unexpected realtime alarm object name: %q", alarms[0].ObjectName)
	}
	if alarms[0].Type != models.AlarmOperator {
		t.Fatalf("unexpected realtime alarm type: %s", alarms[0].Type)
	}
}

func TestCASLProvider_UpdateRealtimeAlarmsFromRows_Lifecycle(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1003),
	}
	provider.cachedObjects = []caslGrdObject{record}
	provider.cachedObjectsAt = time.Now()
	provider.objectByInternalID[mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))] = record

	device := caslDevice{
		DeviceID: caslText("23"),
		ObjID:    caslText("25"),
		Number:   caslInt64(1003),
		Type:     caslText("TYPE_DEVICE_CASL"),
	}
	provider.deviceByDeviceID = map[string]caslDevice{"23": device}
	provider.deviceByObjectID = map[string]caslDevice{"25": device}
	provider.deviceByNumber = map[int64]caslDevice{1003: device}
	provider.cachedDevicesAt = time.Now()
	provider.cachedDictionary = map[string]any{"E110": "Пожежна тривога"}
	provider.cachedDictionaryAt = time.Now()
	provider.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	provider.updateRealtimeAlarmsFromRows(context.Background(), []CASLObjectEvent{
		{
			Action:    "GRD_OBJ_NOTIF",
			ObjID:     "25",
			ObjName:   "1004 Будинок Хіміч Н.П.",
			AlarmType: "ALARM_TYPE_OPERATOR",
			Time:      nowMs,
			Type:      "user_action",
		},
	})

	alarms := provider.snapshotRealtimeAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 realtime alarm, got %d", len(alarms))
	}
	if alarms[0].ObjectName != "1003 | 1004 Будинок Хіміч Н.П." {
		t.Fatalf("unexpected realtime alarm object name: %q", alarms[0].ObjectName)
	}
	if alarms[0].Type != models.AlarmOperator {
		t.Fatalf("unexpected realtime alarm type: %s", alarms[0].Type)
	}

	provider.updateRealtimeAlarmsFromRows(context.Background(), []CASLObjectEvent{
		{
			Action: "GRD_OBJ_MGR_CANCEL",
			ObjID:  "25",
			Time:   nowMs + 1000,
			Type:   "user_action",
		},
	})

	alarms = provider.snapshotRealtimeAlarms()
	if len(alarms) != 0 {
		t.Fatalf("expected realtime alarm cache to be empty after cancel, got %d", len(alarms))
	}
}

func TestCASLProvider_MapCASLRowsToEvents_SkipsRowsWithoutTime(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	events, maxEventTime := provider.mapCASLRowsToEvents(context.Background(), []CASLObjectEvent{
		{
			ObjID:   "25",
			Action:  "GRD_OBJ_NOTIF",
			Type:    "user_action",
			Code:    "GRD_OBJ_NOTIF",
			Number:  1,
			Time:    0,
			ObjName: "1004 Будинок Хіміч Н.П.",
		},
	}, 0)

	if len(events) != 0 {
		t.Fatalf("expected rows without time to be skipped, got %+v", events)
	}
	if maxEventTime != 0 {
		t.Fatalf("expected maxEventTime to stay zero, got %d", maxEventTime)
	}
}

func TestCASLProvider_MapCASLRowsToEvents_UsesIsAlarmFlagCorrection(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1004),
	}
	provider.cachedObjects = []caslGrdObject{record}
	provider.cachedObjectsAt = time.Now()
	provider.objectByInternalID[mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))] = record

	device := caslDevice{
		DeviceID: caslText("23"),
		ObjID:    caslText("25"),
		Number:   caslInt64(1004),
		Type:     caslText("SATEL"),
	}
	provider.deviceByDeviceID = map[string]caslDevice{"23": device}
	provider.deviceByObjectID = map[string]caslDevice{"25": device}
	provider.deviceByNumber = map[int64]caslDevice{1004: device}
	provider.cachedDevicesAt = time.Now()
	provider.cachedDictionary = map[string]any{
		"user_device_types": []any{"SATEL"},
	}
	provider.cachedDictionaryAt = time.Now()
	provider.cachedTranslatorAlarms["SATEL"] = map[string]bool{"FIRE_ALARM": false}
	provider.cachedTransAt["SATEL"] = time.Now()
	provider.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	events, _ := provider.mapCASLRowsToEvents(context.Background(), []CASLObjectEvent{
		{
			ObjID:    "25",
			ObjName:  "1004 Будинок Хіміч Н.П.",
			DeviceID: "23",
			Time:     nowMs,
			Type:     "ppk_event",
			Code:     "FIRE_ALARM",
			Number:   7,
		},
	}, 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != models.EventFault {
		t.Fatalf("event type = %s, want %s (is_alarm correction from translator)", events[0].Type, models.EventFault)
	}
}

func TestCASLProvider_UpdateRealtimeAlarmsFromRows_SkipsRowsWithoutTime(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1003),
	}
	provider.cachedObjects = []caslGrdObject{record}
	provider.cachedObjectsAt = time.Now()
	provider.objectByInternalID[mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))] = record
	provider.cachedDictionary = map[string]any{"E110": "Пожежна тривога"}
	provider.cachedDictionaryAt = time.Now()
	provider.mu.Unlock()

	provider.updateRealtimeAlarmsFromRows(context.Background(), []CASLObjectEvent{
		{
			Action:    "GRD_OBJ_NOTIF",
			ObjID:     "25",
			ObjName:   "1004 Будинок Хіміч Н.П.",
			AlarmType: "ALARM_TYPE_OPERATOR",
			Time:      0,
			Type:      "user_action",
		},
	})

	if alarms := provider.snapshotRealtimeAlarms(); len(alarms) != 0 {
		t.Fatalf("expected rows without time to be ignored, got %+v", alarms)
	}
}

func TestCASLProvider_UpdateRealtimeAlarmsFromRows_UsesTranslatorIsAlarmForCustomDevice(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1004),
	}
	provider.cachedObjects = []caslGrdObject{record}
	provider.cachedObjectsAt = time.Now()
	provider.objectByInternalID[mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))] = record

	device := caslDevice{
		DeviceID: caslText("23"),
		ObjID:    caslText("25"),
		Number:   caslInt64(1004),
		Type:     caslText("SATEL"),
	}
	provider.deviceByDeviceID = map[string]caslDevice{"23": device}
	provider.deviceByObjectID = map[string]caslDevice{"25": device}
	provider.deviceByNumber = map[int64]caslDevice{1004: device}
	provider.cachedDevicesAt = time.Now()
	provider.cachedDictionary = map[string]any{
		"user_device_types": []any{"SATEL"},
	}
	provider.cachedDictionaryAt = time.Now()
	provider.cachedTranslatorAlarms["SATEL"] = map[string]bool{"E152": true, "R152": false}
	provider.cachedTranslators["SATEL"] = map[string]string{"E152": "REFRIGERATION_ALARM"}
	provider.cachedTransAt["SATEL"] = time.Now()
	provider.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	provider.updateRealtimeAlarmsFromRows(context.Background(), []CASLObjectEvent{
		{
			ObjID:    "25",
			ObjName:  "1004 Будинок Хіміч Н.П.",
			DeviceID: "23",
			Time:     nowMs,
			Type:     "ppk_event",
			Code:     "152",
			Subtype:  "E",
			Number:   7,
		},
	})

	alarms := provider.snapshotRealtimeAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 realtime alarm, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmNotification {
		t.Fatalf("realtime alarm type = %s, want %s", alarms[0].Type, models.AlarmNotification)
	}
}

func TestCASLProvider_UpdateRealtimeAlarmsFromRows_UsesReadAlarmEventsForStandardDevice(t *testing.T) {
	t.Parallel()

	provider := NewCASLCloudProvider("http://127.0.0.1:50003", "token", 1)
	provider.mu.Lock()
	record := caslGrdObject{
		ObjID:        "25",
		Name:         "1004 Будинок Хіміч Н.П.",
		DeviceID:     caslInt64(23),
		DeviceNumber: caslInt64(1004),
	}
	provider.cachedObjects = []caslGrdObject{record}
	provider.cachedObjectsAt = time.Now()
	provider.objectByInternalID[mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))] = record

	device := caslDevice{
		DeviceID: caslText("23"),
		ObjID:    caslText("25"),
		Number:   caslInt64(1004),
		Type:     caslText("TYPE_DEVICE_CASL"),
	}
	provider.deviceByDeviceID = map[string]caslDevice{"23": device}
	provider.deviceByObjectID = map[string]caslDevice{"25": device}
	provider.deviceByNumber = map[int64]caslDevice{1004: device}
	provider.cachedDevicesAt = time.Now()
	provider.cachedDictionary = map[string]any{
		"user_device_types": []any{"SATEL"},
	}
	provider.cachedDictionaryAt = time.Now()
	provider.cachedAlarmEvents = map[string]bool{"DOOR_OP": true}
	provider.cachedAlarmEventsAt = time.Now()
	provider.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	provider.updateRealtimeAlarmsFromRows(context.Background(), []CASLObjectEvent{
		{
			ObjID:    "25",
			ObjName:  "1004 Будинок Хіміч Н.П.",
			DeviceID: "23",
			Time:     nowMs,
			Type:     "ppk_event",
			Code:     "DOOR_OP",
			Number:   1,
		},
	})

	alarms := provider.snapshotRealtimeAlarms()
	if len(alarms) != 1 {
		t.Fatalf("expected 1 realtime alarm from read_alarm_events standard device path, got %d", len(alarms))
	}
	if alarms[0].Type != models.AlarmNotification {
		t.Fatalf("realtime alarm type = %s, want %s", alarms[0].Type, models.AlarmNotification)
	}
}

func TestCASLProvider_GetObjectsMarksDisconnectedDevicesOffline(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-disconnected","user_id":"u-off","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr 24","status":"Включено","device_id":"23","device_number":1003}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"name":"MAKS PRO","lastPingDate":1774769732941}]}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"offline":1774769732941}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":0,"accum":0,"online":1,"lastPingDate":1774769732941}}`))
			default:
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].Status != models.StatusOffline || objects[0].IsConnState != 0 {
		t.Fatalf("expected offline object from disconnected devices, got %+v", objects[0])
	}
	if objects[0].BlockedArmedOnOff != 0 {
		t.Fatalf("offline object must not stay visually blocked, got %d", objects[0].BlockedArmedOnOff)
	}

	gotByID := provider.GetObjectByID(strconv.Itoa(objects[0].ID))
	if gotByID == nil {
		t.Fatalf("expected object by id")
	}
	if gotByID.Status != models.StatusOffline || gotByID.IsConnState != 0 {
		t.Fatalf("expected offline object by id, got %+v", gotByID)
	}
}
