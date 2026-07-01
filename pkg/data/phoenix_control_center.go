package data

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	phoenixPacketSeparator = "[*]"
	phoenixPingMarker      = "EVENT_CHECK_PING_DO_Z82"
	phoenixPingInterval    = 15 * time.Second
	phoenixCommandTimeout  = 3 * time.Second
)

type phoenixControlPacket struct {
	fields []string
	raw    string
	ack    bool
	action string
	panel  string
	event  string
	state  string
}

func parsePhoenixControlPacket(raw []byte) phoenixControlPacket {
	text := string(raw)
	fields := strings.Split(text, phoenixPacketSeparator)
	return phoenixControlPacket{
		fields: fields,
		raw:    text,
		ack:    phoenixPacketField(fields, 0) == "1",
		action: phoenixPacketField(fields, 5),
		panel:  phoenixPacketField(fields, 6),
		event:  phoenixPacketField(fields, 11),
		state:  phoenixPacketField(fields, 12),
	}
}

func phoenixPacketField(fields []string, index int) string {
	if index < 0 || index >= len(fields) {
		return ""
	}
	return strings.TrimSpace(fields[index])
}

func (packet phoenixControlPacket) key() string {
	return strings.Join([]string{packet.action, packet.panel, packet.event, packet.state}, "\x00")
}

func phoenixControlAck(packet phoenixControlPacket, remote *net.UDPAddr) []byte {
	if len(packet.fields) == 0 {
		return nil
	}
	fields := append([]string(nil), packet.fields...)
	fields[0] = "1"
	if len(fields) > 3 && remote != nil {
		fields[3] = remote.IP.String()
	}
	return []byte(strings.Join(fields, phoenixPacketSeparator))
}

func (p *PhoenixDataProvider) startControlCenterSession() error {
	if p == nil {
		return fmt.Errorf("phoenix UDP: провайдер не ініціалізований")
	}

	p.operatorMu.RLock()
	host := strings.TrimSpace(p.controlCenterHost)
	controlPort := p.controlPort
	clientPort := p.clientPort
	p.operatorMu.RUnlock()
	if host == "" {
		return fmt.Errorf("phoenix UDP: не вказана адреса центру керування")
	}
	if controlPort <= 0 || clientPort <= 0 {
		return fmt.Errorf("phoenix UDP: невалідні порти з PortSettings (центр %d, оператор %d)", controlPort, clientPort)
	}

	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(controlPort)))
	if err != nil {
		return fmt.Errorf("phoenix UDP: адреса центру керування: %w", err)
	}
	localIP := phoenixRouteLocalIP(remote)
	source, _ := os.Hostname()
	if source = strings.TrimSpace(source); source == "" {
		source = "OBJCATALOG"
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: clientPort})
	if err != nil {
		return fmt.Errorf("phoenix UDP: прослуховування Duty Operator порту %d: %w", clientPort, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.controlMu.Lock()
	if p.controlConn != nil {
		p.controlMu.Unlock()
		cancel()
		_ = conn.Close()
		return nil
	}
	p.controlConn = conn
	p.controlRemote = remote
	p.controlCancel = cancel
	p.controlLocalIP = localIP
	p.controlSource = source
	p.controlMu.Unlock()

	p.controlWG.Add(2)
	go p.readControlCenter(ctx, conn)
	go p.controlCenterHeartbeat(ctx)
	log.Info().
		Str("remote", remote.String()).
		Int("localPort", clientPort).
		Str("localIP", localIP).
		Msg("Phoenix UDP: сеанс оператора запущено")
	return nil
}

func phoenixRouteLocalIP(remote *net.UDPAddr) string {
	if remote == nil {
		return ""
	}
	conn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return ""
	}
	defer conn.Close()
	local, _ := conn.LocalAddr().(*net.UDPAddr)
	if local == nil {
		return ""
	}
	return local.IP.String()
}

func (p *PhoenixDataProvider) readControlCenter(ctx context.Context, conn *net.UDPConn) {
	defer p.controlWG.Done()
	buf := make([]byte, 64*1024)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() == nil && !errors.Is(err, net.ErrClosed) {
				log.Error().Err(err).Msg("Phoenix UDP: помилка читання")
			}
			return
		}
		p.controlMu.Lock()
		expected := p.controlRemote
		p.controlMu.Unlock()
		if expected == nil || remote.Port != expected.Port || !remote.IP.Equal(expected.IP) {
			log.Warn().Str("remote", remote.String()).Msg("Phoenix UDP: пакет не від налаштованого центру керування")
			continue
		}
		packet := parsePhoenixControlPacket(buf[:n])
		if packet.ack {
			p.resolveControlPending(packet.key())
		} else {
			if ack := phoenixControlAck(packet, remote); len(ack) > 0 {
				if _, err := conn.WriteToUDP(ack, remote); err != nil {
					log.Warn().Err(err).Str("action", packet.action).Msg("Phoenix UDP: не вдалося надіслати ACK")
				}
			}
		}
		if phoenixPacketChangesData(packet.action) {
			p.controlRevision.Add(1)
			p.invalidatePhoenixCaches()
		}
		log.Debug().
			Str("remote", remote.String()).
			Str("action", packet.action).
			Str("panel", packet.panel).
			Bool("ack", packet.ack).
			Msg("Phoenix UDP: пакет центру керування")
	}
}

func phoenixPacketChangesData(action string) bool {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "", "PING", "CONNECTED", "USERLOGIN":
		return false
	default:
		return true
	}
}

func (p *PhoenixDataProvider) controlCenterHeartbeat(ctx context.Context) {
	defer p.controlWG.Done()
	ticker := time.NewTicker(phoenixPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if _, operator, err := p.alarmOperatorIdentity(); err == nil {
				if err := p.writeControlPacket(p.phoenixPresencePayload("PING", operator, now)); err != nil {
					log.Warn().Err(err).Msg("Phoenix UDP: PING не надіслано")
				}
			}
		}
	}
}

func (p *PhoenixDataProvider) announceControlCenterOperator() {
	operatorID, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return
	}
	now := time.Now()
	if err := p.writeControlPacket(p.phoenixLoginPayload(operatorID, operatorName, now)); err != nil {
		log.Warn().Err(err).Msg("Phoenix UDP: USERLOGIN не надіслано")
		return
	}
	if err := p.writeControlPacket(p.phoenixPresencePayload("CONNECTED", operatorName, now)); err != nil {
		log.Warn().Err(err).Msg("Phoenix UDP: CONNECTED не надіслано")
	}
}

func (p *PhoenixDataProvider) phoenixLoginPayload(operatorID int64, operatorName string, now time.Time) []byte {
	source, host, localIP := p.controlEndpointFields()
	clientCode := p.controlCenterClientCode()
	note := strings.Join([]string{operatorName, strconv.FormatInt(operatorID, 10), localIP, ""}, "\r\n")
	return phoenixJoinFields([]string{
		"0", source, clientCode, host, "CU", "USERLOGIN", "", "0", "0", "", "", "0", "0",
		phoenixControlTime(now), note, host, "0", localIP, "", "",
	})
}

func (p *PhoenixDataProvider) phoenixPresencePayload(action, operatorName string, now time.Time) []byte {
	source, host, localIP := p.controlEndpointFields()
	clientCode := p.controlCenterClientCode()
	message := ""
	if action == "PING" {
		source = localIP
		message = phoenixPingMarker
	}
	return phoenixJoinFields([]string{
		"0", source, clientCode, host, "CU", action, "", "0", "0", "", message, "0", "0",
		phoenixControlTime(now), "", host, "0", localIP, operatorName, "",
	})
}

func (p *PhoenixDataProvider) phoenixAlarmEventPayload(
	panelID string,
	eventID int64,
	state int64,
	status string,
	operatorName string,
	now time.Time,
) []byte {
	source, host, localIP := p.controlEndpointFields()
	clientCode := p.controlCenterClientCode()
	return phoenixJoinFields([]string{
		"0", source, clientCode, host, "CU", "EVENT", strings.TrimSpace(panelID), "1", "0", "", "",
		strconv.FormatInt(eventID, 10), strconv.FormatInt(state, 10), phoenixControlTime(now),
		status, host, "0", localIP, strings.TrimSpace(operatorName), "",
	})
}

func (p *PhoenixDataProvider) phoenixStatusObjectPayload(eventID int64, operatorName string, now time.Time) []byte {
	source, host, localIP := p.controlEndpointFields()
	clientCode := p.controlCenterClientCode()
	return phoenixJoinFields([]string{
		"0", source, clientCode, host, "CU", "STATUS_OBJECT", "", "0", "1", "", "",
		strconv.FormatInt(eventID, 10), "0", phoenixControlTime(now), "", host, "0", localIP,
		strings.TrimSpace(operatorName), "",
	})
}

func (p *PhoenixDataProvider) controlCenterClientCode() string {
	p.operatorMu.RLock()
	defer p.operatorMu.RUnlock()
	if strings.TrimSpace(p.controlClientCode) == "" {
		return "OP"
	}
	return p.controlClientCode
}

func (p *PhoenixDataProvider) controlEndpointFields() (source, host, localIP string) {
	p.controlMu.Lock()
	source = p.controlSource
	localIP = p.controlLocalIP
	if p.controlRemote != nil {
		host = p.controlRemote.IP.String()
	}
	p.controlMu.Unlock()
	return source, host, localIP
}

func phoenixJoinFields(fields []string) []byte {
	return []byte(strings.Join(fields, phoenixPacketSeparator))
}

func phoenixControlTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return t.Format("02.01.2006 15:04:05")
}

func (p *PhoenixDataProvider) sendControlCommand(ctx context.Context, body []byte) error {
	packet := parsePhoenixControlPacket(body)
	wait := make(chan struct{}, 1)
	p.addControlPending(packet.key(), wait)
	defer p.removeControlPending(packet.key(), wait)

	if err := p.writeControlPacket(body); err != nil {
		return err
	}
	timeout := time.NewTimer(phoenixCommandTimeout)
	defer timeout.Stop()
	select {
	case <-wait:
		p.invalidatePhoenixCaches()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timeout.C:
		return fmt.Errorf("phoenix UDP: центр керування не підтвердив %s", packet.action)
	}
}

func (p *PhoenixDataProvider) writeControlPacket(body []byte) error {
	p.controlMu.Lock()
	conn := p.controlConn
	remote := p.controlRemote
	p.controlMu.Unlock()
	if conn == nil || remote == nil {
		return fmt.Errorf("phoenix UDP: сеанс з центром керування не запущено")
	}
	if _, err := conn.WriteToUDP(body, remote); err != nil {
		return fmt.Errorf("phoenix UDP: надсилання %s: %w", parsePhoenixControlPacket(body).action, err)
	}
	return nil
}

func (p *PhoenixDataProvider) addControlPending(key string, wait chan struct{}) {
	p.controlPendingMu.Lock()
	p.controlPending[key] = append(p.controlPending[key], wait)
	p.controlPendingMu.Unlock()
}

func (p *PhoenixDataProvider) resolveControlPending(key string) {
	p.controlPendingMu.Lock()
	waits := p.controlPending[key]
	delete(p.controlPending, key)
	p.controlPendingMu.Unlock()
	for _, wait := range waits {
		select {
		case wait <- struct{}{}:
		default:
		}
	}
}

func (p *PhoenixDataProvider) removeControlPending(key string, target chan struct{}) {
	p.controlPendingMu.Lock()
	defer p.controlPendingMu.Unlock()
	waits := p.controlPending[key]
	for i, wait := range waits {
		if wait == target {
			waits = append(waits[:i], waits[i+1:]...)
			break
		}
	}
	if len(waits) == 0 {
		delete(p.controlPending, key)
	} else {
		p.controlPending[key] = waits
	}
}

func (p *PhoenixDataProvider) invalidatePhoenixCaches() {
	if p == nil {
		return
	}
	p.objectMu.Lock()
	p.cachedObjectsAt = time.Time{}
	p.latestProbeAt = time.Time{}
	p.objectMu.Unlock()
}

func (p *PhoenixDataProvider) activePhoenixAlarmEventID(ctx context.Context, panelID string) int64 {
	if p == nil || p.db == nil {
		return 0
	}
	var eventID int64
	_ = p.db.GetContext(ctx, &eventID, `
SELECT TOP 1 Event_id
FROM Temp WITH (NOLOCK)
WHERE Panel_id = @p1 AND COALESCE(StateEvent, 0) IN (0, 1, 2, 3)
ORDER BY Event_id DESC`, panelID)
	return eventID
}

func (p *PhoenixDataProvider) sendAlarmState(
	ctx context.Context,
	panelID string,
	eventID int64,
	state int64,
	status string,
	operatorName string,
	statusObject bool,
) error {
	if eventID <= 0 {
		return fmt.Errorf("phoenix UDP: для об'єкта %s не знайдено активну подію", panelID)
	}
	now := time.Now()
	if err := p.sendControlCommand(ctx, p.phoenixAlarmEventPayload(
		panelID, eventID, state, status, operatorName, now,
	)); err != nil {
		return err
	}
	if statusObject {
		if err := p.sendControlCommand(ctx, p.phoenixStatusObjectPayload(eventID, operatorName, now)); err != nil {
			return err
		}
	}
	return nil
}

func phoenixResponseGroupNotifyStatus(groupID int64) string {
	if groupID <= 0 {
		return ""
	}
	return fmt.Sprintf("%d\n", groupID)
}

func (p *PhoenixDataProvider) stopControlCenterSession() {
	if p == nil {
		return
	}
	p.controlMu.Lock()
	cancel := p.controlCancel
	conn := p.controlConn
	p.controlCancel = nil
	p.controlConn = nil
	p.controlRemote = nil
	p.controlMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close()
	}
	p.controlWG.Wait()
}

// Shutdown stops the Phoenix Duty Operator UDP listener and heartbeat.
func (p *PhoenixDataProvider) Shutdown() {
	p.stopControlCenterSession()
}
