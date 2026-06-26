package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-zeromq/zmq4"
)

type sniffRecord struct {
	Time            string `json:"time"`
	Topic           string `json:"topic"`
	Frames          int    `json:"frames"`
	PayloadEncoding string `json:"payload_encoding"`
	PayloadText     string `json:"payload_text,omitempty"`
	PayloadJSON     any    `json:"payload_json,omitempty"`
	Truncated       bool   `json:"truncated,omitempty"`
}

func main() {
	host := flag.String("host", "127.0.0.1", "CASL broker host")
	pubPort := flag.Int("pub", 27001, "CASL broker PUB port to subscribe to")
	topicsRaw := flag.String("topics", "", "comma-separated ZeroMQ topics; empty subscribes to all topics")
	outPath := flag.String("out", ".tmp/casl-broker-sniffer.ndjson", "NDJSON output path")
	maxBody := flag.Int("max-body", 1<<20, "maximum decoded payload bytes stored per record")
	flag.Parse()

	if err := run(*host, *pubPort, *topicsRaw, *outPath, *maxBody); err != nil {
		log.Fatal(err)
	}
}

func run(host string, pubPort int, topicsRaw, outPath string, maxBody int) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open output file: %w", err)
	}
	defer out.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sub := zmq4.NewSub(ctx)
	defer sub.Close()
	closeOnCancel := context.AfterFunc(ctx, func() {
		_ = sub.Close()
	})
	defer closeOnCancel()

	for _, topic := range parseTopics(topicsRaw) {
		if err := sub.SetOption(zmq4.OptionSubscribe, topic); err != nil {
			return fmt.Errorf("subscribe to topic %q: %w", topic, err)
		}
	}

	endpoint := "tcp://" + net.JoinHostPort(host, strconv.Itoa(pubPort))
	if err := sub.Dial(endpoint); err != nil {
		return fmt.Errorf("dial broker pub endpoint %s: %w", endpoint, err)
	}

	log.Printf("CASL broker sniffer listening on %s topics=%q output=%s", endpoint, topicsRaw, outPath)

	enc := json.NewEncoder(out)
	for {
		msg, err := sub.Recv()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, zmq4.ErrClosedConn) {
				return nil
			}
			return fmt.Errorf("receive broker message: %w", err)
		}

		record := recordFromMessage(msg, maxBody)
		if err := enc.Encode(record); err != nil {
			return fmt.Errorf("write output record: %w", err)
		}

		log.Printf("broker topic=%q frames=%d payload=%s", record.Topic, record.Frames, preview(record))
	}
}

func parseTopics(raw string) []string {
	if raw == "" {
		return []string{""}
	}

	seen := make(map[string]bool)
	topics := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		topic := strings.TrimSpace(part)
		if topic == "" {
			continue
		}
		if topic == "*" {
			topic = ""
		}
		if seen[topic] {
			continue
		}
		seen[topic] = true
		topics = append(topics, topic)
	}
	if len(topics) == 0 {
		return []string{""}
	}
	return topics
}

func recordFromMessage(msg zmq4.Msg, maxBody int) sniffRecord {
	record := sniffRecord{
		Time:   time.Now().Format(time.RFC3339Nano),
		Frames: len(msg.Frames),
	}
	if len(msg.Frames) == 0 {
		return record
	}

	record.Topic = string(msg.Frames[0])
	if len(msg.Frames) == 1 {
		record.PayloadText = string(msg.Frames[0])
		record.PayloadEncoding = "plain"
		return record
	}

	decoded := decodePayload(msg.Frames[1], maxBody)
	record.PayloadEncoding = decoded.Encoding
	record.Truncated = decoded.Truncated
	if decoded.JSON != nil {
		record.PayloadJSON = decoded.JSON
	} else {
		record.PayloadText = decoded.Text
	}
	return record
}

type decodedPayload struct {
	Encoding  string
	Text      string
	JSON      any
	Truncated bool
}

func decodePayload(payload []byte, maxBody int) decodedPayload {
	encoding := "plain"
	body := payload

	if len(payload) >= 2 && payload[0] == 0x1f && payload[1] == 0x8b {
		if unzipped, err := gunzip(payload); err == nil {
			body = unzipped
			encoding = "gzip"
		} else {
			encoding = "gzip-error"
		}
	}

	truncated := false
	if maxBody > 0 && len(body) > maxBody {
		body = body[:maxBody]
		truncated = true
	}

	var jsonValue any
	if !truncated && json.Unmarshal(body, &jsonValue) == nil {
		return decodedPayload{
			Encoding:  encoding,
			JSON:      jsonValue,
			Truncated: false,
		}
	}

	return decodedPayload{
		Encoding:  encoding,
		Text:      string(body),
		Truncated: truncated,
	}
}

func gunzip(payload []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func preview(record sniffRecord) string {
	if record.PayloadJSON != nil {
		raw, err := json.Marshal(record.PayloadJSON)
		if err == nil {
			return trimForLog(string(raw), 512)
		}
	}
	return trimForLog(record.PayloadText, 512)
}

func trimForLog(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", `\n`)
	text = strings.ReplaceAll(text, "\r", `\r`)
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
