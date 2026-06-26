// Package broker provides a minimal ZeroMQ pub/sub client for CASL services.
//
// The CASL broker is an XPUB/XSUB proxy:
//   - services connect their PUB socket to the broker's XSUB port (subPort) to publish
//   - services connect their SUB socket to the broker's XPUB port (pubPort) to subscribe
//
// Every message on the wire is two ZMQ frames: [topic, jsonPayload].
package broker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/go-zeromq/zmq4"
)

// Client is a combined ZMQ publisher + subscriber connected to a CASL broker.
type Client struct {
	pub    zmq4.Socket
	sub    zmq4.Socket
	msgID  string // prefix for generated msg_id values
	cancel context.CancelFunc
}

// New creates and connects a Client.
// pubPort is the broker XPUB port (services subscribe here).
// subPort is the broker XSUB port (services publish here).
func New(ctx context.Context, host string, pubPort, subPort int, msgIDPrefix string) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)

	pub := zmq4.NewPub(ctx)
	pubEndpoint := "tcp://" + net.JoinHostPort(host, strconv.Itoa(subPort))
	if err := pub.Dial(pubEndpoint); err != nil {
		cancel()
		return nil, fmt.Errorf("broker: dial pub endpoint %s: %w", pubEndpoint, err)
	}

	sub := zmq4.NewSub(ctx)
	subEndpoint := "tcp://" + net.JoinHostPort(host, strconv.Itoa(pubPort))
	if err := sub.Dial(subEndpoint); err != nil {
		_ = pub.Close()
		cancel()
		return nil, fmt.Errorf("broker: dial sub endpoint %s: %w", subEndpoint, err)
	}

	return &Client{
		pub:    pub,
		sub:    sub,
		msgID:  msgIDPrefix,
		cancel: cancel,
	}, nil
}

// Subscribe adds a topic filter on the SUB socket.
func (c *Client) Subscribe(topics ...string) error {
	for _, t := range topics {
		if err := c.sub.SetOption(zmq4.OptionSubscribe, t); err != nil {
			return fmt.Errorf("broker: subscribe %q: %w", t, err)
		}
	}
	return nil
}

// Publish sends a two-frame message [topic, jsonPayload] on the PUB socket.
func (c *Client) Publish(topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("broker: marshal payload for %q: %w", topic, err)
	}
	msg := zmq4.NewMsgFrom([]byte(topic), body)
	if err := c.pub.Send(msg); err != nil {
		return fmt.Errorf("broker: send to %q: %w", topic, err)
	}
	return nil
}

// Recv blocks until a message arrives on the SUB socket.
// Returns (topic, rawJSON, error).
func (c *Client) Recv() (string, []byte, error) {
	msg, err := c.sub.Recv()
	if err != nil {
		if errors.Is(err, zmq4.ErrClosedConn) {
			return "", nil, context.Canceled
		}
		return "", nil, fmt.Errorf("broker: recv: %w", err)
	}
	if len(msg.Frames) < 2 {
		return string(msg.Frames[0]), nil, nil
	}
	return string(msg.Frames[0]), msg.Frames[1], nil
}

// MsgID generates a unique message ID with the client's prefix.
func (c *Client) MsgID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return c.msgID + "-" + hex.EncodeToString(b)
}

// Close shuts down the client.
func (c *Client) Close() {
	c.cancel()
	_ = c.pub.Close()
	_ = c.sub.Close()
}
