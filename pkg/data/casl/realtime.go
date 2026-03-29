package casl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/websocket"
)

type RealtimeService struct {
	client *APIClient

	mu        sync.Mutex
	cancel    context.CancelFunc
	running   bool
	connected bool

	onEvents func(events []ObjectEvent)
}

func NewRealtimeService(client *APIClient) *RealtimeService {
	return &RealtimeService{
		client: client,
	}
}

func (s *RealtimeService) SetEventHandler(h func(events []ObjectEvent)) {
	s.onEvents = h
}

func (s *RealtimeService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	go s.runLoop(ctx)
}

func (s *RealtimeService) runLoop(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.connected = false
		s.mu.Unlock()
	}()

	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.runSession(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Debug().Err(err).Msg("CASL realtime stream: reconnect")
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > RealtimeBackoff {
			backoff = RealtimeBackoff
		}
	}
}

func (s *RealtimeService) runSession(ctx context.Context) error {
	_, wsURL, userID, _ := s.client.GetSessionInfo()
	if wsURL == "" {
		if _, err := s.client.EnsureToken(ctx); err != nil {
			return err
		}
		_, wsURL, userID, _ = s.client.GetSessionInfo()
	}

	if wsURL == "" {
		return fmt.Errorf("casl realtime: empty ws_url")
	}

	origin := s.client.BaseURL()
	if strings.HasPrefix(wsURL, "wss://") {
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

	s.mu.Lock()
	s.connected = true
	s.mu.Unlock()

	if err := s.sendGetID(conn, userID); err != nil {
		return fmt.Errorf("casl realtime send get_id: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var raw []byte
			if err := websocket.Message.Receive(conn, &raw); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			s.handleMessage(ctx, raw)
		}
	}
}

func (s *RealtimeService) sendGetID(conn *websocket.Conn, userID string) error {
	payload := map[string]any{
		"type": "get_id",
	}
	if userID != "" {
		payload["user_id"] = userID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return websocket.Message.Send(conn, body)
}

func (s *RealtimeService) handleMessage(ctx context.Context, raw []byte) {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 { return }
	if body[0] != '{' && body[0] != '[' {
		idx := bytes.IndexAny(body, "{[")
		if idx < 0 { return }
		body = body[idx:]
	}

	var msg any
	if err := json.Unmarshal(body, &msg); err != nil {
		return
	}

	if connID := s.extractConnID(msg); connID != "" {
		go s.subscribe(ctx, connID)
	}

	rows := make([]ObjectEvent, 0, 4)
	s.collectRealtimeRows(msg, "", &rows)
	if len(rows) > 0 && s.onEvents != nil {
		s.onEvents(rows)
	}
}

func (s *RealtimeService) collectRealtimeRows(value any, fallbackType string, rows *[]ObjectEvent) {
	switch typed := value.(type) {
	case map[string]any:
		nextType := fallbackType
		if tag := asString(typed["tag"]); tag != "" { nextType = tag }
		if ev := asString(typed["event"]); ev != "" { nextType = ev }
		if row, ok := s.mapRealtimeRow(typed, nextType); ok {
			*rows = append(*rows, row)
		}
		for _, nested := range typed {
			s.collectRealtimeRows(nested, nextType, rows)
		}
	case []any:
		for _, nested := range typed {
			s.collectRealtimeRows(nested, fallbackType, rows)
		}
	}
}

func (s *RealtimeService) mapRealtimeRow(source map[string]any, fallbackType string) (ObjectEvent, bool) {
	deviceID := asString(source["device_id"])
	if deviceID == "" { deviceID = asString(source["deviceId"]) }
	objID := asString(source["obj_id"])
	if objID == "" { objID = asString(source["object_id"]) }

	ppkNum := int64(parseAnyInt(source["ppk_num"]))
	if ppkNum <= 0 { ppkNum = int64(parseAnyInt(source["ppk"])) }

	if ppkNum <= 0 && deviceID == "" && objID == "" {
		return ObjectEvent{}, false
	}

	code := asString(source["code"])
	if code == "" { code = asString(source["action"]) }

	row := ObjectEvent{
		PPKNum:    Int64(ppkNum),
		DeviceID:  Text(deviceID),
		ObjID:     Text(objID),
		ObjName:   Text(asString(source["obj_name"])),
		Action:    Text(asString(source["action"])),
		Code:      Text(code),
		Type:      asString(source["type"]),
		Number:    Int64(parseAnyInt(source["number"])),
		ContactID: Text(asString(source["contact_id"])),
		Time:      Int64(parseAnyTime(source["time"]).UnixMilli()),
	}
	if row.Type == "" { row.Type = fallbackType }
	return row, true
}

func (s *RealtimeService) extractConnID(msg any) string {
	switch typed := msg.(type) {
	case map[string]any:
		rowType := strings.ToLower(asString(typed["type"]))
		if rowType == "conn_id" || rowType == "get_id" {
			if id := asString(typed["id"]); len(id) > 8 { return id }
		}
		for _, v := range typed {
			if id := s.extractConnID(v); id != "" { return id }
		}
	case []any:
		for _, v := range typed {
			if id := s.extractConnID(v); id != "" { return id }
		}
	}
	return ""
}

func (s *RealtimeService) subscribe(ctx context.Context, connID string) {
	tags := []string{"ppk_in", "user_action", "ppk_service", "system_event"}
	for _, tag := range tags {
		payload := map[string]any{
			"conn_id": connID,
			"tag":     tag,
		}
		_ = s.client.PostCommand(ctx, payload, nil, true)
	}
}

func parseAnyInt(value any) int {
	switch v := value.(type) {
	case int: return v
	case int64: return int(v)
	case float64: return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

func parseAnyTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time: return v
	case float64: return time.UnixMilli(int64(v))
	case int64: return time.UnixMilli(v)
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil { return t }
		i, err := strconv.ParseInt(v, 10, 64)
		if err == nil { return time.UnixMilli(i) }
	}
	return time.Time{}
}
