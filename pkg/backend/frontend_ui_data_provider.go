package backend

import (
	"context"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

type FrontendUIDataProvider struct {
	frontend contracts.FrontendBackend
	fallback contracts.DataProvider
}

func NewFrontendUIDataProvider(frontend contracts.FrontendBackend, fallback contracts.DataProvider) *FrontendUIDataProvider {
	if frontend == nil && fallback == nil {
		return nil
	}
	return &FrontendUIDataProvider{
		frontend: frontend,
		fallback: fallback,
	}
}

func (p *FrontendUIDataProvider) GetObjects() []models.Object {
	summaries, err := p.listObjects()
	if err != nil {
		return p.fallbackObjects()
	}

	fallbackByID := make(map[int]models.Object)
	for _, object := range p.fallbackObjects() {
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
	items, err := p.listEvents()
	if err != nil {
		return p.fallbackEvents()
	}

	fallbackByID := make(map[int]models.Event)
	for _, event := range p.fallbackEvents() {
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

func (p *FrontendUIDataProvider) GetAlarms() []models.Alarm {
	items, err := p.listAlarms()
	if err != nil {
		return p.fallbackAlarms()
	}

	fallbackByID := make(map[int]models.Alarm)
	for _, alarm := range p.fallbackAlarms() {
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
	if p == nil || p.fallback == nil {
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

func (p *FrontendUIDataProvider) GetAlarmProcessingOptions(ctx context.Context, alarm models.Alarm) ([]contracts.AlarmProcessingOption, error) {
	provider, ok := p.fallback.(contracts.AlarmProcessingProvider)
	if !ok {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return provider.GetAlarmProcessingOptions(ctx, alarm)
}

func (p *FrontendUIDataProvider) ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, user string, request contracts.AlarmProcessingRequest) error {
	provider, ok := p.fallback.(contracts.AlarmProcessingProvider)
	if !ok {
		return contracts.ErrFrontendBackendUnavailable
	}
	return provider.ProcessAlarmWithRequest(ctx, alarm, user, request)
}

func (p *FrontendUIDataProvider) listObjects() ([]contracts.FrontendObjectSummary, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.ListObjects(context.Background())
}

func (p *FrontendUIDataProvider) listAlarms() ([]contracts.FrontendAlarmItem, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.ListAlarms(context.Background())
}

func (p *FrontendUIDataProvider) listEvents() ([]contracts.FrontendEventItem, error) {
	if p == nil || p.frontend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.ListEvents(context.Background())
}

func (p *FrontendUIDataProvider) getObjectDetails(objectID int) (contracts.FrontendObjectDetails, error) {
	if p == nil || p.frontend == nil {
		return contracts.FrontendObjectDetails{}, contracts.ErrFrontendBackendUnavailable
	}
	return p.frontend.GetObjectDetails(context.Background(), objectID)
}

func (p *FrontendUIDataProvider) fallbackObjects() []models.Object {
	if p == nil || p.fallback == nil {
		return nil
	}
	return p.fallback.GetObjects()
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
	return p.fallback.GetEvents()
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
	return p.fallback.GetAlarms()
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
