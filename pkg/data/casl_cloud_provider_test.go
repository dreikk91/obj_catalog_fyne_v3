package data

import (
	"encoding/json"
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
			if payload["type"] != "read_grd_object" {
				t.Fatalf("unexpected command type: %v", payload["type"])
			}
			if strings.TrimSpace(asString(payload["token"])) != "token-1" {
				t.Fatalf("expected token-1, got %v", payload["token"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","address":"Addr","device_id":"23","device_number":1003,"rooms":[{"room_id":"1","name":"Room A","description":"Desc","rtsp":""}]}]}`))
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
	if commandCalls != 1 {
		t.Fatalf("expected 1 command call, got %d", commandCalls)
	}

	gotByID := provider.GetObjectByID(strconv.Itoa(objects[0].ID))
	if gotByID == nil {
		t.Fatalf("expected object by ID")
	}
	if gotByID.Name != "Object 24" {
		t.Fatalf("unexpected object name: %q", gotByID.Name)
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
			if strings.TrimSpace(asString(payload["token"])) != "fresh-token" {
				t.Fatalf("expected refreshed token, got %v", payload["token"])
			}
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"25","name":"Object 25","device_id":"31","device_number":1004}]}`))
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
	if commandCalls != 2 {
		t.Fatalf("expected 2 command calls, got %d", commandCalls)
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
			case "read_user":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"user_id":"3","last_name":"Petrenko","first_name":"Ihor","middle_name":"M","role":"IN_CHARGE","phone_numbers":[{"active":true,"number":"+380971112233"}]}]}`))
			case "read_events_by_id":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"ppk_num":1003,"time":1774769226380,"code":"GROUP_ON","type":"ppk_event","number":1,"contact_id":"R401"}]}`))
			case "read_device_state":
				_, _ = w.Write([]byte(`{"status":"ok","state":{"power":-1,"accum":-1,"door":-1,"online":0,"lastPingDate":` + strconv.FormatInt(lastPing, 10) + `,"lines":{},"groups":{},"adapters":{}}}`))
			case "get_statistic":
				_, _ = w.Write([]byte(`{"status":"ok","data":{"device_id":"23","obj_id":"24","responseFrequencies":5,"communicQuality":5,"powerFailure":5,"criminogenicity":0,"customWins":3}}`))
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
	if len(zones) != 1 || zones[0].Name != "Room A" {
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

	signal, testMsg, _, lastMsg := provider.GetExternalData(objectID)
	if !strings.Contains(signal, "online=0") {
		t.Fatalf("unexpected signal payload: %q", signal)
	}
	if !strings.Contains(testMsg, "freq=5") {
		t.Fatalf("unexpected test payload: %q", testMsg)
	}
	if lastMsg.IsZero() {
		t.Fatalf("expected non-zero last message time")
	}
}

func TestCASLProvider_GetEvents_UsesGeneralTape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-tape","user_id":"u-3","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			cmdType := strings.TrimSpace(asString(payload["type"]))
			w.Header().Set("Content-Type", "application/json")

			switch cmdType {
			case "get_general_tape_objects":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"event_id":"845920","time":"2026-03-29T10:10:10Z","obj_id":"24","event_type":"arm","zone":1,"description":"Object armed","user_id":"operator-1"},{"event_id":"845921","time":"2026-03-29T10:11:10Z","obj_id":"24","event_type":"fire","zone":2,"description":"Fire alarm","user_id":"operator-2"}]}`))
			case "read_grd_object":
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"obj_id":"24","name":"Object 24","device_id":"23","device_number":1003}]}`))
			default:
				t.Fatalf("unexpected command type: %s", cmdType)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
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
}
