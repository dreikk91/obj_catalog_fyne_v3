package caslcompat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ContentTypeJSON = "application/json; charset=utf-8"
	fixtureToken    = "fixture-token"
)

type Handler struct {
	fixture     Fixture
	fixtureMu   sync.Mutex
	provider    contracts.DataProvider
	options     ProviderFixtureOptions
	upstream    CommandUpstream
	now         func() time.Time
	wsURL       string
	nextID      atomic.Uint64
	upgrader    websocket.Upgrader
	clientsMu   sync.Mutex
	clients     map[*wsClient]struct{}
	clientsByID map[string]*wsClient
}

// CommandUpstream executes native CASL /command requests for the original CASL source.
type CommandUpstream interface {
	ExecuteCASLCommand(ctx context.Context, payload map[string]any, requireAuth bool) (map[string]any, error)
}

// PpkCommandUpstream executes native /ppk_command requests.
type PpkCommandUpstream interface {
	ExecuteCASLPpkCommand(ctx context.Context, payload map[string]any) (map[string]any, error)
}

// DeviceCommandUpstream executes /api/devices/{deviceNumber}/command requests.
type DeviceCommandUpstream interface {
	ExecuteDeviceCommand(ctx context.Context, deviceNumber int, payload map[string]any) (map[string]any, error)
}

type wsClient struct {
	id     string
	conn   *websocket.Conn
	userID string
	mu     sync.Mutex
	subsMu sync.Mutex
	subs   map[string]map[string]struct{}
}

func NewFixtureHandler() *Handler {
	return NewFixtureHandlerWithFixture(DefaultFixture())
}

func NewFixtureHandlerWithFixture(fixture Fixture) *Handler {
	return &Handler{
		fixture: fixture,
		now:     time.Now,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
		clients:     make(map[*wsClient]struct{}),
		clientsByID: make(map[string]*wsClient),
	}
}

func NewFixtureHandlerWithWSURL(wsURL string) *Handler {
	h := NewFixtureHandlerWithFixture(DefaultFixture())
	h.wsURL = strings.TrimSpace(wsURL)
	return h
}

func NewFixtureHandlerWithFixtureAndWSURL(fixture Fixture, wsURL string) *Handler {
	h := NewFixtureHandlerWithFixture(fixture)
	h.wsURL = strings.TrimSpace(wsURL)
	return h
}

func NewProviderHandlerWithWSURL(provider contracts.DataProvider, options ProviderFixtureOptions, wsURL string) *Handler {
	fixture := BuildFixtureFromDataProvider(provider, options)
	h := NewFixtureHandlerWithFixtureAndWSURL(fixture, wsURL)
	h.provider = provider
	h.options = options
	return h
}

func (h *Handler) SetCommandUpstream(upstream CommandUpstream) {
	if h == nil {
		return
	}
	h.upstream = upstream
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeCASLError(w, http.StatusServiceUnavailable, "casl compatibility gateway is unavailable")
		return
	}

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")
	if path == "" {
		path = "/"
	}
	if strings.HasPrefix(path, "/api/devices/") && strings.HasSuffix(path, "/command") {
		h.handleAPIDeviceCommand(w, r)
		return
	}

	switch path {
	case "/":
		h.handleWebSocket(w, r)
	case "/captchaShow":
		h.handleCaptchaShow(w, r)
	case "/get_time_server":
		h.handleTimeServer(w, r)
	case "/login":
		h.handleLogin(w, r)
	case "/login_technician":
		h.handleLogin(w, r)
	case "/command":
		h.handleCommand(w, r)
	case "/ppk_command":
		h.handlePpkCommand(w, r)
	case "/ecom_command":
		h.handleEcomCommand(w, r)
	case "/subscribe":
		h.handleSubscribe(w, r)
	case "/subscribe_techn":
		h.handleSubscribe(w, r)
	case "/api/version":
		h.handleAPIVersion(w, r)
	case "/api/devices/state":
		h.handleAPIDevicesState(w, r)
	case "/api/report":
		h.handleAPIReport(w, r)
	default:
		writeCASLError(w, http.StatusNotFound, "route not found")
	}
}

func (h *Handler) handleCaptchaShow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeCASLMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{
		"status":               "ok",
		"captchaShow":          false,
		"GoogleCaptchaSiteKey": "",
	})
}

func (h *Handler) handleTimeServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeCASLMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   h.now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}

	wsURL := h.wsURL
	if wsURL == "" {
		wsURL = "ws://" + r.Host
	}
	h.fixtureMu.Lock()
	defer h.fixtureMu.Unlock()
	user := h.fixture.User
	user.WSURL = wsURL
	writeCASLJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"token":       fixtureToken,
		"user_id":     user.UserID,
		"email":       user.Email,
		"role":        user.Role,
		"first_name":  user.FirstName,
		"last_name":   user.LastName,
		"middle_name": user.MiddleName,
		"pult_id":     user.PultID,
		"ws_url":      wsURL,
		"server_control": map[string]any{
			"host": "127.0.0.1",
			"port": 0,
		},
		"data": user,
	})
}

func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req subscribeRequest
	if !decodeCASLBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ConnID) == "" {
		writeCASLError(w, http.StatusBadRequest, "conn_id is required")
		return
	}
	if strings.TrimSpace(req.Tag) == "" {
		writeCASLError(w, http.StatusBadRequest, "tag is required")
		return
	}
	if !h.subscribeClient(req.ConnID, req.PultIDString(), req.Tag) {
		writeCASLError(w, http.StatusBadRequest, "connection is not open")
		return
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req commandRequest
	if !decodeCASLBody(w, r, &req) {
		return
	}
	commandType := strings.TrimSpace(req.Type)

	if h.shouldProxyNativeCASLCommand(commandType, req) {
		if h.proxyNativeCASLCommand(w, r, req) {
			return
		}
	}

	h.fixtureMu.Lock()
	defer h.fixtureMu.Unlock()
	if h.shouldRefreshFromProvider(commandType) {
		h.refreshFixtureFromProvider()
	}

	switch commandType {
	case "read_grd_object":
		writeCASLOK(w, h.guardedObjects())
	case "read_device":
		writeCASLOK(w, h.fixture.Devices)
	case "read_devices_numbers":
		writeCASLOK(w, h.deviceNumbers())
	case "read_connections":
		writeCASLOK(w, h.fixture.Connections)
	case "read_dictionary":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"data":       h.fixture.Dictionary,
			"dictionary": h.fixture.Dictionary,
		})
	case "read_user":
		writeCASLOK(w, h.fixture.Users)
	case "read_mgr":
		writeCASLOK(w, h.fixture.Managers)
	case "read_pult":
		writeCASLOK(w, h.fixture.Pults)
	case "read_count_in_basket":
		writeCASLOK(w, map[string]any{"count": 0})
	case "get_all_access_by_pult":
		writeCASLOK(w, map[string]any{
			"accessDevices":      []any{},
			"accessTimeToTechns": 43_200_000,
		})
	case "get_templates":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"templates": map[string]any{
				"more_alarm_time":     map[string]any{},
				"ignoring_alarm_time": map[string]any{},
			},
		})
	case "get_firmware_list":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"files": []string{
				"4L_v199.hex.enc",
				"4L_v203.hex.enc",
				"4L_v208_beta.hex.enc",
				"4L_v209_beta.hex.enc",
			},
		})
	case "read_geo_zones":
		writeCASLOK(w, h.geoZones())
	case "read_from_basket":
		writeCASLOK(w, []any{})
	case "read_one_from_basket":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"basketElement": map[string]any{"data": "{}"},
		})
	case "save_in_basket", "del_from_basket":
		writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	case "get_grd_object_full":
		writeCASLJSON(w, http.StatusOK, h.fullObject(req))
	case "read_grd_room":
		writeCASLOK(w, h.objectRooms(req.Int("obj_id")))
	case "get_room_links":
		writeCASLJSON(w, http.StatusOK, h.roomLinks(req.String("room_id")))
	case "read_grd_object_by_id":
		writeCASLOK(w, h.objectByID(req.Int("obj_id")))
	case "get_obj_by_alarm_id":
		writeCASLJSON(w, http.StatusOK, h.fullObject(req))
	case "get_objects_statistic":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":          "ok",
			"data":            h.fixture.Statistics,
			"groupStatistics": h.fixture.Statistics["groupStatistics"],
			"countOfRooms":    h.fixture.Statistics["countOfRooms"],
		})
	case "get_statistic":
		writeCASLJSON(w, http.StatusOK, h.statisticResponse(req))
	case "read_alarm_events":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"data":   h.fixture.AlarmEvents,
			"events": h.fixture.AlarmEvents,
		})
	case "read_events", "read_events_by_id":
		writeCASLOK(w, h.eventJournal(req))
	case "read_last_user_action":
		writeCASLOK(w, map[string]any{})
	case "reset_alarm_event", "set_alarm_event":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"code":   req.String("code"),
			"codes":  req.Extra["codes"],
		})
	case "get_all_pults_users":
		users := h.allPultsUsers()
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"data":          users,
			"allPultsUsers": users,
		})
	case "get_pult_msg", "get_own_msg", "get_system_msg":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"data":      []any{},
			"chat_msgs": []any{},
		})
	case "send_msg":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"chat_msg_num": h.nextID.Add(1),
		})
	case "get_user_send_msg":
		writeCASLOK(w, []any{})
	case "get_disconnected_devices":
		writeCASLOK(w, h.fixture.DisconnectedDevices)
	case "get_general_tape_objects":
		writeCASLOK(w, h.fixture.GeneralTape)
	case "get_general_tape_item":
		writeCASLOK(w, h.generalTapeItems(req))
	case "get_record_history":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"records": []any{},
			"data":    []any{},
		})
	case "get_rtsp_url":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"rtsp_url": "",
		})
	case "get_msg_translator_by_device_type":
		writeCASLOK(w, h.messageTranslator(req))
	case "read_device_state":
		writeCASLOK(w, h.deviceState(req))
	case "grd_object_action":
		event := h.applyGuardedObjectAction(req)
		h.broadcastUserAction(event)
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status":           "ok",
			"reacting_pult_id": "1",
		})
	case "grd_object_group_action", "operator_alarm", "group_on_device", "user_action":
		writeCASLOK(w, map[string]any{
			"obj_id":  req.Int("obj_id"),
			"action":  req.String("action"),
			"user_id": h.fixture.User.UserID,
		})
	case "change_disconnected_state":
		writeCASLOK(w, map[string]any{
			"device_id":    req.Int("device_id"),
			"disconnected": req.Extra["disconnected"],
		})
	case "create_grd_object", "update_grd_object", "delete_grd_object",
		"create_grd_room", "update_grd_room", "delete_grd_room",
		"create_device", "update_device", "delete_device",
		"create_device_line", "update_device_line", "delete_device_line",
		"add_line_to_room", "remove_line_from_room",
		"add_user_to_room", "remove_user_from_room", "upd_priority_user_in_room",
		"create_user", "update_user", "delete_user",
		"create_mgr", "update_mgr", "delete_mgr", "add_mgr_user", "remove_mgr_user",
		"create_pult", "update_pult", "delete_pult", "add_user_to_pult", "remove_user_from_pult",
		"create_geo_zone", "update_geo_zone", "delete_geo_zone",
		"create_block_time_template", "update_block_time_template", "delete_block_time_template",
		"dictionary_add", "add_msg_translator", "edit_msg_translator", "remove_msg_translator",
		"add_user_send_msg", "remove_user_send_msg", "remove_all_send_msg_by_user",
		"add_access_to_techn", "delete_access_to_techn",
		"create_image", "delete_image", "ask_device_version":
		writeCASLJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"id":     h.nextID.Add(1),
		})
	default:
		if h.proxyNativeCASLCommand(w, r, req) {
			return
		}
		log.Printf("CASL fixture unsupported command type=%q body=%#v", commandType, req.Extra)
		writeCASLError(w, http.StatusBadRequest, "unsupported command: "+req.Type)
	}
}

func (h *Handler) shouldProxyNativeCASLCommand(commandType string, req commandRequest) bool {
	if h == nil || h.upstream == nil {
		return false
	}
	if req.hasCASLNamespacedID() {
		return true
	}
	switch commandType {
	case "ask_device_version", "ppk_action", "device_action", "group_off_device":
		return true
	default:
		return false
	}
}

func (h *Handler) proxyNativeCASLCommand(w http.ResponseWriter, r *http.Request, req commandRequest) bool {
	if h == nil || h.upstream == nil {
		return false
	}

	payload := req.nativeCASLPayload()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := h.upstream.ExecuteCASLCommand(ctx, payload, true)
	if err != nil {
		log.Printf("CASL upstream command failed type=%q: %v", req.Type, err)
		return false
	}
	writeCASLJSON(w, http.StatusOK, response)
	return true
}

func (h *Handler) shouldRefreshFromProvider(commandType string) bool {
	if h == nil || h.provider == nil {
		return false
	}
	switch commandType {
	case "get_general_tape_objects",
		"get_general_tape_item",
		"read_events",
		"read_events_by_id",
		"get_statistic",
		"get_disconnected_devices":
		return true
	default:
		return false
	}
}

func (h *Handler) refreshFixtureFromProvider() {
	if h == nil || h.provider == nil {
		return
	}
	fixture := BuildFixtureFromDataProvider(h.provider, h.options)
	if h.wsURL != "" {
		fixture.User.WSURL = h.wsURL
	}
	h.fixture = fixture
}

func (h *Handler) handlePpkCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}
	var req map[string]any
	if !decodeCASLBody(w, r, &req) {
		return
	}
	msg, _ := req["message"].(string)
	ppkNum := req["ppkNum"]
	if msg == "" || ppkNum == nil {
		writeCASLError(w, http.StatusBadRequest, "message and ppkNum are required")
		return
	}

	if h.upstream != nil {
		if ppkUpstream, ok := h.upstream.(PpkCommandUpstream); ok {
			ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
			defer cancel()
			resp, err := ppkUpstream.ExecuteCASLPpkCommand(ctx, req)
			if err != nil {
				log.Printf("CASL upstream ppk_command failed: %v", err)
				writeCASLError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeCASLJSON(w, http.StatusOK, resp)
			return
		}
	}

	writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) handleEcomCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}
	var req map[string]any
	if !decodeCASLBody(w, r, &req) {
		return
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"data": map[string]any{
			"command":  "",
			"response": "",
			"host":     "",
			"port":     "",
		},
	})
}

func (h *Handler) handleAPIVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeCASLMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "fixture",
	})
}

func (h *Handler) handleAPIDevicesState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeCASLMethodNotAllowed(w, http.MethodGet)
		return
	}

	data := make(map[string]any, len(h.fixture.Devices))
	for _, device := range h.fixture.Devices {
		data[fmt.Sprintf("%d", device.Number)] = map[string]any{
			"device_id":      device.DeviceID,
			"device_number":  device.Number,
			"online":         device.Offline < 0 && !device.Disconnected,
			"last_ping_date": h.now().UTC().Format(time.RFC3339),
		}
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok", "data": data})
}

func (h *Handler) handleAPIDeviceCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[1] != "api" || parts[2] != "devices" || parts[4] != "command" {
		writeCASLError(w, http.StatusBadRequest, "invalid route")
		return
	}
	deviceNumberStr := parts[3]
	deviceNum, err := strconv.Atoi(deviceNumberStr)
	if err != nil || deviceNum <= 0 {
		writeCASLError(w, http.StatusBadRequest, "invalid device number")
		return
	}

	var req map[string]any
	if !decodeCASLBody(w, r, &req) {
		return
	}

	cmd, _ := req["command"].(string)
	entityName, _ := req["entity_name"].(string)
	devicePassword, _ := req["device_password"].(string)
	deviceLicenseKey, _ := req["device_license_key"].(string)

	if cmd != "turn_on" && cmd != "turn_off" {
		writeCASLError(w, http.StatusBadRequest, "command must be turn_on or turn_off")
		return
	}
	validEntities := map[string]bool{"uk": true, "relay": true, "c": true, "group": true, "radio": true}
	if !validEntities[entityName] {
		writeCASLError(w, http.StatusBadRequest, "invalid entity_name")
		return
	}
	if devicePassword == "" || deviceLicenseKey == "" {
		writeCASLError(w, http.StatusBadRequest, "device_password and device_license_key required")
		return
	}

	entityNum, err := compatToInt(req["entity_number"])
	if err != nil {
		writeCASLError(w, http.StatusBadRequest, "entity_number must be a number")
		return
	}
	if err := validateEntityNumber(entityName, entityNum); err != nil {
		writeCASLError(w, http.StatusBadRequest, err.Error())
		return
	}

	kd, err := ParseLicenseKey(deviceLicenseKey)
	if err != nil {
		writeCASLError(w, http.StatusBadRequest, fmt.Sprintf("license key error: %v", err))
		return
	}
	if deviceNum != kd.PPKNum {
		writeCASLError(w, http.StatusBadRequest, "license key does not match device number")
		return
	}

	if h.upstream != nil {
		if devUpstream, ok := h.upstream.(DeviceCommandUpstream); ok {
			ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
			defer cancel()
			resp, err := devUpstream.ExecuteDeviceCommand(ctx, deviceNum, req)
			if err != nil {
				log.Printf("CASL upstream device command failed: %v", err)
				writeCASLError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeCASLJSON(w, http.StatusOK, resp)
			return
		}
	}

	writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

var validEntityNumbers = map[string][]int{
	"relay": {0, 2},
	"uk":    {2, 3},
	"c":     {1, 2, 3},
	"radio": {1, 2, 3},
	"group": {1, 2, 3, 4},
}

func validateEntityNumber(entityName string, num int) error {
	valid, ok := validEntityNumbers[entityName]
	if !ok {
		return fmt.Errorf("unknown entity_name: %s", entityName)
	}
	for _, v := range valid {
		if v == num {
			return nil
		}
	}
	return fmt.Errorf("invalid entity_number %d for %q; valid: %v", num, entityName, valid)
}

func compatToInt(v interface{}) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case string:
		return strconv.Atoi(n)
	}
	return 0, fmt.Errorf("cannot convert %T to int", v)
}

func (h *Handler) handleAPIReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCASLMethodNotAllowed(w, http.MethodPost)
		return
	}
	var req map[string]any
	if !decodeCASLBody(w, r, &req) {
		return
	}
	if _, ok := req["alarmId"]; !ok {
		writeCASLError(w, http.StatusBadRequest, "alarmId is required")
		return
	}
	writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		writeCASLError(w, http.StatusBadRequest, "websocket upgrade is required")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	connID := fmt.Sprintf("fixture-%d", h.nextID.Add(1))
	client := &wsClient{
		id:   connID,
		conn: conn,
		subs: make(map[string]map[string]struct{}),
	}
	h.addWSClient(client)
	defer func() {
		h.removeWSClient(client)
		_ = conn.Close()
	}()

	if err := client.writeJSON(map[string]any{"type": "conn_id", "id": connID}); err != nil {
		return
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, body, err := conn.ReadMessage()
			if err != nil {
				return
			}
			h.handleWSClientMessage(client, body)
		}
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := client.writeJSON(map[string]any{
				"type": "ping",
				"time": h.now().UTC().Format(time.RFC3339),
			}); err != nil {
				return
			}
		}
	}
}

func (c *wsClient) writeJSON(message any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(message)
}

func (h *Handler) addWSClient(client *wsClient) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	h.clients[client] = struct{}{}
	if strings.TrimSpace(client.id) != "" {
		h.clientsByID[client.id] = client
	}
}

func (h *Handler) removeWSClient(client *wsClient) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	delete(h.clients, client)
	if strings.TrimSpace(client.id) != "" {
		delete(h.clientsByID, client.id)
	}
}

func (h *Handler) handleWSClientMessage(client *wsClient, body []byte) {
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		return
	}
	switch strings.TrimSpace(asWSString(msg["type"])) {
	case "get_id":
		client.userID = strings.TrimSpace(asWSString(msg["user_id"]))
		_ = client.writeJSON(map[string]any{"type": "conn_id", "id": client.id})
	case "storage_change":
		h.broadcastByTagExcept("storage_change", asWSString(msg["pultId"]), client.id, msg)
	case "chat_action":
		h.broadcastByTagExcept("chat_action", "", client.id, msg)
	}
}

func (h *Handler) subscribeClient(connID string, pultID string, tag string) bool {
	connID = strings.TrimSpace(connID)
	tag = strings.TrimSpace(tag)
	if pultID = strings.TrimSpace(pultID); pultID == "" {
		pultID = "1"
	}
	if connID == "" || tag == "" {
		return false
	}

	h.clientsMu.Lock()
	client := h.clientsByID[connID]
	h.clientsMu.Unlock()
	if client == nil {
		return false
	}

	client.subsMu.Lock()
	defer client.subsMu.Unlock()
	if client.subs[pultID] == nil {
		client.subs[pultID] = make(map[string]struct{})
	}
	client.subs[pultID][tag] = struct{}{}
	return true
}

func (client *wsClient) subscribedTo(tag string, pultID string) bool {
	if client == nil {
		return false
	}
	if pultID = strings.TrimSpace(pultID); pultID == "" {
		pultID = "1"
	}
	tag = strings.TrimSpace(tag)
	client.subsMu.Lock()
	defer client.subsMu.Unlock()
	if tags := client.subs[pultID]; tags != nil {
		if _, ok := tags[tag]; ok {
			return true
		}
	}
	if tags := client.subs[""]; tags != nil {
		_, ok := tags[tag]
		return ok
	}
	return false
}

func (h *Handler) broadcastUserAction(event map[string]any) {
	pultID := asWSString(firstWSValue(event["reacting_pult_id"], event["pult_id"], "1"))
	h.broadcastByTag("user_action", pultID, event)
}

func (h *Handler) broadcastByTag(tag string, pultID string, event map[string]any) {
	h.clientsMu.Lock()
	clients := make([]*wsClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.clientsMu.Unlock()

	for _, client := range clients {
		if !client.subscribedTo(tag, pultID) {
			continue
		}
		out := cloneWSMessage(event)
		out["type"] = tag
		if err := client.writeJSON(out); err != nil {
			h.removeWSClient(client)
		}
	}
}

func (h *Handler) broadcastByTagExcept(tag string, pultID string, exceptConnID string, event map[string]any) {
	h.clientsMu.Lock()
	clients := make([]*wsClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.clientsMu.Unlock()

	for _, client := range clients {
		if client.id == exceptConnID || !client.subscribedTo(tag, pultID) {
			continue
		}
		out := cloneWSMessage(event)
		out["type"] = tag
		if err := client.writeJSON(out); err != nil {
			h.removeWSClient(client)
		}
	}
}

func (h *Handler) deviceNumbers() []int {
	result := make([]int, 0, len(h.fixture.Devices))
	for _, device := range h.fixture.Devices {
		result = append(result, device.Number)
	}
	return result
}

func (h *Handler) guardedObjects() []map[string]any {
	result := make([]map[string]any, 0, len(h.fixture.Objects))
	for _, object := range h.fixture.Objects {
		item := h.objectByID(object.ObjID)
		if device := h.deviceByObjectID(object.ObjID); device != nil {
			item["device_id"] = device.DeviceID
			item["device_number"] = device.Number
			item["device_type"] = device.DeviceType
			item["device_blocked"] = false
		}
		result = append(result, item)
	}
	return result
}

func (h *Handler) applyGuardedObjectAction(req commandRequest) map[string]any {
	objID := req.Int("obj_id")
	action := req.String("action")
	if action == "" {
		action = "GRD_OBJ_PICK"
	}
	now := h.now().UnixMilli()
	mgrID := req.String("mgr_id")
	event := map[string]any{
		"type":              "user_action",
		"user_action_type":  "grd_object_action",
		"obj_id":            objID,
		"action":            action,
		"user_id":           h.fixture.User.UserID,
		"user_ip":           "127.0.0.1",
		"time":              now,
		"record_start_time": nil,
		"room_id":           nil,
	}
	if mgrID != "" {
		event["mgr_id"] = mgrID
	}
	if cause := req.String("cause"); cause != "" {
		event["cause"] = cause
	}
	if note := req.String("note"); note != "" {
		event["note"] = note
	}

	for index := range h.fixture.GeneralTape {
		if h.fixture.GeneralTape[index].ObjID != objID {
			continue
		}
		switch action {
		case "GRD_OBJ_PICK", "GRD_OBJ_HIJACK":
			h.fixture.GeneralTape[index].UserID = h.fixture.User.UserID
			if h.fixture.GeneralTape[index].LastAct == "GRD_OBJ_NOTIF" {
				h.fixture.GeneralTape[index].LastAct = "GRD_OBJ_PICK"
			}
		case "GRD_OBJ_ASS_MGR":
			h.fixture.GeneralTape[index].MgrID = mgrID
			h.fixture.GeneralTape[index].LastAct = action
		case "GRD_OBJ_MGR_ARRIVE", "GRD_OBJ_MGR_CANCEL":
			h.fixture.GeneralTape[index].LastAct = action
		case "GRD_OBJ_FINISH":
			h.fixture.GeneralTape = append(h.fixture.GeneralTape[:index], h.fixture.GeneralTape[index+1:]...)
		default:
			h.fixture.GeneralTape[index].LastAct = action
		}
		break
	}

	h.appendGeneralTapeUserAction(objID, action, mgrID, req.String("cause"), req.String("note"), now)
	return event
}

func (h *Handler) appendGeneralTapeUserAction(objID int, action, mgrID, cause, note string, timeMS int64) {
	key := fmt.Sprintf("%d", objID)
	h.fixture.GeneralTapeItems[key] = append(h.fixture.GeneralTapeItems[key], FixtureEvent{
		ObjID:     objID,
		Time:      timeMS,
		Type:      "user_action",
		DictName:  action,
		UserID:    h.fixture.User.UserID,
		MgrID:     mgrID,
		Cause:     cause,
		Note:      note,
		Number:    nil,
		HozUserID: nil,
		ContactID: nil,
	})
}

func (h *Handler) fullObject(req commandRequest) map[string]any {
	objID := req.Int("obj_id")
	if objID == 0 {
		objID = req.Int("alarmId")
	}
	object := h.objectByID(objID)
	device := h.deviceByObjectID(objID)

	result := map[string]any{
		"status": "ok",
		"rooms":  h.objectRooms(objID),
		"images": []any{},
	}
	for key, value := range object {
		if key == "status" {
			result["obj_status"] = value
			result["object_status"] = value
			continue
		}
		result[key] = value
	}
	if device != nil {
		result["device"] = map[string]any{
			"id":          device.DeviceID,
			"device_id":   device.DeviceID,
			"number":      device.Number,
			"type":        device.Type,
			"device_type": device.DeviceType,
			"offline":     device.Offline,
		}
	}
	return result
}

func (h *Handler) objectByID(objID int) map[string]any {
	for _, object := range h.fixture.Objects {
		if object.ObjID == objID || objID == 0 {
			return map[string]any{
				"obj_id":            object.ObjID,
				"display_number":    h.objectDisplayNumber(object),
				"object_number":     h.objectDisplayNumber(object),
				"ppk_num":           h.objectDisplayNumber(object),
				"name":              object.Name,
				"address":           object.Address,
				"lat":               object.Lat,
				"long":              object.Long,
				"description":       object.Description,
				"contract":          object.Contract,
				"status":            object.Status,
				"object_type":       object.ObjectType,
				"pult_id":           fmt.Sprintf("%d", object.ReactingPultID),
				"reacting_pult_id":  fmt.Sprintf("%d", object.ReactingPultID),
				"user_id":           3,
				"start_date":        int64(0),
				"id_request":        "",
				"manager_id":        "1",
				"in_charge":         h.inChargeUserIDs(object.ObjID),
				"block_message":     nil,
				"time_unblock":      nil,
				"note":              "",
				"images":            []any{},
				"rooms":             h.objectRooms(object.ObjID),
				"geo_zone_id":       1,
				"bissnes_coeff":     nil,
				"non_working_hours": "",
			}
		}
	}
	return map[string]any{"obj_id": objID, "rooms": []any{}, "images": []any{}}
}

func (h *Handler) objectDisplayNumber(object FixtureObject) string {
	if number := strings.TrimSpace(object.DisplayNumber); number != "" {
		return number
	}
	if device := h.deviceByObjectID(object.ObjID); device != nil && device.Number > 0 {
		return fmt.Sprintf("%d", device.Number)
	}
	return fmt.Sprintf("%d", object.ObjID)
}

func (h *Handler) deviceByObjectID(objID int) *FixtureDevice {
	for i := range h.fixture.Devices {
		if h.fixture.Devices[i].ObjID == objID || objID == 0 {
			return &h.fixture.Devices[i]
		}
	}
	return nil
}

func (h *Handler) deviceByID(deviceID int) *FixtureDevice {
	for i := range h.fixture.Devices {
		if h.fixture.Devices[i].DeviceID == deviceID {
			return &h.fixture.Devices[i]
		}
	}
	return nil
}

func (h *Handler) objectRooms(objID int) []map[string]any {
	if objID == 0 {
		if len(h.fixture.Objects) == 0 {
			return []map[string]any{}
		}
		objID = h.fixture.Objects[0].ObjID
	}
	rooms := make([]map[string]any, 0)
	for _, room := range h.fixture.Rooms {
		if room.ObjID != fmt.Sprintf("%d", objID) {
			continue
		}
		rooms = append(rooms, map[string]any{
			"room_id":     room.RoomID,
			"obj_id":      room.ObjID,
			"name":        room.Name,
			"description": room.Description,
			"rtsp":        room.RTSP,
			"images":      room.Images,
			"lines":       room.Lines,
			"users":       room.Users,
		})
	}
	if len(rooms) > 0 {
		return rooms
	}

	device := h.deviceByObjectID(objID)
	lines := map[string]any{}
	if device != nil {
		for key, line := range device.Lines {
			roomID := line.RoomID
			if roomID == "" {
				roomID = fmt.Sprintf("%d01", objID)
			}
			lines[key] = map[string]any{
				"line_id":        line.LineID,
				"line_number":    line.LineNumber,
				"adapter_type":   line.AdapterType,
				"line_type":      line.LineType,
				"description":    line.Description,
				"group_number":   line.GroupNumber,
				"room_id":        roomID,
				"device_id":      device.DeviceID,
				"device_number":  device.Number,
				"adapter_number": line.AdapterNumber,
				"is_broken":      line.IsBroken,
				"isBlocked":      line.IsBlocked,
			}
		}
	}
	return []map[string]any{
		{
			"room_id":     fmt.Sprintf("%d01", objID),
			"obj_id":      fmt.Sprintf("%d", objID),
			"name":        "Fixture room",
			"description": "Fixture room",
			"rtsp":        "",
			"images":      []any{},
			"lines":       lines,
			"users":       h.roomUsers(objID),
		},
	}
}

func (h *Handler) inChargeUserIDs(objID int) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, room := range h.fixture.Rooms {
		if room.ObjID != fmt.Sprintf("%d", objID) {
			continue
		}
		for _, user := range room.Users {
			if _, ok := seen[user.UserID]; ok {
				continue
			}
			seen[user.UserID] = struct{}{}
			result = append(result, user.UserID)
		}
	}
	return result
}

func (h *Handler) roomUsers(objID int) []map[string]any {
	userIDs := h.inChargeUserIDs(objID)
	users := make([]map[string]any, 0, len(userIDs))
	for index, userID := range userIDs {
		users = append(users, map[string]any{
			"user_id":  userID,
			"priority": index + 1,
			"hoz_num":  fmt.Sprintf("%d", index+1),
		})
	}
	return users
}

func (h *Handler) roomLinks(roomID string) map[string]any {
	lineLinks := []map[string]any{}
	userLinks := []map[string]any{}
	for _, room := range h.fixture.Rooms {
		if room.RoomID != roomID {
			continue
		}
		for number, line := range room.Lines {
			lineNumber := line.LineNumber
			if lineNumber == 0 {
				_, _ = fmt.Sscanf(number, "%d", &lineNumber)
			}
			lineLinks = append(lineLinks, map[string]any{
				"room_id":       room.RoomID,
				"device_id":     line.DeviceID,
				"device_number": line.DeviceNumber,
				"line_id":       line.LineID,
				"line_number":   lineNumber,
				"number":        lineNumber,
			})
		}
		for _, user := range room.Users {
			userLinks = append(userLinks, map[string]any{
				"room_id":  room.RoomID,
				"user_id":  user.UserID,
				"priority": user.Priority,
				"hoz_num":  user.HozNum,
			})
		}
		break
	}
	return map[string]any{
		"status":     "ok",
		"line_links": lineLinks,
		"user_links": userLinks,
	}
}

func (h *Handler) geoZones() []map[string]any {
	return []map[string]any{
		{
			"geo_zone_id": "1",
			"pult_id":     1,
			"name":        "Fixture zone",
			"mgrs":        []string{"1"},
		},
	}
}

func (h *Handler) allPultsUsers() []map[string]any {
	result := make([]map[string]any, 0, len(h.fixture.Users))
	for _, user := range h.fixture.Users {
		result = append(result, map[string]any{
			"user_id":     user.UserID,
			"email":       user.Email,
			"role":        user.Role,
			"first_name":  user.FirstName,
			"last_name":   user.LastName,
			"middle_name": user.MiddleName,
			"pult_id":     user.PultID,
		})
	}
	return result
}

func (h *Handler) generalTapeItems(req commandRequest) map[string][]map[string]any {
	objIDs := req.IntSlice("objIds")
	if len(objIDs) == 0 {
		objIDs = req.ObjIDs
	}
	if len(req.ObjIDs) == 0 {
		objID := req.Int("obj_id")
		if objID > 0 {
			objIDs = []int{objID}
		}
	}

	if len(objIDs) == 0 {
		return h.generalTapeItemsFor(h.fixture.GeneralTapeItems)
	}

	filtered := make(map[string][]FixtureEvent, len(objIDs))
	for _, id := range objIDs {
		key := fmt.Sprintf("%d", id)
		if events, ok := h.fixture.GeneralTapeItems[key]; ok {
			filtered[key] = events
		}
	}
	return h.generalTapeItemsFor(filtered)
}

func (h *Handler) generalTapeItemsFor(items map[string][]FixtureEvent) map[string][]map[string]any {
	result := make(map[string][]map[string]any, len(items))
	for objID, events := range items {
		rows := make([]map[string]any, 0, len(events))
		for _, event := range events {
			if event.DictName != "" {
				row := map[string]any{
					"user_id":   event.UserID,
					"dict_name": event.DictName,
					"time":      event.Time,
				}
				if event.MgrID != "" {
					row["mgr_id"] = event.MgrID
				}
				if event.Cause != "" {
					row["cause"] = event.Cause
				}
				if event.Note != "" {
					row["note"] = event.Note
				}
				rows = append(rows, row)
				continue
			}
			rows = append(rows, map[string]any{
				"code":        event.Code,
				"time":        event.Time,
				"hoz_user_id": event.HozUserID,
				"contact_id":  event.ContactID,
				"number":      event.Number,
			})
		}
		result[objID] = rows
	}
	return result
}

func (h *Handler) eventJournal(req commandRequest) []map[string]any {
	objIDs := req.IntSlice("objIds")
	if len(objIDs) == 0 {
		objIDs = req.ObjIDs
	}
	objFilter := make(map[int]struct{}, len(objIDs))
	for _, objID := range objIDs {
		objFilter[objID] = struct{}{}
	}
	if objID := req.Int("obj_id"); objID > 0 {
		objFilter[objID] = struct{}{}
	}

	startTime := int64(firstInt(req.Int("time_start"), req.Int("startTime")))
	endTime := int64(firstInt(req.Int("time_end"), req.Int("endTime")))

	events := make([]map[string]any, 0)
	for _, rows := range h.fixture.GeneralTapeItems {
		for _, event := range rows {
			if !fixtureEventMatches(event.ObjID, event.Time, objFilter, startTime, endTime) {
				continue
			}
			events = append(events, fixtureEventJournalRow(event))
		}
	}
	if len(events) > 0 {
		return events
	}
	for _, tape := range h.fixture.GeneralTape {
		if !fixtureEventMatches(tape.ObjID, tape.Time, objFilter, startTime, endTime) {
			continue
		}
		events = append(events, fixtureTapeJournalRow(tape))
	}
	if len(events) > 0 {
		return events
	}
	if startTime > 0 || endTime > 0 {
		for _, rows := range h.fixture.GeneralTapeItems {
			for _, event := range rows {
				if !fixtureEventMatches(event.ObjID, event.Time, objFilter, 0, 0) {
					continue
				}
				events = append(events, fixtureEventJournalRow(event))
			}
		}
		if len(events) > 0 {
			return events
		}
		for _, tape := range h.fixture.GeneralTape {
			if !fixtureEventMatches(tape.ObjID, tape.Time, objFilter, 0, 0) {
				continue
			}
			events = append(events, fixtureTapeJournalRow(tape))
		}
		if len(events) > 0 {
			return events
		}
	}
	return []map[string]any{}
}

func fixtureEventMatches(objID int, eventTime int64, objFilter map[int]struct{}, startTime int64, endTime int64) bool {
	if len(objFilter) > 0 {
		if _, ok := objFilter[objID]; !ok {
			return false
		}
	}
	if startTime > 0 && eventTime < startTime {
		return false
	}
	if endTime > 0 && eventTime > endTime {
		return false
	}
	return true
}

func fixtureEventJournalRow(event FixtureEvent) map[string]any {
	eventType := event.Type
	if eventType == "" {
		eventType = "ppk_event"
	}
	row := map[string]any{
		"type":        eventType,
		"obj_id":      event.ObjID,
		"device_id":   event.DeviceID,
		"ppk_num":     event.PPKNum,
		"time":        event.Time,
		"contact_id":  event.ContactID,
		"hoz_user_id": event.HozUserID,
	}
	if event.Code != 0 {
		row["code"] = event.Code
	}
	if event.TypeEvent != "" {
		row["type_event"] = event.TypeEvent
	}
	if event.AdditionalType != 0 {
		row["additional_type"] = event.AdditionalType
	}
	if event.Msg != "" {
		row["msg"] = event.Msg
	}
	if event.DictName != "" {
		row["action"] = event.DictName
		row["user_action_type"] = "grd_object_action"
		row["user_id"] = event.UserID
		row["user_ip"] = "0.0.0.0"
		row["record_start_time"] = nil
		row["room_id"] = nil
		if event.MgrID != "" {
			row["mgr_id"] = event.MgrID
		}
		if event.Cause != "" {
			row["cause"] = event.Cause
		}
		if event.Note != "" {
			row["note"] = event.Note
		}
	}
	if event.Number != nil {
		row["number"] = event.Number
	} else if event.LineNumber != 0 {
		row["number"] = event.LineNumber
	} else {
		row["number"] = nil
	}
	if event.Line != 0 {
		row["line"] = event.Line
	}
	if event.LineNumber != 0 {
		row["line_number"] = event.LineNumber
	}
	return row
}

func fixtureTapeJournalRow(tape FixtureTapeRow) map[string]any {
	msg := strings.TrimSpace(tape.Description)
	if msg == "" {
		msg = strings.TrimSpace(tape.ReasonAlarm)
	}
	return map[string]any{
		"type":        "ppk_event",
		"obj_id":      tape.ObjID,
		"device_id":   tape.DeviceID,
		"ppk_num":     0,
		"time":        tape.Time,
		"code":        999,
		"type_event":  "E",
		"msg":         msg,
		"number":      nil,
		"contact_id":  nil,
		"hoz_user_id": nil,
	}
}

func (h *Handler) statisticRows(req commandRequest) []map[string]any {
	events := h.eventJournal(req)
	rows := make([]map[string]any, 0, len(events))
	for _, event := range events {
		rows = append(rows, map[string]any{
			"time":        event["time"],
			"obj_id":      event["obj_id"],
			"device_id":   event["device_id"],
			"ppk_num":     event["ppk_num"],
			"event":       event["msg"],
			"code":        event["code"],
			"type_event":  event["type_event"],
			"description": event["msg"],
		})
	}
	return rows
}

func (h *Handler) statisticResponse(req commandRequest) map[string]any {
	if strings.TrimSpace(req.String("name")) == "stats_alarms" {
		return map[string]any{
			"status": "ok",
			"data":   h.alarmStatistic(req),
		}
	}

	rows := h.statisticRows(req)
	return map[string]any{
		"status":      "ok",
		"data":        rows,
		"total_count": len(rows),
	}
}

func (h *Handler) alarmStatistic(req commandRequest) map[string]any {
	objectID := firstInt(req.Int("objectId"), req.Int("obj_id"), req.Int("object_id"))
	deviceID := firstInt(req.Int("deviceId"), req.Int("device_id"))
	if objectID == 0 && deviceID > 0 {
		if device := h.deviceByID(deviceID); device != nil {
			objectID = device.ObjID
		}
	}
	if deviceID == 0 && objectID > 0 {
		if device := h.deviceByObjectID(objectID); device != nil {
			deviceID = device.DeviceID
		}
	}

	startTime := int64(firstInt(req.Int("startTime"), req.Int("time_start")))
	endTime := int64(firstInt(req.Int("endTime"), req.Int("time_end")))

	totalEvents := 0
	alarmEvents := 0
	powerFailures := 0
	for _, rows := range h.fixture.GeneralTapeItems {
		for _, event := range rows {
			if objectID > 0 && event.ObjID != objectID {
				continue
			}
			if deviceID > 0 && event.DeviceID != deviceID {
				continue
			}
			if startTime > 0 && event.Time < startTime {
				continue
			}
			if endTime > 0 && event.Time > endTime {
				continue
			}
			totalEvents++
			if event.AdditionalType > 0 {
				alarmEvents++
			}
			if event.Code == 301 {
				powerFailures++
			}
		}
	}

	communicQuality := 6
	if deviceID > 0 {
		if device := h.deviceByID(deviceID); device != nil {
			switch {
			case device.Disconnected || device.Offline >= 0:
				communicQuality = 0
			case device.SignalLevel > 0:
				communicQuality = device.SignalLevel
			}
		}
	}

	return map[string]any{
		"device_id":           fmt.Sprintf("%d", deviceID),
		"obj_id":              fmt.Sprintf("%d", objectID),
		"responseFrequencies": totalEvents,
		"communicQuality":     communicQuality,
		"powerFailure":        powerFailures,
		"criminogenicity":     alarmEvents,
		"customWins":          totalEvents,
	}
}

func (h *Handler) messageTranslator(req commandRequest) map[string][]FixtureTranslatorRow {
	deviceType := strings.TrimSpace(req.String("device_type"))
	if deviceType == "" {
		deviceType = strings.TrimSpace(req.String("type_protocol"))
	}
	if deviceType == "" {
		deviceType = strings.TrimSpace(req.String("typeDevice"))
	}
	if deviceType == "" {
		return h.fixture.MessageTranslators
	}
	if rows, ok := h.fixture.MessageTranslators[deviceType]; ok {
		return map[string][]FixtureTranslatorRow{deviceType: rows}
	}
	return map[string][]FixtureTranslatorRow{deviceType: nil}
}

func (h *Handler) deviceState(req commandRequest) map[string]any {
	deviceID := req.Int("device_id")
	for _, dev := range h.fixture.Devices {
		if dev.DeviceID == deviceID {
			return map[string]any{
				"device_id":      dev.DeviceID,
				"online":         dev.Offline < 0 && !dev.Disconnected,
				"last_ping_date": h.now().UTC().Format(time.RFC3339),
			}
		}
	}
	return map[string]any{
		"device_id": deviceID,
		"online":    false,
	}
}

func writeCASLOK(w http.ResponseWriter, data any) {
	writeCASLJSON(w, http.StatusOK, map[string]any{"status": "ok", "data": data})
}

func writeCASLError(w http.ResponseWriter, status int, message string) {
	if strings.TrimSpace(message) == "" {
		message = "request failed"
	}
	writeCASLJSON(w, status, map[string]any{"status": "error", "error": message})
}

func writeCASLMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeCASLError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeCASLJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"status":"error","error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func decodeCASLBody(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		writeCASLError(w, http.StatusBadRequest, "request body is required")
		return false
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeCASLError(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

type subscribeRequest struct {
	Token        string `json:"token"`
	ConnID       string `json:"conn_id"`
	Tag          string `json:"tag"`
	PultID       any    `json:"pult_id"`
	IsTechnician bool   `json:"isTechnician"`
}

func (r subscribeRequest) PultIDString() string {
	value := strings.TrimSpace(asWSString(r.PultID))
	if value == "" {
		return "1"
	}
	return value
}

type commandRequest struct {
	Type   string         `json:"type"`
	Token  string         `json:"token"`
	ObjIDs []int          `json:"obj_ids"`
	Extra  map[string]any `json:"-"`
}

func (r *commandRequest) UnmarshalJSON(data []byte) error {
	type alias commandRequest
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = commandRequest(decoded)
	r.Extra = raw
	return nil
}

func (r commandRequest) String(key string) string {
	value, ok := r.Extra[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return fmt.Sprint(typed)
	}
}

func (r commandRequest) Int(key string) int {
	value, ok := r.Extra[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case string:
		var parsed int
		_, _ = fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

func (r commandRequest) IntSlice(key string) []int {
	value, ok := r.Extra[key]
	if !ok || value == nil {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]int, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case float64:
			result = append(result, int(typed))
		case string:
			var parsed int
			_, _ = fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed)
			if parsed != 0 {
				result = append(result, parsed)
			}
		}
	}
	return result
}

func (r commandRequest) hasCASLNamespacedID() bool {
	for _, key := range []string{"obj_id", "objectId", "object_id", "alarmId", "device_id", "deviceId"} {
		if ids.IsCASLObjectID(r.Int(key)) {
			return true
		}
	}
	for _, key := range []string{"obj_ids", "objIds", "objectIds", "alarmIds", "deviceIds"} {
		for _, id := range r.IntSlice(key) {
			if ids.IsCASLObjectID(id) {
				return true
			}
		}
	}
	return false
}

func (r commandRequest) nativeCASLPayload() map[string]any {
	payload := make(map[string]any, len(r.Extra))
	for key, value := range r.Extra {
		if strings.EqualFold(key, "token") {
			continue
		}
		payload[key] = nativeCASLValue(key, value)
	}
	return payload
}

func nativeCASLValue(key string, value any) any {
	switch typed := value.(type) {
	case float64:
		if native, ok := nativeCASLID(key, int(typed)); ok {
			return native
		}
	case string:
		if parsed := parseIntString(typed); parsed > 0 {
			if native, ok := nativeCASLID(key, parsed); ok {
				return fmt.Sprintf("%d", native)
			}
		}
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, nativeCASLValue(key, item))
		}
		return out
	}
	return value
}

func nativeCASLID(key string, id int) (int, bool) {
	if !isCASLIDField(key) || !ids.IsCASLObjectID(id) {
		return id, false
	}
	return id - ids.CASLObjectIDNamespaceStart, true
}

func isCASLIDField(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "obj_id", "objids", "obj_ids", "objectid", "object_id", "objectids", "alarmid", "alarmids":
		return true
	default:
		return false
	}
}

func parseIntString(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	var parsed int
	_, _ = fmt.Sscanf(value, "%d", &parsed)
	return parsed
}

func firstInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func asWSString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	default:
		return fmt.Sprint(typed)
	}
}

func firstWSValue(values ...any) any {
	for _, value := range values {
		if strings.TrimSpace(asWSString(value)) != "" {
			return value
		}
	}
	return nil
}

func cloneWSMessage(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}
