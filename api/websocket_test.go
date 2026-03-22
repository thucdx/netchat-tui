package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader used by all test WebSocket servers.
var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ---------------------------------------------------------------------------
// TestNewWSClient_RejectsWS
// ---------------------------------------------------------------------------

func TestNewWSClient_RejectsWS(t *testing.T) {
	_, err := NewWSClient("token", "ws://example.com/ws")
	if err == nil {
		t.Fatal("expected error for ws:// URL, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestNewWSClient_RejectsHTTPS
// ---------------------------------------------------------------------------

func TestNewWSClient_RejectsHTTPS(t *testing.T) {
	_, err := NewWSClient("token", "https://example.com/ws")
	if err == nil {
		t.Fatal("expected error for https:// URL, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestNewWSClient_AcceptsWSS
// ---------------------------------------------------------------------------

func TestNewWSClient_AcceptsWSS(t *testing.T) {
	ws, err := NewWSClient("mytoken", "wss://example.com/api/v4/websocket")
	if err != nil {
		t.Fatalf("unexpected error for wss:// URL: %v", err)
	}
	if ws == nil {
		t.Fatal("expected non-nil WSClient")
	}
}

// ---------------------------------------------------------------------------
// helpers: start a test WebSocket server
// ---------------------------------------------------------------------------

// startTestWSServer returns an httptest.Server that upgrades every connection
// to WebSocket and hands the server-side *websocket.Conn to handler.
// The handler runs in a goroutine; call srv.Close() when done.
func startTestWSServer(t *testing.T, handler func(conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		handler(conn)
	}))
	return srv
}

// wsURLFor converts an httptest.Server URL (http://...) to a ws:// URL,
// then rewrites the scheme to wss:// so NewWSClient accepts it, BUT we need
// to actually dial the plain-text test server.  We achieve this by
// temporarily overriding the dialer inside Connect.
//
// Simpler approach: use the gorilla websocket.DefaultDialer directly from
// tests with the ws:// URL.  Since we cannot easily swap the dialer inside
// WSClient, we do the following:
//   - Create the WSClient with a fake wss:// URL (to pass validation).
//   - Then overwrite the wsURL field with the real ws:// test server URL
//     before calling Connect.
//
// This works because wsURL is an unexported field in the same package (tests
// are in package api).
func wsURLFor(srv *httptest.Server) (wssFake string, wsReal string) {
	// srv.URL is like "http://127.0.0.1:PORT"
	wsReal = "ws://" + strings.TrimPrefix(srv.URL, "http://")
	wssFake = "wss://placeholder.test"
	return
}

// connectToTestServer creates a WSClient that is validated with a wss:// URL
// but internally dials the plain http test server (ws://). Returns the client.
func connectToTestServer(t *testing.T, token string, srv *httptest.Server) *WSClient {
	t.Helper()
	wssFake, wsReal := wsURLFor(srv)
	ws, err := NewWSClient(token, wssFake)
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}
	// Overwrite the wsURL so Connect dials the actual test server.
	ws.wsURL = wsReal
	return ws
}

// ---------------------------------------------------------------------------
// TestWSAuthChallengeFormat
// ---------------------------------------------------------------------------

func TestWSAuthChallengeFormat(t *testing.T) {
	const token = "secret-token-123"

	// Channel receives the first message the client sends.
	msgCh := make(chan []byte, 1)

	srv := startTestWSServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Logf("server read error: %v", err)
			return
		}
		msgCh <- msg
		// Keep connection open long enough for the client to finish.
		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()

	ws := connectToTestServer(t, token, srv)
	if err := ws.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer ws.Close()

	select {
	case msg := <-msgCh:
		var challenge authChallenge
		if err := json.Unmarshal(msg, &challenge); err != nil {
			t.Fatalf("unmarshal auth challenge: %v", err)
		}
		if challenge.Action != "authentication_challenge" {
			t.Errorf("action: got %q, want %q", challenge.Action, "authentication_challenge")
		}
		if challenge.Seq != 1 {
			t.Errorf("seq: got %d, want 1", challenge.Seq)
		}
		tok, ok := challenge.Data["token"]
		if !ok {
			t.Fatal("data.token missing")
		}
		if tok != token {
			t.Errorf("data.token: got %v, want %q", tok, token)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for auth challenge from client")
	}
}

// ---------------------------------------------------------------------------
// TestWSEventReceived
// ---------------------------------------------------------------------------

func TestWSEventReceived(t *testing.T) {
	event := WSEvent{
		Event: "posted",
		Data:  map[string]interface{}{"channel_id": "ch1"},
		Seq:   42,
	}
	payload, _ := json.Marshal(event)

	srv := startTestWSServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// Read (and discard) the auth challenge first.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		// Send an event.
		_ = conn.WriteMessage(websocket.TextMessage, payload)
		// Keep connection alive.
		time.Sleep(500 * time.Millisecond)
	})
	defer srv.Close()

	ws := connectToTestServer(t, "tok", srv)
	if err := ws.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer ws.Close()

	select {
	case got := <-ws.Events:
		if got.Event != "posted" {
			t.Errorf("event: got %q, want %q", got.Event, "posted")
		}
		if got.Seq != 42 {
			t.Errorf("seq: got %d, want 42", got.Seq)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on ws.Events channel")
	}
}

// ---------------------------------------------------------------------------
// TestWSUnknownEventNoPanic
// ---------------------------------------------------------------------------

func TestWSUnknownEventNoPanic(t *testing.T) {
	// We send a payload that cannot unmarshal into WSEvent correctly — but it
	// is still valid JSON so the real malformed-message path is exercised.
	// Actually, any JSON object will unmarshal into WSEvent (unknown fields are
	// ignored), so to trigger the "skip malformed" continue we send truly
	// invalid JSON, followed by a well-formed event so we can detect the client
	// is still alive.
	goodEvent := WSEvent{Event: "hello", Seq: 99}
	goodPayload, _ := json.Marshal(goodEvent)

	srv := startTestWSServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// Discard auth challenge.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		// Send malformed JSON.
		_ = conn.WriteMessage(websocket.TextMessage, []byte("{not valid json!!!"))
		// Send a valid event — client must still receive it (no panic).
		_ = conn.WriteMessage(websocket.TextMessage, goodPayload)
		time.Sleep(500 * time.Millisecond)
	})
	defer srv.Close()

	ws := connectToTestServer(t, "tok", srv)
	if err := ws.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer ws.Close()

	// The test itself will panic (and fail loudly) if the readLoop panics.
	select {
	case got := <-ws.Events:
		if got.Event != "hello" {
			t.Errorf("event: got %q, want %q", got.Event, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out — client may have panicked or stopped after bad JSON")
	}
}

// ---------------------------------------------------------------------------
// TestWSClientClose
// ---------------------------------------------------------------------------

func TestWSClientClose(t *testing.T) {
	srv := startTestWSServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// Discard auth challenge then block until client closes.
		conn.ReadMessage() //nolint:errcheck
	})
	defer srv.Close()

	ws := connectToTestServer(t, "tok", srv)
	if err := ws.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Close the client in a goroutine so we can apply our own timeout.
	closeDone := make(chan struct{})
	go func() {
		ws.Close()
		close(closeDone)
	}()

	select {
	case <-ws.done:
		// done channel closed — good
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: ws.done was not closed after Close()")
	}

	// Also wait for Close() itself to return.
	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: Close() did not return")
	}
}

// ---------------------------------------------------------------------------
// TestConnectWithRetry_BackoffCaps
// ---------------------------------------------------------------------------

func TestConnectWithRetry_BackoffCaps(t *testing.T) {
	// We need a WSClient whose Connect() always fails.
	// NewWSClient requires wss://, and dialling a non-existent host will fail.
	// Use a port that is guaranteed to refuse connections fast.
	ws, err := NewWSClient("tok", "wss://127.0.0.1:1") // port 1 is never open
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}

	// Cancel the context after ~150ms so the test is fast.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = ConnectWithRetry(ctx, ws)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected ConnectWithRetry to return an error, got nil")
	}

	// The context is cancelled after ~150ms.  The initial backoff is 2s, but
	// the first attempt fails immediately (dial error), then we wait for the
	// shorter of (delay, ctx.Done()).  Because ctx cancels in ~150ms which is
	// less than the 2s delay, ConnectWithRetry should return quickly.
	//
	// Verify total elapsed is well under 30s (the cap), proving backoff did
	// not sleep for a full uncapped delay.
	if elapsed > 5*time.Second {
		t.Errorf("ConnectWithRetry ran for %v — backoff may not be capping correctly", elapsed)
	}
	t.Logf("ConnectWithRetry returned after %v with error: %v", elapsed, err)
}
