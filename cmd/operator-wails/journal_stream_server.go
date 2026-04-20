package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
	"obj_catalog_fyne_v3/pkg/wailsbridge"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	defaultJournalWSAddr   = "127.0.0.1:17891"
	defaultJournalWSPath   = "/ws/frontend/v1/journal"
	journalBroadcastTick   = 2 * time.Second
	journalBootstrapEvents = 100
	journalWriteTimeout    = 3 * time.Second
	journalShutdownTimeout = 3 * time.Second
	journalMaxInboundBytes = 1024
)

type journalStreamMessage struct {
	Kind               string                  `json:"kind"`
	Events             []frontendv1.EventItem  `json:"events,omitempty"`
	AlarmGroups        []frontendv1.AlarmGroup `json:"alarmGroups,omitempty"`
	AlarmGroupsChanged bool                    `json:"alarmGroupsChanged,omitempty"`
	SentAt             string                  `json:"sentAt"`
}

type journalCacheState struct {
	events          []frontendv1.EventItem
	eventKeys       map[string]struct{}
	alarmGroups     []frontendv1.AlarmGroup
	alarmGroupsHash string
}

type journalStreamServer struct {
	bridge *wailsbridge.FrontendV1Service
	server *http.Server

	cancel context.CancelFunc

	mu       sync.RWMutex
	clients  map[*websocket.Conn]struct{}
	upgrader websocket.Upgrader

	stateMu sync.RWMutex
	state   journalCacheState
}

func startJournalStreamServer(bridge *wailsbridge.FrontendV1Service) (*journalStreamServer, error) {
	if bridge == nil {
		return nil, errors.New("journal websocket bridge is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	instance := &journalStreamServer{
		bridge:  bridge,
		cancel:  cancel,
		clients: make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
	initialState, err := instance.fetchState()
	if err != nil {
		cancel()
		return nil, err
	}
	instance.state = initialState

	mux := http.NewServeMux()
	mux.HandleFunc(defaultJournalWSPath, instance.handleWS)
	instance.server = &http.Server{
		Addr:              defaultJournalWSAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", defaultJournalWSAddr)
	if err != nil {
		cancel()
		return nil, err
	}

	go instance.runBroadcaster(ctx)
	go func() {
		if serveErr := instance.server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Warn().Err(serveErr).Msg("Operator Wails: journal websocket server stopped with error")
		}
	}()

	log.Info().
		Str("addr", defaultJournalWSAddr).
		Str("path", defaultJournalWSPath).
		Msg("Operator Wails: journal websocket server started")

	return instance, nil
}

func (s *journalStreamServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Operator Wails: websocket upgrade failed")
		return
	}

	s.addClient(conn)
	if bootstrap, bootstrapErr := s.buildBootstrap(); bootstrapErr == nil {
		_ = s.writeJSON(conn, bootstrap)
	}

	go s.readLoop(conn)
}

func (s *journalStreamServer) readLoop(conn *websocket.Conn) {
	defer s.removeClient(conn)

	conn.SetReadLimit(journalMaxInboundBytes)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *journalStreamServer) runBroadcaster(ctx context.Context) {
	ticker := time.NewTicker(journalBroadcastTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nextState, err := s.fetchState()
			if err != nil {
				log.Debug().Err(err).Msg("Operator Wails: failed to fetch journal state")
				continue
			}

			message, ok := s.buildDelta(nextState)
			if !ok {
				continue
			}
			s.broadcast(message)
		}
	}
}

func (s *journalStreamServer) fetchState() (journalCacheState, error) {
	events, err := s.bridge.ListEvents()
	if err != nil {
		return journalCacheState{}, err
	}
	alarmGroups, err := s.bridge.ListAlarmGroups()
	if err != nil {
		return journalCacheState{}, err
	}

	state := journalCacheState{
		events:      slices.Clone(events),
		eventKeys:   make(map[string]struct{}, len(events)),
		alarmGroups: slices.Clone(alarmGroups),
	}
	for _, item := range events {
		state.eventKeys[journalEventKey(item)] = struct{}{}
	}
	state.alarmGroupsHash = hashAlarmGroups(alarmGroups)
	return state, nil
}

func (s *journalStreamServer) buildBootstrap() (journalStreamMessage, error) {
	state := s.snapshotState()
	if len(state.events) == 0 && len(state.alarmGroups) == 0 {
		freshState, err := s.fetchState()
		if err != nil {
			return journalStreamMessage{}, err
		}
		s.setState(freshState)
		state = freshState
	}

	return journalStreamMessage{
		Kind:               "bootstrap",
		Events:             latestEvents(state.events, journalBootstrapEvents),
		AlarmGroups:        slices.Clone(state.alarmGroups),
		AlarmGroupsChanged: true,
		SentAt:             time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *journalStreamServer) buildDelta(nextState journalCacheState) (journalStreamMessage, bool) {
	prevState := s.snapshotState()
	newEvents := diffNewEvents(prevState.eventKeys, nextState.events)
	alarmGroupsChanged := prevState.alarmGroupsHash != nextState.alarmGroupsHash
	s.setState(nextState)

	if len(newEvents) == 0 && !alarmGroupsChanged {
		return journalStreamMessage{}, false
	}

	message := journalStreamMessage{
		Kind:   "delta",
		Events: newEvents,
		SentAt: time.Now().UTC().Format(time.RFC3339),
	}
	if alarmGroupsChanged {
		message.AlarmGroups = slices.Clone(nextState.alarmGroups)
		message.AlarmGroupsChanged = true
	}
	return message, true
}

func (s *journalStreamServer) snapshotState() journalCacheState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return journalCacheState{
		events:          slices.Clone(s.state.events),
		eventKeys:       cloneStringSet(s.state.eventKeys),
		alarmGroups:     slices.Clone(s.state.alarmGroups),
		alarmGroupsHash: s.state.alarmGroupsHash,
	}
}

func (s *journalStreamServer) setState(state journalCacheState) {
	s.stateMu.Lock()
	s.state = journalCacheState{
		events:          slices.Clone(state.events),
		eventKeys:       cloneStringSet(state.eventKeys),
		alarmGroups:     slices.Clone(state.alarmGroups),
		alarmGroupsHash: state.alarmGroupsHash,
	}
	s.stateMu.Unlock()
}

func (s *journalStreamServer) broadcast(payload journalStreamMessage) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Debug().Err(err).Msg("Operator Wails: failed to marshal journal message")
		return
	}

	s.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for conn := range s.clients {
		clients = append(clients, conn)
	}
	s.mu.RUnlock()

	for _, conn := range clients {
		if writeErr := s.writeRaw(conn, data); writeErr != nil {
			s.removeClient(conn)
		}
	}
}

func (s *journalStreamServer) writeJSON(conn *websocket.Conn, payload journalStreamMessage) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.writeRaw(conn, data)
}

func (s *journalStreamServer) writeRaw(conn *websocket.Conn, data []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(journalWriteTimeout))
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (s *journalStreamServer) addClient(conn *websocket.Conn) {
	s.mu.Lock()
	s.clients[conn] = struct{}{}
	s.mu.Unlock()
}

func (s *journalStreamServer) removeClient(conn *websocket.Conn) {
	s.mu.Lock()
	if _, exists := s.clients[conn]; exists {
		delete(s.clients, conn)
	}
	s.mu.Unlock()
	_ = conn.Close()
}

func (s *journalStreamServer) shutdown() {
	if s == nil {
		return
	}

	if s.cancel != nil {
		s.cancel()
	}

	s.mu.Lock()
	for conn := range s.clients {
		_ = conn.Close()
		delete(s.clients, conn)
	}
	s.mu.Unlock()

	if s.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), journalShutdownTimeout)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func latestEvents(events []frontendv1.EventItem, limit int) []frontendv1.EventItem {
	if len(events) == 0 {
		return []frontendv1.EventItem{}
	}

	items := slices.Clone(events)
	slices.SortFunc(items, compareFrontendEventsDesc)
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func diffNewEvents(previous map[string]struct{}, current []frontendv1.EventItem) []frontendv1.EventItem {
	if len(current) == 0 {
		return []frontendv1.EventItem{}
	}

	items := make([]frontendv1.EventItem, 0, len(current))
	for _, item := range current {
		if _, exists := previous[journalEventKey(item)]; exists {
			continue
		}
		items = append(items, item)
	}
	slices.SortFunc(items, compareFrontendEventsDesc)
	return items
}

func compareFrontendEventsDesc(left frontendv1.EventItem, right frontendv1.EventItem) int {
	leftTime := parseFrontendEventTime(left.Time)
	rightTime := parseFrontendEventTime(right.Time)
	switch {
	case leftTime.After(rightTime):
		return -1
	case leftTime.Before(rightTime):
		return 1
	case left.ID > right.ID:
		return -1
	case left.ID < right.ID:
		return 1
	default:
		return 0
	}
}

func parseFrontendEventTime(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func journalEventKey(item frontendv1.EventItem) string {
	return item.Time + "|" + item.TypeCode + "|" + strconv.Itoa(item.ObjectID) + "|" + strconv.Itoa(item.ID)
}

func hashAlarmGroups(groups []frontendv1.AlarmGroup) string {
	data, err := json.Marshal(groups)
	if err != nil {
		return ""
	}
	return string(data)
}

func cloneStringSet(source map[string]struct{}) map[string]struct{} {
	if len(source) == 0 {
		return map[string]struct{}{}
	}
	cloned := make(map[string]struct{}, len(source))
	for key := range source {
		cloned[key] = struct{}{}
	}
	return cloned
}
