package backend

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

type FrontendUIDataProvider struct {
	frontend         contracts.FrontendBackend
	fallback         contracts.DataProvider
	fallbackCacheTTL time.Duration
	fallbackMu       sync.Mutex
	cachedObjects    []models.Object
	cachedObjectsAt  time.Time
	cachedEvents     []models.Event
	cachedEventsAt   time.Time
	cachedAlarms     []models.Alarm
	cachedAlarmsAt   time.Time
}

const frontendFallbackCacheTTL = 2 * time.Second
const frontendReadTimeout = 20 * time.Second

func NewFrontendUIDataProvider(frontend contracts.FrontendBackend, fallback contracts.DataProvider) *FrontendUIDataProvider {
	if frontend == nil && fallback == nil {
		return nil
	}
	return &FrontendUIDataProvider{
		frontend:         frontend,
		fallback:         fallback,
		fallbackCacheTTL: frontendFallbackCacheTTL,
	}
}

// GenerateAcceptedObjectsReport delegates accepted objects report generation to the underlying data provider
func (p *FrontendUIDataProvider) GenerateAcceptedObjectsReport(filePath string) error {
	if reporter, ok := p.fallback.(contracts.ExcelReportingProvider); ok {
		return reporter.GenerateAcceptedObjectsReport(filePath)
	}
	return fmt.Errorf("underlying data provider does not support Excel reporting")
}

// AppendObjectToDeletedReport appends an object to the deleted report
func (p *FrontendUIDataProvider) AppendObjectToDeletedReport(obj *models.Object, contacts []models.Contact, pdfFilePath string, filePath string) error {
	if reporter, ok := p.fallback.(contracts.ExcelReportingProvider); ok {
		return reporter.AppendObjectToDeletedReport(obj, contacts, pdfFilePath, filePath)
	}
	return fmt.Errorf("underlying data provider does not support Excel reporting")
}

func (p *FrontendUIDataProvider) GetObjects() []models.Object {
	return p.GetObjectsContext(context.Background())
}

func (p *FrontendUIDataProvider) GetObjectsContext(ctx context.Context) []models.Object {
	summaries, err := p.listObjects(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return p.fallbackObjects()
	}

	fallbackObjects := p.fallbackObjects()
	fallbackByID := make(map[int]models.Object)
	for _, object := range fallbackObjects {
		fallbackByID[object.ID] = object
	}

	result := make([]models.Object, 0, len(summaries))
	for _, summary := range summaries {
		object := fallbackByID[summary.ID]
		mergeFrontendSummary(&object, summary)
		result = append(result, object)
	}
	return result
}

func (p *FrontendUIDataProvider) GetObjectByID(id string) *models.Object {
	objectID, ok := parseObjectID(id)
	if !ok {
		return p.fallbackObjectByID(id)
	}

	details, err := p.getObjectDetails(objectID)
	if err != nil {
		return p.fallbackObjectByID(id)
	}

	var object models.Object
	if fallback := p.fallbackObjectByID(id); fallback != nil {
		object = *fallback
	}
	mergeFrontendDetails(&object, details)
	return &object
}

// GetObjectBaseDetails returns the object card data using one frontend details request when available.
func (p *FrontendUIDataProvider) GetObjectBaseDetails(objectID string) (*models.Object, []models.Zone, []models.Contact) {
	id, ok := parseObjectID(objectID)
	if !ok {
		return p.fallbackObjectByID(objectID), p.fallbackZones(objectID), p.fallbackEmployees(objectID)
	}
	details, err := p.getObjectDetails(id)
	if err != nil {
		return p.fallbackObjectByID(objectID), p.fallbackZones(objectID), p.fallbackEmployees(objectID)
	}

	var object models.Object
	if fallback := p.fallbackObjectByID(objectID); fallback != nil {
		object = *fallback
	}
	mergeFrontendDetails(&object, details)
	return &object, mapModelZones(details.Zones), mapModelContacts(details.Contacts)
}

func (p *FrontendUIDataProvider) GetZones(objectID string) []models.Zone {
	id, ok := parseObjectID(objectID)
	if !ok {
		return p.fallbackZones(objectID)
	}
	details, err := p.getObjectDetails(id)
	if err != nil {
		return p.fallbackZones(objectID)
	}
	return mapModelZones(details.Zones)
}

func (p *FrontendUIDataProvider) GetEmployees(objectID string) []models.Contact {
	id, ok := parseObjectID(objectID)
	if !ok {
		return p.fallbackEmployees(objectID)
	}
	details, err := p.getObjectDetails(id)
	if err != nil {
		return p.fallbackEmployees(objectID)
	}
	return mapModelContacts(details.Contacts)
}

// GetAllObjectContacts loads contacts through the aggregate data provider when available.
func (p *FrontendUIDataProvider) GetAllObjectContacts(ctx context.Context) (map[int][]models.Contact, error) {
	if p == nil || p.fallback == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	if provider, ok := p.fallback.(contracts.AllObjectContactsProvider); ok {
		return provider.GetAllObjectContacts(ctx)
	}

	result := make(map[int][]models.Contact)
	objects := p.GetObjectsContext(ctx)
	for _, object := range objects {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result[object.ID] = p.GetEmployees(strconv.Itoa(object.ID))
	}
	return result, nil
}

func (p *FrontendUIDataProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	id, ok := parseObjectID(objectID)
	if !ok {
		return p.fallbackExternalData(objectID)
	}
	details, err := p.getObjectDetails(id)
	if err != nil {
		return p.fallbackExternalData(objectID)
	}
	return details.ExternalSignal, details.ExternalTestMessage, details.ExternalLastTest, details.ExternalLastMessage
}

func (p *FrontendUIDataProvider) GetEvents() []models.Event {
	return p.GetEventsContext(context.Background())
}

func (p *FrontendUIDataProvider) GetEventsContext(ctx context.Context) []models.Event {
	items, err := p.listEvents(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return p.fallbackEvents()
	}

	fallbackEvents := p.fallbackEvents()
	fallbackByID := make(map[int]models.Event)
	for _, event := range fallbackEvents {
		fallbackByID[event.ID] = event
	}

	result := make([]models.Event, 0, len(items))
	for _, item := range items {
		event := fallbackByID[item.ID]
		mergeFrontendEvent(&event, item)
		result = append(result, event)
	}
	return result
}

func (p *FrontendUIDataProvider) GetObjectEvents(objectID string) []models.Event {
	id, ok := parseObjectID(objectID)
	if !ok {
		return p.fallbackObjectEvents(objectID)
	}
	details, err := p.getObjectDetails(id)
	if err != nil {
		return p.fallbackObjectEvents(objectID)
	}

	fallbackByID := make(map[int]models.Event)
	for _, event := range p.fallbackObjectEvents(objectID) {
		fallbackByID[event.ID] = event
	}

	result := make([]models.Event, 0, len(details.Events))
	for _, item := range details.Events {
		event := fallbackByID[item.ID]
		mergeFrontendEvent(&event, item)
		result = append(result, event)
	}
	return result
}

func (p *FrontendUIDataProvider) GetObjectEventsRange(objectID string, from time.Time, to time.Time) []models.Event {
	if ranged, ok := p.fallback.(contracts.ObjectEventsRangeProvider); ok {
		return ranged.GetObjectEventsRange(objectID, from, to)
	}
	return filterObjectEventsRange(p.GetObjectEvents(objectID), from, to)
}

func filterObjectEventsRange(events []models.Event, from time.Time, to time.Time) []models.Event {
	result := make([]models.Event, 0, len(events))
	for _, event := range events {
		if !from.IsZero() && event.Time.Before(from) {
			continue
		}
		if !to.IsZero() && event.Time.After(to) {
			continue
		}
		result = append(result, event)
	}
	return result
}

func (p *FrontendUIDataProvider) GetAlarms() []models.Alarm {
	return p.GetAlarmsContext(context.Background())
}

func (p *FrontendUIDataProvider) GetAlarmsContext(ctx context.Context) []models.Alarm {
	items, err := p.listAlarms(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return p.fallbackAlarms()
	}

	fallbackAlarms := p.fallbackAlarms()
	fallbackByID := make(map[int]models.Alarm)
	for _, alarm := range fallbackAlarms {
		fallbackByID[alarm.ID] = alarm
	}

	result := make([]models.Alarm, 0, len(items))
	for _, item := range items {
		alarm := fallbackByID[item.ID]
		mergeFrontendAlarm(&alarm, item)
		result = append(result, alarm)
	}
	return result
}

func (p *FrontendUIDataProvider) ProcessAlarm(id string, user string, note string) error {
	if p == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	if alarmID, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && p.frontend != nil {
		return p.frontend.ProcessAlarm(context.Background(), alarmID, contracts.FrontendAlarmProcessRequest{
			User: user,
			Note: note,
		})
	}
	if p.fallback == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	return p.fallback.ProcessAlarm(id, user, note)
}

func (p *FrontendUIDataProvider) GetTestMessages(objectID string) []models.TestMessage {
	provider, ok := p.fallback.(contracts.TestMessageProvider)
	if !ok {
		return nil
	}
	return provider.GetTestMessages(objectID)
}

func (p *FrontendUIDataProvider) GetAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	provider, ok := p.fallback.(contracts.AlarmHistoryProvider)
	if !ok {
		return nil
	}
	return provider.GetAlarmSourceMessages(alarm)
}

func (p *FrontendUIDataProvider) GetActiveAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	provider, ok := p.fallback.(contracts.ActiveAlarmHistoryProvider)
	if !ok {
		return nil
	}
	return provider.GetActiveAlarmSourceMessages(alarm)
}

func (p *FrontendUIDataProvider) GetObjectMedia(ctx context.Context, objectID int) ([]contracts.ObjectMedia, error) {
	provider, ok := p.fallback.(contracts.ObjectMediaProvider)
	if !ok {
		return nil, nil
	}
	return provider.GetObjectMedia(ctx, objectID)
}

func (p *FrontendUIDataProvider) FetchObjectMedia(ctx context.Context, media contracts.ObjectMedia) ([]byte, error) {
	provider, ok := p.fallback.(contracts.ObjectMediaProvider)
	if !ok {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return provider.FetchObjectMedia(ctx, media)
}

func (p *FrontendUIDataProvider) ListObjectLocations(ctx context.Context) ([]contracts.ObjectLocation, error) {
	provider, ok := p.fallback.(contracts.ObjectLocationProvider)
	if !ok {
		return nil, nil
	}
	return provider.ListObjectLocations(ctx)
}

func (p *FrontendUIDataProvider) GetAlarmProcessingOptions(ctx context.Context, alarm models.Alarm) ([]contracts.AlarmProcessingOption, error) {
	provider, ok := p.fallback.(contracts.AlarmProcessingProvider)
	if !ok {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return provider.GetAlarmProcessingOptions(ctx, alarm)
}

func (p *FrontendUIDataProvider) ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, user string, request contracts.AlarmProcessingRequest) error {
	if p == nil || p.frontend == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.ProcessAlarm(ctx, alarm.ID, contracts.FrontendAlarmProcessRequest{
		User:      strings.TrimSpace(user),
		CauseCode: strings.TrimSpace(request.CauseCode),
		Note:      strings.TrimSpace(request.Note),
	})
}

func (p *FrontendUIDataProvider) PickAlarm(ctx context.Context, alarm models.Alarm, user string) error {
	if p == nil || p.frontend == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.PickAlarm(ctx, alarm.ID, contracts.FrontendAlarmPickRequest{User: strings.TrimSpace(user)})
}

func (p *FrontendUIDataProvider) ListResponseGroupsForAlarm(ctx context.Context, alarm models.Alarm) ([]contracts.FrontendResponseGroup, error) {
	if provider, ok := p.fallback.(contracts.AlarmResponseGroupProvider); ok {
		groups, err := provider.ListResponseGroupsForAlarm(ctx, alarm)
		if err != nil {
			return nil, err
		}
		return mapFrontendResponseGroups(groups), nil
	}

	groups, err := p.ListResponseGroups(ctx)
	if err != nil {
		return nil, err
	}
	source := contracts.DetectFrontendSourceByObjectID(alarm.ObjectID)
	filtered := make([]contracts.FrontendResponseGroup, 0, len(groups))
	for _, group := range groups {
		if group.Source == "" || group.Source == contracts.FrontendSourceUnknown || group.Source == source {
			filtered = append(filtered, group)
		}
	}
	return filtered, nil
}

func mapFrontendResponseGroups(groups []contracts.ResponseGroup) []contracts.FrontendResponseGroup {
	result := make([]contracts.FrontendResponseGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, contracts.FrontendResponseGroup{
			ID:              strings.TrimSpace(group.ID),
			Name:            strings.TrimSpace(group.Name),
			Callsign:        strings.TrimSpace(group.Callsign),
			Phone:           strings.TrimSpace(group.Phone),
			Source:          group.Source,
			Status:          group.Status,
			StatusText:      strings.TrimSpace(group.StatusText),
			ObjectNumber:    strings.TrimSpace(group.ObjectNumber),
			ObjectName:      strings.TrimSpace(group.ObjectName),
			Latitude:        strings.TrimSpace(group.Latitude),
			Longitude:       strings.TrimSpace(group.Longitude),
			StatusChangedAt: group.StatusChangedAt,
		})
	}
	return result
}

func (p *FrontendUIDataProvider) ListResponseGroups(ctx context.Context) ([]contracts.FrontendResponseGroup, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	groups, err := p.frontend.ListResponseGroups(ctx)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (p *FrontendUIDataProvider) AssignResponseGroup(ctx context.Context, alarm models.Alarm, groupID string) error {
	if p == nil || p.frontend == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.AssignResponseGroup(ctx, alarm.ID, contracts.FrontendAlarmGroupActionRequest{
		GroupID: strings.TrimSpace(groupID),
	})
}

func (p *FrontendUIDataProvider) NotifyGroupArrived(ctx context.Context, alarm models.Alarm) error {
	if p == nil || p.frontend == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.NotifyGroupArrived(ctx, alarm.ID)
}

func (p *FrontendUIDataProvider) CancelResponseGroup(ctx context.Context, alarm models.Alarm) error {
	if p == nil || p.frontend == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.CancelResponseGroup(ctx, alarm.ID)
}

func (p *FrontendUIDataProvider) listObjects(parent context.Context) ([]contracts.FrontendObjectSummary, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	ctx, cancel := context.WithTimeout(parent, frontendReadTimeout)
	defer cancel()
	return p.frontend.ListObjects(ctx)
}

func (p *FrontendUIDataProvider) listAlarms(parent context.Context) ([]contracts.FrontendAlarmItem, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	ctx, cancel := context.WithTimeout(parent, frontendReadTimeout)
	defer cancel()
	return p.frontend.ListAlarms(ctx)
}

func (p *FrontendUIDataProvider) listEvents(parent context.Context) ([]contracts.FrontendEventItem, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	ctx, cancel := context.WithTimeout(parent, frontendReadTimeout)
	defer cancel()
	return p.frontend.ListEvents(ctx)
}

func (p *FrontendUIDataProvider) getObjectDetails(objectID int) (contracts.FrontendObjectDetails, error) {
	if p == nil || p.frontend == nil {
		return contracts.FrontendObjectDetails{}, contracts.ErrFrontendBackendUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), frontendReadTimeout)
	defer cancel()
	return p.frontend.GetObjectDetails(ctx, objectID)
}

func (p *FrontendUIDataProvider) fallbackObjects() []models.Object {
	if p == nil || p.fallback == nil {
		return nil
	}
	now := time.Now()
	p.fallbackMu.Lock()
	if p.fallbackCacheTTL > 0 && !p.cachedObjectsAt.IsZero() && now.Sub(p.cachedObjectsAt) <= p.fallbackCacheTTL {
		objects := append([]models.Object(nil), p.cachedObjects...)
		p.fallbackMu.Unlock()
		return objects
	}
	p.fallbackMu.Unlock()

	objects := p.fallback.GetObjects()
	p.fallbackMu.Lock()
	p.cachedObjects = append(p.cachedObjects[:0], objects...)
	p.cachedObjectsAt = now
	out := append([]models.Object(nil), p.cachedObjects...)
	p.fallbackMu.Unlock()
	return out
}

func (p *FrontendUIDataProvider) fallbackObjectByID(id string) *models.Object {
	if p == nil || p.fallback == nil {
		return nil
	}
	return p.fallback.GetObjectByID(id)
}

func (p *FrontendUIDataProvider) fallbackZones(objectID string) []models.Zone {
	if p == nil || p.fallback == nil {
		return nil
	}
	return p.fallback.GetZones(objectID)
}

func (p *FrontendUIDataProvider) fallbackEmployees(objectID string) []models.Contact {
	if p == nil || p.fallback == nil {
		return nil
	}
	return p.fallback.GetEmployees(objectID)
}

func (p *FrontendUIDataProvider) fallbackExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	if p == nil || p.fallback == nil {
		return "", "", time.Time{}, time.Time{}
	}
	return p.fallback.GetExternalData(objectID)
}

func (p *FrontendUIDataProvider) fallbackEvents() []models.Event {
	if p == nil || p.fallback == nil {
		return nil
	}
	now := time.Now()
	p.fallbackMu.Lock()
	if p.fallbackCacheTTL > 0 && !p.cachedEventsAt.IsZero() && now.Sub(p.cachedEventsAt) <= p.fallbackCacheTTL {
		events := append([]models.Event(nil), p.cachedEvents...)
		p.fallbackMu.Unlock()
		return events
	}
	p.fallbackMu.Unlock()

	events := p.fallback.GetEvents()
	p.fallbackMu.Lock()
	p.cachedEvents = append(p.cachedEvents[:0], events...)
	p.cachedEventsAt = now
	out := append([]models.Event(nil), p.cachedEvents...)
	p.fallbackMu.Unlock()
	return out
}

func (p *FrontendUIDataProvider) fallbackObjectEvents(objectID string) []models.Event {
	if p == nil || p.fallback == nil {
		return nil
	}
	return p.fallback.GetObjectEvents(objectID)
}

func (p *FrontendUIDataProvider) fallbackAlarms() []models.Alarm {
	if p == nil || p.fallback == nil {
		return nil
	}
	now := time.Now()
	p.fallbackMu.Lock()
	if p.fallbackCacheTTL > 0 && !p.cachedAlarmsAt.IsZero() && now.Sub(p.cachedAlarmsAt) <= p.fallbackCacheTTL {
		alarms := append([]models.Alarm(nil), p.cachedAlarms...)
		p.fallbackMu.Unlock()
		return alarms
	}
	p.fallbackMu.Unlock()

	alarms := p.fallback.GetAlarms()
	p.fallbackMu.Lock()
	p.cachedAlarms = append(p.cachedAlarms[:0], alarms...)
	p.cachedAlarmsAt = now
	out := append([]models.Alarm(nil), p.cachedAlarms...)
	p.fallbackMu.Unlock()
	return out
}

func parseObjectID(raw string) (int, bool) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func mergeFrontendSummary(object *models.Object, summary contracts.FrontendObjectSummary) {
	if object == nil {
		return
	}
	object.ID = summary.ID
	object.DisplayNumber = strings.TrimSpace(summary.DisplayNumber)
	object.Name = strings.TrimSpace(summary.Name)
	object.Address = strings.TrimSpace(summary.Address)
	object.ContractNum = strings.TrimSpace(summary.ContractNumber)
	object.Phone = strings.TrimSpace(summary.Phone)
	object.Phones1 = firstNonEmpty(strings.TrimSpace(summary.Phone), strings.TrimSpace(object.Phones1))
	object.Status = modelObjectStatus(summary.StatusCode)
	object.StatusText = strings.TrimSpace(summary.StatusText)
	object.DeviceType = strings.TrimSpace(summary.DeviceType)
	object.PanelMark = strings.TrimSpace(summary.PanelMark)
	object.SignalStrength = strings.TrimSpace(summary.SignalStrength)
	object.SIM1 = strings.TrimSpace(summary.SIM1)
	object.SIM2 = strings.TrimSpace(summary.SIM2)
	object.LastTestTime = summary.LastTestTime
	object.LastMessageTime = summary.LastMessageTime
	applyNormalizedObjectState(object, summary)
}

func mergeFrontendDetails(object *models.Object, details contracts.FrontendObjectDetails) {
	if object == nil {
		return
	}
	mergeFrontendSummary(object, details.Summary)
	object.GSMLevel = details.GSMLevel
	object.PowerSource = modelPowerSource(details.PowerSource)
	object.AutoTestHours = details.AutoTestHours
	object.SubServerA = strings.TrimSpace(details.SubServerA)
	object.SubServerB = strings.TrimSpace(details.SubServerB)
	object.ObjChan = details.ChannelCode
	object.AkbState = details.AKBState
	object.PowerFault = details.PowerFault
	if details.TestControl {
		object.TestControl = 1
	} else {
		object.TestControl = 0
	}
	object.TestTime = details.TestIntervalMin
	object.Phones1 = firstNonEmpty(strings.TrimSpace(details.Phones), strings.TrimSpace(object.Phones1), strings.TrimSpace(details.Summary.Phone))
	object.Notes1 = strings.TrimSpace(details.Notes)
	object.Location1 = strings.TrimSpace(details.Location)
	object.LaunchDate = strings.TrimSpace(details.LaunchDate)
	if len(object.Groups) == 0 {
		object.Groups = groupsFromFrontendDetails(details)
	}
}

func mergeFrontendAlarm(alarm *models.Alarm, item contracts.FrontendAlarmItem) {
	if alarm == nil {
		return
	}
	alarm.ID = item.ID
	alarm.ObjectID = item.ObjectID
	alarm.ObjectNumber = strings.TrimSpace(item.ObjectNumber)
	alarm.ObjectName = strings.TrimSpace(item.ObjectName)
	alarm.Address = strings.TrimSpace(item.Address)
	alarm.Time = item.Time
	alarm.Details = strings.TrimSpace(item.Details)
	alarm.Type = models.AlarmType(item.TypeCode)
	alarm.ZoneNumber = item.ZoneNumber
	alarm.ZoneName = strings.TrimSpace(item.ZoneName)
	alarm.IsProcessed = item.IsProcessed
	alarm.ProcessedBy = strings.TrimSpace(item.ProcessedBy)
	alarm.ProcessNote = strings.TrimSpace(item.ProcessNote)
	alarm.IsInProgress = item.IsInProgress
	alarm.InProgressBy = strings.TrimSpace(item.InProgressBy)
	alarm.IsOwnedByMe = item.IsOwnedByMe
	alarm.CanTakeOver = item.CanTakeOver
	alarm.CanProcess = item.CanProcess
	alarm.ResponseGroupID = strings.TrimSpace(item.ResponseGroupID)
	alarm.IsResponseGroupDispatched = item.IsResponseGroupDispatched
	alarm.IsResponseGroupArrived = item.IsResponseGroupArrived
	alarm.VisualSeverity = modelVisualSeverity(item.VisualSeverity)
	if alarm.SC1 == 0 {
		alarm.SC1 = modelSC1FromSeverity(item.VisualSeverity)
	}
}

func mergeFrontendEvent(event *models.Event, item contracts.FrontendEventItem) {
	if event == nil {
		return
	}
	event.ID = item.ID
	event.Time = item.Time
	event.ObjectID = item.ObjectID
	event.ObjectNumber = strings.TrimSpace(item.ObjectNumber)
	event.ObjectName = strings.TrimSpace(item.ObjectName)
	event.Type = models.EventType(item.TypeCode)
	event.TypeLabel = strings.TrimSpace(item.TypeText)
	event.ZoneNumber = item.ZoneNumber
	event.Details = strings.TrimSpace(item.Details)
	event.UserName = strings.TrimSpace(item.UserName)
	event.VisualSeverity = modelVisualSeverity(item.VisualSeverity)
	if event.SC1 == 0 {
		event.SC1 = modelSC1FromSeverity(item.VisualSeverity)
	}
}

func applyNormalizedObjectState(object *models.Object, summary contracts.FrontendObjectSummary) {
	if object == nil {
		return
	}

	switch summary.MonitoringStatus {
	case contracts.FrontendMonitoringStatusBlocked:
		object.MonitoringStatus = models.MonitoringStatusBlocked
		object.BlockedArmedOnOff = 1
	case contracts.FrontendMonitoringStatusDebug:
		object.MonitoringStatus = models.MonitoringStatusDebug
		object.BlockedArmedOnOff = 2
	case contracts.FrontendMonitoringStatusActive:
		object.MonitoringStatus = models.MonitoringStatusActive
		object.BlockedArmedOnOff = 0
	default:
		object.MonitoringStatus = models.MonitoringStatusUnknown
	}

	switch summary.GuardStatus {
	case contracts.FrontendGuardStatusGuarded:
		object.GuardStatus = models.GuardStatusGuarded
		object.GuardState = 1
		object.IsUnderGuard = true
	case contracts.FrontendGuardStatusDisarmed:
		object.GuardStatus = models.GuardStatusDisarmed
		object.GuardState = 0
		object.IsUnderGuard = false
	default:
		object.GuardStatus = models.GuardStatusUnknown
	}

	switch summary.ConnectionStatus {
	case contracts.FrontendConnectionStatusOnline:
		object.ConnectionStatus = models.ConnectionStatusOnline
		object.IsConnState = 1
		object.IsConnOK = true
	case contracts.FrontendConnectionStatusOffline:
		object.ConnectionStatus = models.ConnectionStatusOffline
		object.IsConnState = 0
		object.IsConnOK = false
	default:
		object.ConnectionStatus = models.ConnectionStatusUnknown
	}

	object.HasAssignment = summary.HasAssignment

	switch object.Status {
	case models.StatusFire:
		object.AlarmState = 1
		object.TechAlarmState = 0
	case models.StatusFault:
		object.AlarmState = 0
		object.TechAlarmState = 1
	default:
		object.AlarmState = 0
		object.TechAlarmState = 0
	}
}

func modelObjectStatus(code string) models.ObjectStatus {
	switch strings.TrimSpace(strings.ToLower(code)) {
	case "normal":
		return models.StatusNormal
	case "alarm", "fire":
		return models.StatusFire
	case "fault":
		return models.StatusFault
	case "offline":
		return models.StatusOffline
	default:
		return models.StatusNormal
	}
}

func modelPowerSource(code string) models.PowerSource {
	switch strings.TrimSpace(strings.ToLower(code)) {
	case "battery":
		return models.PowerBattery
	default:
		return models.PowerMains
	}
}

func modelSC1FromSeverity(severity contracts.FrontendVisualSeverity) int {
	switch severity {
	case contracts.FrontendVisualSeverityCritical:
		return 1
	case contracts.FrontendVisualSeverityWarning:
		return 2
	case contracts.FrontendVisualSeverityInfo:
		return 10
	default:
		return 0
	}
}

func modelVisualSeverity(severity contracts.FrontendVisualSeverity) models.VisualSeverity {
	switch severity {
	case contracts.FrontendVisualSeverityCritical:
		return models.VisualSeverityCritical
	case contracts.FrontendVisualSeverityWarning:
		return models.VisualSeverityWarning
	case contracts.FrontendVisualSeverityInfo:
		return models.VisualSeverityInfo
	case contracts.FrontendVisualSeverityNormal:
		return models.VisualSeverityNormal
	default:
		return models.VisualSeverityUnknown
	}
}

func modelZoneStatus(code string) models.ZoneStatus {
	switch strings.TrimSpace(strings.ToLower(code)) {
	case "fire":
		return models.ZoneFire
	case "alarm":
		return models.ZoneAlarm
	case "break":
		return models.ZoneBreak
	case "short":
		return models.ZoneShort
	default:
		return models.ZoneNormal
	}
}

func mapModelZones(items []contracts.FrontendZone) []models.Zone {
	result := make([]models.Zone, 0, len(items))
	for _, item := range items {
		zone := models.Zone{
			Number:         item.Number,
			Name:           strings.TrimSpace(item.Name),
			SensorType:     strings.TrimSpace(item.SensorType),
			Status:         modelZoneStatus(item.Status),
			GroupID:        strings.TrimSpace(item.GroupID),
			GroupNumber:    item.GroupNumber,
			GroupName:      strings.TrimSpace(item.GroupName),
			GroupStateText: strings.TrimSpace(item.GroupStateText),
		}
		if strings.EqualFold(strings.TrimSpace(item.Status), "bypassed") {
			zone.IsBypassed = true
		}
		result = append(result, zone)
	}
	return result
}

func mapModelContacts(items []contracts.FrontendContact) []models.Contact {
	result := make([]models.Contact, 0, len(items))
	for _, item := range items {
		result = append(result, models.Contact{
			Name:           strings.TrimSpace(item.Name),
			Position:       strings.TrimSpace(item.Position),
			Phone:          strings.TrimSpace(item.Phone),
			Priority:       item.Priority,
			CodeWord:       strings.TrimSpace(item.CodeWord),
			GroupID:        strings.TrimSpace(item.GroupID),
			GroupNumber:    item.GroupNumber,
			GroupName:      strings.TrimSpace(item.GroupName),
			GroupStateText: strings.TrimSpace(item.GroupStateText),
		})
	}
	return result
}

func groupsFromFrontendDetails(details contracts.FrontendObjectDetails) []models.ObjectGroup {
	seen := map[string]struct{}{}
	groups := make([]models.ObjectGroup, 0, len(details.Zones)+len(details.Contacts))

	appendGroup := func(id string, number int, name string, state string) {
		key := strings.TrimSpace(id)
		if key == "" && number > 0 {
			key = strconv.Itoa(number)
		}
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		groups = append(groups, models.ObjectGroup{
			ID:        key,
			Number:    number,
			Name:      strings.TrimSpace(name),
			StateText: strings.TrimSpace(state),
		})
	}

	for _, zone := range details.Zones {
		appendGroup(zone.GroupID, zone.GroupNumber, zone.GroupName, zone.GroupStateText)
	}
	for _, contact := range details.Contacts {
		appendGroup(contact.GroupID, contact.GroupNumber, contact.GroupName, contact.GroupStateText)
	}
	return groups
}
