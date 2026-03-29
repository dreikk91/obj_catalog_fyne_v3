package casl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type RealtimeService struct {
	client *APIClient

	mu        sync.Mutex
	cancel    context.CancelFunc
	running   bool
	connected bool

	onEvent func(event ObjectEvent)
}

func NewRealtimeService(client *APIClient) *RealtimeService {
	return &RealtimeService{
		client: client,
	}
}

func (s *RealtimeService) SetEventHandler(h func(event ObjectEvent)) {
	s.onEvent = h
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

func (s *RealtimeService) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
	s.mu.Unlock()
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
			// log.Debug().Err(err).Msg("CASL realtime stream: reconnect")
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
	var msg any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	if connID := s.extractConnID(msg); connID != "" {
		go s.subscribe(ctx, connID)
	}
}

func (s *RealtimeService) extractConnID(msg any) string {
	return ""
}

func (s *RealtimeService) subscribe(ctx context.Context, connID string) {
	tags := []string{"ppk_in", "user_action", "system_event"}
	for _, tag := range tags {
		payload := map[string]any{
			"conn_id": connID,
			"tag":     tag,
		}
		_ = s.client.PostCommand(ctx, payload, nil, true)
	}
}
