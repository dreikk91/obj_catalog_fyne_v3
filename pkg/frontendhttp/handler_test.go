package frontendhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

type frontendBackendStub struct {
	capabilitiesResult contracts.FrontendCapabilities
	capabilitiesErr    error

	objectsResult []contracts.FrontendObjectSummary
	objectsErr    error

	alarmsResult                 []contracts.FrontendAlarmItem
	alarmsErr                    error
	alarmProcessingOptionsResult []contracts.FrontendAlarmProcessingOption
	alarmProcessingOptionsErr    error
	alarmProcessingOptionsID     int
	pickAlarmInput               contracts.FrontendAlarmPickRequest
	pickAlarmErr                 error
	pickAlarmID                  int
	processAlarmInput            contracts.FrontendAlarmProcessRequest
	processAlarmErr              error
	processAlarmID               int

	eventsResult         []contracts.FrontendEventItem
	eventsErr            error
	objectEventsResult   contracts.FrontendEventPage
	objectEventsErr      error
	objectEventsObjectID int
	objectEventsOffset   int
	objectEventsLimit    int

	detailsResult contracts.FrontendObjectDetails
	detailsErr    error
	detailsID     int

	createResult contracts.FrontendObjectMutationResult
	createErr    error
	createInput  contracts.FrontendObjectUpsertRequest

	updateResult contracts.FrontendObjectMutationResult
	updateErr    error
	updateInput  contracts.FrontendObjectUpsertRequest
}

func (s *frontendBackendStub) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	return s.capabilitiesResult, s.capabilitiesErr
}

func (s *frontendBackendStub) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	return s.objectsResult, s.objectsErr
}

func (s *frontendBackendStub) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	return s.alarmsResult, s.alarmsErr
}

func (s *frontendBackendStub) GetAlarmProcessingOptions(_ context.Context, alarmID int) ([]contracts.FrontendAlarmProcessingOption, error) {
	s.alarmProcessingOptionsID = alarmID
	return s.alarmProcessingOptionsResult, s.alarmProcessingOptionsErr
}

func (s *frontendBackendStub) ListAlarmProcessingOptionsCached(context.Context) ([]contracts.FrontendAlarmProcessingOption, error) {
	return s.alarmProcessingOptionsResult, s.alarmProcessingOptionsErr
}

func (s *frontendBackendStub) PickAlarm(_ context.Context, alarmID int, request contracts.FrontendAlarmPickRequest) error {
	s.pickAlarmID = alarmID
	s.pickAlarmInput = request
	return s.pickAlarmErr
}

func (s *frontendBackendStub) ProcessAlarm(_ context.Context, alarmID int, request contracts.FrontendAlarmProcessRequest) error {
	s.processAlarmID = alarmID
	s.processAlarmInput = request
	return s.processAlarmErr
}

func (s *frontendBackendStub) GroupProcessAlarm(context.Context, int, string) error {
	return nil
}

func (s *frontendBackendStub) StandbyObject(context.Context, int, contracts.FrontendStandbyRequest) error {
	return nil
}

func (s *frontendBackendStub) ListResponseGroups(context.Context) ([]contracts.FrontendResponseGroup, error) {
	return nil, nil
}

func (s *frontendBackendStub) AssignResponseGroup(context.Context, int, contracts.FrontendAlarmGroupActionRequest) error {
	return nil
}

func (s *frontendBackendStub) NotifyGroupArrived(context.Context, int) error {
	return nil
}

func (s *frontendBackendStub) CancelResponseGroup(context.Context, int) error {
	return nil
}

func (s *frontendBackendStub) ListEvents(context.Context) ([]contracts.FrontendEventItem, error) {
	return s.eventsResult, s.eventsErr
}

func (s *frontendBackendStub) ListObjectEvents(_ context.Context, objectID int, offset int, limit int) (contracts.FrontendEventPage, error) {
	s.objectEventsObjectID = objectID
	s.objectEventsOffset = offset
	s.objectEventsLimit = limit
	return s.objectEventsResult, s.objectEventsErr
}

func (s *frontendBackendStub) GetObjectDetails(_ context.Context, objectID int) (contracts.FrontendObjectDetails, error) {
	s.detailsID = objectID
	return s.detailsResult, s.detailsErr
}

func (s *frontendBackendStub) CreateObject(_ context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	s.createInput = request
	return s.createResult, s.createErr
}

func (s *frontendBackendStub) UpdateObject(_ context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	s.updateInput = request
	return s.updateResult, s.updateErr
}

func TestHandlerListObjects(t *testing.T) {
	stub := &frontendBackendStub{
		objectsResult: []contracts.FrontendObjectSummary{
			{
				ID:            12,
				Source:        contracts.FrontendSourceBridge,
				DisplayNumber: "12",
				Name:          "Школа",
				StatusCode:    "normal",
				StatusText:    "НОРМА",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/objects", nil)
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Items []frontendv1.ObjectSummary `json:"items"`
	}
	decodeJSON(t, rec, &payload)
	if len(payload.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(payload.Items))
	}
	if payload.Items[0].Name != "Школа" {
		t.Fatalf("items[0].Name = %q, want %q", payload.Items[0].Name, "Школа")
	}
}

func TestHandlerGetObjectDetails(t *testing.T) {
	stub := &frontendBackendStub{
		detailsResult: contracts.FrontendObjectDetails{
			Summary: contracts.FrontendObjectSummary{
				ID:            55,
				Source:        contracts.FrontendSourceCASL,
				DisplayNumber: "4001",
				Name:          "CASL object",
			},
			ExternalSignal:           "GPRS",
			PreferredResponseGroupID: "1",
		},
	}

	req := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/objects/55", nil)
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if stub.detailsID != 55 {
		t.Fatalf("details id = %d, want 55", stub.detailsID)
	}

	var payload frontendv1.ObjectDetails
	decodeJSON(t, rec, &payload)
	if payload.Summary.Name != "CASL object" {
		t.Fatalf("summary.Name = %q, want %q", payload.Summary.Name, "CASL object")
	}
	if payload.PreferredResponseGroupID != "1" {
		t.Fatalf("preferred response group = %q, want 1", payload.PreferredResponseGroupID)
	}
}

func TestHandlerCreateObject(t *testing.T) {
	stub := &frontendBackendStub{
		createResult: contracts.FrontendObjectMutationResult{
			Source:   contracts.FrontendSourceBridge,
			ObjectID: 1004,
			NativeID: "1004",
		},
	}

	body := frontendv1.ObjectUpsertRequest{
		Source: frontendv1.SourceBridge,
		Core: frontendv1.ObjectCoreFields{
			Name:    "Новий об'єкт",
			Address: "Львів",
		},
		Legacy: &frontendv1.LegacyObjectPayload{
			ObjN:      1004,
			ObjTypeID: 7,
			ShortName: "Новий об'єкт",
		},
	}

	req := httptest.NewRequest(http.MethodPost, APIV1BasePath+"/objects", encodeJSONBody(t, body))
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if stub.createInput.Legacy == nil || stub.createInput.Legacy.ObjN != 1004 {
		t.Fatalf("create input objn = %+v, want 1004", stub.createInput.Legacy)
	}
}

func TestHandlerUpdateObjectInjectsPathID(t *testing.T) {
	stub := &frontendBackendStub{
		updateResult: contracts.FrontendObjectMutationResult{
			Source:   contracts.FrontendSourceCASL,
			ObjectID: 1500000042,
			NativeID: "42",
		},
	}

	body := frontendv1.ObjectUpsertRequest{
		Source: frontendv1.SourceCASL,
		Core: frontendv1.ObjectCoreFields{
			Address: "Оновлена адреса",
		},
		CASL: &frontendv1.CASLObjectPayload{
			Status: "active",
		},
	}

	req := httptest.NewRequest(http.MethodPut, APIV1BasePath+"/objects/1500000042", encodeJSONBody(t, body))
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if stub.updateInput.ObjectID != 1500000042 {
		t.Fatalf("update object id = %d, want 1500000042", stub.updateInput.ObjectID)
	}
}

func TestHandlerBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, APIV1BasePath+"/objects", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	NewHandler(&frontendBackendStub{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlerBackendErrorMapping(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "unsupported source",
			err:        contracts.ErrUnsupportedFrontendSource,
			wantStatus: http.StatusNotImplemented,
		},
		{
			name:       "service unavailable",
			err:        contracts.ErrFrontendBackendUnavailable,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "not found",
			err:        errors.New("object not found"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "validation",
			err:        errors.New("invalid request"),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stub := &frontendBackendStub{objectsErr: tc.err}
			req := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/objects", nil)
			rec := httptest.NewRecorder()

			NewHandler(stub).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandlerCapabilities(t *testing.T) {
	lastPing := time.Date(2026, time.April, 22, 6, 11, 55, 0, time.UTC)
	stub := &frontendBackendStub{
		capabilitiesResult: contracts.FrontendCapabilities{
			Sources: []contracts.FrontendSourceCapability{
				{
					Source:            contracts.FrontendSourcePhoenix,
					DisplayName:       contracts.FrontendSourcePhoenix.DisplayName(),
					ReadObjects:       true,
					ReadObjectDetails: true,
					ReadEvents:        true,
					ReadAlarms:        true,
					HealthStatus:      contracts.FrontendSourceHealthStatusDegraded,
					HealthText:        "CASL: API online, але WS не отримує ping понад 12 с",
					APIStatus:         contracts.FrontendConnectionStatusOnline,
					RealtimeStatus:    contracts.FrontendConnectionStatusOffline,
					LastRealtimePing:  lastPing,
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/capabilities", nil)
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload frontendv1.Capabilities
	decodeJSON(t, rec, &payload)
	if len(payload.Sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(payload.Sources))
	}
	if payload.Sources[0].HealthStatus != "degraded" {
		t.Fatalf("health status = %q, want %q", payload.Sources[0].HealthStatus, "degraded")
	}
	if payload.Sources[0].LastRealtimePing != lastPing.Format(time.RFC3339) {
		t.Fatalf("last realtime ping = %q, want %q", payload.Sources[0].LastRealtimePing, lastPing.Format(time.RFC3339))
	}
}

func TestHandlerListEventsAndAlarms(t *testing.T) {
	stub := &frontendBackendStub{
		eventsResult: []contracts.FrontendEventItem{
			{
				ID:         1,
				Source:     contracts.FrontendSourceBridge,
				ObjectID:   100,
				ObjectName: "Об'єкт",
				Time:       time.Now(),
			},
		},
		alarmsResult: []contracts.FrontendAlarmItem{
			{
				ID:         2,
				Source:     contracts.FrontendSourceCASL,
				ObjectID:   1500000001,
				ObjectName: "CASL",
				Time:       time.Now(),
			},
		},
	}

	reqEvents := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/events", nil)
	recEvents := httptest.NewRecorder()
	NewHandler(stub).ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("events status = %d, want %d", recEvents.Code, http.StatusOK)
	}

	reqAlarms := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/alarms", nil)
	recAlarms := httptest.NewRecorder()
	NewHandler(stub).ServeHTTP(recAlarms, reqAlarms)
	if recAlarms.Code != http.StatusOK {
		t.Fatalf("alarms status = %d, want %d", recAlarms.Code, http.StatusOK)
	}
}

func TestHandlerListObjectEventsPage(t *testing.T) {
	stub := &frontendBackendStub{
		objectEventsResult: contracts.FrontendEventPage{
			Items: []contracts.FrontendEventItem{
				{
					ID:         7,
					ObjectID:   55,
					ObjectName: "Об'єкт",
					Time:       time.Now(),
				},
			},
			TotalCount: 140,
			HasMore:    true,
		},
	}

	req := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/objects/55/events?offset=100&limit=100", nil)
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if stub.objectEventsObjectID != 55 || stub.objectEventsOffset != 100 || stub.objectEventsLimit != 100 {
		t.Fatalf("object events args = %d/%d/%d, want 55/100/100", stub.objectEventsObjectID, stub.objectEventsOffset, stub.objectEventsLimit)
	}

	var payload frontendv1.EventPageResponse
	decodeJSON(t, rec, &payload)
	if payload.TotalCount != 140 || !payload.HasMore || len(payload.Items) != 1 {
		t.Fatalf("payload = %+v, want 1 item and total=140 hasMore=true", payload)
	}
}

func TestHandlerAlarmProcessingOptions(t *testing.T) {
	stub := &frontendBackendStub{
		alarmProcessingOptionsResult: []contracts.FrontendAlarmProcessingOption{
			{Code: "CAUSES_FALSE_ALARM", Label: "Хибна тривога"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, APIV1BasePath+"/alarms/77/processing-options", nil)
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if stub.alarmProcessingOptionsID != 77 {
		t.Fatalf("alarm id = %d, want 77", stub.alarmProcessingOptionsID)
	}

	var payload frontendv1.AlarmProcessingOptionsResponse
	decodeJSON(t, rec, &payload)
	if len(payload.Items) != 1 || payload.Items[0].Code != "CAUSES_FALSE_ALARM" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestHandlerProcessAlarm(t *testing.T) {
	stub := &frontendBackendStub{}

	req := httptest.NewRequest(
		http.MethodPost,
		APIV1BasePath+"/alarms/88/process",
		encodeJSONBody(t, frontendv1.AlarmProcessRequest{
			User:      "Підлипний А.М",
			CauseCode: "CAUSES_FALSE_ALARM",
			Note:      "Перевірено",
		}),
	)
	rec := httptest.NewRecorder()

	NewHandler(stub).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if stub.processAlarmID != 88 {
		t.Fatalf("alarm id = %d, want 88", stub.processAlarmID)
	}
	if stub.processAlarmInput.CauseCode != "CAUSES_FALSE_ALARM" || stub.processAlarmInput.Note != "Перевірено" {
		t.Fatalf("process input = %+v", stub.processAlarmInput)
	}
}

func encodeJSONBody(t *testing.T, payload any) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return bytes.NewReader(body)
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}
