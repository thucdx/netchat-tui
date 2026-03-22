package api

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// TestChannelUnmarshal
// ---------------------------------------------------------------------------

func TestChannelUnmarshal(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		wantType string
	}{
		{
			name:     "direct message channel",
			json:     `{"id":"ch1","name":"user1__user2","display_name":"","type":"D","team_id":"","total_msg_count":0,"last_post_at":0,"delete_at":0}`,
			wantType: "D",
		},
		{
			name:     "open/public channel",
			json:     `{"id":"ch2","name":"town-square","display_name":"Town Square","type":"O","team_id":"team1","total_msg_count":42,"last_post_at":1700000000,"delete_at":0}`,
			wantType: "O",
		},
		{
			name:     "private channel",
			json:     `{"id":"ch3","name":"secret","display_name":"Secret","type":"P","team_id":"team1","total_msg_count":5,"last_post_at":1700000001,"delete_at":0}`,
			wantType: "P",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ch Channel
			if err := json.Unmarshal([]byte(tc.json), &ch); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if ch.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", ch.Type, tc.wantType)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsMuted_True / TestIsMuted_False
// ---------------------------------------------------------------------------

func TestIsMuted_True(t *testing.T) {
	cm := ChannelMember{
		NotifyProps: NotifyProps{MarkUnread: "mention"},
	}
	if !cm.IsMuted() {
		t.Error("IsMuted() = false, want true for mark_unread=mention")
	}
}

func TestIsMuted_False(t *testing.T) {
	cm := ChannelMember{
		NotifyProps: NotifyProps{MarkUnread: "all"},
	}
	if cm.IsMuted() {
		t.Error("IsMuted() = true, want false for mark_unread=all")
	}
}

// ---------------------------------------------------------------------------
// TestUnreadCount_*
// ---------------------------------------------------------------------------

func TestUnreadCount_Normal(t *testing.T) {
	ch := Channel{TotalMsgCount: 10}
	cm := ChannelMember{MsgCount: 7}
	got := cm.UnreadCount(ch)
	if got != 3 {
		t.Errorf("UnreadCount() = %d, want 3", got)
	}
}

func TestUnreadCount_Zero(t *testing.T) {
	ch := Channel{TotalMsgCount: 5}
	cm := ChannelMember{MsgCount: 5}
	got := cm.UnreadCount(ch)
	if got != 0 {
		t.Errorf("UnreadCount() = %d, want 0", got)
	}
}

func TestUnreadCount_Clamp(t *testing.T) {
	// MsgCount ahead of TotalMsgCount — should clamp to 0, not go negative.
	ch := Channel{TotalMsgCount: 3}
	cm := ChannelMember{MsgCount: 5}
	got := cm.UnreadCount(ch)
	if got != 0 {
		t.Errorf("UnreadCount() = %d, want 0 (no negative unread)", got)
	}
}

// ---------------------------------------------------------------------------
// TestIsDeleted_True / TestIsDeleted_False
// ---------------------------------------------------------------------------

func TestIsDeleted_True(t *testing.T) {
	ch := Channel{DeleteAt: 12345}
	if !ch.IsDeleted() {
		t.Error("IsDeleted() = false, want true for DeleteAt=12345")
	}
}

func TestIsDeleted_False(t *testing.T) {
	ch := Channel{DeleteAt: 0}
	if ch.IsDeleted() {
		t.Error("IsDeleted() = true, want false for DeleteAt=0")
	}
}

// ---------------------------------------------------------------------------
// TestWSEventUnmarshal
// ---------------------------------------------------------------------------

func TestWSEventUnmarshal(t *testing.T) {
	raw := `{
		"event": "posted",
		"data": {
			"channel_type": "O",
			"team_id": "team1"
		},
		"broadcast": {
			"channel_id": "ch-abc",
			"user_id": "",
			"team_id": ""
		},
		"seq": 7
	}`

	var ev WSEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if ev.Event != "posted" {
		t.Errorf("Event = %q, want \"posted\"", ev.Event)
	}
	if ev.Broadcast.ChannelID != "ch-abc" {
		t.Errorf("Broadcast.ChannelID = %q, want \"ch-abc\"", ev.Broadcast.ChannelID)
	}
	if ev.Seq != 7 {
		t.Errorf("Seq = %d, want 7", ev.Seq)
	}
}

// ---------------------------------------------------------------------------
// TestPostListUnmarshal
// ---------------------------------------------------------------------------

func TestPostListUnmarshal(t *testing.T) {
	raw := `{
		"order": ["post2", "post1"],
		"posts": {
			"post1": {
				"id": "post1",
				"channel_id": "ch1",
				"user_id": "user1",
				"message": "hello world",
				"create_at": 1700000001,
				"update_at": 1700000002,
				"delete_at": 0,
				"type": "",
				"root_id": ""
			},
			"post2": {
				"id": "post2",
				"channel_id": "ch1",
				"user_id": "user2",
				"message": "hi there",
				"create_at": 1700000010,
				"update_at": 1700000010,
				"delete_at": 0,
				"type": "",
				"root_id": ""
			}
		}
	}`

	var pl PostList
	if err := json.Unmarshal([]byte(raw), &pl); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	// Verify order slice
	if len(pl.Order) != 2 {
		t.Fatalf("Order len = %d, want 2", len(pl.Order))
	}
	if pl.Order[0] != "post2" || pl.Order[1] != "post1" {
		t.Errorf("Order = %v, want [post2 post1]", pl.Order)
	}

	// Verify posts map
	if len(pl.Posts) != 2 {
		t.Fatalf("Posts len = %d, want 2", len(pl.Posts))
	}
	p1, ok := pl.Posts["post1"]
	if !ok {
		t.Fatal("Posts[\"post1\"] not found")
	}
	if p1.Message != "hello world" {
		t.Errorf("post1.Message = %q, want \"hello world\"", p1.Message)
	}
	if p1.UserID != "user1" {
		t.Errorf("post1.UserID = %q, want \"user1\"", p1.UserID)
	}
	if p1.CreateAt != 1700000001 {
		t.Errorf("post1.CreateAt = %d, want 1700000001", p1.CreateAt)
	}

	p2, ok := pl.Posts["post2"]
	if !ok {
		t.Fatal("Posts[\"post2\"] not found")
	}
	if p2.Message != "hi there" {
		t.Errorf("post2.Message = %q, want \"hi there\"", p2.Message)
	}
}

// ---------------------------------------------------------------------------
// TestWSPostedEventDoubleUnmarshal
// ---------------------------------------------------------------------------

// In the Mattermost WebSocket protocol, the `posted` event encodes the post
// as a JSON *string* inside the data map — i.e. data["post"] is itself a
// JSON-serialised Post object. The handler must therefore do two rounds of
// json.Unmarshal: first to decode the WSEvent, then to decode the inner
// string value into a Post.
func TestWSPostedEventDoubleUnmarshal(t *testing.T) {
	// Build the inner post JSON string that will be embedded inside the outer JSON.
	innerPost := `{"id":"p999","channel_id":"ch-xyz","user_id":"u1","message":"double encoded","create_at":1700005000,"update_at":1700005001,"delete_at":0,"type":"","root_id":""}`

	// The outer WSEvent JSON — data["post"] is a JSON *string* value.
	outerJSON := `{
		"event": "posted",
		"data": {
			"post": "{\"id\":\"p999\",\"channel_id\":\"ch-xyz\",\"user_id\":\"u1\",\"message\":\"double encoded\",\"create_at\":1700005000,\"update_at\":1700005001,\"delete_at\":0,\"type\":\"\",\"root_id\":\"\"}",
			"channel_type": "O"
		},
		"broadcast": {
			"channel_id": "ch-xyz",
			"user_id": "",
			"team_id": ""
		},
		"seq": 1
	}`

	// First unmarshal: decode the outer WSEvent.
	var ev WSEvent
	if err := json.Unmarshal([]byte(outerJSON), &ev); err != nil {
		t.Fatalf("first unmarshal (WSEvent) failed: %v", err)
	}
	if ev.Event != "posted" {
		t.Errorf("Event = %q, want \"posted\"", ev.Event)
	}

	// Extract data["post"] — it should be a string.
	rawPostVal, ok := ev.Data["post"]
	if !ok {
		t.Fatal("data[\"post\"] key missing from WSEvent.Data")
	}
	postStr, ok := rawPostVal.(string)
	if !ok {
		t.Fatalf("data[\"post\"] is %T, want string", rawPostVal)
	}

	// Sanity-check: the extracted string should equal our innerPost JSON.
	if postStr != innerPost {
		t.Errorf("extracted post string = %q\nwant                  %q", postStr, innerPost)
	}

	// Second unmarshal: decode the inner JSON string into a Post.
	var post Post
	if err := json.Unmarshal([]byte(postStr), &post); err != nil {
		t.Fatalf("second unmarshal (Post) failed: %v", err)
	}

	// Verify Post fields.
	if post.ID != "p999" {
		t.Errorf("Post.ID = %q, want \"p999\"", post.ID)
	}
	if post.ChannelID != "ch-xyz" {
		t.Errorf("Post.ChannelID = %q, want \"ch-xyz\"", post.ChannelID)
	}
	if post.UserID != "u1" {
		t.Errorf("Post.UserID = %q, want \"u1\"", post.UserID)
	}
	if post.Message != "double encoded" {
		t.Errorf("Post.Message = %q, want \"double encoded\"", post.Message)
	}
	if post.CreateAt != 1700005000 {
		t.Errorf("Post.CreateAt = %d, want 1700005000", post.CreateAt)
	}
}
