package wailsbridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

type frontendBackendStub struct {
	capabilities           contracts.FrontendCapabilities
	objects                []contracts.FrontendObjectSummary
	alarms                 []contracts.FrontendAlarmItem
	alarmProcessingOptions []contracts.FrontendAlarmProcessingOption
	events                 []contracts.FrontendEventItem
	objectEvents           contracts.FrontendEventPage
	details                contracts.FrontendObjectDetails

	capErr                    error
	objectsErr                error
	alarmsErr                 error
	alarmProcessingOptionsErr error
	eventsErr                 error
	objectEventsErr           error
	detailsErr                error
}

func (s *frontendBackendStub) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	if s.capErr != nil {
		return contracts.FrontendCapabilities{}, s.capErr
	}
	return s.capabilities, nil
}

func (s *frontendBackendStub) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	if s.objectsErr != nil {
		return nil, s.objectsErr
	}
	return s.objects, nil
}

func (s *frontendBackendStub) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	if s.alarmsErr != nil {
		return nil, s.alarmsErr
	}
	return s.alarms, nil
}

func (s *frontendBackendStub) GetAlarmProcessingOptions(context.Context, int) ([]contracts.FrontendAlarmProcessingOption, error) {
	if s.alarmProcessingOptionsErr != nil {
		return nil, s.alarmProcessingOptionsErr
	}
	return s.alarmProcessingOptions, nil
}

func (s *frontendBackendStub) ListAlarmProcessingOptionsCached(context.Context) ([]contracts.FrontendAlarmProcessingOption, error) {
	if s.alarmProcessingOptionsErr != nil {
		return nil, s.alarmProcessingOptionsErr
	}
	return s.alarmProcessingOptions, nil
}

func (s *frontendBackendStub) PickAlarm(context.Context, int, contracts.FrontendAlarmPickRequest) error {
	return nil
}

func (s *frontendBackendStub) ProcessAlarm(context.Context, int, contracts.FrontendAlarmProcessRequest) error {
	return nil
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
	if s.eventsErr != nil {
		return nil, s.eventsErr
	}
	return s.events, nil
}

func (s *frontendBackendStub) ListObjectEvents(context.Context, int, int, int) (contracts.FrontendEventPage, error) {
	if s.objectEventsErr != nil {
		return contracts.FrontendEventPage{}, s.objectEventsErr
	}
	return s.objectEvents, nil
}

func (s *frontendBackendStub) GetObjectDetails(context.Context, int) (contracts.FrontendObjectDetails, error) {
	if s.detailsErr != nil {
		return contracts.FrontendObjectDetails{}, s.detailsErr
	}
	return s.details, nil
}

func (s *frontendBackendStub) CreateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, nil
}

func (s *frontendBackendStub) UpdateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, nil
}

func TestFrontendV1ServiceUnavailableBackend(t *testing.T) {
	service := NewFrontendV1Service(nil)

	if _, err := service.ListObjects(); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("ListObjects err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if _, err := service.ListAlarms(); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("ListAlarms err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if _, err := service.ListEvents(); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("ListEvents err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if _, err := service.GetAlarmProcessingOptions(1); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("GetAlarmProcessingOptions err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if err := service.PickAlarm(1, frontendv1.AlarmPickRequest{}); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("PickAlarm err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if err := service.ProcessAlarm(1, frontendv1.AlarmProcessRequest{}); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("ProcessAlarm err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if _, err := service.ListObjectEvents(1, 0, 100); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("ListObjectEvents err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
	if _, err := service.GetObjectDetails(1); !errors.Is(err, ErrFrontendBackendUnavailable) {
		t.Fatalf("GetObjectDetails err = %v, want %v", err, ErrFrontendBackendUnavailable)
	}
}

func TestFrontendV1ServiceMapping(t *testing.T) {
	now := time.Date(2026, time.April, 18, 20, 11, 0, 0, time.UTC)
	lastPing := time.Date(2026, time.April, 22, 6, 11, 55, 0, time.UTC)
	backend := &frontendBackendStub{
		capabilities: contracts.FrontendCapabilities{
			Sources: []contracts.FrontendSourceCapability{
				{
					Source:            contracts.FrontendSourceBridge,
					DisplayName:       "МІСТ",
					ReadObjects:       true,
					ReadObjectDetails: true,
					ReadAlarms:        true,
					ReadEvents:        true,
					HealthStatus:      contracts.FrontendSourceHealthStatusOnline,
					HealthText:        "CASL: API і WS online",
					APIStatus:         contracts.FrontendConnectionStatusOnline,
					RealtimeStatus:    contracts.FrontendConnectionStatusOnline,
					LastRealtimePing:  lastPing,
				},
			},
		},
		objects: []contracts.FrontendObjectSummary{
			{
				ID:               11,
				Source:           contracts.FrontendSourceBridge,
				DisplayNumber:    "11",
				Name:             "Obj",
				Address:          "Addr",
				StatusText:       "Норма",
				GuardStatus:      contracts.FrontendGuardStatusGuarded,
				ConnectionStatus: contracts.FrontendConnectionStatusOnline,
				MonitoringStatus: contracts.FrontendMonitoringStatusActive,
			},
		},
		alarms: []contracts.FrontendAlarmItem{
			{
				ID:             77,
				ObjectID:       11,
				ObjectNumber:   "11",
				ObjectName:     "Obj",
				Time:           now,
				TypeText:       "Пожежа",
				VisualSeverity: contracts.FrontendVisualSeverityCritical,
			},
		},
		alarmProcessingOptions: []contracts.FrontendAlarmProcessingOption{
			{Code: "CAUSES_FALSE_ALARM", Label: "Хибна тривога"},
		},
		events: []contracts.FrontendEventItem{
			{
				ID:             88,
				ObjectID:       11,
				ObjectNumber:   "11",
				ObjectName:     "Obj",
				Time:           now,
				TypeText:       "Тест",
				VisualSeverity: contracts.FrontendVisualSeverityInfo,
			},
		},
		objectEvents: contracts.FrontendEventPage{
			Items: []contracts.FrontendEventItem{
				{
					ID:             89,
					ObjectID:       11,
					ObjectNumber:   "11",
					ObjectName:     "Obj",
					Time:           now.Add(time.Minute),
					TypeText:       "Оновлення",
					VisualSeverity: contracts.FrontendVisualSeverityInfo,
				},
			},
			TotalCount: 1,
			HasMore:    false,
		},
		details: contracts.FrontendObjectDetails{
			Summary: contracts.FrontendObjectSummary{
				ID:            11,
				Source:        contracts.FrontendSourceBridge,
				DisplayNumber: "11",
				Name:          "Obj",
			},
			Phones:                   "380...",
			Location:                 "Kyiv",
			PreferredResponseGroupID: "5",
		},
	}
	service := NewFrontendV1Service(backend)

	if !service.IsReady() {
		t.Fatalf("service should be ready")
	}

	caps, err := service.Capabilities()
	if err != nil {
		t.Fatalf("Capabilities err = %v", err)
	}
	if len(caps.Sources) != 1 || caps.Sources[0].Source != frontendv1.SourceBridge {
		t.Fatalf("Capabilities mapping failed: %+v", caps.Sources)
	}
	if caps.Sources[0].LastRealtimePing != lastPing.Format(time.RFC3339) {
		t.Fatalf("Capabilities last ping failed: %+v", caps.Sources[0])
	}

	objects, err := service.ListObjects()
	if err != nil {
		t.Fatalf("ListObjects err = %v", err)
	}
	if len(objects) != 1 || objects[0].ID != 11 || objects[0].Source != frontendv1.SourceBridge {
		t.Fatalf("ListObjects mapping failed: %+v", objects)
	}

	alarms, err := service.ListAlarms()
	if err != nil {
		t.Fatalf("ListAlarms err = %v", err)
	}
	if len(alarms) != 1 || alarms[0].VisualSeverity != frontendv1.VisualSeverityCritical {
		t.Fatalf("ListAlarms mapping failed: %+v", alarms)
	}

	options, err := service.GetAlarmProcessingOptions(77)
	if err != nil {
		t.Fatalf("GetAlarmProcessingOptions err = %v", err)
	}
	if len(options) != 1 || options[0].Code != "CAUSES_FALSE_ALARM" {
		t.Fatalf("GetAlarmProcessingOptions mapping failed: %+v", options)
	}

	events, err := service.ListEvents()
	if err != nil {
		t.Fatalf("ListEvents err = %v", err)
	}
	if len(events) != 1 || events[0].VisualSeverity != frontendv1.VisualSeverityInfo {
		t.Fatalf("ListEvents mapping failed: %+v", events)
	}

	page, err := service.ListObjectEvents(11, 0, 100)
	if err != nil {
		t.Fatalf("ListObjectEvents err = %v", err)
	}
	if len(page.Items) != 1 || page.TotalCount != 1 || page.HasMore {
		t.Fatalf("ListObjectEvents mapping failed: %+v", page)
	}

	details, err := service.GetObjectDetails(11)
	if err != nil {
		t.Fatalf("GetObjectDetails err = %v", err)
	}
	if details.Summary.ID != 11 || details.Location != "Kyiv" || details.PreferredResponseGroupID != "5" {
		t.Fatalf("GetObjectDetails mapping failed: %+v", details)
	}
}
