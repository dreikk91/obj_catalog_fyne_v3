package data

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestCombinedDataProviderFindObjectsBySIMPhoneChecksEverySource(t *testing.T) {
	bridge := &combinedSIMLookupStub{
		combinedStubProvider: &combinedStubProvider{},
		usages:               []contracts.AdminSIMPhoneUsage{{ObjN: 101, Name: "Магазин", Slot: "SIM 1"}},
	}
	phoenix := &combinedSIMLookupStub{
		combinedStubProvider: &combinedStubProvider{},
		usages:               []contracts.AdminSIMPhoneUsage{{ObjN: 200001, DisplayNumber: "P-22", Slot: "SIM 2"}},
	}
	casl := &combinedSIMLookupStub{
		combinedStubProvider: &combinedStubProvider{},
		usages:               []contracts.AdminSIMPhoneUsage{{ObjN: 300001, DisplayNumber: "C-33", Slot: "SIM 1"}},
	}
	provider := NewMultiSourceDataProvider(
		ProviderSource{Name: "bridge", Provider: bridge},
		ProviderSource{Name: "phoenix", Provider: phoenix},
		ProviderSource{Name: "casl", Provider: casl},
	)

	got, err := provider.FindObjectsBySIMPhone("+38 (075) 447-96-15", nil)
	if err != nil {
		t.Fatalf("FindObjectsBySIMPhone() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("FindObjectsBySIMPhone() returned %d usages, want 3: %+v", len(got), got)
	}
	sources := make(map[string]bool, len(got))
	for _, usage := range got {
		sources[usage.Source] = true
	}
	for _, want := range []string{"МІСТ/Firebird", "Phoenix", "CASL Cloud"} {
		if !sources[want] {
			t.Fatalf("missing source %q in %+v", want, got)
		}
	}
}

func TestCombinedDataProviderFindObjectsBySIMPhoneReturnsPartialResultsWithError(t *testing.T) {
	bridge := &combinedSIMLookupStub{
		combinedStubProvider: &combinedStubProvider{},
		usages:               []contracts.AdminSIMPhoneUsage{{ObjN: 101, Slot: "SIM 1"}},
	}
	casl := &combinedSIMLookupStub{
		combinedStubProvider: &combinedStubProvider{},
		err:                  errors.New("API unavailable"),
	}
	provider := NewMultiSourceDataProvider(
		ProviderSource{Name: "bridge", Provider: bridge},
		ProviderSource{Name: "casl", Provider: casl},
	)

	got, err := provider.FindObjectsBySIMPhone("0754479615", nil)
	if err == nil || !strings.Contains(err.Error(), "CASL Cloud") {
		t.Fatalf("FindObjectsBySIMPhone() error = %v, want CASL source error", err)
	}
	if len(got) != 1 || got[0].Source != "МІСТ/Firebird" {
		t.Fatalf("FindObjectsBySIMPhone() partial result = %+v", got)
	}
}

func TestFindSIMPhoneUsagesInObjectsExcludesCurrentObject(t *testing.T) {
	exclude := int64(10)
	objects := []models.Object{
		{ID: 10, DisplayNumber: "10", SIM1: "+38 (075) 447-96-15"},
		{ID: 20, DisplayNumber: "P-20", Name: "Склад", SIM2: "0754479615"},
	}

	got := findSIMPhoneUsagesInObjects(objects, "0754479615", &exclude)
	if len(got) != 1 || got[0].ObjN != 20 || got[0].DisplayNumber != "P-20" || got[0].Slot != "SIM 2" {
		t.Fatalf("findSIMPhoneUsagesInObjects() = %+v", got)
	}
}

type combinedStubProvider struct {
	objects      []models.Object
	zones        map[string][]models.Zone
	employees    map[string][]models.Contact
	events       []models.Event
	objectEvents map[string][]models.Event
	alarms       []models.Alarm
	testMessages map[string][]models.TestMessage
	latestID     int64
	latestErr    error
	processCalls []contracts.AlarmProcessingRequest
	pickUsers    []string
	processErr   error
	processOpts  []contracts.AlarmProcessingOption
	healthInfo   contracts.FrontendSourceHealthInfo
	eventsDelay  time.Duration
	alarmsDelay  time.Duration

	reconnectMu    sync.Mutex
	reconnectCalls []string
}

type contextEventStubProvider struct {
	*combinedStubProvider
}

type blockingLatestEventStubProvider struct {
	*combinedStubProvider
	started chan struct{}
	release chan struct{}
}

func (s *blockingLatestEventStubProvider) GetLatestEventID() (int64, error) {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-s.release
	return 0, errors.New("released")
}

func (s *contextEventStubProvider) GetEventsContext(ctx context.Context) []models.Event {
	<-ctx.Done()
	return append([]models.Event(nil), s.events...)
}

type combinedResponseGroupStub struct {
	*combinedStubProvider
	groups     []contracts.ResponseGroup
	groupCalls int
}

type combinedSIMLookupStub struct {
	*combinedStubProvider
	usages []contracts.AdminSIMPhoneUsage
	err    error
}

type combinedAllContactsStub struct {
	*combinedStubProvider
	contacts map[int][]models.Contact
	err      error
	calls    int
}

func (s *combinedAllContactsStub) GetAllObjectContacts(context.Context) (map[int][]models.Contact, error) {
	s.calls++
	result := make(map[int][]models.Contact, len(s.contacts))
	for objectID, contacts := range s.contacts {
		result[objectID] = append([]models.Contact(nil), contacts...)
	}
	return result, s.err
}

func (s *combinedSIMLookupStub) FindObjectsBySIMPhone(string, *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	return append([]contracts.AdminSIMPhoneUsage(nil), s.usages...), s.err
}

func (s *combinedResponseGroupStub) ListResponseGroups(context.Context) ([]contracts.ResponseGroup, error) {
	s.groupCalls++
	return append([]contracts.ResponseGroup(nil), s.groups...), nil
}

func (s *combinedResponseGroupStub) AssignResponseGroup(context.Context, models.Alarm, string) error {
	return nil
}

func (s *combinedResponseGroupStub) NotifyGroupArrived(context.Context, models.Alarm) error {
	return nil
}

func (s *combinedResponseGroupStub) CancelResponseGroup(context.Context, models.Alarm) error {
	return nil
}

func (s *combinedStubProvider) GetObjects() []models.Object {
	return append([]models.Object(nil), s.objects...)
}

func (s *combinedStubProvider) GetObjectByID(id string) *models.Object {
	for i := range s.objects {
		if strconv.Itoa(s.objects[i].ID) == id {
			obj := s.objects[i]
			return &obj
		}
	}
	return nil
}

func (s *combinedStubProvider) GetZones(objectID string) []models.Zone {
	return append([]models.Zone(nil), s.zones[objectID]...)
}

func (s *combinedStubProvider) GetEmployees(objectID string) []models.Contact {
	return append([]models.Contact(nil), s.employees[objectID]...)
}

func (s *combinedStubProvider) GetEvents() []models.Event {
	if s.eventsDelay > 0 {
		time.Sleep(s.eventsDelay)
	}
	return append([]models.Event(nil), s.events...)
}

func (s *combinedStubProvider) GetObjectEvents(objectID string) []models.Event {
	if src, ok := s.objectEvents[objectID]; ok {
		return append([]models.Event(nil), src...)
	}
	return nil
}

func (s *combinedStubProvider) GetAlarms() []models.Alarm {
	if s.alarmsDelay > 0 {
		time.Sleep(s.alarmsDelay)
	}
	return append([]models.Alarm(nil), s.alarms...)
}

func (s *combinedStubProvider) TriggerReconnect(reason string) {
	s.reconnectMu.Lock()
	s.reconnectCalls = append(s.reconnectCalls, reason)
	s.reconnectMu.Unlock()
}

func (s *combinedStubProvider) reconnectReasons() []string {
	s.reconnectMu.Lock()
	defer s.reconnectMu.Unlock()
	return append([]string(nil), s.reconnectCalls...)
}

func (s *combinedStubProvider) ProcessAlarm(id string, user string, note string) error { return nil }

func (s *combinedStubProvider) GetAlarmProcessingOptions(ctx context.Context, alarm models.Alarm) ([]contracts.AlarmProcessingOption, error) {
	return append([]contracts.AlarmProcessingOption(nil), s.processOpts...), nil
}

func (s *combinedStubProvider) ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, user string, request contracts.AlarmProcessingRequest) error {
	s.processCalls = append(s.processCalls, request)
	return s.processErr
}

func (s *combinedStubProvider) PickAlarm(ctx context.Context, alarm models.Alarm, user string) error {
	s.pickUsers = append(s.pickUsers, user)
	return s.processErr
}

func (s *combinedStubProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	return "", "", time.Time{}, time.Time{}
}

func (s *combinedStubProvider) GetTestMessages(objectID string) []models.TestMessage {
	return append([]models.TestMessage(nil), s.testMessages[objectID]...)
}

func (s *combinedStubProvider) GetLatestEventID() (int64, error) {
	return s.latestID, s.latestErr
}

func (s *combinedStubProvider) FrontendSourceHealth() contracts.FrontendSourceHealthInfo {
	return s.healthInfo
}

func TestCombinedDataProvider_MergesObjectsAndAlarms(t *testing.T) {
	t.Parallel()

	now := time.Now()
	secondaryObjID := ids.CASLObjectIDNamespaceStart + 1

	primary := &combinedStubProvider{
		objects: []models.Object{
			{ID: 10, Name: "DB object"},
		},
		alarms: []models.Alarm{
			{ID: 10, ObjectID: 10, Time: now.Add(-2 * time.Minute)},
		},
	}
	secondary := &combinedStubProvider{
		objects: []models.Object{
			{ID: secondaryObjID, Name: "CASL object"},
		},
		alarms: []models.Alarm{
			{ID: secondaryObjID, ObjectID: secondaryObjID, Time: now.Add(-1 * time.Minute)},
		},
	}

	provider := NewCombinedDataProvider(primary, secondary)

	objects := provider.GetObjects()
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}
	if objects[0].ID != 10 || objects[1].ID != secondaryObjID {
		t.Fatalf("unexpected merged objects order/ids: %+v", objects)
	}

	alarms := provider.GetAlarms()
	if len(alarms) != 2 {
		t.Fatalf("expected 2 alarms, got %d", len(alarms))
	}
	if alarms[0].ObjectID != secondaryObjID {
		t.Fatalf("latest alarm should be CASL alarm")
	}
}

func TestCombinedDataProvider_ListResponseGroupsForAlarmQueriesOnlyOwner(t *testing.T) {
	t.Parallel()

	bridge := &combinedResponseGroupStub{
		combinedStubProvider: &combinedStubProvider{},
		groups: []contracts.ResponseGroup{
			{ID: "bridge-1", Name: "МГР МІСТ"},
		},
	}
	casl := &combinedResponseGroupStub{
		combinedStubProvider: &combinedStubProvider{},
		groups: []contracts.ResponseGroup{
			{ID: "casl-1", Name: "МГР CASL"},
		},
	}
	provider := NewMultiSourceDataProvider(
		ProviderSource{
			Name:         "bridge",
			Provider:     bridge,
			OwnsObjectID: func(id int) bool { return !ids.IsCASLObjectID(id) },
		},
		ProviderSource{
			Name:         "casl",
			Provider:     casl,
			OwnsObjectID: ids.IsCASLObjectID,
		},
	)

	groups, err := provider.ListResponseGroupsForAlarm(context.Background(), models.Alarm{ObjectID: 101})
	if err != nil {
		t.Fatalf("ListResponseGroupsForAlarm() error = %v", err)
	}
	if len(groups) != 1 || groups[0].ID != "bridge-1" {
		t.Fatalf("groups = %+v, want bridge group", groups)
	}
	if bridge.groupCalls != 1 || casl.groupCalls != 0 {
		t.Fatalf("group calls: bridge=%d casl=%d", bridge.groupCalls, casl.groupCalls)
	}
	if groups[0].Source != contracts.FrontendSourceBridge {
		t.Fatalf("group source = %q, want bridge", groups[0].Source)
	}
}

func TestCombinedDataProvider_RoutesByObjectIDNamespace(t *testing.T) {
	t.Parallel()

	secondaryObjID := ids.CASLObjectIDNamespaceStart + 2
	secondaryObjIDStr := strconv.Itoa(secondaryObjID)

	primary := &combinedStubProvider{
		zones: map[string][]models.Zone{
			"42": {{Number: 42, Name: "DB zone"}},
		},
	}
	secondary := &combinedStubProvider{
		zones: map[string][]models.Zone{
			secondaryObjIDStr: {{Number: 2, Name: "CASL zone"}},
		},
	}

	provider := NewCombinedDataProvider(primary, secondary)

	dbZones := provider.GetZones("42")
	if len(dbZones) != 1 || dbZones[0].Name != "DB zone" {
		t.Fatalf("unexpected DB zones: %+v", dbZones)
	}

	caslZones := provider.GetZones(secondaryObjIDStr)
	if len(caslZones) != 1 || caslZones[0].Name != "CASL zone" {
		t.Fatalf("unexpected CASL zones: %+v", caslZones)
	}
}

func TestCombinedDataProviderGetAllObjectContactsUsesBulkProviderAndFallback(t *testing.T) {
	bridge := &combinedAllContactsStub{
		combinedStubProvider: &combinedStubProvider{},
		contacts: map[int][]models.Contact{
			1001: {{Name: "Bridge contact", Phone: "0500000001"}},
		},
	}
	phoenixID := ids.PhoenixObjectIDNamespaceStart + 28
	phoenix := &combinedStubProvider{
		objects: []models.Object{{ID: phoenixID}},
		employees: map[string][]models.Contact{
			strconv.Itoa(phoenixID): {{Name: "Phoenix contact", Phone: "0500000002"}},
		},
	}
	provider := NewMultiSourceDataProvider(
		ProviderSource{Name: "bridge", Provider: bridge},
		ProviderSource{Name: "phoenix", Provider: phoenix, OwnsObjectID: ids.IsPhoenixObjectID},
	)

	contacts, err := provider.GetAllObjectContacts(context.Background())
	if err != nil {
		t.Fatalf("GetAllObjectContacts() error = %v", err)
	}
	if bridge.calls != 1 {
		t.Fatalf("bulk provider calls = %d, want 1", bridge.calls)
	}
	if got := contacts[1001]; len(got) != 1 || got[0].Name != "Bridge contact" {
		t.Fatalf("Bridge contacts = %+v", got)
	}
	if got := contacts[phoenixID]; len(got) != 1 || got[0].Name != "Phoenix contact" {
		t.Fatalf("Phoenix contacts = %+v", got)
	}
}

func TestCombinedDataProvider_GetLatestEventID_ChangesWhenAnySourceChanges(t *testing.T) {
	t.Parallel()

	primary := &combinedStubProvider{latestID: 10}
	secondary := &combinedStubProvider{latestID: 20}
	provider := NewCombinedDataProvider(primary, secondary)

	first, err := provider.GetLatestEventID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := provider.GetLatestEventID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Fatalf("cursor must be stable when sources unchanged: %d != %d", first, second)
	}

	secondary.latestID = 21
	third, err := provider.GetLatestEventID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if third == second {
		t.Fatalf("cursor must change when secondary source changes: %d == %d", third, second)
	}
}

func TestCombinedDataProvider_GetLatestEventID_RecoversFromPausedSource(t *testing.T) {
	blocked := &blockingLatestEventStubProvider{
		combinedStubProvider: &combinedStubProvider{},
		started:              make(chan struct{}, 1),
		release:              make(chan struct{}),
	}
	provider := NewMultiSourceDataProvider(ProviderSource{Name: "bridge", Provider: blocked})
	provider.latestProbeTimeout = 20 * time.Millisecond

	start := time.Now()
	_, err := provider.GetLatestEventID()
	if err == nil {
		t.Fatal("GetLatestEventID() error = nil, want no available cursor")
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("GetLatestEventID() took %v, want a hard timeout", elapsed)
	}
	if reasons := blocked.reconnectReasons(); len(reasons) != 1 || reasons[0] != "combined latest event probe timeout" {
		t.Fatalf("reconnect reasons = %+v", reasons)
	}

	select {
	case <-blocked.started:
	default:
		t.Fatal("latest event probe did not start")
	}
	close(blocked.release)
}

func TestCombinedDataProvider_FrontendSourceCapabilities_IncludeHealth(t *testing.T) {
	t.Parallel()

	provider := NewMultiSourceDataProvider(
		ProviderSource{
			Name: "casl",
			Provider: &combinedStubProvider{
				healthInfo: contracts.FrontendSourceHealthInfo{
					HealthStatus:   contracts.FrontendSourceHealthStatusDegraded,
					HealthText:     "CASL: API online, але WS не отримує ping понад 12 с",
					APIStatus:      contracts.FrontendConnectionStatusOnline,
					RealtimeStatus: contracts.FrontendConnectionStatusOffline,
				},
			},
		},
	)

	caps := provider.FrontendSourceCapabilities()
	if len(caps) != 1 {
		t.Fatalf("len(caps) = %d, want 1", len(caps))
	}
	if caps[0].HealthStatus != contracts.FrontendSourceHealthStatusDegraded {
		t.Fatalf("HealthStatus = %q, want degraded", caps[0].HealthStatus)
	}
	if caps[0].RealtimeStatus != contracts.FrontendConnectionStatusOffline {
		t.Fatalf("RealtimeStatus = %q, want offline", caps[0].RealtimeStatus)
	}
}

func TestCombinedDataProvider_MergesBridgePhoenixAndCASLAlarms(t *testing.T) {
	t.Parallel()

	now := time.Now()
	phoenixObjID := ids.PhoenixObjectIDNamespaceStart + 10
	caslObjID := ids.CASLObjectIDNamespaceStart + 20

	provider := NewMultiSourceDataProvider(
		ProviderSource{
			Name: "bridge",
			Provider: &combinedStubProvider{
				alarms: []models.Alarm{
					{ID: 101, ObjectID: 101, ObjectNumber: "101", ObjectName: "Bridge object", Time: now.Add(-3 * time.Minute)},
				},
			},
		},
		ProviderSource{
			Name:         "phoenix",
			OwnsObjectID: ids.IsPhoenixObjectID,
			OwnsAlarmID:  ids.IsPhoenixObjectID,
			Provider: &combinedStubProvider{
				alarms: []models.Alarm{
					{ID: 201, ObjectID: phoenixObjID, ObjectNumber: "L00028", ObjectName: "Phoenix object", Time: now.Add(-2 * time.Minute)},
				},
			},
		},
		ProviderSource{
			Name:         "casl",
			OwnsObjectID: ids.IsCASLObjectID,
			OwnsAlarmID:  ids.IsCASLObjectID,
			Provider: &combinedStubProvider{
				alarms: []models.Alarm{
					{ID: 301, ObjectID: caslObjID, ObjectNumber: "1004", ObjectName: "CASL object", Time: now.Add(-1 * time.Minute)},
				},
			},
		},
	)

	alarms := provider.GetAlarms()
	if len(alarms) != 3 {
		t.Fatalf("expected 3 merged alarms, got %d", len(alarms))
	}
	if alarms[0].ObjectID != caslObjID {
		t.Fatalf("latest alarm must be CASL, got objectID=%d", alarms[0].ObjectID)
	}
	if alarms[1].ObjectID != phoenixObjID {
		t.Fatalf("second alarm must be Phoenix, got objectID=%d", alarms[1].ObjectID)
	}
	if alarms[2].ObjectID != 101 {
		t.Fatalf("third alarm must be Bridge, got objectID=%d", alarms[2].ObjectID)
	}
}

func TestCombinedDataProvider_ProcessAlarmWithRequest_RoutesToCASLSource(t *testing.T) {
	t.Parallel()

	caslObjID := ids.CASLObjectIDNamespaceStart + 42
	primary := &combinedStubProvider{}
	secondary := &combinedStubProvider{
		processOpts: []contracts.AlarmProcessingOption{
			{Code: "CAUSES_FALSE_ALARM", Label: "Хибна тривога"},
		},
	}

	provider := NewCombinedDataProvider(primary, secondary)
	alarm := models.Alarm{ID: caslObjID, ObjectID: caslObjID}

	options, err := provider.GetAlarmProcessingOptions(context.Background(), alarm)
	if err != nil {
		t.Fatalf("GetAlarmProcessingOptions error: %v", err)
	}
	if len(options) != 1 || options[0].Code != "CAUSES_FALSE_ALARM" {
		t.Fatalf("unexpected options: %+v", options)
	}

	err = provider.ProcessAlarmWithRequest(context.Background(), alarm, "Диспетчер", contracts.AlarmProcessingRequest{
		CauseCode: "CAUSES_FALSE_ALARM",
		Note:      "note",
	})
	if err != nil {
		t.Fatalf("ProcessAlarmWithRequest error: %v", err)
	}
	if len(secondary.processCalls) != 1 {
		t.Fatalf("expected 1 CASL process call, got %d", len(secondary.processCalls))
	}
	if secondary.processCalls[0].CauseCode != "CAUSES_FALSE_ALARM" || secondary.processCalls[0].Note != "note" {
		t.Fatalf("unexpected process request: %+v", secondary.processCalls[0])
	}
	if len(primary.processCalls) != 0 {
		t.Fatalf("primary provider must not receive advanced CASL request")
	}
}

func TestCombinedDataProvider_PickAlarm_RoutesToCASLSource(t *testing.T) {
	t.Parallel()

	caslObjID := ids.CASLObjectIDNamespaceStart + 52
	primary := &combinedStubProvider{}
	secondary := &combinedStubProvider{}

	provider := NewCombinedDataProvider(primary, secondary)
	err := provider.PickAlarm(context.Background(), models.Alarm{ID: caslObjID, ObjectID: caslObjID}, "Оператор")
	if err != nil {
		t.Fatalf("PickAlarm error: %v", err)
	}
	if len(secondary.pickUsers) != 1 || secondary.pickUsers[0] != "Оператор" {
		t.Fatalf("unexpected pick calls: %+v", secondary.pickUsers)
	}
	if len(primary.pickUsers) != 0 {
		t.Fatalf("primary provider must not receive CASL pick request")
	}
}

func TestCombinedDataProvider_GetEvents_TriggersReconnectOnTimeout(t *testing.T) {
	t.Parallel()

	casl := &combinedStubProvider{
		eventsDelay: 80 * time.Millisecond,
	}
	provider := NewMultiSourceDataProvider(ProviderSource{
		Name:         "casl",
		Provider:     casl,
		OwnsObjectID: ids.IsCASLObjectID,
		OwnsAlarmID:  ids.IsCASLObjectID,
	})
	provider.eventsTimeout = 20 * time.Millisecond

	start := time.Now()
	events := provider.GetEvents()
	duration := time.Since(start)

	if len(events) != 0 {
		t.Fatalf("expected no events after timeout, got %+v", events)
	}
	if duration > 250*time.Millisecond {
		t.Fatalf("GetEvents blocked for too long: %v", duration)
	}

	reasons := casl.reconnectReasons()
	if len(reasons) != 1 || reasons[0] != "combined get_events timeout" {
		t.Fatalf("unexpected reconnect reasons: %+v", reasons)
	}
}

func TestCombinedDataProvider_GetEvents_UsesCacheReturnedOnContextCancellation(t *testing.T) {
	t.Parallel()

	bridge := &contextEventStubProvider{combinedStubProvider: &combinedStubProvider{
		events: []models.Event{{ID: 42, ObjectID: 42, Source: models.EventSourceBridge}},
	}}
	provider := NewMultiSourceDataProvider(ProviderSource{Name: "bridge", Provider: bridge})
	provider.eventsTimeout = 20 * time.Millisecond

	events := provider.GetEvents()
	if len(events) != 1 || events[0].ID != 42 {
		t.Fatalf("expected cached bridge event after cancellation, got %+v", events)
	}
	if reasons := bridge.reconnectReasons(); len(reasons) != 0 {
		t.Fatalf("unexpected reconnect after cache result: %+v", reasons)
	}
}

func TestCombinedDataProvider_GetEvents_PreservesLastEventsWhenSourceTimesOut(t *testing.T) {
	t.Parallel()

	bridge := &combinedStubProvider{
		events: []models.Event{{ID: 42, ObjectID: 42, Source: models.EventSourceBridge}},
	}
	provider := NewMultiSourceDataProvider(ProviderSource{Name: "bridge", Provider: bridge})
	provider.eventsTimeout = 20 * time.Millisecond

	if events := provider.GetEvents(); len(events) != 1 || events[0].ID != 42 {
		t.Fatalf("initial bridge events = %+v, want cached event", events)
	}

	bridge.eventsDelay = 80 * time.Millisecond
	events := provider.GetEvents()
	if len(events) != 1 || events[0].ID != 42 {
		t.Fatalf("events after timeout = %+v, want last successful bridge event", events)
	}
	if reasons := bridge.reconnectReasons(); len(reasons) != 1 || reasons[0] != "combined get_events timeout" {
		t.Fatalf("unexpected reconnect reasons: %+v", reasons)
	}
}

func TestCombinedDataProvider_GetAlarms_TriggersReconnectOnTimeout(t *testing.T) {
	t.Parallel()

	casl := &combinedStubProvider{
		alarmsDelay: 80 * time.Millisecond,
	}
	provider := NewMultiSourceDataProvider(ProviderSource{
		Name:         "casl",
		Provider:     casl,
		OwnsObjectID: ids.IsCASLObjectID,
		OwnsAlarmID:  ids.IsCASLObjectID,
	})
	provider.alarmsTimeout = 20 * time.Millisecond

	start := time.Now()
	alarms := provider.GetAlarms()
	duration := time.Since(start)

	if len(alarms) != 0 {
		t.Fatalf("expected no alarms after timeout, got %+v", alarms)
	}
	if duration > 250*time.Millisecond {
		t.Fatalf("GetAlarms blocked for too long: %v", duration)
	}

	reasons := casl.reconnectReasons()
	if len(reasons) != 1 || reasons[0] != "combined get_alarms timeout" {
		t.Fatalf("unexpected reconnect reasons: %+v", reasons)
	}
}
