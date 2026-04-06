package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestCASLProvider_GetCASLObjectEditorSnapshot(t *testing.T) {
	t.Parallel()

	payloads := make(map[string]map[string]any)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-editor","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmd := strings.TrimSpace(asString(payload["type"]))
			payloads[cmd] = payload
			w.Header().Set("Content-Type", "application/json")
			switch cmd {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"29","name":"1007 Офіс","device_id":"28","device_number":1007}]}`))
			case "get_grd_object_full":
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"name":"1007 Офіс",
					"address":"Львів, Зелена 69",
					"lat":"49.813556",
					"long":"24.059843",
					"description":"Опис",
					"pult_id":"1",
					"reacting_pult_id":"2",
					"contract":"1029ос",
					"user_id":"3",
					"note":"Примітка",
					"start_date":1679695200000,
					"object_type":"Офіс",
					"id_request":"123",
					"geo_zone_id":2,
					"bissnes_coeff":1.5,
					"rooms":[
						{
							"room_id":"36",
							"name":"Офіс 1",
							"description":"Немає опису1",
							"images":["data:image/jpeg base64,ZmFrZQ=="],
							"rtsp":"rtsp://camera",
							"users":[{"user_id":"41","priority":1,"hoz_num":null}],
							"lines":{"5":{"adapter_type":"SYS","group_number":1,"adapter_number":0}}
						}
					],
					"device":{"id":"28","number":1007,"name":"MAKS PRO","type":"TYPE_DEVICE_Ajax","lastPingDate":1775336400000},
					"obj_status":"1",
					"images":["48","49"]
				}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[
					{"user_id":"3","last_name":"Менеджер","first_name":"Марія","role":"MANAGER","phone_numbers":[{"active":true,"number":"+380501112233"}]},
					{"user_id":"41","last_name":"Іваненко","first_name":"Іван","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+380671112233"}]},
					{"user_id":"281","last_name":"Технік","first_name":"Тарас","role":"TECHNICIAN"}
				]}`))
			case "read_pult":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"pult_id":"1","name":"Пульт 1"},{"pult_id":"2","name":"Пульт 2"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_Ajax":"Ajax"}}}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[
					{
						"device_id":"28",
						"obj_id":"29",
						"number":1007,
						"name":"MAKS PRO 1",
						"type":"TYPE_DEVICE_Ajax",
						"timeout":3600,
						"sim1":"+38 (050) 329-92-04",
						"sim2":"+38 (063) 123-45-67",
						"technician_id":"281",
						"units":"1",
						"requisites":"req",
						"change_date":1775336400000,
						"reglament_date":1775682000000,
						"licence_key":"lic",
						"passw_remote":"pass",
						"lines":[
							{"line_id":213,"line_number":5,"group_number":1,"adapter_type":"SYS","adapter_number":0,"description":"Штора вікна тил","line_type":"EMPTY","isBlocked":false}
						]
					}
				]}`))
			default:
				t.Fatalf("unexpected command type: %v", payload["type"])
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	internalID := int64(mapCASLObjectID("29", "1007 Офіс", "1007"))
	snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, internalID)
	if err != nil {
		t.Fatalf("GetCASLObjectEditorSnapshot failed: %v", err)
	}

	if snapshot.Object.Name != "1007 Офіс" || snapshot.Object.ManagerID != "3" {
		t.Fatalf("unexpected object snapshot: %+v", snapshot.Object)
	}
	if len(snapshot.Object.Rooms) != 1 || snapshot.Object.Rooms[0].RoomID != "36" {
		t.Fatalf("unexpected rooms: %+v", snapshot.Object.Rooms)
	}
	if len(snapshot.Object.Rooms[0].Images) != 1 || !strings.HasPrefix(snapshot.Object.Rooms[0].Images[0], "data:image/jpeg") {
		t.Fatalf("unexpected room images: %+v", snapshot.Object.Rooms[0].Images)
	}
	if len(snapshot.Object.Device.Lines) != 1 || snapshot.Object.Device.Lines[0].RoomID != "36" {
		t.Fatalf("unexpected device lines: %+v", snapshot.Object.Device.Lines)
	}
	if snapshot.Object.Device.Timeout != 3600 || snapshot.Object.Device.TechnicianID != "281" {
		t.Fatalf("unexpected device details: %+v", snapshot.Object.Device)
	}
	if len(snapshot.Users) != 3 || len(snapshot.Pults) != 2 {
		t.Fatalf("unexpected references: users=%d pults=%d", len(snapshot.Users), len(snapshot.Pults))
	}
	if got := strings.TrimSpace(asString(payloads["get_grd_object_full"]["obj_id"])); got != "29" {
		t.Fatalf("unexpected obj_id in get_grd_object_full: %q", got)
	}
}

func TestCASLProvider_GetCASLObjectEditorSnapshotBootstrap(t *testing.T) {
	t.Parallel()

	payloads := make(map[string]map[string]any)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-editor","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmd := strings.TrimSpace(asString(payload["type"]))
			payloads[cmd] = payload
			w.Header().Set("Content-Type", "application/json")
			switch cmd {
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"3","last_name":"Менеджер","first_name":"Марія","role":"MANAGER"}]}`))
			case "read_pult":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"pult_id":"1","name":"Пульт 1"}]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_Ajax":"Ajax"}}}`))
			default:
				t.Fatalf("unexpected bootstrap command type: %v", payload["type"])
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, 0)
	if err != nil {
		t.Fatalf("GetCASLObjectEditorSnapshot bootstrap failed: %v", err)
	}

	if snapshot.Object.ObjID != "" {
		t.Fatalf("expected empty object in bootstrap snapshot: %+v", snapshot.Object)
	}
	if len(snapshot.Users) != 1 || len(snapshot.Pults) != 1 {
		t.Fatalf("unexpected bootstrap references: users=%d pults=%d", len(snapshot.Users), len(snapshot.Pults))
	}
	if _, ok := payloads["get_grd_object_full"]; ok {
		t.Fatalf("bootstrap snapshot should not call get_grd_object_full")
	}
}

func TestCASLProvider_GetCASLObjectEditorSnapshot_DeviceMapTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-editor-map","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmd := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")
			switch cmd {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"29","name":"1007 Офіс","device_id":"28","device_number":1007}]}`))
			case "get_grd_object_full":
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"name":"1007 Офіс",
					"address":"Львів, Зелена 69",
					"rooms":[],
					"device":{"id":"28","number":1007,"name":"MAKS PRO","type":"TYPE_DEVICE_Ajax"},
					"devices":{"1":{"device_id":"28","obj_id":"29","number":1007,"timeout":360,"sim1":"+38 (050) 329-92-04","lines":{"1":{"line_id":213,"line_number":1,"group_number":1,"adapter_type":"SYS","adapter_number":0,"description":"Штора вікна тил","line_type":"EMPTY","isBlocked":false}}}},
					"obj_status":"1",
					"images":[]
				}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_pult":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_Ajax":"Ajax"}}}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			default:
				t.Fatalf("unexpected command type: %v", payload["type"])
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	internalID := int64(mapCASLObjectID("29", "1007 Офіс", "1007"))
	snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, internalID)
	if err != nil {
		t.Fatalf("GetCASLObjectEditorSnapshot failed: %v", err)
	}

	if snapshot.Object.Device.Timeout != 360 {
		t.Fatalf("unexpected timeout: %+v", snapshot.Object.Device)
	}
	if snapshot.Object.Device.SIM1 != "+38 (050) 329-92-04" {
		t.Fatalf("unexpected SIM1: %+v", snapshot.Object.Device)
	}
	if len(snapshot.Object.Device.Lines) != 1 || snapshot.Object.Device.Lines[0].LineNumber != 1 {
		t.Fatalf("unexpected lines: %+v", snapshot.Object.Device.Lines)
	}
}

func TestCASLProvider_GetCASLObjectEditorSnapshot_BusinessCoeffString(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-editor-coeff","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmd := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")
			switch cmd {
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"29","name":"1007 Офіс","device_id":"28","device_number":1007}]}`))
			case "get_grd_object_full":
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"name":"1007 Офіс",
					"bissnes_coeff":"1.5",
					"rooms":[],
					"device":{"id":"28","number":1007,"name":"MAKS PRO","type":"TYPE_DEVICE_Ajax"},
					"obj_status":"1",
					"images":[]
				}`))
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_pult":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			case "read_dictionary":
				_, _ = w.Write([]byte(`{"status":"ok","dictionary":{"device_types":{"TYPE_DEVICE_Ajax":"Ajax"}}}`))
			case "read_device":
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			default:
				t.Fatalf("unexpected command type: %v", payload["type"])
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	internalID := int64(mapCASLObjectID("29", "1007 Офіс", "1007"))
	snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, internalID)
	if err != nil {
		t.Fatalf("GetCASLObjectEditorSnapshot failed: %v", err)
	}
	if snapshot.Object.BusinessCoeff == nil {
		t.Fatalf("expected business coeff to be decoded")
	}
	if *snapshot.Object.BusinessCoeff != 1.5 {
		t.Fatalf("unexpected business coeff: %v", *snapshot.Object.BusinessCoeff)
	}
}

func TestCASLProvider_ObjectEditorMutations(t *testing.T) {
	t.Parallel()

	payloads := make(map[string]map[string]any)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-editor","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmd := strings.TrimSpace(asString(payload["type"]))
			payloads[cmd] = payload
			w.Header().Set("Content-Type", "application/json")
			if cmd == "create_user" {
				_, _ = w.Write([]byte(`{"status":"ok","user_id":"484"}`))
				return
			}
			if cmd == "read_user" {
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"484","last_name":"Нове","first_name":"Імя","role":"IN_CHARGE"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.UpdateCASLObject(ctx, contracts.CASLGuardObjectUpdate{ObjID: "29", Name: "Obj"}); err != nil {
		t.Fatalf("UpdateCASLObject failed: %v", err)
	}
	if err := provider.UpdateCASLRoom(ctx, contracts.CASLRoomUpdate{ObjID: "29", RoomID: "36", Name: "Room"}); err != nil {
		t.Fatalf("UpdateCASLRoom failed: %v", err)
	}
	if err := provider.CreateCASLRoom(ctx, contracts.CASLRoomCreate{ObjID: "29", Name: "New room"}); err != nil {
		t.Fatalf("CreateCASLRoom failed: %v", err)
	}
	if err := provider.UpdateCASLDevice(ctx, contracts.CASLDeviceUpdate{DeviceID: "28", Number: 1007, DeviceType: "TYPE_DEVICE_Ajax"}); err != nil {
		t.Fatalf("UpdateCASLDevice failed: %v", err)
	}
	lineID := int64(213)
	if err := provider.UpdateCASLDeviceLine(ctx, contracts.CASLDeviceLineMutation{DeviceID: "28", LineID: &lineID, LineNumber: 5, LineType: "EMPTY"}); err != nil {
		t.Fatalf("UpdateCASLDeviceLine failed: %v", err)
	}
	if err := provider.CreateCASLDeviceLine(ctx, contracts.CASLDeviceLineMutation{DeviceID: "28", LineNumber: 6, LineType: "EMPTY"}); err != nil {
		t.Fatalf("CreateCASLDeviceLine failed: %v", err)
	}
	if err := provider.AddCASLLineToRoom(ctx, contracts.CASLLineToRoomBinding{ObjID: "29", DeviceID: "28", RoomID: "36", LineNumber: 5}); err != nil {
		t.Fatalf("AddCASLLineToRoom failed: %v", err)
	}
	if err := provider.AddCASLUserToRoom(ctx, contracts.CASLAddUserToRoomRequest{ObjID: "29", RoomID: "36", UserID: "41", Priority: 1}); err != nil {
		t.Fatalf("AddCASLUserToRoom failed: %v", err)
	}
	if err := provider.RemoveCASLUserFromRoom(ctx, contracts.CASLRemoveUserFromRoomRequest{ObjID: "29", RoomID: "36", UserID: "41"}); err != nil {
		t.Fatalf("RemoveCASLUserFromRoom failed: %v", err)
	}
	if err := provider.CreateCASLImage(ctx, contracts.CASLImageCreateRequest{
		ObjID:     "29",
		RoomID:    "36",
		ImageType: "png",
		ImageData: "ZmFrZQ==",
	}); err != nil {
		t.Fatalf("CreateCASLImage failed: %v", err)
	}
	if err := provider.DeleteCASLImage(ctx, contracts.CASLImageDeleteRequest{
		ObjID:   "29",
		RoomID:  "36",
		ImageID: "381",
	}); err != nil {
		t.Fatalf("DeleteCASLImage failed: %v", err)
	}
	internalID := int64(mapCASLObjectID("29", "1007 Офіс", "1007"))
	provider.mu.Lock()
	provider.objectByInternalID = map[int]caslGrdObject{
		int(internalID): {
			ObjID:        "29",
			Name:         "1007 Офіс",
			DeviceNumber: caslInt64(1007),
		},
	}
	provider.mu.Unlock()
	if err := provider.UpdateCASLRoomUserPriorities(ctx, internalID, []contracts.CASLRoomUserPriority{{UserID: "41", RoomID: "36", Priority: 1}}); err != nil {
		t.Fatalf("UpdateCASLRoomUserPriorities failed: %v", err)
	}
	user, err := provider.CreateCASLUser(ctx, contracts.CASLUserCreateRequest{
		LastName:  "Нове",
		FirstName: "Імя",
		Role:      "IN_CHARGE",
	})
	if err != nil {
		t.Fatalf("CreateCASLUser failed: %v", err)
	}
	if user.UserID != "484" {
		t.Fatalf("unexpected created user: %+v", user)
	}

	expectedCommands := []string{
		"update_grd_object",
		"update_grd_room",
		"create_grd_room",
		"update_device",
		"update_device_line",
		"create_device_line",
		"add_line_to_room",
		"add_user_to_room",
		"remove_user_from_room",
		"create_image",
		"delete_image",
		"upd_priority_user_in_room",
		"create_user",
	}
	for _, cmd := range expectedCommands {
		if _, ok := payloads[cmd]; !ok {
			t.Fatalf("expected command %s to be sent", cmd)
		}
	}

	if got := strings.TrimSpace(asString(payloads["update_grd_object"]["obj_id"])); got != "29" {
		t.Fatalf("unexpected obj_id in update_grd_object: %q", got)
	}
	if got := strings.TrimSpace(asString(payloads["upd_priority_user_in_room"]["obj_id"])); got != "29" {
		t.Fatalf("unexpected obj_id in upd_priority_user_in_room: %q", got)
	}
	if got := strings.TrimSpace(asString(payloads["create_image"]["obj_id"])); got != "29" {
		t.Fatalf("unexpected obj_id in create_image: %q", got)
	}
	if got := strings.TrimSpace(asString(payloads["create_image"]["room_id"])); got != "36" {
		t.Fatalf("unexpected room_id in create_image: %q", got)
	}
	if got := strings.TrimSpace(asString(payloads["create_image"]["image_type"])); got != "png" {
		t.Fatalf("unexpected image_type in create_image: %q", got)
	}
	if got := strings.TrimSpace(asString(payloads["delete_image"]["image_id"])); got != "381" {
		t.Fatalf("unexpected image_id in delete_image: %q", got)
	}
	if got := strings.TrimSpace(asString(payloads["create_device_line"]["line_type"])); got != "EMPTY" {
		t.Fatalf("unexpected line_type in create_device_line: %q", got)
	}
}

func TestCASLProvider_ObjectEditorCreateFlow(t *testing.T) {
	t.Parallel()

	payloads := make(map[string]map[string]any)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-editor","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmd := strings.TrimSpace(asString(payload["type"]))
			payloads[cmd] = payload
			w.Header().Set("Content-Type", "application/json")
			switch cmd {
			case "create_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","obj_id":"278"}`))
			case "read_devices_numbers":
				_, _ = w.Write([]byte(`{"status":"ok","data":[1007,1008,1010]}`))
			case "is_device_number_in_use":
				_, _ = w.Write([]byte(`{"status":"ok","data":false}`))
			case "create_device":
				_, _ = w.Write([]byte(`{"status":"ok","device_id":"278"}`))
			case "created_new_device":
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			default:
				t.Fatalf("unexpected create command type: %v", payload["type"])
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := provider.CreateCASLObject(ctx, contracts.CASLGuardObjectCreate{
		Name:           "Магазин Свіжа Сарделька 2000",
		Address:        "Львів м., Широка вул., 1",
		Long:           "30.522892",
		Lat:            "50.450697",
		Description:    "Реагує Степан",
		Contract:       "11001",
		ManagerID:      "3",
		Note:           "Включено",
		StartDate:      1775336400000,
		Status:         "Включено",
		ReactingPultID: "1",
		GeoZoneID:      4,
	})
	if err != nil {
		t.Fatalf("CreateCASLObject failed: %v", err)
	}
	if objID != "278" {
		t.Fatalf("unexpected objID: %q", objID)
	}

	numbers, err := provider.ReadCASLDeviceNumbers(ctx)
	if err != nil {
		t.Fatalf("ReadCASLDeviceNumbers failed: %v", err)
	}
	if len(numbers) != 3 || numbers[0] != 1007 || numbers[2] != 1010 {
		t.Fatalf("unexpected device numbers: %+v", numbers)
	}

	inUse, err := provider.IsCASLDeviceNumberInUse(ctx, 9998)
	if err != nil {
		t.Fatalf("IsCASLDeviceNumberInUse failed: %v", err)
	}
	if inUse {
		t.Fatalf("expected device number to be free")
	}

	deviceID, err := provider.CreateCASLDevice(ctx, contracts.CASLDeviceCreate{
		Number:       9998,
		Name:         "Тестовий об'єкт",
		DeviceType:   "TYPE_DEVICE_Ajax",
		Timeout:      3600,
		TechnicianID: "281",
		SIM1:         "+38 (067) 123-45-67",
		SIM2:         "+38 (093) 123-45-67",
	})
	if err != nil {
		t.Fatalf("CreateCASLDevice failed: %v", err)
	}
	if deviceID != "278" {
		t.Fatalf("unexpected deviceID: %q", deviceID)
	}

	if got := strings.TrimSpace(asString(payloads["create_grd_object"]["name"])); got != "Магазин Свіжа Сарделька 2000" {
		t.Fatalf("unexpected object name in create_grd_object: %q", got)
	}
	if got := parseCASLAnyInt(payloads["is_device_number_in_use"]["device_number"]); got != 9998 {
		t.Fatalf("unexpected device_number in is_device_number_in_use: %d", got)
	}
	if got := parseCASLAnyInt(payloads["create_device"]["number"]); got != 9998 {
		t.Fatalf("unexpected number in create_device: %d", got)
	}
	if got := strings.TrimSpace(asString(payloads["created_new_device"]["device_number"])); got != "9998" {
		t.Fatalf("unexpected device_number in created_new_device: %q", got)
	}
}

func TestCASLProvider_FetchCASLImagePreview(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/images/45/token-editor":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("fake-image"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "token-editor", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := provider.FetchCASLImagePreview(ctx, "45")
	if err != nil {
		t.Fatalf("FetchCASLImagePreview failed: %v", err)
	}
	if string(body) != "fake-image" {
		t.Fatalf("unexpected image body: %q", string(body))
	}
}
