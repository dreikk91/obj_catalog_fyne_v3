package data

import (
	"bytes"
	"context"
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

	return extractCASLConnIDFromTextEnvelope(string(raw))
}

func extractCASLRealtimeRows(raw []byte) []CASLObjectEvent {
	payload, ok := decodeCASLRealtimePayload(raw)
	if !ok {
		return nil
	}
	return extractCASLRealtimeRowsFromValue(payload)
}

func extractCASLRealtimeRowsFromValue(payload any) []CASLObjectEvent {
	rows := make([]CASLObjectEvent, 0, 4)
	collectCASLRealtimeRows(payload, "", &rows)
	return rows
}

func decodeCASLRealtimePayload(raw []byte) (any, bool) {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 {
		return nil, false
	}
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, false
	}
	if body[0] != '{' && body[0] != '[' {
		return nil, false
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func extractCASLConnIDFromTextEnvelope(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if isLikelyCASLConnID(text) {
		return text
	}

	text = strings.TrimLeft(text, "{([ \t\r\n")
	text = strings.TrimLeft(text, "\"'")
	lower := strings.ToLower(text)

	candidates := []string{"conn_id", "connid", "sid", "connection_id"}
	for _, key := range candidates {
		if !strings.HasPrefix(lower, key) {
			continue
		}
		tail := text[len(key):]
		tail = strings.TrimLeft(tail, " \t\r\n:=\">'`")
		if tail == "" {
			return ""
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
		return ""
	}

	return ""
}

func collectCASLRealtimeRows(value any, fallbackType string, rows *[]CASLObjectEvent) {
	switch typed := value.(type) {
	case map[string]any:
		nextType := caslRealtimeFallbackType(typed, fallbackType)
		if row, ok := mapCASLRealtimeRow(typed, nextType); ok {
			*rows = append(*rows, row)
		}
		for _, nested := range caslRealtimeNestedPayloads(typed) {
			collectCASLRealtimeRows(nested, nextType, rows)
		}
	case []any:
		for _, nested := range typed {
			collectCASLRealtimeRows(nested, fallbackType, rows)
		}
	}
}

func caslRealtimeFallbackType(source map[string]any, fallbackType string) string {
	nextType := fallbackType
	if tag := strings.TrimSpace(asString(source["tag"])); tag != "" {
		nextType = tag
	}
	if eventName := strings.TrimSpace(asString(source["event"])); eventName != "" {
		nextType = eventName
	}
	return nextType
}

func caslRealtimeNestedPayloads(source map[string]any) []any {
	keys := []string{
		"data",
		"rows",
		"events",
		"payload",
		"message",
		"messages",
		"result",
		"body",
		"items",
	}

	nested := make([]any, 0, len(keys))
	for _, key := range keys {
		value, ok := source[key]
		if !ok || value == nil {
			continue
		}
		nested = append(nested, value)
	}
	return nested
}

func mapCASLRealtimeRow(source map[string]any, fallbackType string) (CASLObjectEvent, bool) {
	deviceID := firstCASLTextValue(source["device_id"], source["deviceId"])
	rawObjID := firstCASLTextValue(source["obj_id"], source["object_id"])
	userActionType := firstCASLTextValue(source["user_action_type"])
	mgrActionType := firstCASLTextValue(source["mgr_action_type"])
	ppkActionType := firstCASLTextValue(source["ppk_action_type"])

	ppkNumValue, _ := firstCASLIntValue(
		source["ppk_num"],
		source["ppkNum"],
		source["ppk"],
		source["device_number"],
		source["device_num"],
	)
	ppkNum := int64(ppkNumValue)

	numberValue, _ := firstCASLIntValue(
		source["number"],
		source["zone"],
		source["line"],
		source["line_number"],
		source["group"],
		source["group_number"],
	)
	number := int64(numberValue)

	action := firstCASLTextValue(
		source["action"],
		mgrActionType,
		ppkActionType,
		userActionType,
		source["subtype"],
		source["msg"],
		source["type_event"],
	)
	code := firstCASLTextValue(
		source["code"],
		source["event_code"],
		ppkActionType,
		userActionType,
		mgrActionType,
		source["action"],
		source["subtype"],
		source["msg"],
		source["type_event"],
	)
	contactID := firstCASLTextValue(source["contact_id"], source["contactId"])
	rowType := firstCASLTextValue(source["type"], fallbackType, source["module"])
	if strings.EqualFold(rowType, "m3_in") {
		if userActionType == "" {
			userActionType = "mgr_action"
		}
		if mgrActionType == "" {
			mgrActionType = firstCASLTextValue(source["code"], source["subtype"])
		}
		if action == "" {
			action = mgrActionType
		}
		if code == "" {
			code = mgrActionType
		}
	}
	if code == "" && contactID == "" {
		code = firstCASLTextValue(source["status"])
	}
	if code == "" && contactID == "" {
		code = rowType
	}
	if ppkNum <= 0 && deviceID == "" && rawObjID == "" && !isCASLActionSource(firstCASLValue(userActionType, rowType)) {
		return CASLObjectEvent{}, false
	}

	ts, ok := firstCASLTimeValue(source["time"], source["timestamp"], source["ts"], source["create_date"])
	if !ok {
		return CASLObjectEvent{}, false
	}

	row := CASLObjectEvent{
		PPKNum:         ppkNum,
		DeviceID:       deviceID,
		ObjID:          rawObjID,
		ObjIDs:         extractCASLRealtimeStringList(source["objIds"], source["obj_ids"]),
		ObjName:        firstCASLTextValue(source["obj_name"]),
		ObjAddr:        firstCASLTextValue(source["obj_address"]),
		Action:         action,
		AlarmType:      firstCASLTextValue(source["alarm_type"]),
		AlarmID:        firstCASLTextValue(source["alarm_id"]),
		MgrID:          firstCASLTextValue(source["mgr_id"], source["mgr"]),
		UserID:         firstCASLTextValue(source["user_id"]),
		UserFIO:        firstCASLTextValue(source["user_fio"]),
		Time:           ts.UnixMilli(),
		Code:           code,
		Type:           rowType,
		UserActionType: userActionType,
		MgrActionType:  mgrActionType,
		PPKActionType:  ppkActionType,
		Subtype:        firstCASLTextValue(source["subtype"], source["type_event"], source["typeEvent"]),
		Number:         number,
		ContactID:      contactID,
		HozUserID:      firstCASLTextValue(source["hoz_user_id"]),
		Cause:          firstCASLTextValue(source["cause"]),
		Note:           firstCASLTextValue(source["note"]),
		BlockMessage: firstCASLTextValue(
			source["block_message"],
			source["blockMessage"],
		),
	}
	if timeUnblock, ok := firstCASLIntValue(source["time_unblock"], source["timeUnblock"]); ok {
		row.TimeUnblock = int64(timeUnblock)
	}
	if row.HozUserID == "" {
		row.HozUserID = firstCASLTextValue(source["user_id"])
	}
	if err := validateCASLRealtimeObjectEvent(row, "casl realtime"); err != nil {
		return CASLObjectEvent{}, false
	}
	return row, true
}

func extractCASLRealtimeStringList(values ...any) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, 4)
	for _, value := range values {
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				itemValue := strings.TrimSpace(asString(item))
				if itemValue == "" {
					continue
				}
				if _, exists := seen[itemValue]; exists {
					continue
				}
				seen[itemValue] = struct{}{}
				result = append(result, itemValue)
			}
		case []string:
			for _, item := range typed {
				itemValue := strings.TrimSpace(item)
				if itemValue == "" {
					continue
				}
				if _, exists := seen[itemValue]; exists {
					continue
				}
				seen[itemValue] = struct{}{}
				result = append(result, itemValue)
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
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
			if id := strings.TrimSpace(asString(typed[key])); isLikelyCASLConnID(id) {
				return id
			}
		}
		for _, nested := range caslRealtimeNestedPayloads(typed) {
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
			payload, _ := decodeCASLRealtimePayload(raw)
			if payload != nil {
				p.handleCASLRealtimePayload(payload)
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

			rows := extractCASLRealtimeRowsFromValue(payload)
			if payload == nil {
				rows = extractCASLRealtimeRows(raw)
			}
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

func shouldProcessCASLRealtimeRow(row CASLObjectEvent) bool {
	sourceType := strings.ToLower(strings.TrimSpace(row.Type))
	action := strings.ToUpper(strings.TrimSpace(firstCASLValue(row.Action, row.Code)))
	switch sourceType {
	case "ppk_in", "ppk_service":
		return action != "FIRMWARE_CHUNK" && action != "FIRMWARE_END" && action != "PPK_SER_NUM"
	case "mob_user_action":
		return strings.EqualFold(strings.TrimSpace(row.UserActionType), "MOB_ALARM")
	case "system_action":
		return action == "POST_PROC_ALARM_REPORT"
	case "chat_action", "notif", "ping", "system_info", "storage_change":
		return false
	}
	if action == "CREATE_USER" || action == "DELETE_USER" || action == "UPDATE_USER" {
		return false
	}
	return true
}

func (p *CASLCloudProvider) enrichRealtimeRows(ctx context.Context, rows []CASLObjectEvent) []CASLObjectEvent {
	if len(rows) == 0 {
		return nil
	}

	result := make([]CASLObjectEvent, 0, len(rows))
	for _, row := range rows {
		if !shouldProcessCASLRealtimeRow(row) {
			continue
		}
		if strings.TrimSpace(row.ObjID) == "" && strings.TrimSpace(row.AlarmID) != "" && strings.EqualFold(strings.TrimSpace(row.UserActionType), "mgr_action") {
			if objID, ok := p.getCASLObjectByAlarmID(ctx, row.AlarmID); ok {
				row.ObjID = objID
			}
		}
		result = append(result, row)
	}
	return result
}

func (p *CASLCloudProvider) handleCASLRealtimePayload(payload any) {
	root, ok := payload.(map[string]any)
	if !ok {
		return
	}

	switch strings.ToLower(strings.TrimSpace(asString(root["type"]))) {
	case "storage_change":
		p.handleCASLRealtimeStorageChange(root)
	}
}

func (p *CASLCloudProvider) handleCASLRealtimeStorageChange(payload map[string]any) {
	data, ok := payload["data"].(map[string]any)
	if !ok {
		return
	}

	action := strings.ToUpper(strings.TrimSpace(asString(data["action"])))
	if action == "" {
		return
	}
	info := data["info"]

	p.mu.Lock()
	defer p.mu.Unlock()

	if strings.Contains(action, "GUARDED_OBJECT") || strings.Contains(action, "CONNECTION") || strings.Contains(action, "DEVICE") {
		p.cachedObjects = nil
		p.cachedObjectsAt = time.Time{}
		p.objectByInternalID = make(map[int]caslGrdObject)
		p.deviceByDeviceID = make(map[string]caslDevice)
		p.deviceByObjectID = make(map[string]caslDevice)
		p.deviceByNumber = make(map[int64]caslDevice)
		p.cachedDevicesAt = time.Time{}
		p.cachedObjectEvents = make(map[int][]models.Event)
		p.cachedObjectEventsAt = make(map[int]time.Time)
	}
	if strings.Contains(action, "USER") || strings.Contains(action, "MGR") {
		p.cachedUsers = make(map[string]caslUser)
		p.cachedUsersAt = time.Time{}
	}
	if action == "ADD_NEW_EVENTS_JOURNAL" {
		p.eventsRevision++
	}
	if strings.Contains(action, "GUARDED_OBJECT") || strings.Contains(action, "CONNECTION") || strings.Contains(action, "DEVICE") || strings.Contains(action, "USER") || strings.Contains(action, "MGR") {
		p.eventsRevision++
	}
	if strings.HasPrefix(action, "DELETE_GUARDED_OBJECT") {
		deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, strings.TrimSpace(asString(info)))
	}
}

func (p *CASLCloudProvider) getCASLObjectByAlarmID(ctx context.Context, alarmID string) (string, bool) {
	alarmID = strings.TrimSpace(alarmID)
	if alarmID == "" {
		return "", false
	}

	var resp struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
		Error  string `json:"error"`
	}
	if err := p.postCommand(ctx, map[string]any{
		"type":     "get_obj_by_alarm_id",
		"alarm_id": alarmID,
	}, &resp, true); err != nil {
		return "", false
	}
	if !statusIsOK(resp.Status) {
		return "", false
	}
	value := strings.TrimSpace(asString(resp.Data))
	return value, value != ""
}

func (p *CASLCloudProvider) appendRealtimeRows(ctx context.Context, rows []CASLObjectEvent) error {
	rows = p.enrichRealtimeRows(ctx, rows)
	if len(rows) == 0 {
		return nil
	}

	p.mu.RLock()
	startGate := p.eventsStartAtMs
	p.mu.RUnlock()

	// 1. Обробка для загального журналу подій
	events, maxEventTime := p.mapCASLRowsToEvents(ctx, rows, startGate)
	if len(events) > 0 {
		p.mu.Lock()
		added := p.mergeCachedEventsLocked(events)
		if maxEventTime > p.eventsCursorMs {
			p.eventsCursorMs = maxEventTime
		}
		if added > 0 {
			p.eventsRevision++
		}
		p.mu.Unlock()
	}

	// 2. Оновлення кешу активних тривог.
	// На поточних CASL-інсталяціях окремий realtime тег `tape` може бути відсутній,
	// тому підтримуємо кеш тривог за звичайними realtime подіями (`user_action`,
	// `ppk_in`, `ppk_service` тощо), які вже приймаємо через WebSocket.
	if len(rows) > 0 {
		p.updateRealtimeAlarmsFromRows(ctx, rows)
	}

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
		models.EventAlarmNotification,
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
	resolvedByDeviceID := make(map[string]int64)
	unresolvedByDeviceID := make(map[string]struct{})
	for _, row := range rows {
		ppkNum := row.PPKNum
		if ppkNum <= 0 {
			ppkNum = p.resolveCASLPPKByDeviceIDWithCache(ctx, row.DeviceID, resolvedByDeviceID, unresolvedByDeviceID)
		}
		if ppkNum > 0 {
			ppkFilter[ppkNum] = struct{}{}
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
	customDeviceTypes := p.loadCASLCustomDeviceTypeSet(ctx)
	needStandardAlarmFlags := len(customDeviceTypes) == 0
	if !needStandardAlarmFlags {
		for _, row := range rows {
			ppkNum := row.PPKNum
			if ppkNum <= 0 {
				ppkNum = p.resolveCASLPPKByDeviceIDWithCache(ctx, row.DeviceID, resolvedByDeviceID, unresolvedByDeviceID)
			}

			deviceType := ""
			if ctxItem, ok := contextByPPK[ppkNum]; ok {
				deviceType = strings.TrimSpace(ctxItem.DeviceType)
			} else if objCtx, ok := contextByObject[strings.TrimSpace(row.ObjID)]; ok {
				deviceType = strings.TrimSpace(objCtx.DeviceType)
			}
			if deviceType == "" {
				needStandardAlarmFlags = true
				break
			}
			if _, isCustom := customDeviceTypes[strings.ToUpper(deviceType)]; !isCustom {
				needStandardAlarmFlags = true
				break
			}
		}
	}

	var standardAlarmFlags map[string]bool
	if needStandardAlarmFlags {
		standardAlarmFlags = p.loadAlarmEventsCatalogMap(ctx)
	}

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
		if row.Time <= 0 {
			continue
		}

		rawObjID := strings.TrimSpace(row.ObjID)
		ppkNum := row.PPKNum
		if ppkNum <= 0 {
			ppkNum = p.resolveCASLPPKByDeviceIDWithCache(ctx, row.DeviceID, resolvedByDeviceID, unresolvedByDeviceID)
		}

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}

		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10), strings.TrimSpace(row.DeviceID))
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
		case "MANY_GRD_OBJ_FINISH":
			for _, objID := range row.ObjIDs {
				deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objID)
			}
			if len(row.ObjIDs) == 0 {
				deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objectKey)
			}
			continue
		case "GRD_OBJ_MGR_CANCEL", "GRD_OBJ_FINISH":
			deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objectKey)
			continue
		case "DEVICE_BLOCK":
			p.applyCASLRealtimeDeviceBlockLocked(row, true)
		case "DEVICE_UNBLOCK":
			p.applyCASLRealtimeDeviceBlockLocked(row, false)
		}

		deviceType := ""
		translatorAlarms := map[string]bool(nil)
		isCustomDeviceType := false
		if hasCtx {
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
			translatorAlarms = ctxItem.TranslatorAlarms
		}
		if deviceType != "" {
			_, isCustomDeviceType = customDeviceTypes[strings.ToUpper(deviceType)]
		}

		details := buildCASLUserActionDetails(row, dictMap)
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
		eventType := classifyCASLEventTypeWithContext(classifierCode, strings.TrimSpace(row.ContactID), effectiveCASLSourceType(row), details)
		alarmFlags := translatorAlarms
		if !isCustomDeviceType {
			alarmFlags = standardAlarmFlags
		}
		if isAlarm, ok := resolveCASLAlarmFlagFromMap(alarmFlags, classifierCode, strings.TrimSpace(row.ContactID), strings.TrimSpace(row.Subtype)); ok {
			eventType = classifyCASLActiveAlarmEventType(eventType, isAlarm, true)
		} else if isAlarm, ok := resolveCASLAlarmFlagFromAlarmCatalog(standardAlarmFlags, classifierCode, strings.TrimSpace(row.ContactID), deviceType); ok && !isCustomDeviceType {
			eventType = classifyCASLActiveAlarmEventType(eventType, isAlarm, true)
		}
		alarmType, include := mapEventTypeToAlarmType(eventType)
		if action == "GRD_OBJ_NOTIF" {
			if mapped, ok := mapCASLAlarmType(row.AlarmType); ok {
				alarmType = mapped
				include = true
			} else if !include || alarmType == models.AlarmNotification {
				alarmType = models.AlarmBurglary
				include = true
			}
		}
		if !include && action == "GRD_OBJ_NOTIF" {
			alarmType = models.AlarmBurglary
			include = true
		}

		alarmTime := time.UnixMilli(row.Time).Local()
		_, err = strconv.Atoi(objectNum)
		if err != nil {
			log.Error().Err(err).Msg("CASL: не вдалося перетворити objectNum на int")
		}

		if action == "GRD_OBJ_NOTIF" {
			seed := stableCASLAlarmSeed(action, strings.TrimSpace(row.AlarmType), zoneNumber)
			p.realtimeAlarmByObjID[cacheKey] = models.Alarm{
				ID:           stableCASLAlarmID(objectKey, alarmTime.UnixMilli(), seed),
				ObjectID:     objectID,
				ObjectNumber: objectNum,
				ObjectName:   objectName,
				Address:      strings.TrimSpace(row.ObjAddr),
				Time:         alarmTime,
				Details:      details,
				Type:         alarmType,
				ZoneNumber:   int(row.Number),
				SC1:          mapCASLEventSC1(eventType),
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

		if action != "GRD_OBJ_NOTIF" {
			continue
		}
		if !isCASLEventAlarmCandidate(eventType) || !include {
			continue
		}

		seed := stableCASLAlarmSeed(row.Code, row.ContactID, zoneNumber)
		p.realtimeAlarmByObjID[cacheKey] = models.Alarm{
			ID:           stableCASLAlarmID(objectKey, alarmTime.UnixMilli(), seed),
			ObjectID:     objectID,
			ObjectNumber: objectNum,
			ObjectName:   objectName,
			Address:      strings.TrimSpace(row.ObjAddr),
			Time:         alarmTime,
			Details:      details,
			Type:         alarmType,
			ZoneNumber:   zoneNumber,
			SC1:          mapCASLEventSC1(eventType),
		}
	}
}

func (p *CASLCloudProvider) applyCASLRealtimeDeviceBlockLocked(row CASLObjectEvent, blocked bool) {
	deviceID := strings.TrimSpace(row.DeviceID)
	objID := strings.TrimSpace(row.ObjID)

	updateDevice := func(device caslDevice) {
		device.Blocked = blocked
		if deviceID != "" {
			p.deviceByDeviceID[deviceID] = device
		}
		if deviceObjID := strings.TrimSpace(device.ObjID.String()); deviceObjID != "" {
			p.deviceByObjectID[deviceObjID] = device
			if objID == "" {
				objID = deviceObjID
			}
		}
		if number := device.Number.Int64(); number > 0 {
			p.deviceByNumber[number] = device
		}
	}

	if deviceID != "" {
		if device, ok := p.deviceByDeviceID[deviceID]; ok {
			updateDevice(device)
		}
	}
	if objID != "" {
		if device, ok := p.deviceByObjectID[objID]; ok {
			updateDevice(device)
		}
	}

	if len(p.cachedObjects) == 0 {
		return
	}
	for idx := range p.cachedObjects {
		record := &p.cachedObjects[idx]
		if objID != "" && strings.TrimSpace(record.ObjID) != objID {
			continue
		}
		if deviceID != "" && strconv.FormatInt(record.DeviceID.Int64(), 10) != deviceID && objID == "" {
			continue
		}

		record.DeviceBlocked = blocked
		if blocked {
			record.BlockMessage = caslText(strings.TrimSpace(row.BlockMessage))
			if row.TimeUnblock > 0 {
				record.TimeUnblock = caslText(strconv.FormatInt(row.TimeUnblock, 10))
			}
		} else {
			record.BlockMessage = ""
			record.TimeUnblock = ""
		}

		internalID := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))
		p.objectByInternalID[internalID] = *record
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
