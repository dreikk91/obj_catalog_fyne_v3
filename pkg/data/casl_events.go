package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

func (p *CASLCloudProvider) GetEvents() []models.Event {
	p.ensureRealtimeStream()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()
	if events, err := p.readEventsJournalAsEvents(ctx); err == nil {
		return events
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]models.Event(nil), p.cachedEvents...)
}

func (p *CASLCloudProvider) getEventsFromBasketFallback(ctx context.Context) []models.Event {
	basketCount, err := p.readBasketCount(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося отримати count з кошика тривог")
		p.mu.RLock()
		events := append([]models.Event(nil), p.cachedEvents...)
		p.mu.RUnlock()
		return events
	}

	objectID, objectName := p.primaryObjectContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.hasBasketCount {
		p.lastBasketCount = basketCount
		p.hasBasketCount = true
		return append([]models.Event(nil), p.cachedEvents...)
	}

	if basketCount != p.lastBasketCount {
		eventType := models.EventFault
		eventSC1 := 2
		if basketCount == 0 {
			eventType = models.EventRestore
			eventSC1 = 5
		}

		event := models.Event{
			ID:         nextCASLEventID(),
			Time:       time.Now(),
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			Details:    fmt.Sprintf("CASL Cloud: активних тривог у кошику %d (було %d)", basketCount, p.lastBasketCount),
			SC1:        eventSC1,
		}

		p.cachedEvents = append([]models.Event{event}, p.cachedEvents...)
		if len(p.cachedEvents) > caslMaxCachedEvents {
			p.cachedEvents = p.cachedEvents[:caslMaxCachedEvents]
		}
		p.lastBasketCount = basketCount
	}

	return append([]models.Event(nil), p.cachedEvents...)
}

func (p *CASLCloudProvider) readEventsJournalAsEvents(ctx context.Context) ([]models.Event, error) {
	now := time.Now().UnixMilli()
	p.mu.RLock()
	start := p.eventsCursorMs
	startGate := p.eventsStartAtMs
	if start <= 0 {
		start = p.eventsStartAtMs
	}
	if start <= 0 {
		start = now
	}
	p.mu.RUnlock()

	rows, err := p.ReadEventsJournal(ctx, CASLReadEventsRequest{
		TimeStart:   start,
		TimeEnd:     now,
		TimeRequest: now,
	})
	if err != nil {
		return nil, err
	}
	logCASLReadEventsRows(start, now, rows)

	events, maxEventTime := p.mapCASLRowsToEvents(ctx, rows, startGate)
	p.updateRealtimeAlarmsFromRows(ctx, rows)
	p.mu.Lock()
	if maxEventTime > p.eventsCursorMs {
		p.eventsCursorMs = maxEventTime
	}
	if now > p.eventsCursorMs {
		p.eventsCursorMs = now
	}
	added := p.mergeCachedEventsLocked(events)
	if added > 0 {
		p.eventsRevision++
	}
	cached := append([]models.Event(nil), p.cachedEvents...)
	p.mu.Unlock()

	return cached, nil
}

func logCASLReadEventsRows(start, end int64, rows []CASLObjectEvent) {
	log.Debug().
		Int64("time_start", start).
		Int64("time_end", end).
		Int("rows", len(rows)).
		Msg("CASL read_events: отримано події")

	const maxRowsToLog = 200
	for idx, row := range rows {
		if idx >= maxRowsToLog {
			log.Debug().
				Int("logged_rows", maxRowsToLog).
				Int("total_rows", len(rows)).
				Msg("CASL read_events: лог скорочено")
			return
		}

		log.Debug().
			Int("idx", idx).
			Int64("ppk_num", row.PPKNum).
			Str("device_id", strings.TrimSpace(row.DeviceID)).
			Str("obj_id", strings.TrimSpace(row.ObjID)).
			Str("code", strings.TrimSpace(row.Code)).
			Str("contact_id", strings.TrimSpace(row.ContactID)).
			Int64("number", row.Number).
			Str("type", strings.TrimSpace(row.Type)).
			Int64("time", row.Time).
			Msg("CASL read_events row")
	}
}

func (p *CASLCloudProvider) mapCASLRowsToEvents(ctx context.Context, rows []CASLObjectEvent, startGate int64) ([]models.Event, int64) {
	ppkFilter := make(map[int64]struct{}, len(rows))
	objFilter := make(map[string]struct{}, len(rows))
	filteredRows := make([]CASLObjectEvent, 0, len(rows))
	resolvedByDeviceID := make(map[string]int64)
	unresolvedByDeviceID := make(map[string]struct{})
	maxEventTime := int64(0)
	for _, sourceRow := range rows {
		row := sourceRow
		if row.PPKNum <= 0 {
			deviceID := strings.TrimSpace(row.DeviceID)
			if deviceID != "" {
				if resolved, ok := resolvedByDeviceID[deviceID]; ok {
					row.PPKNum = resolved
				} else if _, failed := unresolvedByDeviceID[deviceID]; !failed {
					if resolved, ok := p.resolvePPKByDeviceID(ctx, deviceID); ok {
						resolvedByDeviceID[deviceID] = resolved
						row.PPKNum = resolved
					} else {
						unresolvedByDeviceID[deviceID] = struct{}{}
					}
				}
			}
		}
		rawObjID := strings.TrimSpace(row.ObjID)
		if row.PPKNum <= 0 && rawObjID == "" {
			continue
		}
		if startGate > 0 && row.Time > 0 && row.Time < startGate {
			continue
		}
		if row.Time > maxEventTime {
			maxEventTime = row.Time
		}
		filteredRows = append(filteredRows, row)
		if row.PPKNum > 0 {
			ppkFilter[row.PPKNum] = struct{}{}
		}
		if rawObjID != "" {
			objFilter[rawObjID] = struct{}{}
		}
	}
	if len(filteredRows) == 0 {
		return nil, maxEventTime
	}

	contextByPPK, err := p.loadEventContextsByPPK(ctx, ppkFilter)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося побудувати контексти ППК для журналу подій")
	}
	contextByObject := p.loadEventContextsByObjectNum(ctx, objFilter, contextByPPK)
	dictMap := p.loadDictionaryMap(ctx)

	events := make([]models.Event, 0, len(filteredRows))
	for _, row := range filteredRows {
		ppkNum := row.PPKNum
		number := int(row.Number)
		code := strings.TrimSpace(row.Code)
		contactID := strings.TrimSpace(row.ContactID)
		sourceType := strings.TrimSpace(row.Type)
		rawObjID := strings.TrimSpace(row.ObjID)

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}
		objectID := mapCASLObjectID(strconv.FormatInt(ppkNum, 10))
		objectName := strings.TrimSpace(row.ObjName)
		if objectName == "" {
			if ppkNum > 0 {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			} else if rawObjID != "" {
				objectName = "Об'єкт #" + rawObjID
			} else {
				objectName = "Об'єкт"
			}
		}
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)
		if hasCtx {
			objectID = ctxItem.ObjectID
			objectNum = ctxItem.ObjectNum
			objectName = ctxItem.ObjectName
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		translator := map[string]string(nil)
		lineNames := map[int]string(nil)
		deviceType := ""
		if hasCtx {
			translator = ctxItem.Translator
			lineNames = ctxItem.LineNames
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := buildCASLUserActionDetails(row)
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, number, deviceType)
		}
		if details == "" {
			switch {
			case contactID != "" && code != "":
				details = fmt.Sprintf("%s (%s)", contactID, code)
			case contactID != "":
				details = contactID
			case code != "":
				details = code
			default:
				details = "CASL подія"
			}
		}
		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[number]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}

		classifierCode := code
		if classifierCode == "" {
			classifierCode = strings.TrimSpace(row.Action)
		}
		eventType := classifyCASLEventTypeWithContext(classifierCode, contactID, sourceType, details)
		eventTime := time.Now()
		if row.Time > 0 {
			eventTime = time.UnixMilli(row.Time).Local()
		}
		eventTS := row.Time
		if eventTS <= 0 {
			eventTS = eventTime.UnixMilli()
		}

		seed := code + "|" + contactID + "|" + strconv.Itoa(number) + "|" + strings.TrimSpace(row.HozUserID)
		events = append(events, models.Event{
			ID:         stableCASLEventID(strconv.FormatInt(ppkNum, 10), eventTS, seed, 0),
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: number,
			Details:    details,
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	sortEvents(events)
	return events, maxEventTime
}

func (p *CASLCloudProvider) loadEventContextsByObjectNum(ctx context.Context, objFilter map[string]struct{}, byPPK map[int64]caslEventContext) map[string]caslEventContext {
	if len(objFilter) == 0 {
		return nil
	}

	contexts := make(map[string]caslEventContext, len(objFilter))
	for _, ctxItem := range byPPK {
		if objNum := strings.TrimSpace(ctxItem.ObjectNum); objNum != "" {
			if _, need := objFilter[objNum]; need {
				contexts[objNum] = ctxItem
			}
		}
	}
	if len(contexts) == len(objFilter) {
		return contexts
	}

	records, err := p.loadObjects(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося завантажити об'єкти для object context")
		return contexts
	}
	_, _ = p.loadDevices(ctx)

	for _, record := range records {
		objNum := strings.TrimSpace(record.ObjID)
		if objNum == "" {
			continue
		}
		if _, need := objFilter[objNum]; !need {
			continue
		}
		if _, exists := contexts[objNum]; exists {
			continue
		}

		ppkNum := record.DeviceNumber.Int64()
		if ppkNum > 0 {
			if ctxItem, ok := byPPK[ppkNum]; ok {
				contexts[objNum] = ctxItem
				continue
			}
		}

		ctxItem := caslEventContext{
			ObjectID:  mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(ppkNum, 10)),
			ObjectNum: preferredCASLObjectNumber(record.ObjID, record.Name, ppkNum),
		}
		ctxItem.ObjectName = strings.TrimSpace(record.Name)
		if ctxItem.ObjectName == "" {
			ctxItem.ObjectName = "Об'єкт #" + ctxItem.ObjectNum
		}
		ctxItem.ObjectName = formatCASLJournalObjectName(ctxItem.ObjectNum, ctxItem.ObjectName)

		device, hasDevice := p.resolveDeviceForObject(record)
		if hasDevice {
			ctxItem.DeviceType = strings.TrimSpace(device.Type.String())
			ctxItem.LineNames = buildCASLLineNameIndex(device.Lines)
			ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)
		}
		contexts[objNum] = ctxItem
	}

	return contexts
}

func (p *CASLCloudProvider) mergeCachedEventsLocked(events []models.Event) int {
	if len(events) == 0 {
		return 0
	}

	seen := make(map[int]struct{}, len(p.cachedEvents)+len(events))
	for _, item := range p.cachedEvents {
		seen[item.ID] = struct{}{}
	}
	added := 0
	for _, item := range events {
		if _, exists := seen[item.ID]; exists {
			continue
		}
		seen[item.ID] = struct{}{}
		p.cachedEvents = append(p.cachedEvents, item)
		added++
	}
	if added > 0 {
		sortEvents(p.cachedEvents)
	}
	if len(p.cachedEvents) > caslMaxCachedEvents {
		p.cachedEvents = p.cachedEvents[:caslMaxCachedEvents]
	}
	return added
}

func (p *CASLCloudProvider) readGeneralTapeAsEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := p.ReadGeneralTapeObjects(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	objectNames := map[string]string{}
	if records, loadErr := p.loadObjects(ctx); loadErr == nil {
		for _, record := range records {
			objID := strings.TrimSpace(record.ObjID)
			if objID == "" {
				continue
			}
			name := strings.TrimSpace(record.Name)
			if name == "" {
				name = "Об'єкт #" + objID
			}
			objectNames[objID] = name
		}
	}
	dictMap := p.loadDictionaryMap(ctx)

	events := make([]models.Event, 0, len(rows))
	for idx, row := range rows {
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		objectID := mapCASLObjectID(rawObjID, asString(row["number"]), asString(row["device_number"]))

		objectName := strings.TrimSpace(objectNames[rawObjID])
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["obj_name"]))
		}
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name"]))
		}
		if objectName == "" {
			objectName = "Об'єкт #" + strings.TrimSpace(rawObjID)
		}
		objectName = formatCASLJournalObjectName(rawObjID, objectName)

		eventTime := parseCASLAnyTime(row["time"])
		if eventTime.IsZero() {
			eventTime = time.Now()
		}

		sourceType := strings.TrimSpace(asString(row["event_type"]))
		code := strings.TrimSpace(asString(row["code"]))
		contactID := strings.TrimSpace(asString(row["contact_id"]))
		details := strings.TrimSpace(asString(row["description"]))
		deviceType := strings.TrimSpace(asString(row["device_type"]))
		if details == "" {
			details = decodeCASLEventDescription(nil, dictMap, code, contactID, parseCASLAnyInt(row["zone"]), deviceType)
		}
		if details == "" {
			reason := strings.TrimSpace(asString(row["reasonAlarm"]))
			if reason != "" {
				if translated := decodeCASLEventDescription(nil, dictMap, reason, "", parseCASLAnyInt(row["zone"]), deviceType); translated != "" {
					details = translated
				} else {
					details = "Причина: " + reason
				}
			}
		}
		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)

		eventID := parseCASLAnyInt(row["event_id"])
		if eventID <= 0 {
			eventID = stableCASLEventID(rawObjID, eventTime.UnixMilli(), sourceType+"|"+code+"|"+contactID, idx)
		}

		events = append(events, models.Event{
			ID:         eventID,
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: parseCASLAnyInt(row["zone"]),
			Details:    details,
			UserName:   strings.TrimSpace(asString(row["user_id"])),
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	sortEvents(events)
	if len(events) > caslMaxCachedEvents {
		events = events[:caslMaxCachedEvents]
	}
	return events, nil
}

func (p *CASLCloudProvider) readFromBasketAsAlarms(ctx context.Context) ([]models.Alarm, error) {
	rows, err := p.ReadFromBasket(ctx, 0, 10_000)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	ppkFilter := make(map[int64]struct{}, len(rows))
	objFilter := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		ppkNum := int64(parseCASLAnyInt(row["ppk_num"]))
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(row["device_number"]))
		}
		if ppkNum > 0 {
			ppkFilter[ppkNum] = struct{}{}
		}
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		if rawObjID != "" {
			objFilter[rawObjID] = struct{}{}
		}
	}

	contextByPPK, ctxErr := p.loadEventContextsByPPK(ctx, ppkFilter)
	if ctxErr != nil {
		log.Debug().Err(ctxErr).Msg("CASL: не вдалося побудувати контексти ППК для кошика тривог")
	}
	contextByObject := p.loadEventContextsByObjectNum(ctx, objFilter, contextByPPK)
	dictMap := p.loadDictionaryMap(ctx)

	alarms := make([]models.Alarm, 0, len(rows))
	for _, row := range rows {
		ppkNum := int64(parseCASLAnyInt(row["ppk_num"]))
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(row["device_number"]))
		}
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		if ppkNum <= 0 && rawObjID == "" {
			continue
		}
		number := parseCASLAnyInt(row["number"])
		if number <= 0 {
			number = parseCASLAnyInt(row["zone"])
		}
		code := strings.TrimSpace(asString(row["code"]))
		contactID := strings.TrimSpace(asString(row["contact_id"]))
		sourceType := strings.TrimSpace(asString(row["event_type"]))
		if sourceType == "" {
			sourceType = strings.TrimSpace(asString(row["type"]))
		}
		description := strings.TrimSpace(asString(row["description"]))

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}
		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10))
		objectName := strings.TrimSpace(asString(row["obj_name"]))
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name_obj"]))
		}
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name"]))
		}
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)
		if hasCtx {
			objectID = ctxItem.ObjectID
			if strings.TrimSpace(ctxItem.ObjectNum) != "" {
				objectNum = strings.TrimSpace(ctxItem.ObjectNum)
			}
			if objectName == "" {
				objectName = ctxItem.ObjectName
			}
		}
		if objectName == "" {
			if objectNum != "" {
				objectName = "Об'єкт #" + objectNum
			} else {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			}
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		translator := map[string]string(nil)
		lineNames := map[int]string(nil)
		deviceType := ""
		if hasCtx {
			translator = ctxItem.Translator
			lineNames = ctxItem.LineNames
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := description
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, number, deviceType)
		}
		if details == "" {
			switch {
			case contactID != "" && code != "":
				details = fmt.Sprintf("%s (%s)", contactID, code)
			case contactID != "":
				details = contactID
			case code != "":
				details = code
			default:
				details = "CASL тривога"
			}
		}
		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[number]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}

		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)
		alarmType, include := mapEventTypeToAlarmType(eventType)
		if !include {
			continue
		}

		tsValue := parseCASLAnyTime(row["time"])
		if tsValue.IsZero() {
			tsValue = parseCASLAnyTime(row["create_date"])
		}
		if tsValue.IsZero() {
			tsValue = time.Now()
		}

		seed := code + "|" + contactID + "|" + strconv.Itoa(number)
		seedObjectID := strconv.FormatInt(ppkNum, 10)
		if strings.TrimSpace(seedObjectID) == "" || seedObjectID == "0" {
			seedObjectID = objectNum
		}
		if seedObjectID == "" {
			seedObjectID = "casl"
		}
		alarms = append(alarms, models.Alarm{
			ID:         stableCASLEventID(seedObjectID, tsValue.UnixMilli(), seed, 0),
			ObjectID:   objectID,
			ObjectName: objectName,
			Address:    strings.TrimSpace(asString(row["address"])),
			Time:       tsValue.Local(),
			Details:    details,
			Type:       alarmType,
			ZoneNumber: number,
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	sortCASLAlarms(alarms)

	return alarms, nil
}

func (p *CASLCloudProvider) GetObjectEvents(objectID string) []models.Event {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	now := time.Now()

	p.mu.RLock()
	if ts, ok := p.cachedObjectEventsAt[internalID]; ok && now.Sub(ts) <= caslObjectEventsTTL {
		events := append([]models.Event(nil), p.cachedObjectEvents[internalID]...)
		p.mu.RUnlock()
		return events
	}
	p.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	rawEvents, err := p.readEventsByID(ctx, record)
	if err != nil {
		log.Debug().Err(err).Int("objectID", internalID).Msg("CASL: не вдалося отримати події об'єкта")
		return nil
	}

	events := p.mapCASLObjectEvents(ctx, record, rawEvents)
	sortEvents(events)

	p.mu.Lock()
	p.cachedObjectEvents[internalID] = append([]models.Event(nil), events...)
	p.cachedObjectEventsAt[internalID] = now
	p.mu.Unlock()

	return events
}

func (p *CASLCloudProvider) GetAlarms() []models.Alarm {
	p.ensureRealtimeStream()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()
	if _, err := p.readEventsJournalAsEvents(ctx); err != nil {
		log.Debug().Err(err).Msg("CASL: read_events недоступний під час формування активних тривог")
	}

	alarms := p.snapshotRealtimeAlarms()
	if len(alarms) == 0 {
		rows, err := p.readGeneralTapeItemRows(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("CASL: get_general_tape_item недоступний під час формування активних тривог")
		} else if len(rows) > 0 {
			logCASLGeneralTapeItemRows(rows)
			p.updateRealtimeAlarmsFromRows(ctx, rows)
			alarms = p.snapshotRealtimeAlarms()
		}
	}
	if len(alarms) == 0 {
		return nil
	}
	sortCASLAlarms(alarms)
	return alarms
}

func (p *CASLCloudProvider) readGeneralTapeItemRows(ctx context.Context) ([]CASLObjectEvent, error) {
	records, err := p.loadObjects(ctx)
	if err != nil {
		return nil, err
	}

	objIDs := make([]string, 0, len(records))
	for _, record := range records {
		objID := strings.TrimSpace(record.ObjID)
		if objID == "" {
			continue
		}
		objIDs = append(objIDs, objID)
	}
	if len(objIDs) == 0 {
		return nil, nil
	}

	payload, err := p.ReadGeneralTapeItem(ctx, objIDs)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}

	rows := make([]CASLObjectEvent, 0, len(payload)*2)
	for objID, items := range payload {
		for _, rawRow := range items {
			rowMap := copyStringAnyMap(rawRow)
			if _, exists := rowMap["obj_id"]; !exists || strings.TrimSpace(asString(rowMap["obj_id"])) == "" {
				rowMap["obj_id"] = objID
			}
			if strings.TrimSpace(asString(rowMap["action"])) == "" {
				if dictName := strings.TrimSpace(asString(rowMap["dict_name"])); dictName != "" {
					rowMap["action"] = dictName
				}
			}
			if strings.TrimSpace(asString(rowMap["code"])) == "" {
				if dictName := strings.TrimSpace(asString(rowMap["dict_name"])); dictName != "" {
					rowMap["code"] = dictName
				}
			}

			rowType := strings.TrimSpace(asString(rowMap["type"]))
			if rowType == "" {
				rowType = strings.TrimSpace(asString(rowMap["event_type"]))
			}
			if rowType == "" {
				rowType = "ppk_event"
			}

			row, ok := mapCASLRealtimeRow(rowMap, rowType)
			if !ok {
				continue
			}
			if strings.TrimSpace(row.ObjID) == "" {
				row.ObjID = strings.TrimSpace(objID)
			}
			rows = append(rows, row)
		}
	}

	return rows, nil
}

func logCASLGeneralTapeItemRows(rows []CASLObjectEvent) {
	log.Debug().
		Int("rows", len(rows)).
		Msg("CASL get_general_tape_item: отримано події")

	const maxRowsToLog = 200
	for idx, row := range rows {
		if idx >= maxRowsToLog {
			log.Debug().
				Int("logged_rows", maxRowsToLog).
				Int("total_rows", len(rows)).
				Msg("CASL get_general_tape_item: лог скорочено")
			return
		}

		log.Debug().
			Int("idx", idx).
			Str("obj_id", strings.TrimSpace(row.ObjID)).
			Int64("ppk_num", row.PPKNum).
			Str("code", strings.TrimSpace(row.Code)).
			Str("contact_id", strings.TrimSpace(row.ContactID)).
			Int64("number", row.Number).
			Str("type", strings.TrimSpace(row.Type)).
			Int64("time", row.Time).
			Msg("CASL get_general_tape_item row")
	}
}

func (p *CASLCloudProvider) ProcessAlarm(id string, user string, note string) {
	log.Warn().Str("alarmID", id).Str("user", user).Msg("CASL: ProcessAlarm не підтримується API інтеграцією")
}

func (p *CASLCloudProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return "н/д", "н/д", time.Time{}, time.Time{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return "н/д", "н/д", time.Time{}, time.Time{}
	}

	state, stateErr := p.readDeviceState(ctx, record)
	stats, statsErr := p.readStatsAlarms(ctx, record)

	signalParts := []string{"н/д"}
	testParts := make([]string, 0, 4)

	if stateErr == nil {
		if state.LastPingDate.Int64() > 0 {
			lastMsg = time.UnixMilli(state.LastPingDate.Int64()).Local()
			lastTest = lastMsg
		}
	}

	if statsErr == nil {
		testParts = append(testParts,
			fmt.Sprintf("freq=%d", stats.ResponseFrequencies.Int64()),
			fmt.Sprintf("quality=%d", stats.CommunicQuality.Int64()),
			fmt.Sprintf("alarms=%d", stats.CustomWins.Int64()),
			fmt.Sprintf("power=%d", stats.PowerFailure.Int64()),
		)
	}

	if stateErr != nil && statsErr != nil {
		log.Debug().Err(stateErr).Msg("CASL: не вдалося отримати read_device_state")
		log.Debug().Err(statsErr).Msg("CASL: не вдалося отримати stats_alarms")
		testParts = append(testParts, "н/д")
	}

	return strings.Join(signalParts, "; "), strings.Join(testParts, "; "), lastTest, lastMsg
}

func (p *CASLCloudProvider) GetTestMessages(objectID string) []models.TestMessage {
	events := p.GetObjectEvents(objectID)
	if len(events) == 0 {
		return nil
	}

	messages := make([]models.TestMessage, 0, 32)
	for _, event := range events {
		if event.Type != models.EventTest && !strings.Contains(strings.ToUpper(event.Details), "TEST") {
			continue
		}
		messages = append(messages, models.TestMessage{Time: event.Time, Info: event.GetTypeDisplay(), Details: event.Details})
		if len(messages) >= 200 {
			break
		}
	}
	return messages
}

// GetLatestEventID повертає компактний курсор змін для scheduler.
func (p *CASLCloudProvider) GetLatestEventID() (int64, error) {
	p.ensureRealtimeStream()

	p.mu.RLock()
	revision := p.eventsRevision
	p.mu.RUnlock()
	return revision, nil
}

func (p *CASLCloudProvider) readEventsByID(ctx context.Context, record caslGrdObject) ([]caslObjectEvent, error) {
	if strings.TrimSpace(record.ObjID) == "" || record.DeviceID.Int64() <= 0 {
		return nil, nil
	}

	end := time.Now().UnixMilli()
	start := end - caslObjectEventsSpan.Milliseconds()

	payload := map[string]any{
		"type":             "read_events_by_id",
		"isFullEventsInfo": false,
		"time_start":       start,
		"time_end":         end,
		"time_request":     end,
		"objIds":           []string{strings.TrimSpace(record.ObjID)},
		"deviceIds":        []string{strconv.FormatInt(record.DeviceID.Int64(), 10)},
		"deviceNumbers":    []int64{record.DeviceNumber.Int64()},
	}

	var resp caslReadEventsByIDResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}

	if len(resp.Data) > 0 {
		return append([]caslObjectEvent(nil), resp.Data...), nil
	}
	return append([]caslObjectEvent(nil), resp.Events...), nil
}

func (p *CASLCloudProvider) readDeviceState(ctx context.Context, record caslGrdObject) (caslDeviceState, error) {
	deviceID := record.DeviceID.Int64()
	if deviceID <= 0 {
		return caslDeviceState{}, errors.New("casl: empty device_id")
	}

	payload := map[string]any{"type": "read_device_state", "device_id": strconv.FormatInt(deviceID, 10)}

	var resp caslReadDeviceStateResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return caslDeviceState{}, err
	}
	return resp.State, nil
}

func (p *CASLCloudProvider) readStatsAlarms(ctx context.Context, record caslGrdObject) (caslStatsAlarmsData, error) {
	deviceID := record.DeviceID.Int64()
	if deviceID <= 0 || strings.TrimSpace(record.ObjID) == "" {
		return caslStatsAlarmsData{}, errors.New("casl: empty object/device identifiers")
	}

	end := time.Now().UnixMilli()
	start := end - caslStatsSpan.Milliseconds()

	payload := map[string]any{
		"type":      "get_statistic",
		"name":      "stats_alarms",
		"deviceId":  strconv.FormatInt(deviceID, 10),
		"objectId":  strings.TrimSpace(record.ObjID),
		"startTime": start,
		"endTime":   end,
	}

	var resp caslGetStatisticResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return caslStatsAlarmsData{}, err
	}
	return resp.Data, nil
}

func (p *CASLCloudProvider) mapCASLObjectEvents(ctx context.Context, record caslGrdObject, raw []caslObjectEvent) []models.Event {
	if len(raw) == 0 {
		return nil
	}

	result := make([]models.Event, 0, len(raw))
	objectID := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))
	objectNum := preferredCASLObjectNumber(record.ObjID, record.Name, record.DeviceNumber.Int64())
	objectName := strings.TrimSpace(record.Name)
	if objectName == "" {
		objectName = "Об'єкт #" + objectNum
	}
	objectName = formatCASLJournalObjectName(objectNum, objectName)

	var (
		translator map[string]string
		dictMap    map[string]string
		lineNames  map[int]string
		deviceType string
	)

	device, hasDevice := p.resolveDeviceForObject(record)
	if hasDevice {
		deviceType = strings.TrimSpace(device.Type.String())
		translator = p.loadTranslatorMap(ctx, deviceType)
		lineNames = buildCASLLineNameIndex(device.Lines)
	}
	dictMap = p.loadDictionaryMap(ctx)

	for idx, item := range raw {
		ts := item.Time.Int64()
		eventTime := time.Now()
		if ts > 0 {
			eventTime = time.UnixMilli(ts).Local()
		}

		code := item.Code.String()
		if code == "" {
			code = "UNKNOWN"
		}

		zoneNumber := int(item.Number.Int64())
		contactID := strings.TrimSpace(item.ContactID.String())
		sourceType := strings.TrimSpace(item.Type)

		details := decodeCASLEventDescription(translator, dictMap, code, contactID, zoneNumber, deviceType)
		if details == "" {
			switch {
			case contactID != "" && code != "":
				details = fmt.Sprintf("%s (%s)", contactID, code)
			case contactID != "":
				details = contactID
			default:
				details = code
			}
		}
		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)

		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[zoneNumber]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}
		if sourceType != "" && !strings.EqualFold(sourceType, "ppk_event") {
			details += " | src=" + sourceType
		}

		result = append(result, models.Event{
			ID:         stableCASLEventID(record.ObjID, ts, code, idx),
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: zoneNumber,
			Details:    details,
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	return result
}

func (p *CASLCloudProvider) loadEventContextsByPPK(ctx context.Context, ppkFilter map[int64]struct{}) (map[int64]caslEventContext, error) {
	records, err := p.loadObjects(ctx)
	if err != nil {
		return nil, err
	}
	_, devicesErr := p.loadDevices(ctx)
	if devicesErr != nil {
		log.Debug().Err(devicesErr).Msg("CASL: read_device недоступний для event context")
	}

	contexts := make(map[int64]caslEventContext, len(records))
	for _, record := range records {
		ppkNum := record.DeviceNumber.Int64()
		if ppkNum <= 0 {
			continue
		}
		if len(ppkFilter) > 0 {
			if _, ok := ppkFilter[ppkNum]; !ok {
				continue
			}
		}

		objectNum := preferredCASLObjectNumber(record.ObjID, record.Name, ppkNum)
		ctxItem := caslEventContext{
			ObjectID:  mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(ppkNum, 10)),
			ObjectNum: objectNum,
		}
		ctxItem.ObjectName = strings.TrimSpace(record.Name)
		if ctxItem.ObjectName == "" {
			ctxItem.ObjectName = "Об'єкт #" + objectNum
		}
		ctxItem.ObjectName = formatCASLJournalObjectName(ctxItem.ObjectNum, ctxItem.ObjectName)

		device, hasDevice := p.resolveDeviceForObject(record)
		if hasDevice {
			ctxItem.DeviceType = strings.TrimSpace(device.Type.String())
			ctxItem.LineNames = buildCASLLineNameIndex(device.Lines)
			ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)
		}
		contexts[ppkNum] = ctxItem
	}

	return contexts, nil
}
