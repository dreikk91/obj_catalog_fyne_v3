package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestCASLProvider_RealtimeWatchdogTimeout(t *testing.T) {
	// Set the watchdog timeout to a very small value to trigger it quickly in the test
	originalTimeout := caslRealtimeWatchdogTimeout
	caslRealtimeWatchdogTimeout = 200 * time.Millisecond
	defer func() {
		caslRealtimeWatchdogTimeout = originalTimeout
	}()

	// Channel to signal when client connects to WS
	wsChan := make(chan struct{})
	var closeOnce sync.Once

	// Mock WebSocket server
	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		// Signal that connection is accepted
		closeOnce.Do(func() {
			close(wsChan)
		})
		// Stay connected but send absolutely nothing to trigger the watchdog
		time.Sleep(1 * time.Second)
	}))
	defer wsServer.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")

	// Mock HTTP server for login and subscribe command
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			_, _ = w.Write([]byte(`{"status":"ok","token":"test-token","ws_url":"` + wsURL + `","user_id":"test-user"}`))
			return
		}
		if r.URL.Path == "/subscribe" {
			// Mock successful tag subscriptions
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
	}))
	defer httpServer.Close()

	provider := NewCASLCloudProvider(httpServer.URL, "", 1, "test@email.com", "password")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Call runRealtimeSession directly. It should connect, subscribe, and then
	// return a watchdog timeout error because no ping/message is received within 200ms.
	err := provider.runRealtimeSession(ctx)
	if err == nil {
		t.Fatal("expected watchdog timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "realtime stream watchdog timeout") {
		t.Fatalf("expected watchdog timeout error, got: %v", err)
	}
}
