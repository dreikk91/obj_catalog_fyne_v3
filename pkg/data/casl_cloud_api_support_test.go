package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCASLProvider_ServiceEndpoints(t *testing.T) {
	var loginCalls int
	var subscribeCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslCaptchaShowPath:
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for captchaShow: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","captchaShow":false,"GoogleCaptchaSiteKey":"site-key"}`))
		case caslTimeServerPath:
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for get_time_server: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"time":"2026-03-29T06:40:40.148Z"}`))
		case caslLoginPath:
			loginCalls++
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if strings.TrimSpace(asString(payload["email"])) != "test@lot.lviv.ua" {
				t.Fatalf("unexpected email: %v", payload["email"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-service","user_id":"483","ws_url":"ws://10.0.0.1:23322"}`))
		case caslSubscribePath:
			subscribeCalls++
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if strings.TrimSpace(asString(payload["token"])) != "token-service" {
				t.Fatalf("expected token-service, got %v", payload["token"])
			}
			if strings.TrimSpace(asString(payload["conn_id"])) != "conn-1" {
				t.Fatalf("unexpected conn_id: %v", payload["conn_id"])
			}
			if strings.TrimSpace(asString(payload["tag"])) != "ppk_in" {
				t.Fatalf("unexpected tag: %v", payload["tag"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	captcha, err := provider.GetCaptchaConfig(ctx)
	if err != nil {
		t.Fatalf("GetCaptchaConfig error: %v", err)
	}
	if captcha.GoogleCaptchaSiteKey != "site-key" || captcha.CaptchaShow {
		t.Fatalf("unexpected captcha response: %+v", captcha)
	}

	serverTime, err := provider.GetServerTime(ctx)
	if err != nil {
		t.Fatalf("GetServerTime error: %v", err)
	}
	if serverTime.IsZero() {
		t.Fatalf("expected non-zero server time")
	}

	session, err := provider.EnsureAuthorized(ctx)
	if err != nil {
		t.Fatalf("EnsureAuthorized error: %v", err)
	}
	if session.Token != "token-service" || session.UserID != "483" {
		t.Fatalf("unexpected session: %+v", session)
	}

	if err := provider.Subscribe(ctx, "conn-1", "ppk_in"); err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}

	if loginCalls != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
	if subscribeCalls != 1 {
		t.Fatalf("expected 1 subscribe call, got %d", subscribeCalls)
	}
}

func TestCASLProvider_CommandWrappers(t *testing.T) {
	called := map[string]int{}
	var loginCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			loginCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-api","user_id":"u1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			called[cmdType]++

			if cmdType != "read_pult" {
				if strings.TrimSpace(asString(payload["token"])) != "token-api" {
					t.Fatalf("command %s expected token-api, got %v", cmdType, payload["token"])
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch cmdType {
			case "read_pult":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"pult_id":"1","name":"Pult One"}]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Obj 24","device_id":"23","device_number":1003}]}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"3","last_name":"Petrenko"}]}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23","number":1003}]}`))
			case "read_mgr":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"mgr_id":"11","name":"Crew 11"}]}`))
			case "read_connections":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","device_id":"23","user_id":"3"}]}`))
			case "read_grd_room":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"room_id":"1","name":"Room A"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"roles":["ADMIN"]}}`))
			case "read_alarm_events":
				_, _ = w.Write([]byte(`{"status":"ok","events":[{"code":"DOOR_OP","is_alarm_in_start":1,"is_alarm":1}]}`))
			case "get_objects_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"countOfRooms":204}}`))
			case "get_disconnected_devices":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"device_id":"23"}]}`))
			case "get_all_access_by_pult":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"accessDevices":[],"accessTimeToTechns":43200000}}`))
			case "get_firmware_list":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"file":"fw_v1.hex.enc"}]}`))
			case "read_from_basket":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24"}]}`))
			case "monitor":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"db":"ok","queue":"ok"}}`))
			case "get_msg_translator_by_device_type":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"device_type":"MAKS_PRO","dict":{"100":"Event"}}}`))
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"event_id":"1"}]}`))
			case "get_rtsp_url":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"url":"rtsp://camera"}}`))
			case "read_count_in_basket":
				_, _ = w.Write([]byte(`{"status":"ok","count":101}`))
			case "read_events":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"ppk_num":1003,"time":1774769226380,"code":"TEST","number":1}]}`))
			case "read_events_by_id":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"ppk_num":1003,"time":1774769226380,"code":"GROUP_ON","type":"ppk_event","number":1,"contact_id":"R401"}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":-1,"accum":-1,"door":-1,"online":0,"lastPingDate":1774769732941,"lines":{},"groups":{},"adapters":{}}}`))
			case "get_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"device_id":"23","obj_id":"24","responseFrequencies":6,"communicQuality":6,"powerFailure":6,"criminogenicity":0,"customWins":13}}`))
			case "group_on_device", "group_off_device", "update_grd_object", "grd_obj_pick", "grd_obj_finish":
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pults, err := provider.ReadPults(ctx, 0, 1000)
	if err != nil || len(pults) != 1 {
		t.Fatalf("ReadPults failed: len=%d err=%v", len(pults), err)
	}

	objects, err := provider.ReadGuardObjects(ctx, 0, 1000)
	if err != nil || len(objects) != 1 {
		t.Fatalf("ReadGuardObjects failed: len=%d err=%v", len(objects), err)
	}

	users, err := provider.ReadUsersRaw(ctx, 0, 1000)
	if err != nil || len(users) != 1 {
		t.Fatalf("ReadUsersRaw failed: len=%d err=%v", len(users), err)
	}

	devices, err := provider.ReadDevices(ctx, 0, 1000)
	if err != nil || len(devices) != 1 {
		t.Fatalf("ReadDevices failed: len=%d err=%v", len(devices), err)
	}

	managers, err := provider.ReadManagers(ctx, 0, 1000)
	if err != nil || len(managers) != 1 {
		t.Fatalf("ReadManagers failed: len=%d err=%v", len(managers), err)
	}

	connections, err := provider.ReadConnections(ctx, 0, 1000)
	if err != nil || len(connections) != 1 {
		t.Fatalf("ReadConnections failed: len=%d err=%v", len(connections), err)
	}

	rooms, err := provider.ReadGuardRooms(ctx, 0, 1000)
	if err != nil || len(rooms) != 1 {
		t.Fatalf("ReadGuardRooms failed: len=%d err=%v", len(rooms), err)
	}

	dict, err := provider.ReadDictionary(ctx)
	if err != nil {
		t.Fatalf("ReadDictionary failed: %v", err)
	}
	if _, ok := dict["roles"]; !ok {
		t.Fatalf("expected roles in dictionary")
	}

	alarmEvents, err := provider.ReadAlarmEventsCatalog(ctx)
	if err != nil || len(alarmEvents) != 1 {
		t.Fatalf("ReadAlarmEventsCatalog failed: len=%d err=%v", len(alarmEvents), err)
	}
	if alarmEvents[0].Code != "DOOR_OP" {
		t.Fatalf("unexpected alarm event code: %s", alarmEvents[0].Code)
	}

	statsMap, err := provider.GetObjectsStatistic(ctx)
	if err != nil {
		t.Fatalf("GetObjectsStatistic failed: %v", err)
	}
	if asString(statsMap["countOfRooms"]) != "204" {
		t.Fatalf("unexpected countOfRooms: %v", statsMap["countOfRooms"])
	}

	disconnected, err := provider.GetDisconnectedDevices(ctx)
	if err != nil || len(disconnected) != 1 {
		t.Fatalf("GetDisconnectedDevices failed: len=%d err=%v", len(disconnected), err)
	}

	access, err := provider.GetAllAccessByPult(ctx)
	if err != nil {
		t.Fatalf("GetAllAccessByPult failed: %v", err)
	}
	if asString(access["accessTimeToTechns"]) != "43200000" {
		t.Fatalf("unexpected accessTimeToTechns: %v", access["accessTimeToTechns"])
	}

	firmware, err := provider.GetFirmwareList(ctx)
	if err != nil || len(firmware) != 1 {
		t.Fatalf("GetFirmwareList failed: len=%d err=%v", len(firmware), err)
	}

	fromBasket, err := provider.ReadFromBasket(ctx, 0, 1000)
	if err != nil || len(fromBasket) != 1 {
		t.Fatalf("ReadFromBasket failed: len=%d err=%v", len(fromBasket), err)
	}

	monitor, err := provider.Monitor(ctx)
	if err != nil {
		t.Fatalf("Monitor failed: %v", err)
	}
	if asString(monitor["db"]) != "ok" {
		t.Fatalf("unexpected monitor payload: %#v", monitor)
	}

	translator, err := provider.GetMessageTranslatorByDeviceType(ctx, "MAKS_PRO")
	if err != nil {
		t.Fatalf("GetMessageTranslatorByDeviceType failed: %v", err)
	}
	if _, ok := translator.(map[string]any); !ok {
		t.Fatalf("unexpected translator payload: %#v", translator)
	}

	tape, err := provider.ReadGeneralTapeObjects(ctx)
	if err != nil || len(tape) != 1 {
		t.Fatalf("ReadGeneralTapeObjects failed: len=%d err=%v", len(tape), err)
	}

	rtsp, err := provider.GetRTSPURL(ctx)
	if err != nil {
		t.Fatalf("GetRTSPURL failed: %v", err)
	}
	rtspMap, ok := rtsp.(map[string]any)
	if !ok || strings.TrimSpace(asString(rtspMap["url"])) != "rtsp://camera" {
		t.Fatalf("unexpected rtsp payload: %#v", rtsp)
	}

	basketCount, err := provider.ReadBasketCount(ctx)
	if err != nil {
		t.Fatalf("ReadBasketCount failed: %v", err)
	}
	if basketCount != 101 {
		t.Fatalf("unexpected basket count: %d", basketCount)
	}

	journalEvents, err := provider.ReadEventsJournal(ctx, CASLReadEventsRequest{})
	if err != nil || len(journalEvents) != 1 {
		t.Fatalf("ReadEventsJournal failed: len=%d err=%v", len(journalEvents), err)
	}
	if journalEvents[0].Code != "TEST" {
		t.Fatalf("unexpected journal code: %s", journalEvents[0].Code)
	}

	events, err := provider.ReadEventsByID(ctx, CASLReadEventsByIDRequest{
		ObjIDs:        []string{"24"},
		DeviceIDs:     []string{"23"},
		DeviceNumbers: []int64{1003},
	})
	if err != nil || len(events) != 1 {
		t.Fatalf("ReadEventsByID failed: len=%d err=%v", len(events), err)
	}
	if events[0].Code != "GROUP_ON" {
		t.Fatalf("unexpected event code: %s", events[0].Code)
	}

	state, err := provider.ReadDeviceStateByID(ctx, "23")
	if err != nil {
		t.Fatalf("ReadDeviceStateByID failed: %v", err)
	}
	if state.LastPingDate != 1774769732941 {
		t.Fatalf("unexpected lastPingDate: %d", state.LastPingDate)
	}

	stats, err := provider.GetStatistic(ctx, CASLGetStatisticRequest{DeviceID: "23", ObjectID: "24"})
	if err != nil {
		t.Fatalf("GetStatistic failed: %v", err)
	}
	if stats.CustomWins != 13 {
		t.Fatalf("unexpected CustomWins: %d", stats.CustomWins)
	}

	if err := provider.GroupOnDevice(ctx, 1003, 1); err != nil {
		t.Fatalf("GroupOnDevice failed: %v", err)
	}
	if err := provider.GroupOffDevice(ctx, 1003, 1); err != nil {
		t.Fatalf("GroupOffDevice failed: %v", err)
	}
	if _, err := provider.UpdateGuardObject(ctx, map[string]any{"obj_id": "24", "name": "Updated"}); err != nil {
		t.Fatalf("UpdateGuardObject failed: %v", err)
	}
	if err := provider.PickGuardObject(ctx, "24", "845920"); err != nil {
		t.Fatalf("PickGuardObject failed: %v", err)
	}
	if err := provider.FinishGuardObject(ctx, "24", "845920"); err != nil {
		t.Fatalf("FinishGuardObject failed: %v", err)
	}

	generic, err := provider.ExecuteCASLCommand(ctx, map[string]any{"type": "read_count_in_basket"}, true)
	if err != nil {
		t.Fatalf("ExecuteCASLCommand failed: %v", err)
	}
	if asString(generic["count"]) != strconv.Itoa(101) {
		t.Fatalf("unexpected generic response: %#v", generic)
	}

	expectedCommands := []string{
		"read_pult",
		"read_grd_object",
		"read_user",
		"read_device",
		"read_mgr",
		"read_connections",
		"read_grd_room",
		"read_dictionary",
		"read_alarm_events",
		"get_objects_statistic",
		"get_disconnected_devices",
		"get_all_access_by_pult",
		"get_firmware_list",
		"read_from_basket",
		"monitor",
		"get_msg_translator_by_device_type",
		"get_general_tape_objects",
		"get_rtsp_url",
		"read_count_in_basket",
		"read_events",
		"read_events_by_id",
		"read_device_state",
		"get_statistic",
		"group_on_device",
		"group_off_device",
		"update_grd_object",
		"grd_obj_pick",
		"grd_obj_finish",
	}

	for _, cmd := range expectedCommands {
		if called[cmd] == 0 {
			t.Fatalf("expected command %s to be called", cmd)
		}
	}

	if loginCalls != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
}
