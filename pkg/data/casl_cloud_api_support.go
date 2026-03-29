package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	caslCaptchaShowPath  = "/captchaShow"
	caslTimeServerPath   = "/get_time_server"
	caslSubscribePath    = "/subscribe"
	caslDefaultPageLimit = 1000
)

// CASLSessionInfo keeps the current auth/session details.
type CASLSessionInfo struct {
	Token  string
	WSURL  string
	UserID string
	PultID int64
}

// CASLCaptchaConfig describes /captchaShow response.
type CASLCaptchaConfig struct {
	Status               string `json:"status"`
	CaptchaShow          bool   `json:"captchaShow"`
	GoogleCaptchaSiteKey string `json:"GoogleCaptchaSiteKey"`
	Error                string `json:"error"`
}

// CASLAlarmEventDefinition describes a row from read_alarm_events.
type CASLAlarmEventDefinition struct {
	Code           string
	IsAlarmInStart int
	IsAlarm        int
}

// CASLObjectEvent is an exported shape of read_events_by_id records.
type CASLObjectEvent struct {
	PPKNum    int64
	DeviceID  string
	ObjID     string
	ObjName   string
	ObjAddr   string
	Action    string
	AlarmType string
	MgrID     string
	UserID    string
	UserFIO   string
	Time      int64
	Code      string
	Type      string
	Number    int64
	ContactID string
	HozUserID string
}

// CASLDeviceStateInfo is an exported shape of read_device_state.state.
type CASLDeviceStateInfo struct {
	Power        int64
	Accum        int64
	Door         int64
	Online       int64
	LastPingDate int64
	Lines        any
	Groups       any
	Adapters     any
}

// CASLStatsAlarms is an exported shape of get_statistic(name=stats_alarms).
type CASLStatsAlarms struct {
	DeviceID            string
	ObjectID            string
	ResponseFrequencies int64
	CommunicQuality     int64
	PowerFailure        int64
	Criminogenicity     int64
	CustomWins          int64
}

// CASLReadEventsByIDRequest describes filters for read_events_by_id.
type CASLReadEventsByIDRequest struct {
	IsFullEventsInfo bool
	TimeStart        int64
	TimeEnd          int64
	TimeRequest      int64
	ObjIDs           []string
	DeviceIDs        []string
	DeviceNumbers    []int64
}

// CASLGetStatisticRequest describes get_statistic input.
type CASLGetStatisticRequest struct {
	Name      string
	DeviceID  string
	ObjectID  string
	StartTime int64
	EndTime   int64
	Limit     int
}

// CASLReadEventsRequest describes read_events input.
type CASLReadEventsRequest struct {
	TimeStart   int64
	TimeEnd     int64
	TimeRequest int64
}

// SessionInfo returns the latest cached session values.
func (p *CASLCloudProvider) SessionInfo() CASLSessionInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return CASLSessionInfo{
		Token:  strings.TrimSpace(p.token),
		WSURL:  strings.TrimSpace(p.wsURL),
		UserID: strings.TrimSpace(p.userID),
		PultID: p.pultID,
	}
}

// EnsureAuthorized guarantees valid token (login will be performed if needed).
func (p *CASLCloudProvider) EnsureAuthorized(ctx context.Context) (CASLSessionInfo, error) {
	if _, err := p.ensureToken(ctx); err != nil {
		return CASLSessionInfo{}, err
	}
	return p.SessionInfo(), nil
}

// GetCaptchaConfig executes GET /captchaShow.
func (p *CASLCloudProvider) GetCaptchaConfig(ctx context.Context) (CASLCaptchaConfig, error) {
	body, status, err := p.doJSONGet(ctx, caslCaptchaShowPath)
	if err != nil {
		return CASLCaptchaConfig{}, err
	}

	var resp CASLCaptchaConfig
	if err := json.Unmarshal(body, &resp); err != nil {
		return CASLCaptchaConfig{}, fmt.Errorf("casl decode captchaShow response: %w", err)
	}

	if !statusIsOK(resp.Status) && !statusIsOK(status.Status) {
		errText := strings.TrimSpace(resp.Error)
		if errText == "" {
			errText = strings.TrimSpace(status.Error)
		}
		return CASLCaptchaConfig{}, fmt.Errorf("casl captchaShow status=%q error=%q", resp.Status, errText)
	}

	return resp, nil
}

// GetServerTime executes GET /get_time_server.
func (p *CASLCloudProvider) GetServerTime(ctx context.Context) (time.Time, error) {
	body, _, err := p.doJSONGet(ctx, caslTimeServerPath)
	if err != nil {
		return time.Time{}, err
	}

	var resp struct {
		Time  string `json:"time"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return time.Time{}, fmt.Errorf("casl decode get_time_server response: %w", err)
	}

	raw := strings.TrimSpace(resp.Time)
	if raw == "" {
		if strings.TrimSpace(resp.Error) != "" {
			return time.Time{}, fmt.Errorf("casl get_time_server error=%q", strings.TrimSpace(resp.Error))
		}
		return time.Time{}, fmt.Errorf("casl get_time_server: empty time")
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, raw)
	if parseErr != nil {
		parsed, parseErr = time.Parse(time.RFC3339, raw)
	}
	if parseErr != nil {
		return time.Time{}, fmt.Errorf("casl parse get_time_server time: %w", parseErr)
	}
	return parsed, nil
}

// Subscribe executes POST /subscribe with token.
func (p *CASLCloudProvider) Subscribe(ctx context.Context, connID string, tag string) error {
	connID = strings.TrimSpace(connID)
	tag = strings.TrimSpace(tag)
	if connID == "" {
		return fmt.Errorf("casl subscribe: conn_id is empty")
	}
	if tag == "" {
		return fmt.Errorf("casl subscribe: tag is empty")
	}

	token, err := p.ensureToken(ctx)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"conn_id": connID,
		"tag":     tag,
		"token":   token,
	}

	body, status, err := p.doJSONRequest(ctx, caslSubscribePath, payload)
	if err != nil {
		return err
	}

	var resp caslStatusOnlyResponse
	_ = json.Unmarshal(body, &resp)
	if !statusIsOK(resp.Status) && !statusIsOK(status.Status) {
		errText := strings.TrimSpace(resp.Error)
		if errText == "" {
			errText = strings.TrimSpace(status.Error)
		}
		return fmt.Errorf("casl subscribe status=%q error=%q", resp.Status, errText)
	}

	return nil
}

// ExecuteCASLCommand is a generic /command executor for known and future command types.
func (p *CASLCloudProvider) ExecuteCASLCommand(ctx context.Context, payload map[string]any, requireAuth bool) (map[string]any, error) {
	var response map[string]any
	if err := p.postCommand(ctx, payload, &response, requireAuth); err != nil {
		return nil, err
	}
	if response == nil {
		response = map[string]any{}
	}
	return response, nil
}

// ReadPults calls read_pult command (works without token).
func (p *CASLCloudProvider) ReadPults(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_pult", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, false)
}

// ReadGuardObjects calls read_grd_object command.
func (p *CASLCloudProvider) ReadGuardObjects(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_grd_object", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// ReadUsersRaw calls read_user command.
func (p *CASLCloudProvider) ReadUsersRaw(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_user", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// ReadDevices calls read_device command.
func (p *CASLCloudProvider) ReadDevices(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_device", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// ReadManagers calls read_mgr command.
func (p *CASLCloudProvider) ReadManagers(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_mgr", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// ReadConnections calls read_connections command.
func (p *CASLCloudProvider) ReadConnections(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_connections", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// ReadGuardRooms calls read_grd_room command.
func (p *CASLCloudProvider) ReadGuardRooms(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_grd_room", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// ReadDictionary calls read_dictionary command.
func (p *CASLCloudProvider) ReadDictionary(ctx context.Context) (map[string]any, error) {
	var resp struct {
		Status     string         `json:"status"`
		Dictionary map[string]any `json:"dictionary"`
		Error      string         `json:"error"`
	}

	if err := p.postCommand(ctx, map[string]any{"type": "read_dictionary"}, &resp, true); err != nil {
		return nil, err
	}
	if resp.Dictionary == nil {
		return map[string]any{}, nil
	}
	return resp.Dictionary, nil
}

// ReadAlarmEventsCatalog calls read_alarm_events command.
func (p *CASLCloudProvider) ReadAlarmEventsCatalog(ctx context.Context) ([]CASLAlarmEventDefinition, error) {
	var resp struct {
		Status string `json:"status"`
		Events []struct {
			Code           string    `json:"code"`
			IsAlarmInStart caslInt64 `json:"is_alarm_in_start"`
			IsAlarm        caslInt64 `json:"is_alarm"`
		} `json:"events"`
		Data []struct {
			Code           string    `json:"code"`
			IsAlarmInStart caslInt64 `json:"is_alarm_in_start"`
			IsAlarm        caslInt64 `json:"is_alarm"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := p.postCommand(ctx, map[string]any{"type": "read_alarm_events"}, &resp, true); err != nil {
		return nil, err
	}

	source := resp.Events
	if len(source) == 0 {
		source = resp.Data
	}

	result := make([]CASLAlarmEventDefinition, 0, len(source))
	for _, item := range source {
		result = append(result, CASLAlarmEventDefinition{
			Code:           strings.TrimSpace(item.Code),
			IsAlarmInStart: int(item.IsAlarmInStart.Int64()),
			IsAlarm:        int(item.IsAlarm.Int64()),
		})
	}
	return result, nil
}

// GetObjectsStatistic calls get_objects_statistic command.
func (p *CASLCloudProvider) GetObjectsStatistic(ctx context.Context) (map[string]any, error) {
	return p.readCommandDataAsMap(ctx, map[string]any{"type": "get_objects_statistic"}, true)
}

// GetDisconnectedDevices calls get_disconnected_devices command.
func (p *CASLCloudProvider) GetDisconnectedDevices(ctx context.Context) ([]map[string]any, error) {
	return p.readCommandDataAsMaps(ctx, map[string]any{"type": "get_disconnected_devices"}, true)
}

// GetAllAccessByPult calls get_all_access_by_pult command.
func (p *CASLCloudProvider) GetAllAccessByPult(ctx context.Context) (map[string]any, error) {
	return p.readCommandDataAsMap(ctx, map[string]any{"type": "get_all_access_by_pult"}, true)
}

// GetFirmwareList calls get_firmware_list command.
func (p *CASLCloudProvider) GetFirmwareList(ctx context.Context) ([]map[string]any, error) {
	return p.readCommandDataAsMaps(ctx, map[string]any{"type": "get_firmware_list"}, true)
}

// ReadFromBasket calls read_from_basket command.
func (p *CASLCloudProvider) ReadFromBasket(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	payload := map[string]any{"type": "read_from_basket", "skip": normalizePage(skip), "limit": normalizeLimit(limit)}
	return p.readCommandDataAsMaps(ctx, payload, true)
}

// Monitor calls monitor command.
func (p *CASLCloudProvider) Monitor(ctx context.Context) (map[string]any, error) {
	return p.readCommandDataAsMap(ctx, map[string]any{"type": "monitor"}, true)
}

// GetMessageTranslatorByDeviceType calls get_msg_translator_by_device_type command.
func (p *CASLCloudProvider) GetMessageTranslatorByDeviceType(ctx context.Context, deviceType string) (any, error) {
	key := strings.TrimSpace(deviceType)
	if key == "" {
		// Для translator endpoint не робимо auto-relogin на WRONG_FORMAT,
		// інакше отримаємо шторм login+command запитів.
		return p.readCommandDataAsAnyNoRelogin(ctx, map[string]any{"type": "get_msg_translator_by_device_type"}, true)
	}

	// У різних інсталяціях CASL зустрічаються обидва варіанти:
	// typeDevice (frontend) та device_type.
	primary := map[string]any{
		"type":       "get_msg_translator_by_device_type",
		"typeDevice": key,
	}
	data, err := p.readCommandDataAsAnyNoRelogin(ctx, primary, true)
	if err == nil {
		return data, nil
	}
	if !isCASLWrongFormatErr(err) {
		return nil, err
	}

	fallback := map[string]any{
		"type":        "get_msg_translator_by_device_type",
		"device_type": key,
	}
	return p.readCommandDataAsAnyNoRelogin(ctx, fallback, true)
}

// ReadGeneralTapeObjects calls get_general_tape_objects command.
func (p *CASLCloudProvider) ReadGeneralTapeObjects(ctx context.Context) ([]map[string]any, error) {
	return p.readCommandDataAsMaps(ctx, map[string]any{"type": "get_general_tape_objects"}, true)
}

// ReadGeneralTapeItem calls get_general_tape_item command for selected object IDs.
func (p *CASLCloudProvider) ReadGeneralTapeItem(ctx context.Context, objIDs []string) (map[string][]map[string]any, error) {
	filtered := make([]string, 0, len(objIDs))
	seen := make(map[string]struct{}, len(objIDs))
	for _, rawID := range objIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		filtered = append(filtered, id)
	}

	payload := map[string]any{"type": "get_general_tape_item"}
	if len(filtered) > 0 {
		payload["obj_ids"] = filtered
	}

	data, err := p.readCommandDataAsAny(ctx, payload, true)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return map[string][]map[string]any{}, nil
	}

	root, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("casl command %q: expected object data, got %T", asString(payload["type"]), data)
	}

	result := make(map[string][]map[string]any, len(root))
	for rawObjID, rawRows := range root {
		objID := strings.TrimSpace(rawObjID)
		if objID == "" {
			continue
		}

		rows := make([]map[string]any, 0, 8)
		switch typed := rawRows.(type) {
		case []any:
			for _, row := range typed {
				if mapped, ok := row.(map[string]any); ok {
					rows = append(rows, mapped)
				}
			}
		case map[string]any:
			rows = append(rows, typed)
		}
		if len(rows) == 0 {
			continue
		}
		result[objID] = rows
	}

	return result, nil
}

// GetRTSPURL calls get_rtsp_url command and returns data payload.
func (p *CASLCloudProvider) GetRTSPURL(ctx context.Context) (any, error) {
	return p.readCommandDataAsAny(ctx, map[string]any{"type": "get_rtsp_url"}, true)
}

// ReadEventsJournal executes read_events (general journal) command.
func (p *CASLCloudProvider) ReadEventsJournal(ctx context.Context, req CASLReadEventsRequest) ([]CASLObjectEvent, error) {
	end := req.TimeEnd
	if end <= 0 {
		end = time.Now().UnixMilli()
	}
	start := req.TimeStart
	if start <= 0 {
		start = end - caslObjectEventsSpan.Milliseconds()
	}
	timeRequest := req.TimeRequest
	if timeRequest <= 0 {
		timeRequest = end
	}

	payload := map[string]any{
		"type":         "read_events",
		"time_start":   start,
		"time_end":     end,
		"time_request": timeRequest,
	}

	var resp struct {
		Status string            `json:"status"`
		Data   []caslObjectEvent `json:"data"`
		Events []caslObjectEvent `json:"events"`
		Error  string            `json:"error"`
	}
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}

	rows := resp.Data
	if len(rows) == 0 {
		rows = resp.Events
	}

	result := make([]CASLObjectEvent, 0, len(rows))
	for _, item := range rows {
		result = append(result, CASLObjectEvent{
			PPKNum:    item.PPKNum.Int64(),
			DeviceID:  strings.TrimSpace(item.DeviceID.String()),
			ObjID:     strings.TrimSpace(item.ObjID.String()),
			ObjName:   strings.TrimSpace(item.ObjName.String()),
			ObjAddr:   strings.TrimSpace(item.ObjAddr.String()),
			Action:    strings.TrimSpace(item.Action.String()),
			AlarmType: strings.TrimSpace(item.AlarmType.String()),
			MgrID:     strings.TrimSpace(item.MgrID.String()),
			UserID:    strings.TrimSpace(item.UserID.String()),
			UserFIO:   strings.TrimSpace(item.UserFIO.String()),
			Time:      item.Time.Int64(),
			Code:      strings.TrimSpace(item.Code.String()),
			Type:      strings.TrimSpace(item.Type),
			Number:    item.Number.Int64(),
			ContactID: strings.TrimSpace(item.ContactID.String()),
			HozUserID: strings.TrimSpace(item.HozUserID.String()),
		})
	}
	return result, nil
}

// ReadBasketCount exposes read_count_in_basket command.
func (p *CASLCloudProvider) ReadBasketCount(ctx context.Context) (int, error) {
	return p.readBasketCount(ctx)
}

// ReadEventsByID executes read_events_by_id with explicit filters.
func (p *CASLCloudProvider) ReadEventsByID(ctx context.Context, req CASLReadEventsByIDRequest) ([]CASLObjectEvent, error) {
	end := req.TimeEnd
	if end <= 0 {
		end = time.Now().UnixMilli()
	}
	start := req.TimeStart
	if start <= 0 {
		start = end - caslObjectEventsSpan.Milliseconds()
	}
	timeRequest := req.TimeRequest
	if timeRequest <= 0 {
		timeRequest = end
	}

	payload := map[string]any{
		"type":             "read_events_by_id",
		"isFullEventsInfo": req.IsFullEventsInfo,
		"time_start":       start,
		"time_end":         end,
		"time_request":     timeRequest,
		"objIds":           append([]string(nil), req.ObjIDs...),
		"deviceIds":        append([]string(nil), req.DeviceIDs...),
		"deviceNumbers":    append([]int64(nil), req.DeviceNumbers...),
	}

	var resp caslReadEventsByIDResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}

	rawEvents := resp.Data
	if len(rawEvents) == 0 {
		rawEvents = resp.Events
	}

	result := make([]CASLObjectEvent, 0, len(rawEvents))
	for _, item := range rawEvents {
		result = append(result, CASLObjectEvent{
			PPKNum:    item.PPKNum.Int64(),
			DeviceID:  strings.TrimSpace(item.DeviceID.String()),
			ObjID:     strings.TrimSpace(item.ObjID.String()),
			ObjName:   strings.TrimSpace(item.ObjName.String()),
			ObjAddr:   strings.TrimSpace(item.ObjAddr.String()),
			Action:    strings.TrimSpace(item.Action.String()),
			AlarmType: strings.TrimSpace(item.AlarmType.String()),
			MgrID:     strings.TrimSpace(item.MgrID.String()),
			UserID:    strings.TrimSpace(item.UserID.String()),
			UserFIO:   strings.TrimSpace(item.UserFIO.String()),
			Time:      item.Time.Int64(),
			Code:      strings.TrimSpace(item.Code.String()),
			Type:      strings.TrimSpace(item.Type),
			Number:    item.Number.Int64(),
			ContactID: strings.TrimSpace(item.ContactID.String()),
			HozUserID: strings.TrimSpace(item.HozUserID.String()),
		})
	}

	return result, nil
}

// ReadDeviceStateByID executes read_device_state for the given device id.
func (p *CASLCloudProvider) ReadDeviceStateByID(ctx context.Context, deviceID string) (CASLDeviceStateInfo, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return CASLDeviceStateInfo{}, fmt.Errorf("casl read_device_state: device_id is empty")
	}

	payload := map[string]any{"type": "read_device_state", "device_id": deviceID}

	var resp caslReadDeviceStateResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return CASLDeviceStateInfo{}, err
	}

	state := resp.State
	return CASLDeviceStateInfo{
		Power:        state.Power.Int64(),
		Accum:        state.Accum.Int64(),
		Door:         state.Door.Int64(),
		Online:       state.Online.Int64(),
		LastPingDate: state.LastPingDate.Int64(),
		Lines:        state.Lines,
		Groups:       state.Groups,
		Adapters:     state.Adapters,
	}, nil
}

// GetStatistic executes get_statistic command.
func (p *CASLCloudProvider) GetStatistic(ctx context.Context, req CASLGetStatisticRequest) (CASLStatsAlarms, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "stats_alarms"
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	objectID := strings.TrimSpace(req.ObjectID)
	if name == "stats_alarms" && (deviceID == "" || objectID == "") {
		return CASLStatsAlarms{}, fmt.Errorf("casl get_statistic: deviceID and objectID are required")
	}

	end := req.EndTime
	if end <= 0 {
		end = time.Now().UnixMilli()
	}
	start := req.StartTime
	if start <= 0 {
		start = end - caslStatsSpan.Milliseconds()
	}

	payload := map[string]any{
		"type":      "get_statistic",
		"name":      name,
		"startTime": start,
		"endTime":   end,
	}
	if deviceID != "" {
		payload["deviceId"] = deviceID
	}
	if objectID != "" {
		payload["objectId"] = objectID
	}
	if req.Limit > 0 {
		payload["limit"] = req.Limit
	}

	var resp caslGetStatisticResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return CASLStatsAlarms{}, err
	}

	stats := resp.Data
	return CASLStatsAlarms{
		DeviceID:            strings.TrimSpace(stats.DeviceID),
		ObjectID:            strings.TrimSpace(stats.ObjectID),
		ResponseFrequencies: stats.ResponseFrequencies.Int64(),
		CommunicQuality:     stats.CommunicQuality.Int64(),
		PowerFailure:        stats.PowerFailure.Int64(),
		Criminogenicity:     stats.Criminogenicity.Int64(),
		CustomWins:          stats.CustomWins.Int64(),
	}, nil
}

// GetStatisticReport executes generic get_statistic(name=...) query and returns rows.
func (p *CASLCloudProvider) GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("casl get_statistic report: name is required")
	}

	payload := map[string]any{
		"type": "get_statistic",
		"name": name,
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	return p.readCommandDataAsMaps(ctx, payload, true)
}

// GroupOnDevice executes group_on_device action command.
func (p *CASLCloudProvider) GroupOnDevice(ctx context.Context, deviceNumber int64, groupNumber int) error {
	if deviceNumber <= 0 {
		return fmt.Errorf("casl group_on_device: deviceNumber must be > 0")
	}
	if groupNumber <= 0 {
		return fmt.Errorf("casl group_on_device: groupNumber must be > 0")
	}
	_, err := p.ExecuteCASLCommand(ctx, map[string]any{
		"type":          "group_on_device",
		"device_number": deviceNumber,
		"group_number":  groupNumber,
	}, true)
	return err
}

// GroupOffDevice executes group_off_device action command.
func (p *CASLCloudProvider) GroupOffDevice(ctx context.Context, deviceNumber int64, groupNumber int) error {
	if deviceNumber <= 0 {
		return fmt.Errorf("casl group_off_device: deviceNumber must be > 0")
	}
	if groupNumber <= 0 {
		return fmt.Errorf("casl group_off_device: groupNumber must be > 0")
	}
	_, err := p.ExecuteCASLCommand(ctx, map[string]any{
		"type":          "group_off_device",
		"device_number": deviceNumber,
		"group_number":  groupNumber,
	}, true)
	return err
}

// UpdateGuardObject executes update_grd_object action command.
func (p *CASLCloudProvider) UpdateGuardObject(ctx context.Context, object map[string]any) (map[string]any, error) {
	payload := copyStringAnyMap(object)
	payload["type"] = "update_grd_object"
	return p.ExecuteCASLCommand(ctx, payload, true)
}

// PickGuardObject executes grd_obj_pick action command.
func (p *CASLCloudProvider) PickGuardObject(ctx context.Context, objID string, eventID string) error {
	payload := map[string]any{"type": "grd_obj_pick"}
	if strings.TrimSpace(objID) != "" {
		payload["obj_id"] = strings.TrimSpace(objID)
	}
	if strings.TrimSpace(eventID) != "" {
		payload["event_id"] = strings.TrimSpace(eventID)
	}
	if len(payload) == 1 {
		return fmt.Errorf("casl grd_obj_pick: objID or eventID is required")
	}
	_, err := p.ExecuteCASLCommand(ctx, payload, true)
	return err
}

// FinishGuardObject executes grd_obj_finish action command.
func (p *CASLCloudProvider) FinishGuardObject(ctx context.Context, objID string, eventID string) error {
	payload := map[string]any{"type": "grd_obj_finish"}
	if strings.TrimSpace(objID) != "" {
		payload["obj_id"] = strings.TrimSpace(objID)
	}
	if strings.TrimSpace(eventID) != "" {
		payload["event_id"] = strings.TrimSpace(eventID)
	}
	if len(payload) == 1 {
		return fmt.Errorf("casl grd_obj_finish: objID or eventID is required")
	}
	_, err := p.ExecuteCASLCommand(ctx, payload, true)
	return err
}

func (p *CASLCloudProvider) readCommandDataAsMap(ctx context.Context, payload map[string]any, requireAuth bool) (map[string]any, error) {
	data, err := p.readCommandDataAsAny(ctx, payload, requireAuth)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return map[string]any{}, nil
	}
	if mapped, ok := data.(map[string]any); ok {
		return mapped, nil
	}
	return nil, fmt.Errorf("casl command %q: expected object data, got %T", asString(payload["type"]), data)
}

func (p *CASLCloudProvider) readCommandDataAsMaps(ctx context.Context, payload map[string]any, requireAuth bool) ([]map[string]any, error) {
	data, err := p.readCommandDataAsAny(ctx, payload, requireAuth)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	asSlice, ok := data.([]any)
	if !ok {
		if asMap, isMap := data.(map[string]any); isMap {
			return []map[string]any{asMap}, nil
		}
		return nil, fmt.Errorf("casl command %q: expected array data, got %T", asString(payload["type"]), data)
	}

	result := make([]map[string]any, 0, len(asSlice))
	for _, item := range asSlice {
		mapped, isMap := item.(map[string]any)
		if !isMap {
			continue
		}
		result = append(result, mapped)
	}

	return result, nil
}

func (p *CASLCloudProvider) readCommandDataAsAny(ctx context.Context, payload map[string]any, requireAuth bool) (any, error) {
	return p.readCommandDataAsAnyWithRelogin(ctx, payload, requireAuth, true)
}

func (p *CASLCloudProvider) readCommandDataAsAnyNoRelogin(ctx context.Context, payload map[string]any, requireAuth bool) (any, error) {
	return p.readCommandDataAsAnyWithRelogin(ctx, payload, requireAuth, false)
}

func (p *CASLCloudProvider) readCommandDataAsAnyWithRelogin(ctx context.Context, payload map[string]any, requireAuth bool, allowRelogin bool) (any, error) {
	var resp struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
		Error  string          `json:"error"`
	}
	if err := p.postCommandWithRetry(ctx, payload, &resp, requireAuth, allowRelogin); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, nil
	}

	var data any
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("casl decode %q data: %w", asString(payload["type"]), err)
	}

	return data, nil
}

func (p *CASLCloudProvider) doJSONGet(ctx context.Context, path string) ([]byte, caslStatusOnlyResponse, error) {
	startedAt := time.Now()
	log.Debug().
		Str("method", http.MethodGet).
		Str("path", path).
		Msg("CASL HTTP request")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl read response: %w", readErr)
	}

	log.Debug().
		Str("method", http.MethodGet).
		Str("path", path).
		Int("statusCode", resp.StatusCode).
		Dur("duration", time.Since(startedAt)).
		Msg("CASL HTTP response")
	logCASLHTTPBody(http.MethodGet, path, "response", body)

	var status caslStatusOnlyResponse
	_ = json.Unmarshal(body, &status)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(status.Error) != "" {
			return nil, status, fmt.Errorf("casl http %d: %s", resp.StatusCode, status.Error)
		}
		return nil, status, fmt.Errorf("casl unexpected http status: %d", resp.StatusCode)
	}

	return body, status, nil
}

func normalizePage(skip int) int {
	if skip < 0 {
		return 0
	}
	return skip
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return caslDefaultPageLimit
	}
	if limit > caslReadLimit {
		return caslReadLimit
	}
	return limit
}
