package data

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func (p *CASLCloudProvider) loadDevices(ctx context.Context) ([]caslDevice, error) {
	p.mu.RLock()
	cacheValid := len(p.deviceByDeviceID) > 0 && time.Since(p.cachedDevicesAt) < caslObjectsCacheTTL
	if cacheValid {
		result := make([]caslDevice, 0, len(p.deviceByDeviceID))
		for _, item := range p.deviceByDeviceID {
			result = append(result, item)
		}
		p.mu.RUnlock()
		return result, nil
	}
	p.mu.RUnlock()

	devices, err := p.readDevices(ctx)
	if err == nil && len(devices) > 0 {
		p.applyCASLCoreSnapshot(nil, devices, nil)
		return append([]caslDevice(nil), devices...), nil
	}

	connObjects, connDevices, connUsers, connErr := p.readConnectionsCoreSnapshot(ctx)
	if connErr == nil && len(connDevices) > 0 {
		p.applyCASLCoreSnapshot(connObjects, connDevices, connUsers)
		return append([]caslDevice(nil), connDevices...), nil
	}

	if err != nil {
		return nil, err
	}
	return nil, connErr
}

func (p *CASLCloudProvider) resolveDeviceForObject(record caslGrdObject) (caslDevice, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	deviceID := strconv.FormatInt(record.DeviceID.Int64(), 10)
	if deviceID != "" && deviceID != "0" {
		if device, ok := p.deviceByDeviceID[deviceID]; ok {
			return device, true
		}
	}

	objID := strings.TrimSpace(record.ObjID)
	if objID != "" {
		if device, ok := p.deviceByObjectID[objID]; ok {
			return device, true
		}
	}

	deviceNumber := record.DeviceNumber.Int64()
	if deviceNumber > 0 {
		if device, ok := p.deviceByNumber[deviceNumber]; ok {
			return device, true
		}
	}

	return caslDevice{}, false
}

func (p *CASLCloudProvider) resolvePPKByDeviceID(ctx context.Context, deviceID string) (int64, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return 0, false
	}

	p.mu.RLock()
	if device, ok := p.deviceByDeviceID[deviceID]; ok {
		ppkNum := device.Number.Int64()
		p.mu.RUnlock()
		if ppkNum > 0 {
			return ppkNum, true
		}
		return 0, false
	}
	p.mu.RUnlock()

	if _, err := p.loadDevices(ctx); err != nil {
		return 0, false
	}

	p.mu.RLock()
	device, ok := p.deviceByDeviceID[deviceID]
	p.mu.RUnlock()
	if !ok {
		return 0, false
	}

	ppkNum := device.Number.Int64()
	if ppkNum <= 0 {
		return 0, false
	}
	return ppkNum, true
}

func buildCASLDeviceIndexes(devices []caslDevice) (map[string]caslDevice, map[string]caslDevice, map[int64]caslDevice) {
	byDeviceID := make(map[string]caslDevice, len(devices))
	byObjectID := make(map[string]caslDevice, len(devices))
	byNumber := make(map[int64]caslDevice, len(devices))

	for _, device := range devices {
		deviceID := strings.TrimSpace(device.DeviceID.String())
		if deviceID != "" {
			byDeviceID[deviceID] = device
		}
		objectID := strings.TrimSpace(device.ObjID.String())
		if objectID != "" {
			byObjectID[objectID] = device
		}
		number := device.Number.Int64()
		if number > 0 {
			byNumber[number] = device
		}
	}

	return byDeviceID, byObjectID, byNumber
}

func (p *CASLCloudProvider) readDevices(ctx context.Context) ([]caslDevice, error) {
	payload := map[string]any{"type": "read_device", "skip": 0, "limit": caslReadLimit}

	var resp caslReadDeviceResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	if err := validateCASLDevices(resp.Data); err != nil {
		return nil, err
	}

	return append([]caslDevice(nil), resp.Data...), nil
}

func decodeCASLDeviceType(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "—"
	}
	if translated, ok := caslDeviceTypeDisplayNames[value]; ok {
		return translated
	}
	return value
}

func selectCASLDevice(ok bool, device caslDevice) *caslDevice {
	if !ok {
		return nil
	}
	value := device
	return &value
}

func decodeCASLDeviceLines(raw json.RawMessage) []caslDeviceLine {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 || bytes.Equal(body, []byte("null")) {
		return nil
	}

	var generic any
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil
	}

	decoded := decodeCASLLinePayloads(generic)
	lines := make([]caslDeviceLine, 0, len(decoded))
	for _, item := range decoded {
		lines = append(lines, mapCASLDecodedLineToDeviceLine(item))
	}
	return lines
}

type caslDecodedLine struct {
	LineID        *int64
	LineNumber    int
	GroupNumber   int
	AdapterType   string
	AdapterNumber int
	Description   string
	LineType      string
	IsBlocked     bool
	RoomID        string
}

func decodeCASLLinePayloads(raw any) []caslDecodedLine {
	lines := make([]caslDecodedLine, 0, 16)
	switch typed := raw.(type) {
	case []any:
		for idx, item := range typed {
			if line, ok := decodeCASLLineFromAny(item, strconv.Itoa(idx+1)); ok {
				lines = append(lines, line)
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if line, ok := decodeCASLLineFromAny(typed[key], key); ok {
				lines = append(lines, line)
			}
		}
	}

	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].LineNumber < lines[j].LineNumber
	})
	return lines
}

func decodeCASLLineFromAny(value any, fallbackKey string) (caslDecodedLine, bool) {
	line := caslDecodedLine{
		LineNumber: parseCASLID(fallbackKey),
	}
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return line, false
		}
		line.Description = text
		line.LineType = "EMPTY"
		return line, true
	case map[string]any:
		lineID := int64(parseCASLAnyInt(firstCASLAny(typed["line_id"], typed["id"])))
		if lineID > 0 {
			line.LineID = &lineID
		}
		if parsed := parseCASLAnyInt(firstCASLAny(typed["line_number"], typed["number"])); parsed > 0 {
			line.LineNumber = parsed
		}
		line.GroupNumber = parseCASLAnyInt(typed["group_number"])
		line.AdapterType = strings.TrimSpace(asString(typed["adapter_type"]))
		line.AdapterNumber = parseCASLAnyInt(typed["adapter_number"])
		line.Description = strings.TrimSpace(firstCASLString(typed["description"], typed["name"]))
		line.LineType = strings.TrimSpace(firstCASLString(typed["line_type"], typed["type"]))
		line.IsBlocked = parseCASLAnyInt(typed["isBlocked"]) > 0 || strings.EqualFold(strings.TrimSpace(asString(typed["isBlocked"])), "true")
		line.RoomID = strings.TrimSpace(asString(typed["room_id"]))
		return line, line.LineNumber > 0 || line.LineID != nil
	default:
		return line, false
	}
}

func mapCASLDecodedLineToDeviceLine(line caslDecodedLine) caslDeviceLine {
	result := caslDeviceLine{
		GroupNumber:   caslInt64(line.GroupNumber),
		AdapterType:   caslText(line.AdapterType),
		AdapterNumber: caslInt64(line.AdapterNumber),
		Description:   caslText(line.Description),
		LineType:      caslText(line.LineType),
		IsBlocked:     line.IsBlocked,
		RoomID:        caslText(line.RoomID),
	}
	if line.LineID != nil && *line.LineID > 0 {
		result.ID = caslInt64(*line.LineID)
	}
	if line.LineNumber > 0 {
		result.Number = caslInt64(line.LineNumber)
	}
	if result.Description.String() != "" {
		result.Name = result.Description
	}
	if result.LineType.String() != "" {
		result.Type = result.LineType
	}
	if result.ID.Int64() <= 0 && result.Number.Int64() > 0 {
		result.ID = result.Number
	}
	if result.Number.Int64() <= 0 && result.ID.Int64() > 0 {
		result.Number = result.ID
	}
	return result
}

func mapCASLDecodedLineToEditorLine(line caslDecodedLine) contracts.CASLDeviceLineDetails {
	result := contracts.CASLDeviceLineDetails{
		LineNumber:    line.LineNumber,
		GroupNumber:   line.GroupNumber,
		AdapterType:   line.AdapterType,
		AdapterNumber: line.AdapterNumber,
		Description:   line.Description,
		LineType:      line.LineType,
		IsBlocked:     line.IsBlocked,
		RoomID:        line.RoomID,
	}
	if line.LineID != nil && *line.LineID > 0 {
		lineID := *line.LineID
		result.LineID = &lineID
	}
	return result
}
