package caslcompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/ids"

	"github.com/gorilla/websocket"
)

func TestFixtureHandler_LoginReturnsTokenAndWSURL(t *testing.T) {
	handler := NewFixtureHandler()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"operator@example.com","pwd":"x","pult_id":1}`))
	req.Host = "127.0.0.1:50003"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	decodeTestJSON(t, rec.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Fatalf("unexpected status: %#v", body["status"])
	}
	if body["token"] != fixtureToken {
		t.Fatalf("unexpected token: %#v", body["token"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is %T", body["data"])
	}
	if data["ws_url"] != "ws://127.0.0.1:50003" {
		t.Fatalf("unexpected ws_url: %#v", data["ws_url"])
	}
	if data["role"] != "ADMIN" {
		t.Fatalf("unexpected data role: %#v", data["role"])
	}
	if body["role"] != "ADMIN" {
		t.Fatalf("unexpected top-level role: %#v", body["role"])
	}
	if body["ws_url"] != "ws://127.0.0.1:50003" {
		t.Fatalf("unexpected top-level ws_url: %#v", body["ws_url"])
	}
}

func TestFixtureHandler_LoginUsesConfiguredWSURL(t *testing.T) {
	handler := NewFixtureHandlerWithWSURL("ws://127.0.0.1:23322")
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"operator@example.com","pwd":"x","pult_id":1}`))
	req.Host = "127.0.0.1:50003"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	decodeTestJSON(t, rec.Body.Bytes(), &body)
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is %T", body["data"])
	}
	if data["ws_url"] != "ws://127.0.0.1:23322" {
		t.Fatalf("unexpected ws_url: %#v", data["ws_url"])
	}
	if body["ws_url"] != "ws://127.0.0.1:23322" {
		t.Fatalf("unexpected top-level ws_url: %#v", body["ws_url"])
	}
}

func TestFixtureHandler_UsesInjectedFixture(t *testing.T) {
	fixture := buildFixtureFromUnified(UnifiedFixture{
		Admin: UnifiedUser{ID: "900", Email: "custom@example.com", Role: "ADMIN", FirstName: "Custom", LastName: "Admin", PultID: 9},
		Objects: []UnifiedObject{
			{ID: 9001, DisplayNumber: "C-9001", Name: "Custom object", Address: "Custom address", ReactingPultID: 9, ResponsibleIDs: []string{"901"}, Room: UnifiedRoom{ID: "900101", Name: "Custom room"}},
		},
		Devices: []UnifiedDevice{
			{ID: 9101, ObjectID: 9001, Number: 9001, Name: "Custom device", Type: "CUSTOM_DEVICE", Lines: []UnifiedLine{
				{ID: 1, Number: 1, AdapterType: "SYS", GroupNumber: 1, RoomID: "900101"},
			}},
		},
		DeviceTypes: []UnifiedDeviceType{
			{Type: "CUSTOM_DEVICE", NameUK: "Custom", NameRU: "Custom", NameEN: "Custom", MaxLines: 1, MaxGroups: 1},
		},
	})
	handler := NewFixtureHandlerWithFixtureAndWSURL(fixture, "ws://127.0.0.1:29999")

	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"custom@example.com","pwd":"x","pult_id":9}`))
	loginReq.Host = "127.0.0.1:50003"
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	var login map[string]any
	decodeTestJSON(t, loginRec.Body.Bytes(), &login)
	if login["user_id"] != "900" || login["ws_url"] != "ws://127.0.0.1:29999" {
		t.Fatalf("login = %#v", login)
	}

	objectsBody := postCommand(t, handler, `{"type":"read_grd_object","token":"fixture-token"}`)
	objects := objectsBody["data"].([]any)
	object := objects[0].(map[string]any)
	if object["obj_id"] != float64(9001) || object["device_type"] != "CUSTOM_DEVICE" {
		t.Fatalf("read_grd_object object = %#v", object)
	}
	if object["display_number"] != "C-9001" || object["object_number"] != "C-9001" {
		t.Fatalf("read_grd_object display number = %#v", object)
	}
}

func TestStaticSiteHandlerServesCASLUIAndAPIOnSameOrigin(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "<html>casl ui</html>")
	writeTestFile(t, filepath.Join(root, "public", "static", "js", "main.js"), "console.log('casl')")
	writeTestFile(t, filepath.Join(root, "public", "static", "js", "650.js"), "const userCodes=t?M[Ae]:[];")
	writeTestFile(t, filepath.Join(root, "configurator_4L", "index.html"), "<html>conf</html>")
	writeTestFile(t, filepath.Join(root, "casl-technic", "index.html"), "<html>tech</html>")

	handler := NewStaticSiteHandler(NewFixtureHandlerWithWSURL("ws://127.0.0.1:23322"), StaticSiteOptions{
		CASLRootDir: root,
	})

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "root index", path: "/", want: "casl ui"},
		{name: "static asset", path: "/static/js/main.js", want: "console.log('casl')"},
		{name: "patched casl chunk", path: "/static/js/650.js", want: "t&&M?M[Ae]:[]"},
		{name: "spa fallback", path: "/objects/7001001", want: "casl ui"},
		{name: "configurator", path: "/conf/", want: "conf"},
		{name: "technic", path: "/tech/", want: "tech"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("body %q does not contain %q", rec.Body.String(), tt.want)
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/captchaShow", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("captcha status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	decodeTestJSON(t, rec.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Fatalf("captcha was not served by api handler: %#v", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api version status = %d body=%s", rec.Code, rec.Body.String())
	}
	decodeTestJSON(t, rec.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Fatalf("api version was not served by api handler: %#v", body)
	}
}

func TestStaticSiteHandlerAcceptsRootWebSocket(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "<html>casl ui</html>")

	server := httptest.NewServer(NewStaticSiteHandler(NewFixtureHandler(), StaticSiteOptions{
		CASLRootDir: root,
	}))
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Logf("close websocket: %v", err)
		}
	})

	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read conn_id: %v", err)
	}
	if msg["type"] != "conn_id" {
		t.Fatalf("unexpected ws message: %#v", msg)
	}
}

func TestFixtureHandler_CommandReadDeviceAndTranslator(t *testing.T) {
	handler := NewFixtureHandler()

	objectsBody := postCommand(t, handler, `{"type":"read_grd_object","token":"fixture-token"}`)
	objects, ok := objectsBody["data"].([]any)
	if !ok || len(objects) != 2 {
		t.Fatalf("read_grd_object data = %#v", objectsBody["data"])
	}
	firstObject := objects[0].(map[string]any)
	if firstObject["device_number"] != float64(7001001) {
		t.Fatalf("unexpected first object device_number: %#v", firstObject["device_number"])
	}
	if firstObject["display_number"] != "7001001" || firstObject["object_number"] != "7001001" {
		t.Fatalf("unexpected first object display numbers: %#v", firstObject)
	}
	if firstObject["device_id"] != float64(7101001) {
		t.Fatalf("unexpected first object device_id: %#v", firstObject["device_id"])
	}

	deviceBody := postCommand(t, handler, `{"type":"read_device","token":"fixture-token","skip":0,"limit":100000}`)
	devices, ok := deviceBody["data"].([]any)
	if !ok || len(devices) != 2 {
		t.Fatalf("read_device data = %#v", deviceBody["data"])
	}
	firstDevice := devices[0].(map[string]any)
	if firstDevice["type"] != "PHOENIXDB_GENERIC" {
		t.Fatalf("unexpected first device type: %#v", firstDevice["type"])
	}
	if _, ok := firstDevice["moreAlarmTime"].([]any); !ok {
		t.Fatalf("first device moreAlarmTime = %#v", firstDevice["moreAlarmTime"])
	}
	if _, ok := firstDevice["ignoringAlarmTime"].([]any); !ok {
		t.Fatalf("first device ignoringAlarmTime = %#v", firstDevice["ignoringAlarmTime"])
	}

	translatorBody := postCommand(t, handler, `{"type":"get_msg_translator_by_device_type","token":"fixture-token","device_type":"PHOENIXDB_GENERIC"}`)
	translator, ok := translatorBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("translator data = %#v", translatorBody["data"])
	}
	rows, ok := translator["PHOENIXDB_GENERIC"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("PHOENIXDB_GENERIC rows = %#v", translator["PHOENIXDB_GENERIC"])
	}
	if len(rows) < 10 {
		t.Fatalf("PHOENIXDB_GENERIC: expected at least 10 event rows, got %d", len(rows))
	}
	hasFireAlarm := false
	for _, r := range rows {
		if m, ok := r.(map[string]any); ok {
			if code, _ := m["code"].(float64); int(code) == 110 {
				if m["type_event"] == "E" {
					hasFireAlarm = true
					break
				}
			}
		}
	}
	if !hasFireAlarm {
		t.Fatalf("PHOENIXDB_GENERIC: no fire alarm (code=110 E) row found in %d rows", len(rows))
	}

	translatorBody = postCommand(t, handler, `{"type":"get_msg_translator_by_device_type","token":"fixture-token","typeDevice":"PHOENIXDB_GENERIC"}`)
	translator, ok = translatorBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("translator by typeDevice data = %#v", translatorBody["data"])
	}
	if _, ok := translator["PHOENIXDB_GENERIC"].([]any); !ok {
		t.Fatalf("translator by typeDevice PHOENIXDB_GENERIC = %#v", translator["PHOENIXDB_GENERIC"])
	}
}

func TestFixtureHandler_CommandShapesExpectedByCASLUI(t *testing.T) {
	handler := NewFixtureHandler()

	statistics := postCommand(t, handler, `{"type":"get_objects_statistic","token":"fixture-token"}`)
	statisticsData, ok := statistics["data"].(map[string]any)
	if !ok {
		t.Fatalf("get_objects_statistic data = %#v", statistics["data"])
	}
	groupStatistics, ok := statisticsData["groupStatistics"].(map[string]any)
	if !ok {
		t.Fatalf("get_objects_statistic groupStatistics = %#v", statisticsData["groupStatistics"])
	}
	if _, ok := groupStatistics["7001001"].(map[string]any); !ok {
		t.Fatalf("groupStatistics[7001001] = %#v", groupStatistics["7001001"])
	}
	if _, ok := statistics["groupStatistics"].(map[string]any); !ok {
		t.Fatalf("top-level groupStatistics = %#v", statistics["groupStatistics"])
	}

	dictionaryBody := postCommand(t, handler, `{"type":"read_dictionary","token":"fixture-token"}`)
	dictionary, ok := dictionaryBody["dictionary"].(map[string]any)
	if !ok {
		t.Fatalf("read_dictionary dictionary = %#v", dictionaryBody["dictionary"])
	}
	if devices, ok := dictionary["devices"].([]any); !ok || len(devices) == 0 {
		t.Fatalf("read_dictionary devices = %#v", dictionary["devices"])
	}
	if _, ok := dictionary["msg_translator"].(map[string]any); !ok {
		t.Fatalf("read_dictionary msg_translator = %#v", dictionary["msg_translator"])
	}
	devicesDict := dictionary["devices"].([]any)
	firstDeviceDict := devicesDict[0].(map[string]any)
	if firstDeviceDict["max_lines"] != float64(999) {
		t.Fatalf("read_dictionary max_lines = %#v", firstDeviceDict["max_lines"])
	}
	if _, ok := dictionary["adapter_types"].([]any); !ok {
		t.Fatalf("read_dictionary adapter_types = %#v", dictionary["adapter_types"])
	}
	if alarmCauses, ok := dictionary["alarm_causes"].([]any); !ok || len(alarmCauses) == 0 {
		t.Fatalf("read_dictionary alarm_causes = %#v", dictionary["alarm_causes"])
	}

	deviceNumbers := postCommand(t, handler, `{"type":"read_devices_numbers","token":"fixture-token"}`)
	if numbers, ok := deviceNumbers["data"].([]any); !ok || len(numbers) == 0 {
		t.Fatalf("read_devices_numbers data = %#v", deviceNumbers["data"])
	}

	templates := postCommand(t, handler, `{"type":"get_templates","token":"fixture-token"}`)
	if _, ok := templates["templates"].(map[string]any); !ok {
		t.Fatalf("get_templates templates = %#v", templates["templates"])
	}

	firmware := postCommand(t, handler, `{"type":"get_firmware_list","token":"fixture-token"}`)
	if files, ok := firmware["files"].([]any); !ok || len(files) == 0 {
		t.Fatalf("get_firmware_list files = %#v", firmware["files"])
	}

	geoZones := postCommand(t, handler, `{"type":"read_geo_zones","token":"fixture-token"}`)
	geoZonesData, ok := geoZones["data"].([]any)
	if !ok || len(geoZonesData) == 0 {
		t.Fatalf("read_geo_zones data = %#v", geoZones["data"])
	}
	geoZone := geoZonesData[0].(map[string]any)
	if mgrs, ok := geoZone["mgrs"].([]any); !ok || len(mgrs) == 0 {
		t.Fatalf("read_geo_zones mgrs = %#v", geoZone["mgrs"])
	}

	alarmEvents := postCommand(t, handler, `{"type":"read_alarm_events","token":"fixture-token"}`)
	if _, ok := alarmEvents["events"].([]any); !ok {
		t.Fatalf("read_alarm_events events = %#v", alarmEvents["events"])
	}

	fullObject := postCommand(t, handler, `{"type":"get_grd_object_full","token":"fixture-token","obj_id":7001001}`)
	if fullObject["obj_id"] != float64(7001001) {
		t.Fatalf("get_grd_object_full obj_id = %#v", fullObject["obj_id"])
	}
	if rooms, ok := fullObject["rooms"].([]any); !ok || len(rooms) == 0 {
		t.Fatalf("get_grd_object_full rooms = %#v", fullObject["rooms"])
	}
	if inCharge, ok := fullObject["in_charge"].([]any); !ok || len(inCharge) == 0 {
		t.Fatalf("get_grd_object_full in_charge = %#v", fullObject["in_charge"])
	}

	rooms := postCommand(t, handler, `{"type":"read_grd_room","token":"fixture-token","obj_id":7001001}`)
	data, ok := rooms["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatalf("read_grd_room data = %#v", rooms["data"])
	}
	room := data[0].(map[string]any)
	lines, ok := room["lines"].(map[string]any)
	if !ok || len(lines) == 0 {
		t.Fatalf("read_grd_room lines = %#v", room["lines"])
	}
	line := lines["1"].(map[string]any)
	if line["device_id"] != float64(7101001) {
		t.Fatalf("read_grd_room line device_id = %#v", line["device_id"])
	}
	if line["adapter_type"] != "SYS" {
		t.Fatalf("read_grd_room line adapter_type = %#v", line["adapter_type"])
	}
	roomUsers, ok := room["users"].([]any)
	if !ok || len(roomUsers) == 0 {
		t.Fatalf("read_grd_room users = %#v", room["users"])
	}

	roomLinks := postCommand(t, handler, `{"type":"get_room_links","token":"fixture-token","room_id":"700100101"}`)
	lineLinks, ok := roomLinks["line_links"].([]any)
	if !ok || len(lineLinks) == 0 {
		t.Fatalf("get_room_links line_links = %#v", roomLinks["line_links"])
	}
	userLinks, ok := roomLinks["user_links"].([]any)
	if !ok || len(userLinks) == 0 {
		t.Fatalf("get_room_links user_links = %#v", roomLinks["user_links"])
	}

	pultUsers := postCommand(t, handler, `{"type":"get_all_pults_users","token":"fixture-token"}`)
	users, ok := pultUsers["allPultsUsers"].([]any)
	if !ok || len(users) == 0 {
		t.Fatalf("get_all_pults_users allPultsUsers = %#v", pultUsers["allPultsUsers"])
	}

	readUsers := postCommand(t, handler, `{"type":"read_user","token":"fixture-token"}`)
	readUsersData, ok := readUsers["data"].([]any)
	if !ok || len(readUsersData) < 2 {
		t.Fatalf("read_user data = %#v", readUsers["data"])
	}
	for _, rawUser := range readUsersData {
		userRow := rawUser.(map[string]any)
		if phones, ok := userRow["phone_numbers"].([]any); !ok || len(phones) == 0 {
			t.Fatalf("read_user %v phone_numbers = %#v", userRow["user_id"], userRow["phone_numbers"])
		}
		if _, ok := userRow["device_ids"].([]any); !ok {
			t.Fatalf("read_user %v device_ids = %#v", userRow["user_id"], userRow["device_ids"])
		}
	}
	responsible := readUsersData[1].(map[string]any)
	if responsible["role"] != "IN_CHARGE" {
		t.Fatalf("read_user responsible role = %#v", responsible["role"])
	}
	if phones, ok := responsible["phone_numbers"].([]any); !ok || len(phones) == 0 {
		t.Fatalf("read_user responsible phone_numbers = %#v", responsible["phone_numbers"])
	}

	access := postCommand(t, handler, `{"type":"get_all_access_by_pult","token":"fixture-token"}`)
	accessData, ok := access["data"].(map[string]any)
	if !ok {
		t.Fatalf("get_all_access_by_pult data = %#v", access["data"])
	}
	if _, ok := accessData["accessDevices"].([]any); !ok {
		t.Fatalf("get_all_access_by_pult accessDevices = %#v", accessData["accessDevices"])
	}
	if accessData["accessTimeToTechns"] != float64(43200000) {
		t.Fatalf("get_all_access_by_pult accessTimeToTechns = %#v", accessData["accessTimeToTechns"])
	}

	managers := postCommand(t, handler, `{"type":"read_mgr","token":"fixture-token"}`)
	managersData, ok := managers["data"].([]any)
	if !ok || len(managersData) == 0 {
		t.Fatalf("read_mgr data = %#v", managers["data"])
	}
	manager := managersData[0].(map[string]any)
	if manager["name"] == "" {
		t.Fatalf("read_mgr name = %#v", manager["name"])
	}
	if _, ok := manager["users"].([]any); !ok {
		t.Fatalf("read_mgr users = %#v", manager["users"])
	}

	for _, typ := range []string{"get_pult_msg", "get_own_msg", "get_system_msg"} {
		body := postCommand(t, handler, `{"type":"`+typ+`","token":"fixture-token"}`)
		if _, ok := body["chat_msgs"].([]any); !ok {
			t.Fatalf("%s chat_msgs = %#v", typ, body["chat_msgs"])
		}
	}
}

func TestFixtureHandler_OperatorCompatibilityCommands(t *testing.T) {
	handler := NewFixtureHandler()

	for _, typ := range []string{"read_events", "read_events_by_id"} {
		body := postCommand(t, handler, `{"type":"`+typ+`","token":"fixture-token","obj_ids":[7001001]}`)
		events, ok := body["data"].([]any)
		if !ok || len(events) == 0 {
			t.Fatalf("%s data = %#v", typ, body["data"])
		}
		event := events[0].(map[string]any)
		if event["type"] != "ppk_event" {
			t.Fatalf("%s event type = %#v", typ, event["type"])
		}

		camelBody := postCommand(t, handler, `{"type":"`+typ+`","token":"fixture-token","objIds":[7001001]}`)
		camelEvents, ok := camelBody["data"].([]any)
		if !ok || len(camelEvents) == 0 {
			t.Fatalf("%s camelCase data = %#v", typ, camelBody["data"])
		}

		windowBody := postCommand(t, handler, `{"type":"`+typ+`","token":"fixture-token","time_start":1777150000000,"time_end":1777159999999}`)
		windowEvents, ok := windowBody["data"].([]any)
		if !ok || len(windowEvents) == 0 {
			t.Fatalf("%s window data = %#v", typ, windowBody["data"])
		}
	}

	statistic := postCommand(t, handler, `{"type":"get_statistic","token":"fixture-token"}`)
	if rows, ok := statistic["data"].([]any); !ok || len(rows) == 0 {
		t.Fatalf("get_statistic data = %#v", statistic["data"])
	}
	if statistic["total_count"] != float64(2) {
		t.Fatalf("get_statistic total_count = %#v", statistic["total_count"])
	}

	alarmStatistic := postCommand(t, handler, `{"type":"get_statistic","token":"fixture-token","name":"stats_alarms","deviceId":70010011,"objectId":7001001}`)
	alarmStatisticData, ok := alarmStatistic["data"].(map[string]any)
	if !ok {
		t.Fatalf("stats_alarms data = %#v", alarmStatistic["data"])
	}
	if alarmStatisticData["obj_id"] != "7001001" || alarmStatisticData["device_id"] != "70010011" {
		t.Fatalf("stats_alarms identity = %#v", alarmStatisticData)
	}
	if _, ok := alarmStatisticData["responseFrequencies"].(float64); !ok {
		t.Fatalf("stats_alarms responseFrequencies = %#v", alarmStatisticData["responseFrequencies"])
	}

	history := postCommand(t, handler, `{"type":"get_record_history","token":"fixture-token","obj_id":7001001}`)
	if _, ok := history["records"].([]any); !ok {
		t.Fatalf("get_record_history records = %#v", history["records"])
	}

	rtsp := postCommand(t, handler, `{"type":"get_rtsp_url","token":"fixture-token"}`)
	if _, ok := rtsp["rtsp_url"].(string); !ok {
		t.Fatalf("get_rtsp_url rtsp_url = %#v", rtsp["rtsp_url"])
	}

	basket := postCommand(t, handler, `{"type":"read_one_from_basket","token":"fixture-token","basket_id":1}`)
	if _, ok := basket["basketElement"].(map[string]any); !ok {
		t.Fatalf("read_one_from_basket basketElement = %#v", basket["basketElement"])
	}

	for _, typ := range []string{
		"save_in_basket",
		"del_from_basket",
		"get_user_send_msg",
		"grd_object_group_action",
		"operator_alarm",
		"group_on_device",
		"user_action",
		"change_disconnected_state",
		"create_grd_object",
		"update_grd_object",
		"delete_grd_object",
	} {
		postCommand(t, handler, `{"type":"`+typ+`","token":"fixture-token","obj_id":7001001}`)
	}
}

func TestFixtureHandler_ReadEventsFallsBackToGeneralTape(t *testing.T) {
	fixture := DefaultFixture()
	fixture.GeneralTapeItems = nil
	handler := NewFixtureHandlerWithFixture(fixture)

	body := postCommand(t, handler, `{"type":"read_events","token":"fixture-token","time_start":1777150000000,"time_end":1777159999999}`)
	events, ok := body["data"].([]any)
	if !ok || len(events) == 0 {
		t.Fatalf("fallback read_events data = %#v", body["data"])
	}
	event := events[0].(map[string]any)
	if event["obj_id"] != float64(7001001) {
		t.Fatalf("fallback event = %#v", event)
	}
	if msg, _ := event["msg"].(string); msg == "" || strings.Contains(msg, "UNIFIED_") {
		t.Fatalf("fallback event should use human msg: %#v", event)
	}
}

func TestFixtureHandler_CommandGeneralTapeItemsFiltersByObjectID(t *testing.T) {
	handler := NewFixtureHandler()

	tape := postCommand(t, handler, `{"type":"get_general_tape_objects","token":"fixture-token"}`)
	tapeRows, ok := tape["data"].([]any)
	if !ok || len(tapeRows) == 0 {
		t.Fatalf("get_general_tape_objects data = %#v", tape["data"])
	}
	tapeRow := tapeRows[0].(map[string]any)
	for _, key := range []string{"name", "address", "pult_id", "description", "reasonAlarm", "last_act"} {
		if _, ok := tapeRow[key]; !ok {
			t.Fatalf("get_general_tape_objects missing %s in %#v", key, tapeRow)
		}
	}

	body := postCommand(t, handler, `{"type":"get_general_tape_item","token":"fixture-token","obj_ids":[7001001]}`)
	items, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("items data = %#v", body["data"])
	}
	rawRows, ok := items["7001001"].([]any)
	if !ok || len(rawRows) != 2 {
		t.Fatalf("expected object key 7001001 in %#v", items)
	}
	ppkEvent := rawRows[0].(map[string]any)
	if ppkEvent["code"] != float64(110) || ppkEvent["number"] != float64(2) {
		t.Fatalf("unexpected ppk event = %#v", ppkEvent)
	}
	userAction := rawRows[1].(map[string]any)
	if userAction["dict_name"] != "GRD_OBJ_NOTIF" {
		t.Fatalf("unexpected user action = %#v", userAction)
	}

	camelBody := postCommand(t, handler, `{"type":"get_general_tape_item","token":"fixture-token","objIds":[7001001]}`)
	camelItems, ok := camelBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("camel items data = %#v", camelBody["data"])
	}
	if _, ok := camelItems["7001001"].([]any); !ok {
		t.Fatalf("expected camelCase object key 7001001 in %#v", camelItems)
	}
	if _, ok := camelItems["7002002"]; ok {
		t.Fatalf("unexpected camelCase object 7002002 in %#v", camelItems)
	}
}

func TestFixtureHandler_GuardedObjectActionUpdatesAlarmWorkflow(t *testing.T) {
	handler := NewFixtureHandler()

	pick := postCommand(t, handler, `{"type":"grd_object_action","token":"fixture-token","obj_id":7001001,"action":"GRD_OBJ_PICK"}`)
	if pick["reacting_pult_id"] != "1" {
		t.Fatalf("GRD_OBJ_PICK reacting_pult_id = %#v", pick["reacting_pult_id"])
	}

	tape := postCommand(t, handler, `{"type":"get_general_tape_objects","token":"fixture-token"}`)
	rows, ok := tape["data"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("get_general_tape_objects after pick = %#v", tape["data"])
	}
	row := rows[0].(map[string]any)
	if row["user_id"] != "100" {
		t.Fatalf("picked user_id = %#v", row["user_id"])
	}
	if row["last_act"] != "GRD_OBJ_PICK" {
		t.Fatalf("picked last_act = %#v", row["last_act"])
	}

	postCommand(t, handler, `{"type":"grd_object_action","token":"fixture-token","obj_id":7001001,"action":"GRD_OBJ_ASS_MGR","mgr_id":"1"}`)
	tape = postCommand(t, handler, `{"type":"get_general_tape_objects","token":"fixture-token"}`)
	row = tape["data"].([]any)[0].(map[string]any)
	if row["mgr_id"] != "1" || row["last_act"] != "GRD_OBJ_ASS_MGR" {
		t.Fatalf("assigned mgr row = %#v", row)
	}

	postCommand(t, handler, `{"type":"grd_object_action","token":"fixture-token","obj_id":7001001,"action":"GRD_OBJ_MGR_ARRIVE","mgr_id":"1"}`)
	tape = postCommand(t, handler, `{"type":"get_general_tape_objects","token":"fixture-token"}`)
	row = tape["data"].([]any)[0].(map[string]any)
	if row["last_act"] != "GRD_OBJ_MGR_ARRIVE" {
		t.Fatalf("mgr arrive last_act = %#v", row["last_act"])
	}

	postCommand(t, handler, `{"type":"grd_object_action","token":"fixture-token","obj_id":7001001,"action":"GRD_OBJ_FINISH","cause":"CAUSES_CUSTOM_WINS","note":"done"}`)
	tape = postCommand(t, handler, `{"type":"get_general_tape_objects","token":"fixture-token"}`)
	if rows, ok := tape["data"].([]any); !ok || len(rows) != 0 {
		t.Fatalf("get_general_tape_objects after finish = %#v", tape["data"])
	}

	items := postCommand(t, handler, `{"type":"get_general_tape_item","token":"fixture-token","obj_ids":[7001001]}`)
	itemMap, ok := items["data"].(map[string]any)
	if !ok {
		t.Fatalf("items data = %#v", items["data"])
	}
	itemRows, ok := itemMap["7001001"].([]any)
	if !ok || len(itemRows) < 5 {
		t.Fatalf("expected user actions in tape item = %#v", itemMap["7001001"])
	}
	lastAction := itemRows[len(itemRows)-1].(map[string]any)
	if lastAction["dict_name"] != "GRD_OBJ_FINISH" || lastAction["cause"] != "CAUSES_CUSTOM_WINS" {
		t.Fatalf("finish action row = %#v", lastAction)
	}
}

func TestFixtureHandler_ProxiesCASLNamespacedCommandToUpstream(t *testing.T) {
	handler := NewFixtureHandler()
	upstream := &testCommandUpstream{
		response: map[string]any{"status": "ok", "native": true},
	}
	handler.SetCommandUpstream(upstream)

	caslObjID := ids.CASLObjectIDNamespaceStart + 27
	body := postCommand(t, handler, `{"type":"grd_object_action","token":"fixture-token","obj_id":`+fmt.Sprintf("%d", caslObjID)+`,"action":"GRD_OBJ_PICK"}`)

	if body["native"] != true {
		t.Fatalf("expected native upstream response, got %#v", body)
	}
	if len(upstream.calls) != 1 {
		t.Fatalf("upstream calls = %d, want 1", len(upstream.calls))
	}
	call := upstream.calls[0]
	if _, ok := call["token"]; ok {
		t.Fatalf("fixture token must not be forwarded upstream: %#v", call)
	}
	if call["obj_id"] != 27 {
		t.Fatalf("obj_id was not normalized for upstream: %#v", call)
	}
}

func TestFixtureHandler_BCApiCompatibilityRoutes(t *testing.T) {
	handler := NewFixtureHandler()

	stateReq := httptest.NewRequest(http.MethodGet, "/api/devices/state", nil)
	stateRec := httptest.NewRecorder()
	handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state status = %d body=%s", stateRec.Code, stateRec.Body.String())
	}
	var body map[string]any
	decodeTestJSON(t, stateRec.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Fatalf("state status payload = %#v", body)
	}
	if _, ok := body["data"].(map[string]any); !ok {
		t.Fatalf("state data = %#v", body["data"])
	}

	commandReq := httptest.NewRequest(
		http.MethodPost,
		"/api/devices/123/command",
		bytes.NewBufferString(`{
			"command": "turn_on",
			"entity_name": "relay",
			"entity_number": 0,
			"device_password": "x",
			"device_license_key": "180-244-6-132-4-200"
		}`),
	)
	commandRec := httptest.NewRecorder()
	handler.ServeHTTP(commandRec, commandReq)
	if commandRec.Code != http.StatusOK {
		t.Fatalf("device command status = %d body=%s", commandRec.Code, commandRec.Body.String())
	}

	reportReq := httptest.NewRequest(http.MethodPost, "/api/report", bytes.NewBufferString(`{"alarmId":1}`))
	reportRec := httptest.NewRecorder()
	handler.ServeHTTP(reportRec, reportReq)
	if reportRec.Code != http.StatusOK {
		t.Fatalf("report status = %d body=%s", reportRec.Code, reportRec.Body.String())
	}
}

func TestFixtureHandler_SubscribeAcceptsKnownShape(t *testing.T) {
	handler := NewFixtureHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	connID := readWSConnID(t, conn)
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(`{"token":"fixture-token","conn_id":"`+connID+`","tag":"ppk_in","pult_id":1}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFixtureHandler_WebSocketSendsConnID(t *testing.T) {
	server := httptest.NewServer(NewFixtureHandler())
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read conn_id: %v", err)
	}
	if msg["type"] != "conn_id" {
		t.Fatalf("unexpected ws message: %#v", msg)
	}
	if strings.TrimSpace(msg["id"].(string)) == "" {
		t.Fatalf("empty conn_id: %#v", msg)
	}
}

func TestFixtureHandler_WebSocketBroadcastsGuardedObjectAction(t *testing.T) {
	handler := NewFixtureHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	var connMsg map[string]any
	if err := conn.ReadJSON(&connMsg); err != nil {
		t.Fatalf("read conn_id: %v", err)
	}
	connID, _ := connMsg["id"].(string)
	if connID == "" {
		t.Fatalf("conn_id is empty: %#v", connMsg)
	}

	subscribeResp, err := http.Post(
		server.URL+"/subscribe",
		ContentTypeJSON,
		strings.NewReader(`{"token":"fixture-token","conn_id":"`+connID+`","tag":"user_action","pult_id":1}`),
	)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer subscribeResp.Body.Close()
	if subscribeResp.StatusCode != http.StatusOK {
		t.Fatalf("subscribe status = %d", subscribeResp.StatusCode)
	}

	resp, err := http.Post(
		server.URL+"/command",
		ContentTypeJSON,
		strings.NewReader(`{"type":"grd_object_action","token":"fixture-token","obj_id":7001001,"action":"GRD_OBJ_PICK"}`),
	)
	if err != nil {
		t.Fatalf("post command: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post command status = %d", resp.StatusCode)
	}

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var actionMsg map[string]any
	if err := conn.ReadJSON(&actionMsg); err != nil {
		t.Fatalf("read user_action: %v", err)
	}
	if actionMsg["type"] != "user_action" || actionMsg["action"] != "GRD_OBJ_PICK" {
		t.Fatalf("unexpected user action message: %#v", actionMsg)
	}
	if actionMsg["user_id"] != "100" {
		t.Fatalf("unexpected user_id: %#v", actionMsg["user_id"])
	}
}

func postCommand(t *testing.T, handler http.Handler, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var decoded map[string]any
	decodeTestJSON(t, rec.Body.Bytes(), &decoded)
	if decoded["status"] != "ok" {
		t.Fatalf("unexpected command status: %#v", decoded)
	}
	return decoded
}

func readWSConnID(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read conn_id: %v", err)
	}
	connID, _ := msg["id"].(string)
	if msg["type"] != "conn_id" || strings.TrimSpace(connID) == "" {
		t.Fatalf("unexpected conn_id message: %#v", msg)
	}
	return connID
}

type testCommandUpstream struct {
	calls    []map[string]any
	response map[string]any
}

func (u *testCommandUpstream) ExecuteCASLCommand(_ context.Context, payload map[string]any, _ bool) (map[string]any, error) {
	copied := make(map[string]any, len(payload))
	for key, value := range payload {
		copied[key] = value
	}
	u.calls = append(u.calls, copied)
	return u.response, nil
}

func decodeTestJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode json %s: %v", string(body), err)
	}
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestGenerateLicenseKey(t *testing.T) {
	tail := string([]byte{0x00, 0x7b, 0x01, 0xc8}) // PPKNum = 123 (0x007b), Key = 456 (0x01c8)
	dCh := dChecksumDecStr(tail)
	calc := checksumHex16(dCh + tail)
	c0, _ := strconv.ParseInt(calc[0:2], 16, 32)
	c1, _ := strconv.ParseInt(calc[2:4], 16, 32)
	
	decoded := string([]byte{byte(c0), byte(c1), 0x00, 0x7b, 0x01, 0xc8})
	
	b := -1
	out := make([]byte, len(decoded))
	for i := 0; i < len(decoded); i++ {
		b++
		if b == len(digitSet)-1 {
			b = 0
		}
		shift, _ := strconv.Atoi(string(digitSet[b]))
		out[i] = decoded[i] + byte(shift)
	}
	
	parts := make([]string, 6)
	for i, v := range out {
		parts[i] = strconv.Itoa(int(v))
	}
	keyStr := strings.Join(parts, "-")
	t.Logf("Generated Key: %s", keyStr)
}

