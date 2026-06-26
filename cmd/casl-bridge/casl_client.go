package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// CASLString handles float64/string/null types from JSON.
type CASLString string

func (cs *CASLString) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*cs = CASLString(stringFromCASL(raw))
	return nil
}

// CASLInt handles float64/string/null types from JSON.
type CASLInt int

func (ci *CASLInt) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*ci = CASLInt(intFromCASL(raw))
	return nil
}

type CASLDeviceLine struct {
	LineID        CASLString `json:"line_id"`
	DeviceID      CASLString `json:"device_id"`
	GroupNumber   CASLInt    `json:"group_number"`
	AdapterType   CASLString `json:"adapter_type"`
	DictLineType  CASLInt    `json:"dict_line_type"`
	Description   CASLString `json:"description"`
	LineType      CASLString `json:"line_type"`
	AdapterNumber CASLInt    `json:"adapter_number"`
	RoomID        CASLString `json:"room_id"`
	IsBroken      CASLInt    `json:"is_broken"`
}

type CASLDevice struct {
	DeviceID     CASLString                 `json:"device_id"`
	DeviceNumber CASLInt                    `json:"number"`
	Name         CASLString                 `json:"name"`
	Timeout      CASLInt                    `json:"timeout"`
	Sim1         CASLString                 `json:"sim1"`
	Sim2         CASLString                 `json:"sim2"`
	Lines        map[string]*CASLDeviceLine `json:"lines"`
}

type ReadOneDeviceResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error"`
	Data   *CASLDevice `json:"data"`
	Device *CASLDevice `json:"device"`
}

func (r *ReadOneDeviceResponse) GetDevice() *CASLDevice {
	if r.Data != nil {
		return r.Data
	}
	return r.Device
}

type LinkedObjectItem struct {
	ObjectID    CASLString `json:"obj_id"`
	ObjectName  CASLString `json:"obj_name"`
	Address     CASLString `json:"address"`
	Description CASLString `json:"description"`
}

type GetGrdObjectLastActByDeviceResponse struct {
	Status   string                      `json:"status"`
	Error    string                      `json:"error"`
	InfoMsgs map[string]*LinkedObjectItem `json:"infoMsgs"`
}

type CASLRoom struct {
	RoomID      CASLString `json:"room_id"`
	GrdObjID    CASLInt    `json:"grd_obj_id"`
	Name        CASLString `json:"name"`
	Note        CASLString `json:"note"`
	Description CASLString `json:"description"`
}

func (cr *CASLRoom) GetDescription() string {
	if cr.Note != "" {
		return string(cr.Note)
	}
	return string(cr.Description)
}

type ReadGrdRoomResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error"`
	Data   []*CASLRoom `json:"data"`
}

type CASLObject struct {
	GrdObjID CASLInt    `json:"grd_obj_id"`
	Name     CASLString `json:"name"`
	Address  CASLString `json:"address"`
	Note     CASLString `json:"note"`
	PultID   CASLInt    `json:"pult_id"`
}

type ReadGrdObjectByIDResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error"`
	Object *CASLObject `json:"object"`
}

type RoomUser struct {
	UserID   int64 `json:"user_id"`
	Priority int64 `json:"priority"`
}

type GetRoomLinksResponse struct {
	Status        string     `json:"status"`
	Error         string     `json:"error"`
	DeviceLineIDs []int64    `json:"device_line_ids"`
	Users         []RoomUser `json:"users"`
}

type CASLClient struct {
	request func(ctx context.Context, payload map[string]any) (map[string]any, error)
}

func NewCASLClient(request func(ctx context.Context, payload map[string]any) (map[string]any, error)) *CASLClient {
	return &CASLClient{request: request}
}

func (c *CASLClient) requestWithRetry(ctx context.Context, payload map[string]any) (map[string]any, error) {
	// Rate limiting: sleep briefly between calls to avoid spamming the broker
	time.Sleep(50 * time.Millisecond)

	var lastErr error
	backoff := 500 * time.Millisecond
	maxBackoff := 5 * time.Second

	for i := 0; i < 3; i++ {
		resp, err := c.request(ctx, payload)
		if err == nil {
			return resp, nil
		}

		errMsg := strings.ToLower(err.Error())
		isTimeout := strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline")
		isTransport := strings.Contains(errMsg, "broker") || strings.Contains(errMsg, "publish") || strings.Contains(errMsg, "connection")

		if !isTimeout && !isTransport {
			return nil, err
		}

		lastErr = err
		log.Warn().Err(err).Int("attempt", i+1).Msg("CASL API request failed, retrying...")

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
	return nil, fmt.Errorf("after retries: %w", lastErr)
}

func (c *CASLClient) ReadOneDevice(ctx context.Context, pultID int, ppkNum int) (*CASLDevice, error) {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":          "read_one_device",
		"device_number": ppkNum,
		"_user_id":      "0",
		"_pult_id":      strconv.Itoa(pultID),
	})
	if err != nil {
		return nil, err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return nil, fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var typedResp ReadOneDeviceResponse
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return nil, err
	}

	dev := typedResp.GetDevice()
	if dev == nil {
		return nil, fmt.Errorf("read_one_device: response data is missing device info")
	}
	return dev, nil
}

func (c *CASLClient) GetGrdObjectLastActByDevice(ctx context.Context, pultID int, ppkNums []any) (map[int]*LinkedObjectItem, error) {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":     "get_grd_object_last_act_by_device",
		"num_ppks": ppkNums,
		"time":     strconv.FormatInt(time.Now().UnixMilli(), 10),
		"_user_id": "0",
		"_pult_id": strconv.Itoa(pultID),
	})
	if err != nil {
		return nil, err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return nil, fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var typedResp struct {
		InfoMsgs map[string]*LinkedObjectItem `json:"infoMsgs"`
	}
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return nil, err
	}

	result := make(map[int]*LinkedObjectItem)
	for k, item := range typedResp.InfoMsgs {
		n, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		result[n] = item
	}
	return result, nil
}

func (c *CASLClient) CreateDevice(ctx context.Context, pultID int, ppkNum int, name, sim1, sim2 string, timeout int) (string, error) {
	payload := map[string]any{
		"type":        "create_device",
		"number":      ppkNum,
		"name":        name,
		"device_type": "FULL_SURGARD",
		"_user_id":    "0",
		"_pult_id":    strconv.Itoa(pultID),
	}
	if sim1 != "" {
		payload["sim1"] = sim1
	}
	if sim2 != "" {
		payload["sim2"] = sim2
	}
	if timeout > 0 {
		payload["timeout"] = timeout
	}

	resp, err := c.requestWithRetry(ctx, payload)
	if err != nil {
		return "", err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return "", fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}

	var typedResp struct {
		DeviceID CASLString `json:"device_id"`
	}
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return "", err
	}

	return string(typedResp.DeviceID), nil
}

func (c *CASLClient) CreateGrdObject(ctx context.Context, pultID int, ppkNum int, name, address, source string) (string, error) {
	payload := map[string]any{
		"type":             "create_grd_object",
		"name":             name,
		"address":          address,
		"description":      name,
		"contract":         fmt.Sprintf("%s:%d", source, ppkNum),
		"status":           "GRD_OFF",
		"object_type":      strings.ToUpper(source),
		"id_request":       fmt.Sprintf("casl-bridge:%s:%d", source, ppkNum),
		"reacting_pult_id": strconv.Itoa(pultID),
		"_user_id":         "0",
		"_pult_id":         strconv.Itoa(pultID),
	}

	resp, err := c.requestWithRetry(ctx, payload)
	if err != nil {
		return "", err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return "", fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}

	var typedResp struct {
		ObjectID CASLString `json:"obj_id"`
	}
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return "", err
	}

	return string(typedResp.ObjectID), nil
}

func (c *CASLClient) ReadGrdRoom(ctx context.Context, pultID int, objID string) ([]*CASLRoom, error) {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":        "read_grd_room",
		"obj_id":      objID,
		"grd_obj_id":  objID,
		"_user_id":    "0",
		"_pult_id":    strconv.Itoa(pultID),
	})
	if err != nil {
		return nil, err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return nil, fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var typedResp struct {
		Data []*CASLRoom `json:"data"`
	}
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return nil, err
	}

	return typedResp.Data, nil
}

func (c *CASLClient) CreateGrdRoom(ctx context.Context, pultID int, objID, roomName, description string) (string, error) {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":        "create_grd_room",
		"obj_id":      objID,
		"name":        roomName,
		"description": description,
		"rtsp":        "",
		"_user_id":    "0",
		"_pult_id":    strconv.Itoa(pultID),
	})
	if err != nil {
		return "", err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return "", fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}

	var typedResp struct {
		RoomID CASLString `json:"room_id"`
	}
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return "", err
	}

	return string(typedResp.RoomID), nil
}

func (c *CASLClient) GetRoomLinks(ctx context.Context, pultID int, roomID string) (*GetRoomLinksResponse, error) {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":     "get_room_links",
		"room_id":  roomID,
		"_user_id": "0",
		"_pult_id": strconv.Itoa(pultID),
	})
	if err != nil {
		return nil, err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return nil, fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var typedResp GetRoomLinksResponse
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return nil, err
	}

	return &typedResp, nil
}

func (c *CASLClient) AddLineToRoom(ctx context.Context, pultID int, deviceID string, lineNumber int, roomID, objID string) error {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":        "add_line_to_room",
		"device_id":   deviceID,
		"line_number": lineNumber,
		"room_id":     roomID,
		"obj_id":      objID,
		"_user_id":    "0",
		"_pult_id":    strconv.Itoa(pultID),
	})
	if err != nil {
		return err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return caslAPIError{op: "add_line_to_room", code: caslErrorCode(resp)}
	}
	return nil
}

func (c *CASLClient) EnsureDeviceLine(ctx context.Context, pultID int, deviceID string, lineNumber, groupNumber int, description string) error {
	payload := map[string]any{
		"type":           "create_device_line",
		"device_id":      deviceID,
		"line_number":    lineNumber,
		"group_number":   groupNumber,
		"adapter_type":   "SYS",
		"adapter_number": 0,
		"_user_id":       "0",
		"_pult_id":       strconv.Itoa(pultID),
	}
	if strings.TrimSpace(description) != "" {
		payload["description"] = strings.TrimSpace(description)
	}

	resp, err := c.requestWithRetry(ctx, payload)
	if err != nil {
		return err
	}
	s, _ := resp["status"].(string)
	errCode, _ := resp["error"].(string)
	if s == "ok" || errCode == "LINE_NUMBER_IN_USE" {
		return nil
	}
	return fmt.Errorf("create_device_line: %s", errCode)
}

func (c *CASLClient) UpdateDevice(ctx context.Context, pultID int, deviceID string, updates map[string]any) error {
	payload := map[string]any{
		"type":      "update_device",
		"device_id": deviceID,
		"_user_id":  "0",
		"_pult_id":  strconv.Itoa(pultID),
	}
	for k, v := range updates {
		payload[k] = v
	}

	resp, err := c.requestWithRetry(ctx, payload)
	if err != nil {
		return err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return fmt.Errorf("update_device: %s", caslErrorCode(resp))
	}
	return nil
}

func (c *CASLClient) UpdateDeviceLine(ctx context.Context, pultID int, deviceID string, lineNumber int, deviceLineID string, updates map[string]any) error {
	payload := map[string]any{
		"type":           "update_device_line",
		"device_id":      deviceID,
		"line_number":    lineNumber,
		"device_line_id": deviceLineID,
		"number":         lineNumber,
		"_user_id":       "0",
		"_pult_id":       strconv.Itoa(pultID),
	}
	for k, v := range updates {
		payload[k] = v
	}

	resp, err := c.requestWithRetry(ctx, payload)
	if err != nil {
		return err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return fmt.Errorf("update_device_line: %s", caslErrorCode(resp))
	}
	return nil
}

func (c *CASLClient) ReadGrdObjectByID(ctx context.Context, pultID int, objID string) (*CASLObject, error) {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":        "read_grd_object_by_id",
		"grd_obj_id":  objID,
		"obj_id":      objID,
		"_user_id":    "0",
		"_pult_id":    strconv.Itoa(pultID),
	})
	if err != nil {
		return nil, err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return nil, fmt.Errorf("status=%s, error=%v", s, resp["error"])
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var typedResp ReadGrdObjectByIDResponse
	if err := json.Unmarshal(raw, &typedResp); err != nil {
		return nil, err
	}

	if typedResp.Object == nil {
		return nil, fmt.Errorf("read_grd_object_by_id: response object is missing")
	}
	return typedResp.Object, nil
}

func (c *CASLClient) UpdateGrdRoom(ctx context.Context, pultID int, roomID string, roomName string, description string) error {
	resp, err := c.requestWithRetry(ctx, map[string]any{
		"type":        "update_grd_room",
		"room_id":     roomID,
		"name":        roomName,
		"description": description,
		"_user_id":    "0",
		"_pult_id":    strconv.Itoa(pultID),
	})
	if err != nil {
		return err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return fmt.Errorf("update_grd_room: %s", caslErrorCode(resp))
	}
	return nil
}

type caslAPIError struct {
	op   string
	code string
}

func (e caslAPIError) Error() string {
	if e.code == "" {
		return e.op
	}
	return e.op + ": " + e.code
}

func caslErrorCode(resp map[string]any) string {
	if code, _ := resp["error"].(string); code != "" {
		return code
	}
	return fmt.Sprint(resp["error"])
}

func stringFromCASL(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return fmt.Sprint(value)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return ""
	}
}

func intFromCASL(v any) int {
	switch value := v.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(value))
		return n
	default:
		return 0
	}
}

