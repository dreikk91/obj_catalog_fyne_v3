package data

import (
	"context"
	"database/sql"
	"fmt"
	"obj_catalog_fyne_v3/pkg/models"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/ids"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const (
	phoenixSourceName          = "phoenix"
	phoenixBlockedStateText    = "ЗАБЛОКОВАНО"
	phoenixStandStateText      = "СТЕНДИ"
	phoenixDisarmedStateText   = "БЕЗ ОХОРОНИ"
	phoenixPartialDisarmedText = "ЧАСТКОВО БЕЗ ОХОРОНИ"
)

type PhoenixDataProvider struct {
	db  *sqlx.DB
	dsn string

	idMu      sync.RWMutex
	panelByID map[int]string
	idByPanel map[string]int

	objectMu         sync.RWMutex
	cachedObjects    []models.Object
	cachedObjectsAt  time.Time
	objectCacheTTL   time.Duration
	latestProbeAt    time.Time
	latestProbeValue int64

	eventMu      sync.RWMutex
	lastEventID  int64
	cachedEvents []models.Event
}

func NewPhoenixDataProvider(db *sqlx.DB, dsn string) *PhoenixDataProvider {
	return &PhoenixDataProvider{
		db:             db,
		dsn:            dsn,
		panelByID:      make(map[int]string),
		idByPanel:      make(map[string]int),
		objectCacheTTL: 15 * time.Second,
	}
}

func (p *PhoenixDataProvider) GetObjects() []models.Object {
	if p == nil || p.db == nil {
		return nil
	}

	p.objectMu.RLock()
	if len(p.cachedObjects) > 0 && time.Since(p.cachedObjectsAt) < p.objectCacheTTL {
		cached := append([]models.Object(nil), p.cachedObjects...)
		p.objectMu.RUnlock()
		return cached
	}
	p.objectMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var rows []phoenixObjectGroupRow
	if err := p.db.SelectContext(ctx, &rows, phoenixObjectsListQuery); err != nil {
		log.Error().Err(err).Msg("Phoenix: помилка отримання списку об'єктів")
		return nil
	}

	objects := p.buildObjects(rows)

	p.objectMu.Lock()
	p.cachedObjects = append([]models.Object(nil), objects...)
	p.cachedObjectsAt = time.Now()
	p.objectMu.Unlock()

	return objects
}

func (p *PhoenixDataProvider) GetObjectByID(objectID string) *models.Object {
	panelID, ok := p.resolvePanelID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var groupRows []phoenixObjectGroupRow
	if err := p.db.SelectContext(ctx, &groupRows, phoenixObjectDetailGroupsQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання деталей об'єкта")
		return nil
	}
	if len(groupRows) == 0 {
		return nil
	}

	object := p.buildObjects(groupRows)
	if len(object) == 0 {
		return nil
	}
	obj := object[0]

	var channelRows []phoenixChannelRow
	if err := p.db.SelectContext(ctx, &channelRows, phoenixChannelInfoQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання інформації про канал")
	} else if len(channelRows) > 0 {
		p.applyChannelInfo(&obj, channelRows[0])
	}

	return &obj
}

func (p *PhoenixDataProvider) GetZones(objectID string) []models.Zone {
	panelID, ok := p.resolvePanelID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var rows []phoenixZoneRow
	if err := p.db.SelectContext(ctx, &rows, phoenixZonesQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання зон")
		return nil
	}

	zones := make([]models.Zone, 0, len(rows))
	for _, row := range rows {
		effectiveDisabled := phoenixEffectiveDisabled(row.GroupDisabled, row.PanelDisabled)
		zones = append(zones, models.Zone{
			Number:         row.ZoneNo,
			Name:           nullString(row.ZoneName),
			SensorType:     phoenixZoneTypeText(row.IsAlarmButton),
			Status:         phoenixZoneStatus(row.Status),
			IsBypassed:     nullBool(row.IsBypass),
			GroupID:        buildPhoenixGroupID(row.PanelID, row.GroupNo),
			GroupNumber:    row.GroupNo,
			GroupName:      phoenixGroupName(row.GroupNo, row.GroupName),
			GroupStateText: phoenixGroupStateText(row.GroupIsOpen, effectiveDisabled, row.TestPanel, sql.NullInt64{}),
		})
	}
	return zones
}

func (p *PhoenixDataProvider) GetEmployees(objectID string) []models.Contact {
	panelID, ok := p.resolvePanelID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var rows []phoenixResponsibleRow
	if err := p.db.SelectContext(ctx, &rows, phoenixResponsiblesQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання відповідальних")
		return nil
	}

	contacts := make([]models.Contact, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(nullString(row.ContactValue))
		if value == "" {
			continue
		}
		effectiveDisabled := phoenixEffectiveDisabled(row.GroupDisabled, row.PanelDisabled)
		label := strings.TrimSpace(nullString(row.ContactLabel))
		if label == "" {
			label = strings.TrimSpace(nullString(row.ContactKind))
		}
		contacts = append(contacts, models.Contact{
			Name:           strings.TrimSpace(nullString(row.ResponsibleName)),
			Position:       label,
			Phone:          value,
			Priority:       int(nullInt64(row.CallOrder)),
			CodeWord:       strings.TrimSpace(nullString(row.ResponsibleAddr)),
			GroupID:        buildPhoenixGroupID(row.PanelID, row.GroupNo),
			GroupNumber:    row.GroupNo,
			GroupName:      phoenixGroupName(row.GroupNo, row.GroupName),
			GroupStateText: phoenixGroupStateText(row.GroupIsOpen, effectiveDisabled, row.TestPanel, sql.NullInt64{}),
		})
	}
	return contacts
}

func (p *PhoenixDataProvider) GetEvents() []models.Event {
	if p == nil || p.db == nil {
		return nil
	}

	p.eventMu.Lock()
	defer p.eventMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if p.lastEventID == 0 {
		var latest sql.NullInt64
		if err := p.db.GetContext(ctx, &latest, phoenixLatestEventIDQuery); err != nil {
			log.Error().Err(err).Msg("Phoenix: помилка отримання стартового курсора подій")
			return append([]models.Event(nil), p.cachedEvents...)
		}

		p.lastEventID = nullInt64(latest)
		return append([]models.Event(nil), p.cachedEvents...)
	}

	var rows []phoenixEventRow
	if err := p.db.SelectContext(ctx, &rows, phoenixIncrementalEventsQuery, p.lastEventID); err != nil {
		log.Error().Err(err).Int64("lastEventID", p.lastEventID).Msg("Phoenix: помилка інкрементального завантаження подій")
		return append([]models.Event(nil), p.cachedEvents...)
	}
	if len(rows) == 0 {
		return append([]models.Event(nil), p.cachedEvents...)
	}

	newEvents := mapPhoenixEventRows(rows, p.mapEventRow)
	p.lastEventID = maxPhoenixEventID(rows, p.lastEventID)
	reversePhoenixEvents(newEvents)
	p.cachedEvents = append(newEvents, p.cachedEvents...)
	if len(p.cachedEvents) > 5000 {
		p.cachedEvents = p.cachedEvents[:5000]
	}
	return append([]models.Event(nil), p.cachedEvents...)
}

func (p *PhoenixDataProvider) GetObjectEvents(objectID string) []models.Event {
	panelID, ok := p.resolvePanelID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var rows []phoenixEventRow
	if err := p.db.SelectContext(ctx, &rows, phoenixObjectEventsQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання подій об'єкта")
		return nil
	}

	events := make([]models.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, p.mapEventRow(row))
	}
	return events
}

func (p *PhoenixDataProvider) GetAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	if len(alarm.SourceMsgs) > 0 {
		return append([]models.AlarmMsg(nil), alarm.SourceMsgs...)
	}

	events := p.GetObjectEvents(strconv.Itoa(alarm.ObjectID))
	return buildAlarmSourceMessagesFromEvents(alarm, events)
}

func (p *PhoenixDataProvider) GetLatestEventID() (int64, error) {
	if p == nil || p.db == nil {
		return 0, fmt.Errorf("phoenix database is not initialized")
	}

	p.objectMu.RLock()
	if !p.latestProbeAt.IsZero() && time.Since(p.latestProbeAt) < 10*time.Second {
		value := p.latestProbeValue
		p.objectMu.RUnlock()
		return value, nil
	}
	p.objectMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var latest sql.NullInt64
	if err := p.db.GetContext(ctx, &latest, phoenixLatestEventIDQuery); err != nil {
		return 0, err
	}
	value := nullInt64(latest)

	p.objectMu.Lock()
	p.latestProbeAt = time.Now()
	p.latestProbeValue = value
	p.objectMu.Unlock()

	return value, nil
}

func (p *PhoenixDataProvider) GetAlarms() []models.Alarm {
	if p == nil || p.db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var activeRows []phoenixActiveAlarmRow
	if err := p.db.SelectContext(ctx, &activeRows, phoenixActiveAlarmsQuery); err != nil {
		log.Error().Err(err).Msg("Phoenix: помилка отримання активних тривог із Temp")
	} else if len(activeRows) > 0 {
		return p.buildPhoenixActiveAlarms(activeRows)
	}

	var rows []phoenixObjectGroupRow
	if err := p.db.SelectContext(ctx, &rows, phoenixObjectsListQuery); err != nil {
		log.Error().Err(err).Msg("Phoenix: помилка отримання активних тривог")
		return nil
	}

	return p.buildPhoenixAlarms(rows)
}

func (p *PhoenixDataProvider) ProcessAlarm(id string, user string, note string) {}

func (p *PhoenixDataProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	panelID, ok := p.resolvePanelID(objectID)
	if !ok {
		return "", "", time.Time{}, time.Time{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var channels []phoenixChannelRow
	if err := p.db.SelectContext(ctx, &channels, phoenixChannelInfoQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання зовнішніх даних каналу")
	} else if len(channels) > 0 {
		signal = phoenixSignalText(channels[0].SignalLevel)
		lastTest = nullTime(channels[0].LastTest)
		testMsg = phoenixTestControlText(channels[0].TestTimeout)
	}

	var groups []phoenixObjectGroupRow
	if err := p.db.SelectContext(ctx, &groups, phoenixObjectDetailGroupsQuery, panelID); err != nil {
		log.Error().Err(err).Str("panelID", panelID).Msg("Phoenix: помилка отримання груп для зовнішніх даних")
	} else {
		for _, row := range groups {
			if ts := nullTime(row.GroupTime); ts.After(lastMsg) {
				lastMsg = ts
			}
		}
	}

	if signal == "" {
		signal = "—"
	}
	if testMsg == "" {
		testMsg = "—"
	}
	return signal, testMsg, lastTest, lastMsg
}

func (p *PhoenixDataProvider) GetTestMessages(objectID string) []models.TestMessage {
	return nil
}

func (p *PhoenixDataProvider) buildObjects(rows []phoenixObjectGroupRow) []models.Object {
	if len(rows) == 0 {
		return nil
	}

	objectsByPanel := make(map[string]*models.Object)
	order := make([]string, 0, len(rows))

	for _, row := range rows {
		panelID := strings.TrimSpace(row.PanelID)
		if panelID == "" {
			continue
		}

		obj := objectsByPanel[panelID]
		if obj == nil {
			obj = &models.Object{
				ID:            p.registerPanelID(panelID),
				DisplayNumber: panelID,
				PanelMark:     panelID,
				Name:          phoenixObjectName(panelID, row.CompanyName, row.GroupName),
				Address:       strings.TrimSpace(nullString(row.Address)),
				Phone:         strings.TrimSpace(nullString(row.Telephones)),
				Phones1:       strings.TrimSpace(nullString(row.Telephones)),
				ContractNum:   panelID,
				DeviceType:    strings.TrimSpace(nullString(row.TypeName)),
				Groups:        make([]models.ObjectGroup, 0, 4),
				IsConnOK:      true,
				IsConnState:   1,
				GuardState:    1,
				LaunchDate:    phoenixDateText(row.CreateDate),
			}
			objectsByPanel[panelID] = obj
			order = append(order, panelID)
		}

		effectiveDisabled := phoenixEffectiveDisabled(row.GroupDisabled, row.PanelDisabled)
		groupStateText := phoenixGroupStateText(row.IsOpen, effectiveDisabled, row.TestPanel, row.StateEvent)
		obj.Groups = append(obj.Groups, models.ObjectGroup{
			ID:        buildPhoenixGroupID(panelID, row.GroupNo),
			Source:    phoenixSourceName,
			Number:    row.GroupNo,
			Name:      phoenixGroupName(row.GroupNo, row.GroupName),
			Armed:     !nullBool(row.IsOpen),
			StateText: groupStateText,
		})

		p.applyPhoenixObjectState(obj, row)
	}

	result := make([]models.Object, 0, len(order))
	for _, panelID := range order {
		obj := objectsByPanel[panelID]
		p.finalizePhoenixObjectState(obj)
		sort.SliceStable(obj.Groups, func(i int, j int) bool {
			return obj.Groups[i].Number < obj.Groups[j].Number
		})
		result = append(result, *obj)
	}
	return result
}

func (p *PhoenixDataProvider) buildPhoenixAlarms(rows []phoenixObjectGroupRow) []models.Alarm {
	if len(rows) == 0 {
		return nil
	}

	alarms := make([]models.Alarm, 0, len(rows))
	for _, row := range rows {
		panelID := strings.TrimSpace(row.PanelID)
		if panelID == "" {
			continue
		}
		if !phoenixStateIsAlarm(row.StateEvent) {
			continue
		}
		if nullBool(row.TestPanel) || nullBool(row.GroupDisabled) || nullBool(row.PanelDisabled) {
			continue
		}

		objectID := p.registerPanelID(panelID)
		objectName := phoenixObjectName(panelID, row.CompanyName, row.GroupName)
		details := phoenixGroupName(row.GroupNo, row.GroupName)
		if details == "" {
			details = "Тривога Phoenix"
		}

		alarmTime := time.Now()
		if ts := nullTime(row.GroupTime); !ts.IsZero() {
			alarmTime = normalizePhoenixEventTime(ts)
		}

		alarms = append(alarms, models.Alarm{
			ID:           ids.StablePhoenixID(panelID, strconv.Itoa(row.GroupNo), "alarm"),
			ObjectID:     objectID,
			ObjectNumber: panelID,
			ObjectName:   objectName,
			Address:      strings.TrimSpace(nullString(row.Address)),
			Time:         alarmTime,
			Details:      details,
			Type:         models.AlarmFire,
			SC1:          1,
		})
	}

	sort.SliceStable(alarms, func(i, j int) bool {
		left := alarms[i].Time
		right := alarms[j].Time
		if left.Equal(right) {
			return alarms[i].ID > alarms[j].ID
		}
		return left.After(right)
	})

	return alarms
}

func (p *PhoenixDataProvider) buildPhoenixActiveAlarms(rows []phoenixActiveAlarmRow) []models.Alarm {
	if len(rows) == 0 {
		return nil
	}

	groupedMessages := make(map[string][]phoenixActiveAlarmMessage, len(rows))
	groupOrder := make([]string, 0, len(rows))

	for _, row := range rows {
		panelID := strings.TrimSpace(row.PanelID)
		if panelID == "" {
			continue
		}

		groupKey := phoenixActiveAlarmCaseKey(row)
		if groupKey == "" {
			continue
		}
		if _, exists := groupedMessages[groupKey]; !exists {
			groupOrder = append(groupOrder, groupKey)
		}
		groupedMessages[groupKey] = append(groupedMessages[groupKey], buildPhoenixActiveAlarmMessage(row))
	}

	alarms := make([]models.Alarm, 0, len(groupedMessages))
	for _, groupKey := range groupOrder {
		messages := groupedMessages[groupKey]
		if len(messages) == 0 {
			continue
		}

		sort.SliceStable(messages, func(i, j int) bool {
			left := messages[i].Time
			right := messages[j].Time
			if left.Equal(right) {
				return messages[i].SortID > messages[j].SortID
			}
			return left.After(right)
		})

		selected, ok := selectPhoenixActiveAlarmMessage(messages)
		if !ok {
			continue
		}
		selectedRow := selected.Row
		panelID := strings.TrimSpace(selectedRow.PanelID)
		if panelID == "" {
			continue
		}

		alarmType, mapped := mapEventTypeToAlarmType(selected.EventType)
		if !mapped {
			alarmType = models.AlarmSystemEvent
		}

		details := strings.TrimSpace(selected.Details)
		if details == "" {
			details = "Тривога Phoenix"
		}

		alarmID := ids.StablePhoenixID(
			panelID,
			groupKey,
			"alarm_case",
		)
		rowSC1 := resolvePhoenixGroupedAlarmSC1(messages, phoenixActiveAlarmMessageSC1(selected))

		alarms = append(alarms, models.Alarm{
			ID:           alarmID,
			ObjectID:     p.registerPanelID(panelID),
			ObjectNumber: panelID,
			ObjectName:   phoenixObjectName(panelID, selectedRow.CompanyName, selectedRow.GroupName),
			Address:      strings.TrimSpace(nullString(selectedRow.Address)),
			Time:         selected.Time,
			Details:      details,
			Type:         alarmType,
			ZoneNumber:   int(nullInt64(selectedRow.ZoneNo)),
			ZoneName:     strings.TrimSpace(nullString(selectedRow.ZoneName)),
			SC1:          rowSC1,
			SourceMsgs:   mapPhoenixActiveAlarmMessagesToAlarmMsgs(messages),
		})
	}

	sort.SliceStable(alarms, func(i, j int) bool {
		left := alarms[i].Time
		right := alarms[j].Time
		if left.Equal(right) {
			return alarms[i].ID > alarms[j].ID
		}
		return left.After(right)
	})

	return alarms
}

type phoenixActiveAlarmMessage struct {
	Row       phoenixActiveAlarmRow
	Time      time.Time
	Details   string
	EventType models.EventType
	IsAlarm   bool
	SortID    int64
}

func buildPhoenixActiveAlarmMessage(row phoenixActiveAlarmRow) phoenixActiveAlarmMessage {
	details := strings.TrimSpace(phoenixActiveAlarmDetails(row))
	eventType := phoenixActiveAlarmEventType(row, details)
	isAlarm := phoenixRowIsAlarm(row.GroupSent, row.AutoReset)

	eventTime := time.Now()
	if ts := nullTime(row.TimeEvent); !ts.IsZero() {
		eventTime = normalizePhoenixEventTime(ts)
	}

	sortID := int64(0)
	if row.EventID.Valid && row.EventID.Int64 > 0 {
		sortID = row.EventID.Int64
	} else if row.EventParentID.Valid && row.EventParentID.Int64 > 0 {
		sortID = row.EventParentID.Int64
	}

	return phoenixActiveAlarmMessage{
		Row:       row,
		Time:      eventTime,
		Details:   details,
		EventType: eventType,
		IsAlarm:   isAlarm,
		SortID:    sortID,
	}
}

func selectPhoenixActiveAlarmMessage(messages []phoenixActiveAlarmMessage) (phoenixActiveAlarmMessage, bool) {
	if len(messages) == 0 {
		return phoenixActiveAlarmMessage{}, false
	}

	// Пріоритет 1: тривожні події типу "пожежа/проникнення/паніка/...".
	for _, msg := range messages {
		if msg.IsAlarm && isPrimaryAlarmEventType(msg.EventType) {
			return msg, true
		}
	}
	// Пріоритет 2: інші тривожні стани (fault/offline/...).
	for _, msg := range messages {
		if msg.IsAlarm {
			return msg, true
		}
	}

	return messages[0], true
}

func resolvePhoenixGroupedAlarmSC1(messages []phoenixActiveAlarmMessage, fallback int) int {
	if len(messages) == 0 {
		return fallback
	}

	latest := messages[0]
	if latest.EventType == models.EventFault && hasPrimaryPhoenixAlarmMessage(messages) {
		// Спецправило: після тривоги несправність не знімає "пожежний" колір головного рядка.
		return 1
	}

	latestSC1 := phoenixActiveAlarmMessageSC1(latest)
	if latestSC1 != 0 {
		return latestSC1
	}
	for _, msg := range messages {
		msgSC1 := phoenixActiveAlarmMessageSC1(msg)
		if msgSC1 != 0 {
			return msgSC1
		}
	}
	return fallback
}

func hasPrimaryPhoenixAlarmMessage(messages []phoenixActiveAlarmMessage) bool {
	for _, msg := range messages {
		if msg.IsAlarm && isPrimaryAlarmEventType(msg.EventType) {
			return true
		}
	}
	return false
}

func phoenixActiveAlarmMessageSC1(msg phoenixActiveAlarmMessage) int {
	return phoenixEventSC1(
		msg.Row.TypeCodeID,
		msg.Row.EventCode,
		msg.Row.ContactIDCode,
		msg.Row.TypeMessage,
		msg.Row.AccessCode,
		msg.Row.SystemFlag,
		msg.Details,
	)
}

func mapPhoenixActiveAlarmMessagesToAlarmMsgs(messages []phoenixActiveAlarmMessage) []models.AlarmMsg {
	if len(messages) == 0 {
		return nil
	}

	result := make([]models.AlarmMsg, 0, len(messages))
	for _, msg := range messages {
		result = append(result, models.AlarmMsg{
			Time:    msg.Time,
			Code:    strings.TrimSpace(nullString(msg.Row.EventCode)),
			Number:  int(nullInt64(msg.Row.ZoneNo)),
			Details: strings.TrimSpace(msg.Details),
			SC1:     phoenixActiveAlarmMessageSC1(msg),
			IsAlarm: msg.IsAlarm,
		})
	}
	return result
}

func phoenixActiveAlarmCaseKey(row phoenixActiveAlarmRow) string {
	panelID := strings.TrimSpace(row.PanelID)
	if panelID == "" {
		return ""
	}
	// Для стрічки активних тривог Phoenix групуємо події на рівні об'єкта:
	// один рядок = один panel_id, а в SourceMsgs зберігаємо повну хронологію.
	return panelID
}

func (p *PhoenixDataProvider) applyPhoenixObjectState(obj *models.Object, row phoenixObjectGroupRow) {
	if obj == nil {
		return
	}

	if ts := nullTime(row.GroupTime); ts.After(obj.LastMessageTime) {
		obj.LastMessageTime = ts
	}
	if name := strings.TrimSpace(nullString(row.TypeName)); obj.DeviceType == "" && name != "" {
		obj.DeviceType = name
	}

	switch {
	case nullInt64(row.StateEvent) == 2 || nullInt64(row.StateEvent) == 3:
		obj.Status = models.StatusFire
		obj.StatusText = "ТРИВОГА"
		obj.AlarmState = 1
	case obj.Status != models.StatusFire && nullBool(row.TestPanel):
		obj.BlockedArmedOnOff = 2
		obj.Status = models.StatusNormal
		obj.StatusText = phoenixStandStateText
	case obj.Status != models.StatusFire && (nullBool(row.GroupDisabled) || nullBool(row.PanelDisabled)):
		if obj.BlockedArmedOnOff != 2 {
			obj.BlockedArmedOnOff = 1
		}
		obj.Status = models.StatusNormal
		obj.StatusText = phoenixBlockedStateText
	case obj.StatusText == "":
		if nullBool(row.IsOpen) {
			obj.StatusText = phoenixDisarmedStateText
		} else {
			obj.StatusText = "ПІД ОХОРОНОЮ"
		}
		obj.Status = models.StatusNormal
	}
}

func (p *PhoenixDataProvider) finalizePhoenixObjectState(obj *models.Object) {
	if obj == nil {
		return
	}

	if len(obj.Groups) == 0 {
		switch obj.BlockedArmedOnOff {
		case 1:
			obj.GuardState = 1
			obj.IsUnderGuard = false
			if obj.StatusText == "" {
				obj.StatusText = phoenixBlockedStateText
			}
		case 2:
			obj.GuardState = 1
			obj.IsUnderGuard = false
			if obj.StatusText == "" {
				obj.StatusText = phoenixStandStateText
			}
		default:
			obj.IsUnderGuard = true
			if obj.GuardState == 0 {
				obj.GuardState = 1
			}
			if obj.StatusText == "" {
				obj.StatusText = "ПІД ОХОРОНОЮ"
			}
		}
		return
	}

	if obj.BlockedArmedOnOff == 1 {
		obj.GuardState = 1
		obj.IsUnderGuard = false
		if obj.Status == models.StatusNormal {
			obj.StatusText = phoenixBlockedStateText
		}
		return
	}

	if obj.BlockedArmedOnOff == 2 {
		obj.GuardState = 1
		obj.IsUnderGuard = false
		if obj.Status == models.StatusNormal {
			obj.StatusText = phoenixStandStateText
		}
		return
	}

	armedCount := 0
	disarmedCount := 0
	for _, group := range obj.Groups {
		if group.Armed {
			armedCount++
		} else {
			disarmedCount++
		}
	}

	switch {
	case disarmedCount == len(obj.Groups):
		obj.GuardState = 0
		obj.IsUnderGuard = false
		if obj.Status == models.StatusNormal {
			obj.StatusText = phoenixDisarmedStateText
		}
	case armedCount == len(obj.Groups):
		obj.GuardState = 1
		obj.IsUnderGuard = true
		if obj.Status == models.StatusNormal {
			obj.StatusText = "ПІД ОХОРОНОЮ"
		}
	default:
		obj.GuardState = 1
		obj.IsUnderGuard = true
		if obj.Status == models.StatusNormal {
			obj.StatusText = phoenixPartialDisarmedText
		}
	}
}

func (p *PhoenixDataProvider) applyChannelInfo(obj *models.Object, row phoenixChannelRow) {
	if obj == nil {
		return
	}
	if name := strings.TrimSpace(nullString(row.DeviceName)); name != "" {
		obj.PanelMark = name
	}
	if value := strings.TrimSpace(nullString(row.ChannelNo)); value != "" {
		obj.ContractNum = value
	}
	if value := strings.TrimSpace(nullString(row.ChannelType)); value != "" {
		switch {
		case row.OpenInternetChannelID.Valid:
			obj.ObjChan = 5
		case strings.Contains(strings.ToLower(value), "автодозв"), strings.Contains(strings.ToLower(value), "autodial"):
			obj.ObjChan = 1
		}
	}
	if sim := strings.TrimSpace(nullString(row.Sim1Number)); sim != "" {
		obj.SIM1 = sim
	}
	if sim := strings.TrimSpace(nullString(row.Sim2Number)); sim != "" {
		obj.SIM2 = sim
	}
	obj.SignalStrength = phoenixSignalText(row.SignalLevel)
	obj.LastTestTime = nullTime(row.LastTest)
	if timeoutMinutes := phoenixTimeoutMinutes(row.TestTimeout); timeoutMinutes > 0 {
		obj.TestControl = 1
		obj.TestTime = timeoutMinutes
		if timeoutMinutes%60 == 0 {
			obj.AutoTestHours = int(timeoutMinutes / 60)
		}
	}
}

func (p *PhoenixDataProvider) mapEventRow(row phoenixEventRow) models.Event {
	panelID := strings.TrimSpace(row.PanelID)
	objectID := p.registerPanelID(panelID)

	details := phoenixAlarmDetails(row.CodeMessage, row.ZoneName, int(nullInt64(row.GroupNo)), row.GroupName)
	typeLabel := strings.TrimSpace(nullString(row.TypeMessage))

	return models.Event{
		ID:           stablePhoenixEventID(panelID, row.EventID),
		Time:         normalizePhoenixEventTime(row.TimeEvent),
		ObjectID:     objectID,
		ObjectNumber: panelID,
		ObjectName:   phoenixObjectName(panelID, row.CompanyName, row.GroupName),
		Type: phoenixEventType(
			row.EventCode,
			row.ContactIDCode,
			row.TypeCodeID,
			row.TypeMessage,
			row.AccessCode,
			row.SystemFlag,
			details,
		),
		TypeLabel:  typeLabel,
		ZoneNumber: int(nullInt64(row.ZoneNo)),
		Details:    details,
		SC1: phoenixEventSC1(
			row.TypeCodeID,
			row.EventCode,
			row.ContactIDCode,
			row.TypeMessage,
			row.AccessCode,
			row.SystemFlag,
			details,
		),
	}
}

func phoenixAlarmDetails(codeMessage sql.NullString, zoneName sql.NullString, groupNo int, groupName sql.NullString) string {
	details := strings.TrimSpace(nullString(codeMessage))
	if zone := strings.TrimSpace(nullString(zoneName)); zone != "" {
		if details != "" {
			details += " [" + zone + "]"
		} else {
			details = zone
		}
	}

	group := strings.TrimSpace(nullString(groupName))
	if group == "" && groupNo > 0 {
		group = fmt.Sprintf("Група %d", groupNo)
	}
	if group == "" {
		return details
	}
	if details != "" {
		return details + " | " + group
	}
	return group
}

func phoenixActiveAlarmDetails(row phoenixActiveAlarmRow) string {
	details := strings.TrimSpace(nullString(row.CodeMessage))
	zone := strings.TrimSpace(nullString(row.ZoneName))
	if zone != "" && !strings.Contains(strings.ToLower(details), strings.ToLower(zone)) {
		if details != "" {
			details += " [" + zone + "]"
		} else {
			details = zone
		}
	}
	group := strings.TrimSpace(nullString(row.GroupMessage))
	if group == "" {
		group = strings.TrimSpace(nullString(row.GroupName))
	}
	if group == "" && row.GroupNo > 0 {
		group = fmt.Sprintf("Група %d", row.GroupNo)
	}
	if details != "" && group != "" {
		return details + " | " + group
	}
	if details != "" {
		return details
	}
	return group
}

func phoenixActiveAlarmEventType(row phoenixActiveAlarmRow, details string) models.EventType {
	if nullBool(row.IsAlarmButton) {
		return models.EventPanic
	}
	return phoenixEventType(
		row.EventCode,
		row.ContactIDCode,
		row.TypeCodeID,
		row.TypeMessage,
		row.AccessCode,
		row.SystemFlag,
		details,
	)
}

func mapPhoenixEventRows(rows []phoenixEventRow, mapRow func(phoenixEventRow) models.Event) []models.Event {
	events := make([]models.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapRow(row))
	}
	return events
}

func reversePhoenixEvents(events []models.Event) {
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
}

func maxPhoenixEventID(rows []phoenixEventRow, current int64) int64 {
	maxID := current
	for _, row := range rows {
		if row.EventID > maxID {
			maxID = row.EventID
		}
	}
	return maxID
}

func (p *PhoenixDataProvider) resolvePanelID(objectID string) (string, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(objectID))
	if err != nil {
		return "", false
	}

	p.idMu.RLock()
	panelID, ok := p.panelByID[parsed]
	p.idMu.RUnlock()
	if ok {
		return panelID, true
	}

	objects := p.GetObjects()
	for _, obj := range objects {
		if obj.ID == parsed {
			return strings.TrimSpace(obj.DisplayNumber), true
		}
	}
	return "", false
}

func (p *PhoenixDataProvider) registerPanelID(panelID string) int {
	panelID = strings.TrimSpace(panelID)
	if panelID == "" {
		return 0
	}

	p.idMu.Lock()
	defer p.idMu.Unlock()

	if id, ok := p.idByPanel[panelID]; ok {
		return id
	}

	candidate := ids.StablePhoenixID(panelID)
	for {
		existing, occupied := p.panelByID[candidate]
		if !occupied || existing == panelID {
			p.panelByID[candidate] = panelID
			p.idByPanel[panelID] = candidate
			return candidate
		}
		candidate++
		if candidate > ids.PhoenixObjectIDNamespaceEnd {
			candidate = ids.PhoenixObjectIDNamespaceStart
		}
	}
}

func buildPhoenixGroupID(panelID string, groupNo int) string {
	return fmt.Sprintf("phoenix:panel=%s:group=%d", strings.TrimSpace(panelID), groupNo)
}

func phoenixObjectName(panelID string, companyName sql.NullString, groupName sql.NullString) string {
	if name := strings.TrimSpace(nullString(companyName)); name != "" {
		return name
	}
	if name := strings.TrimSpace(nullString(groupName)); name != "" {
		return name
	}
	return strings.TrimSpace(panelID)
}

func phoenixGroupName(groupNo int, name sql.NullString) string {
	if value := strings.TrimSpace(nullString(name)); value != "" {
		return value
	}
	return fmt.Sprintf("Група %d", groupNo)
}

func phoenixEffectiveDisabled(groupDisabled sql.NullBool, panelDisabled sql.NullBool) sql.NullBool {
	if nullBool(groupDisabled) || nullBool(panelDisabled) {
		return sql.NullBool{Bool: true, Valid: true}
	}
	if groupDisabled.Valid || panelDisabled.Valid {
		return sql.NullBool{Bool: false, Valid: true}
	}
	return sql.NullBool{}
}

func phoenixGroupStateText(isOpen sql.NullBool, groupDisabled sql.NullBool, testPanel sql.NullBool, stateEvent sql.NullInt64) string {
	switch {
	case phoenixStateIsAlarm(stateEvent):
		return "ТРИВОГА"
	case nullBool(testPanel):
		return phoenixStandStateText
	case nullBool(groupDisabled):
		return phoenixBlockedStateText
	case nullBool(isOpen):
		return phoenixDisarmedStateText
	default:
		return "ПІД ОХОРОНОЮ"
	}
}

func phoenixStateIsAlarm(stateEvent sql.NullInt64) bool {
	return nullInt64(stateEvent) == 2 || nullInt64(stateEvent) == 3
}

func phoenixZoneStatus(status sql.NullInt64) models.ZoneStatus {
	switch nullInt64(status) {
	case 1:
		return models.ZoneNormal
	case 2:
		return models.ZoneAlarm
	default:
		return models.ZoneNormal
	}
}

func phoenixZoneTypeText(isAlarmButton sql.NullBool) string {
	if nullBool(isAlarmButton) {
		return "Тривожна кнопка"
	}
	return "Охоронна"
}

func phoenixSignalText(level sql.NullInt64) string {
	if !level.Valid {
		return "—"
	}
	return fmt.Sprintf("%d", level.Int64)
}

func phoenixEventType(
	code sql.NullString,
	contactIDCode sql.NullString,
	typeCodeID sql.NullInt64,
	typeMessage sql.NullString,
	accessCode sql.NullString,
	systemFlag sql.NullBool,
	details string,
) models.EventType {
	if nullBool(systemFlag) {
		return models.SystemEvent
	}
	if strings.TrimSpace(nullString(accessCode)) == "1" {
		return models.EventArm
	}

	codeValue := strings.ToUpper(strings.TrimSpace(nullString(code)))
	contactCodeValue := strings.ToUpper(strings.TrimSpace(nullString(contactIDCode)))
	typeMessageValue := strings.TrimSpace(nullString(typeMessage))

	if eventType, ok := phoenixEventTypeByTypeMessage(typeMessageValue); ok {
		return eventType
	}

	baseType := models.EventType("")
	if typeCodeID.Valid {
		if eventType, ok := phoenixEventTypeByTypeCode(typeCodeID.Int64); ok {
			baseType = eventType
		}
	}

	codeAndDetails := strings.TrimSpace(typeMessageValue + " " + details)
	if contactCodeValue != "" {
		derivedType, hasDerivedType := phoenixEventTypeByCodeAndDetails(contactCodeValue, codeAndDetails)
		if baseType != "" {
			if hasDerivedType && shouldOverridePhoenixBaseEventType(baseType, derivedType) {
				return derivedType
			}
			return baseType
		}
		if hasDerivedType {
			return derivedType
		}
	}

	derivedType, hasDerivedType := phoenixEventTypeByCodeAndDetails(codeValue, codeAndDetails)
	if baseType != "" {
		if hasDerivedType && shouldOverridePhoenixBaseEventType(baseType, derivedType) {
			return derivedType
		}
		return baseType
	}
	if hasDerivedType {
		return derivedType
	}
	return models.SystemEvent
}

func phoenixEventTypeByTypeMessage(typeMessage string) (models.EventType, bool) {
	value := strings.ToLower(strings.TrimSpace(typeMessage))
	if value == "" {
		return "", false
	}

	switch value {
	case "пожежа", "ймовірна пожежа", "пожежна тривога з клавіатури":
		return models.EventFire, true
	case "тривога":
		return models.EventBurglary, true
	case "тривожна кнопка", "мобільна тривожна кнопка", "напад":
		return models.EventPanic, true
	case "медична тривога":
		return models.EventMedical, true
	case "витік газу":
		return models.EventGas, true
	case "тривога тампера":
		return models.EventTamper, true

	case "норма", "норма несправності", "норма контролю патруля", "норма сирени",
		"норма системної помилки", "норма лінії", "норма тесту", "норма акб",
		"норма тампера", "норма зв'язку", "норма зв'язку з лінд",
		"норма зв'язку з пристроєм", "норма зв'язку з пцс", "скасування тривоги":
		return models.EventRestore, true

	case "постановка", "безумовна постановка", "постановка ключем", "постановка кодом",
		"дистанційна постановка", "постановка з радіобрелока", "лишаюся вдома",
		"постановка з мобільного", "початок постановки", "постановка забороненим ключем":
		return models.EventArm, true
	case "зняття", "дистанційне зняття", "зняття кодом", "зняття з радіобрелока",
		"зняття з мобільного", "зняття ключем", "початок зняття", "зняття забороненим ключем":
		return models.EventDisarm, true

	case "тест", "режим тестування радіодатчиків":
		return models.EventTest, true

	case "втрата основного живлення":
		return models.EventPowerFail, true
	case "норма основного живлення":
		return models.EventPowerOK, true
	case "проблема акб", "відключення заряду акб":
		return models.EventBatteryLow, true

	case "втрата зв'язку з лінд", "втрата зв'язку з пцс", "втрата зв'язку з пристроєм":
		return models.EventOffline, true
	case "увімкнення зв'язку з пцс":
		return models.EventOnline, true

	case "звіт", "звіт. із тривогами", "система відправки email", "viber розсилка",
		"перевір дзвінок із об'єкта", "покази електролічильника", "зображення",
		"gps координати правильні":
		return models.EventNotification, true

	case "планшетник", "дозвіл камери", "заборона камери", "вхід на рівень доступу",
		"віддалене керування":
		return models.EventOperatorAction, true

	case "система", "системна подія від sur-gard", "системний запис":
		return models.SystemEvent, true

	case "несправність", "проблема тесту", "несправність шлейфу", "помилка під час звіту",
		"системна несправність", "помилка", "проблема з сиреною", "кз лінії",
		"проблема опитування", "проблема орлан-gprs", "системна помилка",
		"gps координати не правильні", "порушення контролю патруля", "втрата подій",
		"переповнення буфера подій", "об'єкт не знято з охорони", "об'єкт не закрито",
		"помилковий код не підтверджено", "заборона постановки":
		return models.EventFault, true

	case "контроль патруля", "увімкнення контролю сирени", "прив'язка пристрою",
		"віддалене конфігурування", "виведення зі стендів", "увімкнення контролю 220в",
		"початок посилки gps координат", "увімкнення режиму очікування",
		"увімкнення запалювання", "вихід замкнено", "вимкнення контролю сирени",
		"вимкнення контролю акб", "увімкнення виходу", "скидання", "вимкнення контролю 220в",
		"скидання з пцс", "вихід розімкнено", "увімкнення шлейфу", "сирену вимкнено",
		"зміна sim карти", "вимкнення виходу", "запит стану", "повернення на gps сервіс",
		"переведення до стендів", "патруль", "увімкнення живлення датчиків",
		"вимкнення режиму очікування", "оновлення прошивки", "зняття заборони",
		"увімкнення контролю акб", "датчик руху", "функцію реле ввімкнено",
		"реле вимкнено", "очищення буфера", "вимкнення живлення gps приймача",
		"сирену ввімкнено", "датчик удару", "вимкнення запалювання", "увімкнення ппк",
		"вимкнення зв'язку з пцс", "відключення живлення датчиків",
		"функцію реле вимкнено", "реле ввімкнено", "вимкнення шлейфу",
		"увімкнення живлення gps приймача", "увімкнення заряду акб":
		return models.EventService, true
	}

	switch {
	case strings.Contains(value, "пожеж"):
		return models.EventFire, true
	case strings.Contains(value, "тривожн"), strings.Contains(value, "напад"):
		return models.EventPanic, true
	case strings.Contains(value, "медич"):
		return models.EventMedical, true
	case strings.Contains(value, "газ"):
		return models.EventGas, true
	case strings.Contains(value, "тампер"):
		if strings.Contains(value, "норма") {
			return models.EventRestore, true
		}
		return models.EventTamper, true
	case strings.Contains(value, "втрата основного живлення"):
		return models.EventPowerFail, true
	case strings.Contains(value, "основного живлення"), strings.Contains(value, "220в"):
		if strings.Contains(value, "норма") {
			return models.EventPowerOK, true
		}
		return models.EventPowerFail, true
	case strings.Contains(value, "акб"):
		if strings.Contains(value, "норма") || strings.Contains(value, "увімкнення заряду") {
			return models.EventRestore, true
		}
		return models.EventBatteryLow, true
	case strings.Contains(value, "зв'язку"):
		if strings.Contains(value, "норма") || strings.Contains(value, "увімкнення") {
			return models.EventOnline, true
		}
		return models.EventOffline, true
	case strings.Contains(value, "норма"), strings.Contains(value, "скасування"):
		return models.EventRestore, true
	case strings.Contains(value, "знят"):
		return models.EventDisarm, true
	case strings.Contains(value, "постан"), strings.Contains(value, "взят"):
		return models.EventArm, true
	case strings.Contains(value, "тест"):
		return models.EventTest, true
	case strings.Contains(value, "несправ"), strings.Contains(value, "помилка"),
		strings.Contains(value, "проблема"), strings.Contains(value, "кз"),
		strings.Contains(value, "втрата подій"):
		return models.EventFault, true
	case strings.Contains(value, "звіт"), strings.Contains(value, "розсилка"),
		strings.Contains(value, "дзвінок"), strings.Contains(value, "зображення"):
		return models.EventNotification, true
	case strings.Contains(value, "система"), strings.Contains(value, "sur-gard"):
		return models.SystemEvent, true
	default:
		return "", false
	}
}

func phoenixEventTypeByCodeAndDetails(code string, details string) (models.EventType, bool) {
	if eventType, ok := phoenixEventTypeByCIDCode(code); ok {
		return eventType, true
	}

	text := strings.ToLower(strings.TrimSpace(code + " " + details))
	switch {
	case (strings.HasPrefix(strings.TrimSpace(code), "R") && hasPhoenixCodeDigits(code)),
		strings.Contains(text, "віднов"),
		strings.Contains(text, "норма"):
		return models.EventRestore, true
	case strings.Contains(text, "220"), strings.Contains(text, "основного живлення"):
		if strings.Contains(text, "втрата") || strings.Contains(text, "проблем") {
			return models.EventPowerFail, true
		}
		return models.EventPowerOK, true
	case strings.Contains(text, "акб"):
		if strings.Contains(text, "проблем") || strings.Contains(text, "низь") {
			return models.EventBatteryLow, true
		}
		return models.EventRestore, true
	case strings.Contains(text, "напад"), strings.Contains(text, "тривожн"):
		return models.EventPanic, true
	case strings.Contains(text, "медич"):
		return models.EventMedical, true
	case strings.Contains(text, "газ"):
		return models.EventGas, true
	case strings.Contains(text, "тампер"):
		if strings.Contains(text, "норма") || strings.Contains(text, "віднов") {
			return models.EventRestore, true
		}
		return models.EventTamper, true
	case strings.Contains(text, "знят"):
		return models.EventDisarm, true
	case strings.Contains(text, "постан"), strings.Contains(text, "взят"):
		return models.EventArm, true
	case strings.Contains(text, "зв'язку"), strings.Contains(text, "offline"):
		return models.EventOffline, true
	case strings.Contains(text, "несправ"), strings.Contains(text, "обрив"), strings.Contains(text, "кз"):
		return models.EventFault, true
	case strings.Contains(text, "проник"), strings.Contains(text, "охорон"):
		return models.EventBurglary, true
	case strings.Contains(text, "пожеж"), strings.Contains(text, "тривог"):
		return models.EventFire, true
	default:
		return "", false
	}
}

func hasPhoenixCodeDigits(code string) bool {
	for _, ch := range code {
		if ch >= '0' && ch <= '9' {
			return true
		}
	}
	return false
}

func shouldOverridePhoenixBaseEventType(base models.EventType, candidate models.EventType) bool {
	if candidate == "" || candidate == base {
		return false
	}
	switch base {
	case models.EventFault, models.SystemEvent, models.EventService, models.EventNotification:
		return true
	default:
		return false
	}
}

func phoenixEventTypeByCIDCode(code string) (models.EventType, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return "", false
	}

	cid, ok := extractPhoenixCIDCode(code)
	if !ok {
		return "", false
	}

	isRestore := strings.HasPrefix(code, "R")
	if isRestore {
		switch cid {
		case 301:
			return models.EventPowerOK, true
		case 350:
			return models.EventOnline, true
		case 401:
			return models.EventDisarm, true
		default:
			if cid >= 100 && cid < 200 {
				return models.EventRestore, true
			}
		}
	}

	switch cid {
	case 110:
		return models.EventFire, true
	case 120:
		return models.EventPanic, true
	case 130:
		return models.EventBurglary, true
	case 151:
		return models.EventGas, true
	case 301:
		return models.EventPowerFail, true
	case 302:
		return models.EventBatteryLow, true
	case 350:
		return models.EventOffline, true
	case 383:
		return models.EventTamper, true
	case 401:
		return models.EventArm, true
	default:
		return "", false
	}
}

func extractPhoenixCIDCode(code string) (int, bool) {
	digits := make([]rune, 0, 3)
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			continue
		}
		digits = append(digits, ch)
		if len(digits) == 3 {
			break
		}
	}
	if len(digits) < 3 {
		return 0, false
	}

	value, err := strconv.Atoi(string(digits))
	if err != nil {
		return 0, false
	}
	return value, true
}

func phoenixEventTypeByTypeCode(typeCodeID int64) (models.EventType, bool) {
	switch typeCodeID {
	case 0:
		return models.SystemEvent, true

	case 1, 22, 102, 116:
		return models.EventBurglary, true
	case 10, 124, 128:
		return models.EventPanic, true
	case 14, 129, 131:
		return models.EventFire, true
	case 28:
		return models.EventTamper, true
	case 85:
		return models.EventMedical, true
	case 156:
		return models.EventGas, true

	case 2, 29, 30, 31, 34, 53, 55, 92, 122, 139:
		return models.EventRestore, true
	case 3, 89, 119, 123, 125, 133, 143, 146, 148, 150:
		return models.EventArm, true
	case 4, 23, 130, 132, 144, 145, 147, 149:
		return models.EventDisarm, true
	case 5, 153, 158:
		return models.EventTest, true
	case 6:
		return models.EventPowerFail, true
	case 7:
		return models.EventPowerOK, true
	case 8:
		return models.EventBatteryLow, true
	case 18, 32, 50, 56, 117:
		return models.EventOffline, true
	case 19, 35, 51, 57, 99, 113:
		return models.EventOnline, true

	case 11, 67, 70, 74, 75, 76, 78, 93, 94, 95, 104, 137, 138:
		return models.EventNotification, true
	case 36, 79, 87, 88, 136:
		return models.EventOperatorAction, true

	case 15, 16, 17, 21, 33, 52, 54, 66, 69, 90, 91, 121, 126:
		return models.EventFault, true

	case 20, 68, 127, 157:
		return models.SystemEvent, true

	case 12, 13, 24, 25, 26, 27, 37, 38, 39, 40, 41, 42, 49, 58,
		59, 60, 61, 62, 63, 64, 65, 71, 72, 73, 80, 81, 82, 83, 84,
		96, 97, 98, 100, 101, 103, 106, 107, 108, 109, 110, 111, 112,
		114, 115, 118, 120, 134, 135, 140, 141, 142, 151, 152:
		return models.EventService, true
	default:
		return "", false
	}
}

func phoenixEventSC1(
	typeCodeID sql.NullInt64,
	code sql.NullString,
	contactIDCode sql.NullString,
	typeMessage sql.NullString,
	accessCode sql.NullString,
	systemFlag sql.NullBool,
	details string,
) int {
	switch phoenixEventType(code, contactIDCode, typeCodeID, typeMessage, accessCode, systemFlag, details) {
	case models.EventFire, models.EventBurglary, models.EventPanic, models.EventMedical, models.EventGas, models.EventTamper:
		return 1
	case models.EventFault, models.EventBatteryLow, models.EventPowerFail:
		return 2
	case models.EventOffline:
		return 12
	case models.EventRestore, models.EventOnline, models.EventPowerOK:
		return 5
	case models.EventArm:
		return 10
	case models.EventDisarm:
		return 11
	default:
		return 0
	}
}

func phoenixRowIsAlarm(groupSent sql.NullBool, autoReset sql.NullBool) bool {
	if nullBool(groupSent) {
		return true
	}
	if nullBool(autoReset) {
		return false
	}
	return false
}

func phoenixDateText(value sql.NullTime) string {
	if !value.Valid || value.Time.IsZero() {
		return ""
	}
	return value.Time.Format("02.01.2006")
}

func phoenixTimeoutMinutes(value sql.NullTime) int64 {
	if !value.Valid || value.Time.IsZero() {
		return 0
	}

	hours, minutes, seconds := value.Time.Clock()
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second
	if duration <= 0 {
		return 0
	}

	return int64(duration / time.Minute)
}

func phoenixTestControlText(value sql.NullTime) string {
	minutes := phoenixTimeoutMinutes(value)
	if minutes <= 0 {
		return ""
	}
	if minutes%60 == 0 {
		hours := minutes / 60
		if hours == 1 {
			return "кожну 1 год"
		}
		return fmt.Sprintf("кожні %d год", hours)
	}
	return fmt.Sprintf("кожні %d хв", minutes)
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullBool(value sql.NullBool) bool {
	return value.Valid && value.Bool
}

func nullInt64(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func nullTime(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func normalizePhoenixEventTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	year, month, day := value.Date()
	hour, minute, second := value.Clock()
	return time.Date(year, month, day, hour, minute, second, value.Nanosecond(), time.Local)
}
