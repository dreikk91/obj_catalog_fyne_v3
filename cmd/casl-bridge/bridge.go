package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/broker"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

// Bridge polls Firebird and Phoenix, translates events to CASL broker messages.
type Bridge struct {
	client   *broker.Client
	bridge   *data.DBDataProvider
	phoenix  *data.PhoenixDataProvider
	cfg      bridgeConfig
	moduleID string
	casl     *CASLClient

	pendingMu sync.Mutex
	pending   map[string]chan json.RawMessage

	eventMu      sync.Mutex
	seenEvents   map[string]map[string]bool
	primedEvents map[string]bool
}

func newBridge(client *broker.Client, fb *data.DBDataProvider, ph *data.PhoenixDataProvider, cfg bridgeConfig) *Bridge {
	b := &Bridge{
		client:   client,
		bridge:   fb,
		phoenix:  ph,
		cfg:      cfg,
		moduleID: randomID(9),
		pending:  make(map[string]chan json.RawMessage),
		seenEvents: map[string]map[string]bool{
			"bridge":  {},
			"phoenix": {},
		},
		primedEvents: make(map[string]bool),
	}
	b.casl = NewCASLClient(b.apiRequest)
	return b
}

// Run starts all loops and blocks until ctx is cancelled.
func (b *Bridge) Run(ctx context.Context) {
	topics := []string{"api_in", "api_out"}
	if err := b.client.Subscribe(topics...); err != nil {
		log.Error().Err(err).Msg("bridge: subscribe failed")
		return
	}

	pollTick := time.NewTicker(b.cfg.PollInterval.Duration())
	heartbeatTick := time.NewTicker(b.cfg.HeartbeatInterval.Duration())
	monitorTick := time.NewTicker(10 * time.Second)
	defer pollTick.Stop()
	defer heartbeatTick.Stop()
	defer monitorTick.Stop()

	// recv loop in background
	recvErr := make(chan error, 1)
	go func() {
		for {
			topic, payload, err := b.client.Recv()
			if err != nil {
				recvErr <- err
				return
			}
			b.handleIncoming(ctx, topic, payload)
		}
	}()

	// start provisioner (creates missing objects in CASL DB)
	b.startProvisioner(ctx)
	phoenixWake := b.startPhoenixEventWatcher(ctx)

	// initial heartbeat
	b.sendHeartbeats(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-recvErr:
			if ctx.Err() == nil {
				log.Error().Err(err).Msg("bridge: broker recv error")
			}
			return
		case <-pollTick.C:
			b.pollEvents(ctx)
		case <-phoenixWake:
			b.pollPhoenixEvents(ctx)
		case <-heartbeatTick.C:
			b.sendHeartbeats(ctx)
		case <-monitorTick.C:
			b.sendMonitorHeartbeat()
		}
	}
}

// ── incoming api_in ───────────────────────────────────────────────────────────

func (b *Bridge) handleIncoming(_ context.Context, topic string, raw []byte) {
	switch topic {
	case "api_in":
		if len(raw) == 0 {
			return
		}
		var msg map[string]any
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		msgType, _ := msg["type"].(string)
		msgID, _ := msg["msg_id"].(string)
		switch msgType {
		case "get_devices_timeouts":
			b.replyDevicesTimeouts(msgID)
		}
	case "api_out":
		b.routeAPIOut(raw)
	}
}

func (b *Bridge) routeAPIOut(raw []byte) {
	var msg struct {
		MsgID string `json:"msg_id"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil || msg.MsgID == "" {
		return
	}
	b.pendingMu.Lock()
	ch, ok := b.pending[msg.MsgID]
	b.pendingMu.Unlock()
	if ok {
		dst := make([]byte, len(raw))
		copy(dst, raw)
		select {
		case ch <- json.RawMessage(dst):
		default:
		}
	}
}

func (b *Bridge) apiRequest(ctx context.Context, payload map[string]any) (map[string]any, error) {
	msgID := b.client.MsgID()
	payload["msg_id"] = msgID
	if _, ok := payload["ip_addr"]; !ok {
		payload["ip_addr"] = "127.0.0.1"
	}

	ch := make(chan json.RawMessage, 1)
	b.pendingMu.Lock()
	b.pending[msgID] = ch
	b.pendingMu.Unlock()
	defer func() {
		b.pendingMu.Lock()
		delete(b.pending, msgID)
		b.pendingMu.Unlock()
	}()

	if err := b.client.Publish("api_in", payload); err != nil {
		return nil, fmt.Errorf("api_in publish: %w", err)
	}

	select {
	case raw := <-ch:
		var result map[string]any
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("api_out unmarshal: %w", err)
		}
		return result, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("api request timeout: type=%v msg_id=%s", payload["type"], msgID)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *Bridge) replyDevicesTimeouts(replyTo string) {
	type entry struct {
		Number  int `json:"number"`
		Timeout int `json:"timeout"`
	}

	var items []entry
	timeout := int(b.cfg.DeviceTimeout.Duration().Seconds())

	if b.bridge != nil {
		for _, obj := range b.bridge.GetObjects() {
			if n, ok := ids.BridgePPKNum(obj.DisplayNumber); ok {
				items = append(items, entry{Number: n, Timeout: timeout})
			}
		}
	}
	if b.phoenix != nil {
		for _, obj := range b.phoenix.GetObjects() {
			if n, ok := ids.PhoenixPPKNum(obj.DisplayNumber); ok {
				items = append(items, entry{Number: n, Timeout: timeout})
			}
		}
	}

	payload := map[string]any{
		"msg_id": replyTo,
		"data":   items,
	}
	if err := b.client.Publish("api_out", payload); err != nil {
		log.Error().Err(err).Msg("bridge: reply get_devices_timeouts")
	}
}

// ── event polling ─────────────────────────────────────────────────────────────

func (b *Bridge) pollEvents(ctx context.Context) {
	if b.bridge != nil {
		b.publishFreshEvents(ctx, b.bridge.GetEvents(), "bridge")
	}
	b.pollPhoenixEvents(ctx)
}

func (b *Bridge) pollPhoenixEvents(ctx context.Context) {
	if b.phoenix != nil {
		b.publishFreshEvents(ctx, b.phoenix.GetEvents(), "phoenix")
	}
}

func (b *Bridge) publishFreshEvents(ctx context.Context, events []models.Event, sourceModel string) {
	if len(events) == 0 {
		return
	}

	fresh := make([]models.Event, 0, len(events))
	b.eventMu.Lock()
	seen := b.seenEvents[sourceModel]
	if seen == nil {
		seen = make(map[string]bool)
		b.seenEvents[sourceModel] = seen
	}
	primed := b.primedEvents[sourceModel]
	for _, ev := range events {
		key := eventKey(ev)
		if seen[key] {
			continue
		}
		seen[key] = true
		if primed || b.cfg.PublishInitialEvents {
			fresh = append(fresh, ev)
		}
	}
	if !primed {
		b.primedEvents[sourceModel] = true
	}
	b.eventMu.Unlock()

	for _, ev := range fresh {
		b.publishEvent(ctx, ev, sourceModel)
	}
}

func (b *Bridge) publishEvent(_ context.Context, ev models.Event, sourceModel string) {
	ppkNum := b.resolvePPKNum(ev.ObjectNumber, sourceModel)
	if ppkNum == 0 {
		return
	}

	code, _ := eventCode(ev.Type)

	payload := map[string]any{
		"msg_id":  b.client.MsgID(),
		"ppk_num": ppkNum,
		"time":    ev.Time.UnixMilli(),
		"code":    code,
		"model":   sourceModel,
	}
	if ev.ZoneNumber > 0 {
		payload["number"] = ev.ZoneNumber
	}

	if err := b.client.Publish("ppk_in", payload); err != nil {
		log.Error().Err(err).Int("ppk_num", ppkNum).Msg("bridge: publish ppk_in")
	} else {
		log.Debug().Int("ppk_num", ppkNum).Str("type", string(ev.Type)).Int("code", code).Msg("bridge: ppk_in published")
	}
}

// ── heartbeats ────────────────────────────────────────────────────────────────

func (b *Bridge) sendHeartbeats(_ context.Context) {
	now := time.Now().UnixMilli()

	if b.bridge != nil {
		for _, obj := range b.bridge.GetObjects() {
			n, ok := ids.BridgePPKNum(obj.DisplayNumber)
			if !ok {
				continue
			}
			payload := map[string]any{
				"msg_id":  b.client.MsgID(),
				"ppk_num": n,
				"time":    now,
				"model":   "bridge",
				"code":    "ping",
			}
			if err := b.client.Publish("ppk_empty", payload); err != nil {
				log.Error().Err(err).Int("ppk_num", n).Msg("bridge: publish ppk_empty")
			}
		}
	}

	if b.phoenix != nil {
		for _, obj := range b.phoenix.GetObjects() {
			n, ok := ids.PhoenixPPKNum(obj.DisplayNumber)
			if !ok {
				continue
			}
			payload := map[string]any{
				"msg_id":  b.client.MsgID(),
				"ppk_num": n,
				"time":    now,
				"model":   "phoenix",
				"code":    "ping",
			}
			if err := b.client.Publish("ppk_empty", payload); err != nil {
				log.Error().Err(err).Int("ppk_num", n).Msg("bridge: publish ppk_empty")
			}
		}
	}
}

func (b *Bridge) sendMonitorHeartbeat() {
	payload := map[string]any{
		"data":   "module",
		"id":     b.moduleID,
		"module": "casl-bridge",
	}
	if err := b.client.Publish("monitor_in", payload); err != nil {
		log.Error().Err(err).Msg("bridge: publish monitor_in")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (b *Bridge) resolvePPKNum(objectNumber, sourceModel string) int {
	objectNumber = strings.TrimSpace(objectNumber)
	if objectNumber == "" {
		return 0
	}
	switch sourceModel {
	case "bridge":
		n, ok := ids.BridgePPKNum(objectNumber)
		if !ok {
			// try parsing raw int as fallback
			if v, err := strconv.Atoi(objectNumber); err == nil && v > 0 {
				return v
			}
		}
		return n
	case "phoenix":
		n, _ := ids.PhoenixPPKNum(objectNumber)
		return n
	}
	return 0
}

func randomID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, n)
	_, _ = randRead(b)
	for i, v := range b {
		b[i] = chars[int(v)%len(chars)]
	}
	return string(b)
}

func eventKey(ev models.Event) string {
	if ev.ID != 0 {
		return strconv.Itoa(ev.ID)
	}
	return strings.Join([]string{
		strconv.FormatInt(ev.Time.UnixMilli(), 10),
		ev.ObjectNumber,
		string(ev.Type),
		strconv.Itoa(ev.ZoneNumber),
		ev.Details,
	}, "\x00")
}
