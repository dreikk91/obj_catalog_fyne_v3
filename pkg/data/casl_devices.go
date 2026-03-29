package data

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
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

	if body[0] == '[' {
		var lines []caslDeviceLine
		if err := json.Unmarshal(body, &lines); err == nil {
			return lines
		}
		return nil
	}

	if body[0] != '{' {
		return nil
	}

	var source map[string]any
	if err := json.Unmarshal(body, &source); err != nil {
		return nil
	}
	if len(source) == 0 {
		return nil
	}

	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]caslDeviceLine, 0, len(keys))
	for _, key := range keys {
		line, ok := decodeCASLDeviceLineFromAny(source[key], key)
		if !ok {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func decodeCASLDeviceLineFromAny(value any, fallbackKey string) (caslDeviceLine, bool) {
	var line caslDeviceLine
	fallbackNum := parseCASLID(fallbackKey)
	if fallbackNum > 0 {
		line.Number = caslInt64(fallbackNum)
		line.ID = caslInt64(fallbackNum)
	}

	switch typed := value.(type) {
	case string:
		line.Name = caslText(strings.TrimSpace(typed))
		return line, strings.TrimSpace(line.Name.String()) != ""
	case map[string]any:
		if encoded, err := json.Marshal(typed); err == nil {
			_ = json.Unmarshal(encoded, &line)
		}
		if line.ID.Int64() <= 0 && fallbackNum > 0 {
			line.ID = caslInt64(fallbackNum)
		}
		if line.Number.Int64() <= 0 && fallbackNum > 0 {
			line.Number = caslInt64(fallbackNum)
		}
		if line.Name.String() == "" {
			if text := strings.TrimSpace(asString(typed["description"])); text != "" {
				line.Name = caslText(text)
			}
		}
		return line, line.Name.String() != "" || line.ID.Int64() > 0 || line.Number.Int64() > 0
	default:
		return line, false
	}
}
