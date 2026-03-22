package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// GetChannelsForUser tests
// ---------------------------------------------------------------------------

func TestGetChannelsForUser_Success(t *testing.T) {
	t.Parallel()

	const responseBody = `[
		{"id":"ch-1","name":"general","display_name":"General","type":"O","team_id":"team-1"},
		{"id":"ch-2","name":"dm__alice__bob","display_name":"","type":"D","team_id":""}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	channels, err := c.GetChannelsForUser("user-1", "team-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	// First channel: type O (open/public)
	if channels[0].ID != "ch-1" {
		t.Errorf("channels[0].ID: got %q, want %q", channels[0].ID, "ch-1")
	}
	if channels[0].Type != "O" {
		t.Errorf("channels[0].Type: got %q, want %q", channels[0].Type, "O")
	}
	if channels[0].Name != "general" {
		t.Errorf("channels[0].Name: got %q, want %q", channels[0].Name, "general")
	}

	// Second channel: type D (direct message)
	if channels[1].ID != "ch-2" {
		t.Errorf("channels[1].ID: got %q, want %q", channels[1].ID, "ch-2")
	}
	if channels[1].Type != "D" {
		t.Errorf("channels[1].Type: got %q, want %q", channels[1].Type, "D")
	}
}

func TestGetChannelsForUser_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.GetChannelsForUser("user-1", "team-1")
	if err == nil {
		t.Fatal("expected an error for 500 response, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetChannelMembersForUser tests
// ---------------------------------------------------------------------------

func TestGetChannelMembersForUser_Success(t *testing.T) {
	t.Parallel()

	// First member: mark_unread="mention" → IsMuted() should be true.
	// Second member: mark_unread="all"     → IsMuted() should be false.
	const responseBody = `[
		{
			"channel_id":"ch-1",
			"user_id":"user-1",
			"msg_count":10,
			"mention_count":2,
			"notify_props":{"mark_unread":"mention","desktop":"default","push":"default","ignore_channel_mentions":"default"},
			"last_viewed_at":1700000001
		},
		{
			"channel_id":"ch-2",
			"user_id":"user-1",
			"msg_count":5,
			"mention_count":0,
			"notify_props":{"mark_unread":"all","desktop":"all","push":"none","ignore_channel_mentions":"off"},
			"last_viewed_at":1700000002
		}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	members, err := c.GetChannelMembersForUser("user-1", "team-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	// First member should be muted (mark_unread="mention").
	if !members[0].IsMuted() {
		t.Errorf("members[0].IsMuted(): got false, want true (mark_unread=%q)", members[0].NotifyProps.MarkUnread)
	}

	// Second member should NOT be muted (mark_unread="all").
	if members[1].IsMuted() {
		t.Errorf("members[1].IsMuted(): got true, want false (mark_unread=%q)", members[1].NotifyProps.MarkUnread)
	}

	// Spot-check a few other fields on the second member.
	if members[1].ChannelID != "ch-2" {
		t.Errorf("members[1].ChannelID: got %q, want %q", members[1].ChannelID, "ch-2")
	}
	if members[1].NotifyProps.Desktop != "all" {
		t.Errorf("members[1].NotifyProps.Desktop: got %q, want %q", members[1].NotifyProps.Desktop, "all")
	}
}

func TestGetChannelMembersForUser_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	members, err := c.GetChannelMembersForUser("user-1", "team-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected empty slice, got %d members", len(members))
	}
}

// ---------------------------------------------------------------------------
// GetPreferences tests
// ---------------------------------------------------------------------------

func TestGetPreferences_Success(t *testing.T) {
	t.Parallel()

	const responseBody = `[
		{"user_id":"user-1","category":"display_settings","name":"channel_display_mode","value":"full"},
		{"user_id":"user-1","category":"tutorial_step","name":"user-1","value":"999"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	prefs, err := c.GetPreferences("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prefs) != 2 {
		t.Fatalf("expected 2 preferences, got %d", len(prefs))
	}

	// First preference.
	if prefs[0].Category != "display_settings" {
		t.Errorf("prefs[0].Category: got %q, want %q", prefs[0].Category, "display_settings")
	}
	if prefs[0].Name != "channel_display_mode" {
		t.Errorf("prefs[0].Name: got %q, want %q", prefs[0].Name, "channel_display_mode")
	}
	if prefs[0].Value != "full" {
		t.Errorf("prefs[0].Value: got %q, want %q", prefs[0].Value, "full")
	}

	// Second preference.
	if prefs[1].Category != "tutorial_step" {
		t.Errorf("prefs[1].Category: got %q, want %q", prefs[1].Category, "tutorial_step")
	}
	if prefs[1].Value != "999" {
		t.Errorf("prefs[1].Value: got %q, want %q", prefs[1].Value, "999")
	}
}

// ---------------------------------------------------------------------------
// MarkChannelRead tests
// ---------------------------------------------------------------------------

func TestMarkChannelRead_Success(t *testing.T) {
	t.Parallel()

	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	}))
	defer srv.Close()

	const channelID = "ch-xyz"
	c := newTestClientPointing(srv, "tok")
	err := c.MarkChannelRead(channelID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the request body contains the channel_id.
	var bodyMap map[string]interface{}
	if jsonErr := json.Unmarshal(capturedBody, &bodyMap); jsonErr != nil {
		t.Fatalf("request body is not valid JSON: %v (body=%q)", jsonErr, capturedBody)
	}

	gotChannelID, ok := bodyMap["channel_id"]
	if !ok {
		t.Fatalf("request body missing \"channel_id\" key; body=%q", capturedBody)
	}
	if gotChannelID != channelID {
		t.Errorf("channel_id in request body: got %q, want %q", gotChannelID, channelID)
	}
}

func TestMarkChannelRead_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	err := c.MarkChannelRead("ch-xyz")
	if err == nil {
		t.Fatal("expected an error for 500 response, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetUsersByIDs tests
// ---------------------------------------------------------------------------

func TestGetUsersByIDs_Success(t *testing.T) {
	t.Parallel()

	const responseBody = `[
		{"id":"uid-1","username":"alice","first_name":"Alice","last_name":"Smith","nickname":"","email":"alice@example.com","delete_at":0},
		{"id":"uid-2","username":"bob","first_name":"Bob","last_name":"Jones","nickname":"Bobby","email":"bob@example.com","delete_at":0}
	]`

	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	ids := []string{"uid-1", "uid-2"}
	c := newTestClientPointing(srv, "tok")
	users, err := c.GetUsersByIDs(ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	// Verify first user fields.
	if users[0].ID != "uid-1" {
		t.Errorf("users[0].ID: got %q, want %q", users[0].ID, "uid-1")
	}
	if users[0].Username != "alice" {
		t.Errorf("users[0].Username: got %q, want %q", users[0].Username, "alice")
	}
	if users[0].FirstName != "Alice" {
		t.Errorf("users[0].FirstName: got %q, want %q", users[0].FirstName, "Alice")
	}
	if users[0].LastName != "Smith" {
		t.Errorf("users[0].LastName: got %q, want %q", users[0].LastName, "Smith")
	}

	// Verify second user fields.
	if users[1].ID != "uid-2" {
		t.Errorf("users[1].ID: got %q, want %q", users[1].ID, "uid-2")
	}
	if users[1].Nickname != "Bobby" {
		t.Errorf("users[1].Nickname: got %q, want %q", users[1].Nickname, "Bobby")
	}

	// Verify request body is a JSON array of IDs.
	var sentIDs []string
	if jsonErr := json.Unmarshal(capturedBody, &sentIDs); jsonErr != nil {
		t.Fatalf("request body is not a JSON array: %v (body=%q)", jsonErr, capturedBody)
	}
	if len(sentIDs) != 2 {
		t.Fatalf("expected 2 IDs in request body, got %d", len(sentIDs))
	}
	if sentIDs[0] != "uid-1" {
		t.Errorf("sentIDs[0]: got %q, want %q", sentIDs[0], "uid-1")
	}
	if sentIDs[1] != "uid-2" {
		t.Errorf("sentIDs[1]: got %q, want %q", sentIDs[1], "uid-2")
	}
}

func TestGetUsersByIDs_Empty(t *testing.T) {
	t.Parallel()

	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	users, err := c.GetUsersByIDs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected empty slice, got %d users", len(users))
	}

	// Verify the request body is a JSON empty array.
	var sentIDs []string
	if jsonErr := json.Unmarshal(capturedBody, &sentIDs); jsonErr != nil {
		t.Fatalf("request body is not valid JSON: %v (body=%q)", jsonErr, capturedBody)
	}
	if len(sentIDs) != 0 {
		t.Errorf("expected empty JSON array in request body, got %d elements", len(sentIDs))
	}
}
