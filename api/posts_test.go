package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// GetPostsForChannel tests
// ---------------------------------------------------------------------------

func TestGetPostsForChannel_Success(t *testing.T) {
	t.Parallel()

	// Two posts in the PostList. Order is newest-first: post-2, post-1.
	const responseBody = `{
		"order": ["post-2","post-1"],
		"posts": {
			"post-1": {"id":"post-1","channel_id":"ch-1","user_id":"uid-1","message":"Hello world","create_at":1700000001,"update_at":1700000001,"delete_at":0,"type":"","root_id":"","edit_at":0,"is_pinned":false},
			"post-2": {"id":"post-2","channel_id":"ch-1","user_id":"uid-2","message":"Hi there","create_at":1700000002,"update_at":1700000002,"delete_at":0,"type":"","root_id":"","edit_at":0,"is_pinned":false}
		},
		"next_post_id": "",
		"prev_post_id": ""
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	pl, err := c.GetPostsForChannel("ch-1", 0, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Order is preserved (two entries in the correct order).
	if len(pl.Order) != 2 {
		t.Fatalf("expected 2 entries in Order, got %d", len(pl.Order))
	}
	if pl.Order[0] != "post-2" {
		t.Errorf("Order[0]: got %q, want %q", pl.Order[0], "post-2")
	}
	if pl.Order[1] != "post-1" {
		t.Errorf("Order[1]: got %q, want %q", pl.Order[1], "post-1")
	}

	// Verify the Posts map has both entries.
	if len(pl.Posts) != 2 {
		t.Fatalf("expected 2 posts in Posts map, got %d", len(pl.Posts))
	}

	// Check post-1 fields.
	p1, ok := pl.Posts["post-1"]
	if !ok {
		t.Fatal("Posts map missing key \"post-1\"")
	}
	if p1.ID != "post-1" {
		t.Errorf("post-1 ID: got %q, want %q", p1.ID, "post-1")
	}
	if p1.ChannelID != "ch-1" {
		t.Errorf("post-1 ChannelID: got %q, want %q", p1.ChannelID, "ch-1")
	}
	if p1.UserID != "uid-1" {
		t.Errorf("post-1 UserID: got %q, want %q", p1.UserID, "uid-1")
	}
	if p1.Message != "Hello world" {
		t.Errorf("post-1 Message: got %q, want %q", p1.Message, "Hello world")
	}

	// Check post-2 fields.
	p2, ok := pl.Posts["post-2"]
	if !ok {
		t.Fatal("Posts map missing key \"post-2\"")
	}
	if p2.ID != "post-2" {
		t.Errorf("post-2 ID: got %q, want %q", p2.ID, "post-2")
	}
	if p2.UserID != "uid-2" {
		t.Errorf("post-2 UserID: got %q, want %q", p2.UserID, "uid-2")
	}
	if p2.Message != "Hi there" {
		t.Errorf("post-2 Message: got %q, want %q", p2.Message, "Hi there")
	}
}

func TestGetPostsForChannel_PageParams(t *testing.T) {
	t.Parallel()

	var capturedURL string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order":[],"posts":{}}`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.GetPostsForChannel("ch-abc", 1, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The request URL query string must contain page=1 and per_page=30.
	if !strings.Contains(capturedURL, "page=1") {
		t.Errorf("expected query param page=1 in URL %q, but not found", capturedURL)
	}
	if !strings.Contains(capturedURL, "per_page=30") {
		t.Errorf("expected query param per_page=30 in URL %q, but not found", capturedURL)
	}
}

func TestGetPostsForChannel_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.GetPostsForChannel("ch-1", 0, 30)
	if err == nil {
		t.Fatal("expected an error for 500 response, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetPostsSince tests
// ---------------------------------------------------------------------------

func TestGetPostsSince_Success(t *testing.T) {
	t.Parallel()

	const since int64 = 1700000000000

	var capturedURL string

	const responseBody = `{
		"order": ["post-99"],
		"posts": {
			"post-99": {"id":"post-99","channel_id":"ch-1","user_id":"uid-1","message":"New message","create_at":1700000000001,"update_at":1700000000001,"delete_at":0,"type":"","root_id":"","edit_at":0,"is_pinned":false}
		},
		"next_post_id": "",
		"prev_post_id": ""
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	pl, err := c.GetPostsSince("ch-1", since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The request URL must include the since query param.
	if !strings.Contains(capturedURL, "since=1700000000000") {
		t.Errorf("expected query param since=1700000000000 in URL %q", capturedURL)
	}

	// Verify returned PostList.
	if len(pl.Order) != 1 || pl.Order[0] != "post-99" {
		t.Errorf("Order: got %v, want [post-99]", pl.Order)
	}
	p, ok := pl.Posts["post-99"]
	if !ok {
		t.Fatal("Posts map missing key \"post-99\"")
	}
	if p.Message != "New message" {
		t.Errorf("post-99 Message: got %q, want %q", p.Message, "New message")
	}
}

func TestGetPostsSince_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order":[],"posts":{}}`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	pl, err := c.GetPostsSince("ch-1", 1700000000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pl.Order) != 0 {
		t.Errorf("expected empty Order slice, got %v", pl.Order)
	}
	if len(pl.Posts) != 0 {
		t.Errorf("expected empty Posts map, got %d entries", len(pl.Posts))
	}
}

func TestGetPostsSince_ZeroSince(t *testing.T) {
	t.Parallel()

	// No server needed — the guard fires before any network call.
	c := &Client{}
	_, err := c.GetPostsSince("ch-1", 0)
	if err == nil {
		t.Fatal("expected an error when since=0, got nil")
	}
	if !strings.Contains(err.Error(), "since must be > 0") {
		t.Errorf("error message %q does not contain %q", err.Error(), "since must be > 0")
	}
}

// ---------------------------------------------------------------------------
// CreatePost tests
// ---------------------------------------------------------------------------

func TestCreatePost_Success(t *testing.T) {
	t.Parallel()

	const responseBody = `{
		"id":"post-new",
		"channel_id":"ch-1",
		"user_id":"uid-1",
		"message":"Hello channel",
		"create_at":1700000010,
		"update_at":1700000010,
		"delete_at":0,
		"type":"",
		"root_id":"",
		"edit_at":0,
		"is_pinned":false
	}`

	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	p, err := c.CreatePost("ch-1", "Hello channel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the returned Post has the expected fields.
	if p.ChannelID != "ch-1" {
		t.Errorf("returned Post ChannelID: got %q, want %q", p.ChannelID, "ch-1")
	}
	if p.Message != "Hello channel" {
		t.Errorf("returned Post Message: got %q, want %q", p.Message, "Hello channel")
	}
	if p.ID != "post-new" {
		t.Errorf("returned Post ID: got %q, want %q", p.ID, "post-new")
	}

	// Verify the request body contains the correct channel_id and message.
	var bodyMap map[string]interface{}
	if jsonErr := json.Unmarshal(capturedBody, &bodyMap); jsonErr != nil {
		t.Fatalf("request body is not valid JSON: %v (body=%q)", jsonErr, capturedBody)
	}

	gotChannelID, ok := bodyMap["channel_id"]
	if !ok {
		t.Fatalf("request body missing \"channel_id\" key; body=%q", capturedBody)
	}
	if gotChannelID != "ch-1" {
		t.Errorf("channel_id in request body: got %q, want %q", gotChannelID, "ch-1")
	}

	gotMessage, ok := bodyMap["message"]
	if !ok {
		t.Fatalf("request body missing \"message\" key; body=%q", capturedBody)
	}
	if gotMessage != "Hello channel" {
		t.Errorf("message in request body: got %q, want %q", gotMessage, "Hello channel")
	}
}

func TestCreatePost_MessageNotInPath(t *testing.T) {
	t.Parallel()

	// Use a distinctive message that must not appear in the URL path.
	const sensitiveMsg = "SECRET_MESSAGE_MUST_NOT_BE_IN_URL"

	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"p1","channel_id":"ch-1","user_id":"uid-1","message":"` + sensitiveMsg + `","create_at":1,"update_at":1,"delete_at":0,"type":"","root_id":"","edit_at":0,"is_pinned":false}`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.CreatePost("ch-1", sensitiveMsg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Regression guard: the message content must NOT appear in the request URL.
	if strings.Contains(capturedPath, sensitiveMsg) {
		t.Errorf("message content %q must not appear in request URL %q (it should only be in the body)", sensitiveMsg, capturedPath)
	}
}

func TestCreatePost_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.CreatePost("ch-1", "some message")
	if err == nil {
		t.Fatal("expected an error for 400 response, got nil")
	}
}
