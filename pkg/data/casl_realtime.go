package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/websocket"
)

func extractCASLRealtimeConnID(raw []byte) string {
	payload, ok := decodeCASLRealtimePayload(raw)
	if ok {
		if id := extractCASLConnIDFromAny(payload); id != "" {
			return id
		}
	}

	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	if isLikelyCASLConnID(text) {
		return text
	}

	lower := strings.ToLower(text)
	candidates := []string{"conn_id", "connid", "sid", "connection_id"}
	for _, key := range candidates {
		idx := strings.Index(lower, key)
		if idx < 0 {
			continue
		}
		tail := text[idx+len(key):]
		tail = strings.TrimLeft(tail, " \t\r\n:=\">'`")
		if tail == "" {
			continue
		}

		end := len(tail)
		for i, r := range tail {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				continue
			}
			end = i
			break
		}
		id := strings.TrimSpace(tail[:end])
		if isLikelyCASLConnID(id) {
			return id
		}
	}

	return ""
}

func extractCASLRealtimeRows(raw []byte) []CASLObjectEvent {
	payload, ok := decodeCASLRealtimePayload(raw)
	if !ok {
		return nil
	}
	rows := make([]CASLObjectEvent, 0, 4)
	collectCASLRealtimeRows(payload, "", &rows)
	return rows
}

func decodeCASLRealtimePayload(raw []byte) (any, bool) {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 {
		return nil, false
	}
	if body[0] != '{' && body[0] != '[' {
		idx := bytes.IndexAny(body, "{[")
		if idx < 0 {
			return nil, false
		}
		body = body[idx:]
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func collectCASLRealtimeRows(value any, fallbackType string, rows *[]CASLObjectEvent) {
	switch typed := value.(type) {
	case map[string]any:
		nextType := fallbackType
		if tag := strings.TrimSpace(asString(typed["tag"])); tag != "" {
			nextType = tag
		}
		if eventName := strings.TrimSpace(asString(typed["event"])); eventName != "" {
			nextType = eventName
		}
		if row, ok := mapCASLRealtimeRow(typed, nextType); ok {
			*rows = append(*rows, row)
		}
		for _, nested := range typed {
			collectCASLRealtimeRows(nested, nextType, rows)
		}
	case []any:
		for _, nested := range typed {
			collectCASLRealtimeRows(nested, fallbackType, rows)
		}
	}
}

func mapCASLRealtimeRow(source map[string]any, fallbackType string) (CASLObjectEvent, bool) {
	deviceID := strings.TrimSpace(asString(source["device_id"]))
	if deviceID == "" {
		deviceID = strings.TrimSpace(asString(source["deviceId"]))
	}
	rawObjID := strings.TrimSpace(asString(source["obj_id"]))
	if rawObjID == "" {
		rawObjID = strings.TrimSpace(asString(source["object_id"]))
	}

	ppkNum := int64(parseCASLAnyInt(source["ppk_num"]))
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["ppkNum"]))
	}
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["ppk"]))
	}
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["device_number"]))
	}
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["device_num"]))
	}
	if ppkNum <= 0 && deviceID == "" && rawObjID == "" {
		return CASLObjectEvent{}, false
	}

	number := int64(parseCASLAnyInt(source["number"]))
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["zone"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["line"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["line_number"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["group"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["group_number"]))
	}

	code := strings.TrimSpace(asString(source["code"]))
	if code == "" {
		code = strings.TrimSpace(asString(source["event_code"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["ppk_action_type"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["user_action_type"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["mgr_action_type"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["action"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["subtype"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["msg"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["type_event"]))
	}
	contactID := strings.TrimSpace(asString(source["contact_id"]))
	if contactID == "" {
		contactID = strings.TrimSpace(asString(source["contactId"]))
	}
	rowType := strings.TrimSpace(asString(source["type"]))
	if rowType == "" {
		rowType = fallbackType
	}
	if rowType == "" {
		rowType = strings.TrimSpace(asString(source["user_action_type"]))
	}
	if rowType == "" {
		rowType = strings.TrimSpace(asString(source["module"]))
	}
	if code == "" && contactID == "" {
		code = strings.TrimSpace(asString(source["status"]))
	}
	if code == "" && contactID == "" {
		code = rowType
	}

	ts := parseCASLAnyTime(source["time"])
	if ts.IsZero() {
		ts = parseCASLAnyTime(source["timestamp"])
	}
	if ts.IsZero() {
		ts = parseCASLAnyTime(source["ts"])
	}
	if ts.IsZero() {
		ts = parseCASLAnyTime(source["create_date"])
	}

	row := CASLObjectEvent{
		PPKNum:    ppkNum,
		DeviceID:  deviceID,
		ObjID:     rawObjID,
		ObjName:   strings.TrimSpace(asString(source["obj_name"])),
		ObjAddr:   strings.TrimSpace(asString(source["obj_address"])),
		Action:    strings.TrimSpace(asString(source["action"])),
		AlarmType: strings.TrimSpace(asString(source["alarm_type"])),
		MgrID:     strings.TrimSpace(asString(source["mgr_id"])),
		UserID:    strings.TrimSpace(asString(source["user_id"])),
		UserFIO:   strings.TrimSpace(asString(source["user_fio"])),
		Time:      0,
		Code:      code,
		Type:      rowType,
		Number:    number,
		ContactID: contactID,
		HozUserID: strings.TrimSpace(asString(source["hoz_user_id"])),
	}
	if row.HozUserID == "" {
		row.HozUserID = strings.TrimSpace(asString(source["user_id"]))
	}
	if !ts.IsZero() {
		row.Time = ts.UnixMilli()
	}
	if row.Code == "" && row.ContactID == "" && row.Number == 0 && row.Action == "" {
		return CASLObjectEvent{}, false
	}
	return row, true
}

func extractCASLConnIDFromAny(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		rowType := strings.ToLower(strings.TrimSpace(asString(typed["type"])))
		if rowType == "conn_id" || rowType == "get_id" {
			if id := strings.TrimSpace(asString(typed["id"])); isLikelyCASLConnID(id) {
				return id
			}
		}
		candidates := []string{"conn_id", "connId", "sid", "connection_id"}
		for _, key := range candidates {
			if id := strings.TrimSpace(asString(typed[key])); id != "" {
				return id
			}
		}
		for _, nested := range typed {
			if id := extractCASLConnIDFromAny(nested); id != "" {
				return id
			}
		}
	case []any:
		for _, nested := range typed {
			if id := extractCASLConnIDFromAny(nested); id != "" {
				return id
			}
		}
	}
	return ""
}

func randomCASLConnID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

func extractCASLConnIDFromWSURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	query := parsed.Query()
	for _, key := range []string{"conn_id", "connId", "sid", "connection_id", "id"} {
		value := strings.TrimSpace(query.Get(key))
		if isLikelyCASLConnID(value) {
			return value
		}
	}
	return ""
}

func sendCASLRealtimeGetID(conn *websocket.Conn, userID string) error {
	payload := map[string]any{
		"type": "get_id",
	}
	if strings.TrimSpace(userID) != "" {
		payload["user_id"] = strings.TrimSpace(userID)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return websocket.Message.Send(conn, body)
}

func isLikelyCASLConnID(value string) bool {
	id := strings.TrimSpace(value)
	if len(id) < 8 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func (p *CASLCloudProvider) ensureRealtimeStream() {
	p.mu.RLock()
	wsURL := strings.TrimSpace(p.wsURL)
	p.mu.RUnlock()
	if wsURL == "" {
		ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
		_, _ = p.ensureToken(ctx)
		cancel()
		p.mu.RLock()
		wsURL = strings.TrimSpace(p.wsURL)
		p.mu.RUnlock()
	}
	if wsURL == "" {
		return
	}

	p.realtimeMu.Lock()
	if p.realtimeRunning {
		p.realtimeMu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.realtimeCancel = cancel
	p.realtimeRunning = true
	p.realtimeMu.Unlock()

	go p.runRealtimeLoop(ctx)
}

func (p *CASLCloudProvider) runRealtimeLoop(ctx context.Context) {
	defer func() {
		p.realtimeMu.Lock()
		if p.realtimeCancel != nil {
			p.realtimeCancel = nil
		}
		p.realtimeRunning = false
		p.realtimeSubscribed = false
		p.realtimeMu.Unlock()
	}()

	backoff := time.Second
	lastLogAt := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := p.runRealtimeSession(ctx); err != nil && !errors.Is(err, context.Canceled) {
			now := time.Now()
			if lastLogAt.IsZero() || now.Sub(lastLogAt) >= 30*time.Second {
				log.Debug().Err(err).Msg("CASL realtime stream: reconnect")
				lastLogAt = now
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > caslRealtimeBackoff {
			backoff = caslRealtimeBackoff
		}
	}
}

func (p *CASLCloudProvider) runRealtimeSession(ctx context.Context) error {
	p.realtimeMu.Lock()
	p.realtimeSubscribed = false
	p.realtimeMu.Unlock()

	ensureCtx, ensureCancel := context.WithTimeout(ctx, caslHTTPTimeout)
	if _, err := p.ensureToken(ensureCtx); err != nil {
		ensureCancel()
		return err
	}
	ensureCancel()

	p.mu.RLock()
	wsURL := strings.TrimSpace(p.wsURL)
	userID := strings.TrimSpace(p.userID)
	p.mu.RUnlock()
	if wsURL == "" {
		return errors.New("casl realtime: empty ws_url")
	}

	origin := p.baseURL
	if parsed, err := url.Parse(wsURL); err == nil && parsed.Scheme == "wss" {
		origin = strings.Replace(origin, "http://", "https://", 1)
	}

	cfg, err := websocket.NewConfig(wsURL, origin)
	if err != nil {
		return fmt.Errorf("casl realtime config: %w", err)
	}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return fmt.Errorf("casl realtime dial: %w", err)
	}
	defer conn.Close()
	if getIDErr := sendCASLRealtimeGetID(conn, userID); getIDErr != nil {
		log.Debug().Err(getIDErr).Msg("CASL realtime: get_id send failed")
	} else {
		log.Debug().Msg("CASL realtime: get_id sent")
	}

	rawCh := make(chan []byte, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(rawCh)
		for {
			var raw []byte
			if recvErr := websocket.Message.Receive(conn, &raw); recvErr != nil {
				select {
				case errCh <- recvErr:
				default:
				}
				return
			}
			if len(raw) == 0 {
				continue
			}
			msg := append([]byte(nil), raw...)
			select {
			case rawCh <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	connID := extractCASLConnIDFromWSURL(wsURL)
	if connID != "" {
		log.Debug().Str("conn_id", connID).Msg("CASL realtime conn_id extracted from ws_url")
	}
	subscribed := false
	subscribeTicker := time.NewTicker(1500 * time.Millisecond)
	defer subscribeTicker.Stop()
	lastSubscribeErrAt := time.Time{}
	lastSubscribeErrText := ""

	subscribeOnce := func(candidate string) {
		if subscribed {
			return
		}
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}

		subCtx, subCancel := context.WithTimeout(ctx, caslHTTPTimeout)
		subErr := p.subscribeRealtimeTags(subCtx, candidate)
		subCancel()
		if subErr != nil {
			errText := strings.TrimSpace(subErr.Error())
			if errText == "" {
				errText = "unknown"
			}
			now := time.Now()
			if errText != lastSubscribeErrText || lastSubscribeErrAt.IsZero() || now.Sub(lastSubscribeErrAt) >= 15*time.Second {
				log.Debug().Err(subErr).Str("conn_id", candidate).Msg("CASL realtime subscribe failed")
				lastSubscribeErrText = errText
				lastSubscribeErrAt = now
			}
			return
		}

		subscribed = true
		p.realtimeMu.Lock()
		p.realtimeSubscribed = true
		p.realtimeMu.Unlock()
		log.Debug().Str("conn_id", candidate).Msg("CASL realtime subscribed")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recvErr := <-errCh:
			if recvErr == nil {
				return io.EOF
			}
			return recvErr
		case raw, ok := <-rawCh:
			if !ok {
				return io.EOF
			}

			if connID == "" {
				if extracted := extractCASLRealtimeConnID(raw); extracted != "" {
					connID = extracted
					log.Debug().Str("conn_id", connID).Msg("CASL realtime conn_id extracted")
				}
			}
			if !subscribed && connID != "" {
				subscribeOnce(connID)
			}

			rows := extractCASLRealtimeRows(raw)
			if len(rows) > 0 {
				if appendErr := p.appendRealtimeRows(ctx, rows); appendErr != nil {
					log.Debug().Err(appendErr).Msg("CASL realtime append failed")
				}
			}
		case <-subscribeTicker.C:
			if subscribed {
				continue
			}
			if connID == "" {
				continue
			}
			subscribeOnce(connID)
		}
	}
}

func (p *CASLCloudProvider) subscribeRealtimeTags(ctx context.Context, connID string) error {
	type tagSpec struct {
		name     string
		required bool
	}

	tags := []tagSpec{
		{name: "ppk_in", required: true},
		{name: "user_action", required: true},
		{name: "ppk_service", required: false},
		{name: "ppk_out", required: false},
		{name: "system_event", required: false},
		{name: "system_action", required: false},
		{name: "m3_in", required: false},
		{name: "mob_user_action", required: false},
		{name: "system_info", required: false},
		{name: "storage_change", required: false},
		{name: "notif", required: false},
		{name: "chat_action", required: false},
		{name: "rtsp_action", required: false},
		{name: "ping", required: false},
	}

	requiredSubscribed := 0
	totalSubscribed := 0
	for _, tag := range tags {
		if err := p.Subscribe(ctx, connID, tag.name); err != nil {
			if isCASLUnknownTagErr(err) {
				if tag.required {
					return fmt.Errorf("casl subscribe tag=%s: %w", tag.name, err)
				}
				continue
			}
			if !tag.required {
				log.Debug().Err(err).Str("tag", tag.name).Msg("CASL realtime subscribe: optional tag subscribe failed")
				continue
			}
			return fmt.Errorf("casl subscribe tag=%s: %w", tag.name, err)
		}
		totalSubscribed++
		if tag.required {
			requiredSubscribed++
		}
	}

	if totalSubscribed == 0 {
		return errors.New("casl subscribe: no tags were subscribed")
	}
	if requiredSubscribed == 0 {
		log.Debug().Msg("CASL realtime subscribe: required tags are unavailable, running on optional tags only")
	}

	return nil
}

func (p *CASLCloudProvider) appendRealtimeRows(ctx context.Context, rows []CASLObjectEvent) error {
	p.mu.RLock()
	startGate := p.eventsStartAtMs
	p.mu.RUnlock()

	events, maxEventTime := p.mapCASLRowsToEvents(ctx, rows, startGate)
	if len(events) == 0 {
		p.updateRealtimeAlarmsFromRows(ctx, rows)
		return nil
	}

	p.mu.Lock()
	added := p.mergeCachedEventsLocked(events)
	if maxEventTime > p.eventsCursorMs {
		p.eventsCursorMs = maxEventTime
	}
	if added > 0 {
		p.eventsRevision++
	}
	p.mu.Unlock()

	p.updateRealtimeAlarmsFromRows(ctx, rows)
	return nil
}

func (p *CASLCloudProvider) snapshotRealtimeAlarms() []models.Alarm {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.realtimeAlarmByObjID) == 0 {
		return nil
	}

	alarms := make([]models.Alarm, 0, len(p.realtimeAlarmByObjID))
	for _, alarm := range p.realtimeAlarmByObjID {
		alarms = append(alarms, alarm)
	}
	return alarms
}

func mergeCASLAlarms(primary []models.Alarm, secondary []models.Alarm) []models.Alarm {
	if len(secondary) == 0 {
		return append([]models.Alarm(nil), primary...)
	}

	out := append([]models.Alarm(nil), primary...)
	seen := make(map[string]int, len(out))
	for i := range out {
		seen[caslAlarmMergeKey(out[i])] = i
	}

	for _, alarm := range secondary {
		key := caslAlarmMergeKey(alarm)
		if idx, exists := seen[key]; exists {
			if alarm.Time.After(out[idx].Time) {
				out[idx] = alarm
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, alarm)
	}

	return out
}

func caslAlarmMergeKey(alarm models.Alarm) string {
	return strconv.Itoa(alarm.ObjectID) + "|" + strconv.Itoa(alarm.ZoneNumber) + "|" + string(alarm.Type)
}

func sortCASLAlarms(alarms []models.Alarm) {
	sort.SliceStable(alarms, func(i, j int) bool {
		left := alarms[i].Time
		right := alarms[j].Time
		if left.Equal(right) {
			return alarms[i].ID > alarms[j].ID
		}
		return left.After(right)
	})
}

func canonicalCASLRealtimeObjectKey(rawObjID string, objectNum string, objectID int) string {
	if number := strings.TrimSpace(objectNum); number != "" {
		return number
	}
	if value := strings.TrimSpace(rawObjID); value != "" {
		return value
	}
	if objectID > 0 {
		return strconv.Itoa(objectID)
	}
	return ""
}

func canonicalCASLRealtimeAlarmKey(objectKey string, zoneNumber int) string {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return ""
	}
	if zoneNumber <= 0 {
		return key + "|z0"
	}
	return key + "|z" + strconv.Itoa(zoneNumber)
}

func deleteRealtimeAlarmsByObjectKey(cache map[string]models.Alarm, objectKey string) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return
	}
	prefix := objectKey + "|"
	for key := range cache {
		if strings.HasPrefix(key, prefix) || key == objectKey {
			delete(cache, key)
		}
	}
}

func isCASLEventAlarmCandidate(eventType models.EventType) bool {
	switch eventType {
	case models.EventFire,
		models.EventBurglary,
		models.EventPanic,
		models.EventMedical,
		models.EventGas,
		models.EventTamper,
		models.EventFault,
		models.EventPowerFail,
		models.EventBatteryLow,
		models.EventOffline:
		return true
	default:
		return false
	}
}

func (p *CASLCloudProvider) updateRealtimeAlarmsFromRows(ctx context.Context, rows []CASLObjectEvent) {
	if len(rows) == 0 {
		return
	}

	ppkFilter := make(map[int64]struct{}, len(rows))
	objFilter := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.PPKNum > 0 {
			ppkFilter[row.PPKNum] = struct{}{}
		}
		if objID := strings.TrimSpace(row.ObjID); objID != "" {
			objFilter[objID] = struct{}{}
		}
	}

	contextByPPK, err := p.loadEventContextsByPPK(ctx, ppkFilter)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося завантажити контексти ППК для realtime alarm cache")
	}
	contextByObject := p.loadEventContextsByObjectNum(ctx, objFilter, contextByPPK)
	dictMap := p.loadDictionaryMap(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, row := range rows {
		action := strings.ToUpper(strings.TrimSpace(row.Action))
		if action == "" {
			action = strings.ToUpper(strings.TrimSpace(row.Code))
		}
		if action == "" {
			continue
		}

		rawObjID := strings.TrimSpace(row.ObjID)
		ppkNum := row.PPKNum

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}

		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10))
		objectName := strings.TrimSpace(row.ObjName)
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)

		if hasCtx {
			objectID = ctxItem.ObjectID
			if strings.TrimSpace(ctxItem.ObjectNum) != "" {
				objectNum = strings.TrimSpace(ctxItem.ObjectNum)
			}
			if objectName == "" {
				objectName = strings.TrimSpace(ctxItem.ObjectName)
			}
		}

		if objectName == "" {
			if objectNum != "" {
				objectName = "Об'єкт #" + objectNum
			} else if ppkNum > 0 {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			} else {
				objectName = "Об'єкт"
			}
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		objectKey := canonicalCASLRealtimeObjectKey(rawObjID, objectNum, objectID)
		if objectKey == "" {
			continue
		}
		zoneNumber := int(row.Number)
		cacheKey := canonicalCASLRealtimeAlarmKey(objectKey, zoneNumber)

		switch action {
		case "GRD_OBJ_MGR_CANCEL", "GRD_OBJ_FINISH":
			deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objectKey)
			continue
		}

		deviceType := ""
		if hasCtx {
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := buildCASLUserActionDetails(row)
		if details == "" {
			details = decodeCASLEventDescription(nil, dictMap, row.Code, row.ContactID, int(row.Number), deviceType)
		}
		if details == "" {
			details = strings.TrimSpace(row.Action)
		}
		if details == "" {
			details = strings.TrimSpace(row.Code)
		}
		if details == "" {
			details = "CASL подія"
		}

		classifierCode := strings.TrimSpace(row.Code)
		if classifierCode == "" {
			classifierCode = action
		}
		eventType := classifyCASLEventTypeWithContext(classifierCode, strings.TrimSpace(row.ContactID), strings.TrimSpace(row.Type), details)
		alarmType, include := mapEventTypeToAlarmType(eventType)
		if !include && action == "GRD_OBJ_NOTIF" {
			alarmType = models.AlarmNotification
			include = true
		}

		alarmTime := time.Now()
		if row.Time > 0 {
			alarmTime = time.UnixMilli(row.Time).Local()
		}
		objectNumint, err := strconv.Atoi(objectNum)
		if err != nil {
			log.Error().Err(err).Msg("CASL: не вдалося перетворити objectNum на int")
		}

		if action == "GRD_OBJ_NOTIF" {
			seed := action + "|" + strings.TrimSpace(row.AlarmType) + "|" + strconv.Itoa(zoneNumber)
			p.realtimeAlarmByObjID[cacheKey] = models.Alarm{
				ID:         stableCASLEventID(cacheKey, alarmTime.UnixMilli(), seed, 0),
				ObjectID:   objectNumint,
				ObjectName: objectName,
				Address:    strings.TrimSpace(row.ObjAddr),
				Time:       alarmTime,
				Details:    details,
				Type:       alarmType,
				ZoneNumber: int(row.Number),
				SC1:        mapCASLEventSC1(eventType),
			}
			continue
		}

		if action == "GRD_OBJ_PICK" || action == "GRD_OBJ_ASS_MGR" {

			for key, existing := range p.realtimeAlarmByObjID {
				if !strings.HasPrefix(key, objectKey+"|") {
					continue
				}
				existing.ObjectID = objectID
				existing.ObjectName = objectName
				if strings.TrimSpace(row.ObjAddr) != "" {
					existing.Address = strings.TrimSpace(row.ObjAddr)
				}
				existing.Time = alarmTime
				if details != "" {
					existing.Details = details
				}
				p.realtimeAlarmByObjID[key] = existing
			}
			continue
		}

		if eventType == models.EventRestore || eventType == models.EventPowerOK || eventType == models.EventOnline {
			if zoneNumber > 0 {
				delete(p.realtimeAlarmByObjID, cacheKey)
			} else {
				deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objectKey)
			}
			continue
		}

		if !isCASLEventAlarmCandidate(eventType) || !include {
			continue
		}

		seed := strings.TrimSpace(row.Code) + "|" + strings.TrimSpace(row.ContactID) + "|" + strconv.Itoa(zoneNumber)
		p.realtimeAlarmByObjID[cacheKey] = models.Alarm{
			ID:         stableCASLEventID(cacheKey, alarmTime.UnixMilli(), seed, 0),
			ObjectID:   objectNumint,
			ObjectName: objectName,
			Address:    strings.TrimSpace(row.ObjAddr),
			Time:       alarmTime,
			Details:    details,
			Type:       alarmType,
			ZoneNumber: zoneNumber,
			SC1:        mapCASLEventSC1(eventType),
		}
	}
}

func (p *CASLCloudProvider) restartRealtimeStream() {
	p.realtimeMu.Lock()
	cancel := p.realtimeCancel
	p.realtimeCancel = nil
	p.realtimeRunning = false
	p.realtimeSubscribed = false
	p.realtimeMu.Unlock()

	if cancel != nil {
		cancel()
	}
	p.ensureRealtimeStream()
}
