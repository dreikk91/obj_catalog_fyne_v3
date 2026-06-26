package ami

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Host      string
	Port      int
	Username  string
	Secret    string
	Extension string // operator extension/group, e.g. 8880
	Context   string // from-internal
}

type AMIEvent struct {
	Name   string
	Fields map[string]string
}

type CallSession struct {
	ActionID string
	Channels map[string]bool
}

type Client struct {
	cfg Config

	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex

	actionID atomic.Int64

	Events chan AMIEvent
	stopCh chan struct{}

	sessions map[string]*CallSession
	sessMu   sync.Mutex

	closeOnce sync.Once
}

func normalizeCfg(cfg Config) Config {
	if cfg.Port == 0 {
		cfg.Port = 5038
	}
	if strings.TrimSpace(cfg.Context) == "" {
		cfg.Context = "from-internal"
	}
	if strings.TrimSpace(cfg.Extension) == "" {
		cfg.Extension = "8880"
	}
	return cfg
}

func newBaseClient(cfg Config) *Client {
	return &Client{
		cfg:      normalizeCfg(cfg),
		Events:   make(chan AMIEvent, 256),
		stopCh:   make(chan struct{}),
		sessions: make(map[string]*CallSession),
	}
}

func NewClient(cfg Config) (*Client, error) {
	c := newBaseClient(cfg)
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewClientLazy creates AMI client without failing startup when AMI is down.
// It returns a client instance and starts background reconnect attempts.
func NewClientLazy(cfg Config) (*Client, error) {
	c := newBaseClient(cfg)
	go c.reconnectLoop()
	return c, nil
}

func (c *Client) connect() error {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(conn)

	// greeting
	_, _ = reader.ReadString('\n')

	login := fmt.Sprintf(
		"Action: Login\r\nUsername: %s\r\nSecret: %s\r\nEvents: on\r\n\r\n",
		c.cfg.Username, c.cfg.Secret,
	)

	if _, err := fmt.Fprint(conn, login); err != nil {
		_ = conn.Close()
		return err
	}

	resp, err := readMessage(reader)
	if err != nil {
		_ = conn.Close()
		return err
	}

	if resp["Response"] != "Success" {
		_ = conn.Close()
		return fmt.Errorf("AMI login failed: %s", resp["Message"])
	}

	c.mu.Lock()
	oldConn := c.conn
	c.conn = conn
	c.reader = reader
	c.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}

	log.Println("[AMI] connected")
	go c.readLoop(conn, reader)

	return nil
}

// Originate starts call to destination through an operator group.
// Returns action/call ID that can be passed to Hangup(callID).
func (c *Client) Originate(destination string, group string) (string, error) {
	return c.originate(destination, group, "Alarm Monitor")
}

// OriginateCall is UI-friendly wrapper using configured extension/group and callerID.
func (c *Client) OriginateCall(destination string, callerID string) error {
	group := strings.TrimSpace(c.cfg.Extension)
	_, err := c.originate(destination, group, callerID)
	return err
}

func (c *Client) originate(destination string, group string, callerID string) (string, error) {
	destination = strings.TrimSpace(destination)
	group = strings.TrimSpace(group)
	callerID = strings.TrimSpace(callerID)

	if destination == "" {
		return "", fmt.Errorf("empty destination")
	}
	if group == "" {
		group = c.cfg.Extension
	}
	if group == "" {
		group = "8880"
	}
	if callerID == "" {
		callerID = "Alarm Monitor"
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		go c.reconnectLoop()
		return "", fmt.Errorf("AMI not connected")
	}

	actionID := fmt.Sprintf("orig-%d", c.actionID.Add(1))

	msg := strings.Join([]string{
		"Action: Originate",
		fmt.Sprintf("Channel: Local/%s@%s", group, c.cfg.Context),
		fmt.Sprintf("Exten: %s", destination),
		fmt.Sprintf("Context: %s", c.cfg.Context),
		"Priority: 1",
		fmt.Sprintf("CallerID: %s", callerID),
		fmt.Sprintf("Variable: __CALL_ID=%s", actionID),
		fmt.Sprintf("Variable: __ORIGINATE_ID=%s", actionID),
		"Timeout: 30000",
		"Async: true",
		fmt.Sprintf("ActionID: %s", actionID),
		"", "",
	}, "\r\n")

	if _, err := fmt.Fprint(c.conn, msg); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		go c.reconnectLoop()
		return "", err
	}

	c.sessMu.Lock()
	c.sessions[actionID] = &CallSession{
		ActionID: actionID,
		Channels: make(map[string]bool),
	}
	c.sessMu.Unlock()

	log.Printf("[AMI] CALL -> group %s -> %s", group, destination)
	return actionID, nil
}

func (c *Client) Hangup(actionID string) {
	c.sessMu.Lock()
	session := c.sessions[actionID]
	c.sessMu.Unlock()
	if session == nil {
		return
	}

	for ch := range session.Channels {
		c.sendHangup(ch)
	}
}

// HangupAll sends Hangup for all tracked channels.
func (c *Client) HangupAll() error {
	c.sessMu.Lock()
	defer c.sessMu.Unlock()

	for _, sess := range c.sessions {
		for ch := range sess.Channels {
			c.sendHangup(ch)
		}
	}
	return nil
}

func (c *Client) sendHangup(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return
	}

	msg := fmt.Sprintf("Action: Hangup\r\nChannel: %s\r\n\r\n", channel)
	_, _ = fmt.Fprint(c.conn, msg)
}

func (c *Client) readLoop(expectedConn net.Conn, expectedReader *bufio.Reader) {
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		msg, err := readMessage(expectedReader)
		if err != nil {
			log.Println("[AMI] read error:", err)

			c.mu.Lock()
			if c.conn == expectedConn {
				c.conn = nil
				c.reader = nil
			}
			c.mu.Unlock()

			go c.reconnectLoop()
			return
		}

		if ev, ok := msg["Event"]; ok {
			c.handleEvent(ev, msg)
			select {
			case c.Events <- AMIEvent{Name: ev, Fields: msg}:
			default:
			}
		}
	}
}

func (c *Client) handleEvent(name string, f map[string]string) {
	actionID := extractOriginateID(f)
	channel := f["Channel"]

	if actionID == "" || channel == "" {
		return
	}

	c.sessMu.Lock()
	defer c.sessMu.Unlock()

	sess := c.sessions[actionID]
	if sess == nil {
		return
	}

	switch name {
	case "Newchannel":
		sess.Channels[channel] = true
	case "Hangup":
		delete(sess.Channels, channel)
		if len(sess.Channels) == 0 {
			delete(c.sessions, actionID)
			log.Printf("[AMI] call %s finished", actionID)
		}
	}
}

func extractOriginateID(fields map[string]string) string {
	candidates := []string{"Variable___ORIGINATE_ID", "__ORIGINATE_ID", "ORIGINATE_ID", "ActionID"}
	for _, k := range candidates {
		if v := strings.TrimSpace(fields[k]); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(fields["Variable"]); v != "" {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "__ORIGINATE_ID" {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func (c *Client) reconnectLoop() {
	for {
		select {
		case <-c.stopCh:
			return
		case <-time.After(5 * time.Second):
			if c.IsConnected() {
				return
			}
			log.Println("[AMI] reconnecting...")
			if err := c.connect(); err != nil {
				log.Println("[AMI] reconnect failed:", err)
				continue
			}
			log.Println("[AMI] reconnected")
			return
		}
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.stopCh)
		c.mu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
			c.reader = nil
		}
		c.mu.Unlock()
		close(c.Events)
	})
}

func (c *Client) Cfg() Config {
	return c.cfg
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *Client) ReconnectNow() {
	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.reader = nil
	}
	c.mu.Unlock()
	go c.reconnectLoop()
}

func readMessage(r *bufio.Reader) (map[string]string, error) {
	fields := make(map[string]string)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		if i := strings.Index(line, ": "); i != -1 {
			fields[line[:i]] = line[i+2:]
		}
	}

	return fields, nil
}
