package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient manages a persistent WebSocket connection to the Mattermost server.
type WSClient struct {
	mu        sync.Mutex
	token     string
	wsURL     string // e.g. "wss://netchat.viettel.vn/api/v4/websocket"
	conn      *websocket.Conn
	connected bool
	Events    chan WSEvent  // buffered channel (size 64); caller reads from this
	done      chan struct{} // closed when the reader goroutine stops
}

// authChallenge is the JSON structure for the Mattermost auth challenge message.
type authChallenge struct {
	Seq    int                    `json:"seq"`
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
}

// NewWSClient creates a WSClient. wsURL must start with "wss://".
func NewWSClient(token, wsURL string) (*WSClient, error) {
	if !strings.HasPrefix(wsURL, "wss://") {
		return nil, fmt.Errorf("api: wsURL must start with wss://")
	}
	return &WSClient{
		token:  token,
		wsURL:  wsURL,
		Events: make(chan WSEvent, 64),
		done:   make(chan struct{}),
	}, nil
}

// connectOnce dials the WebSocket, sends the auth challenge, and starts the
// reader goroutine. It is the shared implementation used by Connect and
// ConnectWithRetry. ctx is used for the dial so that cancellation works during
// the handshake.
func (ws *WSClient) connectOnce(ctx context.Context) error {
	// Fix 4 (SECURITY): use a private dialer instead of the mutable DefaultDialer.
	dialer := &websocket.Dialer{
		TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
		HandshakeTimeout: 10 * time.Second,
	}

	// Read wsURL under the mutex so we see any update from ConnectWithRetry.
	ws.mu.Lock()
	wsURL := ws.wsURL
	done := ws.done
	ws.mu.Unlock()

	// Fix 4 cont.: use DialContext so ctx cancellation aborts the dial.
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("api: websocket dial failed: %w", err)
	}

	// Build and send the auth challenge as the very first message.
	challenge := authChallenge{
		Seq:    1,
		Action: "authentication_challenge",
		Data:   map[string]interface{}{"token": ws.token},
	}
	payload, err := json.Marshal(challenge)
	if err != nil {
		conn.Close()
		return fmt.Errorf("api: marshalling auth challenge: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		conn.Close()
		return fmt.Errorf("api: sending auth challenge: %w", err)
	}

	// Fix 1 (DATA RACE): assign conn and done under the mutex, then start the
	// goroutine. The goroutine receives its own copies as arguments and never
	// reads ws.conn / ws.done directly.
	ws.mu.Lock()
	ws.conn = conn
	ws.done = done // keep done consistent (may have been reset by ConnectWithRetry)
	ws.connected = true
	ws.mu.Unlock()

	// Start the reader goroutine; it owns conn and done — not ws fields.
	go ws.readLoop(conn, done)

	return nil
}

// Connect establishes the WebSocket connection, sends the auth challenge,
// and starts the reader goroutine that pumps events into the Events channel.
// The auth challenge is the FIRST message sent after connection.
func (ws *WSClient) Connect() error {
	return ws.connectOnce(context.Background())
}

// readLoop reads JSON messages from the WebSocket in a loop, decodes them into
// WSEvent, and sends them to the Events channel. It is non-blocking: if the
// channel is full the event is dropped. On any read error the loop exits and
// done is closed.
//
// Fix 1 (DATA RACE): conn and done are passed as arguments so the goroutine
// owns its local copies and never races with other goroutines reading ws.conn
// or ws.done.
func (ws *WSClient) readLoop(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Connection closed or network error — exit without reconnecting.
			return
		}
		var event WSEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			// Skip malformed messages and keep reading.
			continue
		}
		// Non-blocking send: drop the event if the channel is full.
		select {
		case ws.Events <- event:
		default:
		}
	}
}

// Close cleanly shuts down the connection and the reader goroutine.
func (ws *WSClient) Close() {
	// Fix 1 + Fix 2: snapshot fields under the mutex.
	ws.mu.Lock()
	conn := ws.conn
	done := ws.done
	isConnected := ws.connected
	ws.mu.Unlock()

	// Fix 2 (DEADLOCK): if never connected, there is no reader goroutine
	// waiting on done, so we must not block on <-done.
	if !isConnected || conn == nil {
		return
	}

	// Send a close frame and wait briefly for the reader goroutine to finish.
	_ = conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	conn.Close()

	// Wait for the reader goroutine to stop.
	<-done
}

// ConnectWithRetry connects a WSClient with exponential backoff.
// Initial delay: 2s. Doubles each attempt. Max delay: 30s.
// Stops retrying when ctx is cancelled.
func ConnectWithRetry(ctx context.Context, ws *WSClient) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 30 * time.Second
	)
	delay := initialDelay
	for {
		// Fix 3 (UNSAFE RESET): reset all mutable state under the mutex at the
		// START of each iteration, before calling connectOnce. This ensures the
		// new done channel is in place before the goroutine might close the old
		// one, and that ws.conn is not stale from a previous attempt.
		ws.mu.Lock()
		ws.conn = nil
		ws.connected = false
		ws.done = make(chan struct{})
		ws.mu.Unlock()

		err := ws.connectOnce(ctx)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
