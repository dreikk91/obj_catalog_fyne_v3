package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	if wantID := caslObjectIDNamespaceStart + 123; obj.ID != wantID {
		t.Fatalf("unexpected ID: got %d, want %d", obj.ID, wantID)
	}
	if !isCASLObjectID(obj.ID) {
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

func TestBuildCASLUserActionDetails(t *testing.T) {
	t.Parallel()

	details := buildCASLUserActionDetails(CASLObjectEvent{
		Action:    "GRD_OBJ_NOTIF",
		ObjID:     "25",
		ObjName:   "1004 Будинок Хіміч Н.П.",
		AlarmType: "ALARM_TYPE_OPERATOR",
	})
	if !strings.Contains(details, "Нова тривога") || !strings.Contains(details, "1004 Будинок Хіміч Н.П.") {
		t.Fatalf("unexpected notif details: %q", details)
	}

	details = buildCASLUserActionDetails(CASLObjectEvent{
		Action:  "GRD_OBJ_PICK",
		UserFIO: "Островська Марина",
	})
	if !strings.Contains(details, "взято в роботу") || !strings.Contains(details, "Островська") {
		t.Fatalf("unexpected pick details: %q", details)
	}
}

func TestClassifyCASLEventTypeWithContext_UserActionByCode(t *testing.T) {
	t.Parallel()

	got := classifyCASLEventTypeWithContext("GRD_OBJ_NOTIF", "", "user_action", "")
	if got != models.EventType(models.AlarmBurglary) {
		t.Fatalf("expected EventBurglary for GRD_OBJ_NOTIF, got %s", got)
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
		{code: "PANIC_ALARM", expected: models.EventPanic},
		{code: "MEDICAL_ALARM", expected: models.EventMedical},
		{code: "GAS_ALARM", expected: models.EventGas},
		{code: "SABOTAGE_AD", expected: models.EventTamper},
		{code: "NO_220", expected: models.EventPowerFail},
		{code: "OO_OK_220", expected: models.EventRestore},
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
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","obj_id":"24","number":1003,"name":"Ломбард","type":"TYPE_DEVICE_CASL","sim1":"+380501234567","sim2":"+380671234567","lines":[{"id":1,"name":"Вхідні двері"}]}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":0,"accum":0,"online":1,"lastPingDate":1774769732941}}`))
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
	if loginCalls != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
	if commandCalls != 3 {
		t.Fatalf("expected 3 command calls after GetObjects, got %d", commandCalls)
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
	if commandCalls != 4 {
		t.Fatalf("expected 4 command calls after GetObjectByID, got %d", commandCalls)
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
	if loginCalls != 1 {
		t.Fatalf("expected 1 relogin call, got %d", loginCalls)
	}
	if commandCalls != 4 {
		t.Fatalf("expected 4 command calls, got %d", commandCalls)
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
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"3","last_name":"Petrenko","first_name":"Ihor","middle_name":"M","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+380971112233"}]}]}`))
			case "read_events_by_id":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"ppk_num":1003,"time":1774769226380,"code":"GROUP_ON","type":"ppk_event","number":1,"contact_id":"R401"}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":-1,"accum":-1,"door":-1,"online":0,"lastPingDate":` + strconv.FormatInt(lastPing, 10) + `,"lines":{},"groups":{},"adapters":{}}}`))
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

func TestCASLProvider_GetAlarms_FromReadEventsRows(t *testing.T) {
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
	if len(alarms) != 2 {
		t.Fatalf("expected 2 alarms from read_events, got %d", len(alarms))
	}
	if alarms[0].ObjectName != "1003 | Object 24" {
		t.Fatalf("unexpected object name: %q", alarms[0].ObjectName)
	}
	if alarms[0].Type != models.AlarmPowerFail && alarms[0].Type != models.AlarmFire {
		t.Fatalf("unexpected alarm type: %s", alarms[0].Type)
	}
}

func TestCASLProvider_GetAlarms_FromReadEventsRowsWithoutPPK(t *testing.T) {
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
		t.Fatalf("expected 1 alarm from read_events row without ppk_num, got %d", len(alarms))
	}
	if alarms[0].ObjectName != "1003 | Object 24" {
		t.Fatalf("unexpected object name: %q", alarms[0].ObjectName)
	}
}

func TestCASLProvider_GetAlarms_FallbackToGeneralTapeItem(t *testing.T) {
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
	if len(alarms) != 1 {
		t.Fatalf("expected 1 alarm from get_general_tape_item fallback, got %d", len(alarms))
	}
	if alarms[0].ObjectName != "1003 | Object 215" {
		t.Fatalf("unexpected object name: %q", alarms[0].ObjectName)
	}
	if alarms[0].ZoneNumber != 13 {
		t.Fatalf("unexpected zone number: %d", alarms[0].ZoneNumber)
	}
	if strings.TrimSpace(alarms[0].Details) == "" {
		t.Fatalf("expected non-empty alarm details")
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
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token", 1)
	provider.mu.Lock()
	provider.realtimeAlarmByObjID["24"] = models.Alarm{
		ID:         1,
		ObjectID:   mapCASLObjectID("24"),
		ObjectName: "24 | Object 24",
		Time:       time.Now(),
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
