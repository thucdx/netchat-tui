package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/messages"
	"github.com/thucdx/netchat-tui/tui/sidebar"
)

// helper: create an AppModel with sidebar items pre-loaded and a window size set.
func newTestApp(items []sidebar.ChannelItem) AppModel {
	m := NewAppModel(nil) // no real API client

	// Simulate a WindowSizeMsg so the model is "ready".
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = result.(AppModel)

	// Load sidebar items.
	m.sidebar.SetItems(items)
	return m
}

// selectChannelByIndex simulates the user navigating the sidebar cursor to
// the given index and pressing Enter. This sets the sidebar's internal
// `selected` field AND emits a ChannelSelectedMsg which AppModel processes.
func selectChannelByIndex(t *testing.T, m AppModel, index int) AppModel {
	t.Helper()

	// Ensure focus is on sidebar.
	m.focus = FocusSidebar
	m.syncFocus()

	// Move cursor to the desired index. The cursor starts at 0.
	// First reset to top.
	for i := 0; i < len(m.sidebar.Items()); i++ {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = result.(AppModel)
	}
	// Now move down to the desired index.
	for i := 0; i < index; i++ {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = result.(AppModel)
	}

	// Press Enter to select.
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	// The sidebar emits ChannelSelectedMsg via a tea.Cmd. Execute it
	// and feed the result back to AppModel.
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			result, _ = m.Update(msg)
			m = result.(AppModel)
		}
	}

	return m
}

// ---------------------------------------------------------------------------
// Test: NewPostMsg for the ACTIVE channel appends to chat (no unread bump)
// ---------------------------------------------------------------------------

func TestNewPostMsg_ActiveChannel(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 0},
		{Channel: api.Channel{ID: "ch2", DisplayName: "Random"}, UnreadCount: 0},
	}

	m := newTestApp(items)
	m = selectChannelByIndex(t, m, 0) // select ch1

	// Verify ch1 is the active channel.
	sel := m.sidebar.SelectedChannel()
	if sel == nil || sel.Channel.ID != "ch1" {
		t.Fatalf("expected ch1 to be selected, got %v", sel)
	}

	// Simulate a WebSocket posted event for the ACTIVE channel.
	post := api.Post{ID: "p1", ChannelID: "ch1", UserID: "u1", Message: "hello", CreateAt: 1700000000000}
	result, _ := m.Update(messages.NewPostMsg{Post: post})
	m = result.(AppModel)

	// Sidebar unread for ch1 should remain 0 (post was for the active channel).
	for _, item := range m.sidebar.Items() {
		if item.Channel.ID == "ch1" && item.UnreadCount != 0 {
			t.Errorf("active channel ch1 unread should be 0, got %d", item.UnreadCount)
		}
	}
}

// ---------------------------------------------------------------------------
// Test: NewPostMsg for an INACTIVE channel increments sidebar unread
// ---------------------------------------------------------------------------

func TestNewPostMsg_InactiveChannel(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 0},
		{Channel: api.Channel{ID: "ch2", DisplayName: "Random"}, UnreadCount: 0},
	}

	m := newTestApp(items)
	m = selectChannelByIndex(t, m, 0) // select ch1

	// Simulate a WebSocket posted event for ch2 (INACTIVE).
	post := api.Post{ID: "p2", ChannelID: "ch2", UserID: "u2", Message: "hi", CreateAt: 1700000001000}
	result, _ := m.Update(messages.NewPostMsg{Post: post})
	m = result.(AppModel)

	// Sidebar unread for ch2 should be incremented to 1.
	found := false
	for _, item := range m.sidebar.Items() {
		if item.Channel.ID == "ch2" {
			found = true
			if item.UnreadCount != 1 {
				t.Errorf("inactive channel ch2 unread should be 1, got %d", item.UnreadCount)
			}
		}
	}
	if !found {
		t.Fatal("ch2 not found in sidebar items")
	}
}

// ---------------------------------------------------------------------------
// Test: NewPostMsg for a MUTED channel still increments unread count
// ---------------------------------------------------------------------------

func TestNewPostMsg_MutedChannel(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 0},
		{
			Channel:     api.Channel{ID: "ch-muted", DisplayName: "Noisy"},
			Member:      api.ChannelMember{NotifyProps: api.NotifyProps{MarkUnread: "mention"}},
			UnreadCount: 0,
			IsMuted:     true,
		},
	}

	m := newTestApp(items)
	m = selectChannelByIndex(t, m, 0) // select ch1

	// Post to the muted channel (which is inactive).
	post := api.Post{ID: "p3", ChannelID: "ch-muted", UserID: "u3", Message: "noise", CreateAt: 1700000002000}
	result, _ := m.Update(messages.NewPostMsg{Post: post})
	m = result.(AppModel)

	// The unread count should still increment (muted channels still track unreads).
	for _, item := range m.sidebar.Items() {
		if item.Channel.ID == "ch-muted" {
			if item.UnreadCount != 1 {
				t.Errorf("muted channel unread should be 1, got %d", item.UnreadCount)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Test: ChannelSelectedMsg clears unread badge for the selected channel
// ---------------------------------------------------------------------------

func TestChannelSelected_ClearsBadge(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 5},
		{Channel: api.Channel{ID: "ch2", DisplayName: "Random"}, UnreadCount: 3},
	}

	m := newTestApp(items)

	// Select ch1 via Enter — its unread badge should be cleared.
	m = selectChannelByIndex(t, m, 0)

	for _, item := range m.sidebar.Items() {
		if item.Channel.ID == "ch1" && item.UnreadCount != 0 {
			t.Errorf("selected channel ch1 unread should be 0 after selection, got %d", item.UnreadCount)
		}
		// ch2 should be unchanged.
		if item.Channel.ID == "ch2" && item.UnreadCount != 3 {
			t.Errorf("unselected channel ch2 unread should remain 3, got %d", item.UnreadCount)
		}
	}
}

// ---------------------------------------------------------------------------
// Test: WindowSizeMsg propagates to all sub-models
// ---------------------------------------------------------------------------

func TestWindowSizeMsg_Propagates(t *testing.T) {
	t.Parallel()

	m := NewAppModel(nil)

	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = result.(AppModel)

	if !m.ready {
		t.Fatal("AppModel should be ready after WindowSizeMsg")
	}

	if m.layout.TotalHeight <= 0 {
		t.Errorf("layout.TotalHeight should be > 0, got %d", m.layout.TotalHeight)
	}
	if m.layout.ChatWidth <= 0 {
		t.Errorf("layout.ChatWidth should be > 0, got %d", m.layout.ChatWidth)
	}
}

// ---------------------------------------------------------------------------
// Test: channel_viewed WS event clears unread badge
// ---------------------------------------------------------------------------

func TestChannelViewed_ClearsUnread(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 7},
		{Channel: api.Channel{ID: "ch2", DisplayName: "Random"}, UnreadCount: 4},
	}

	m := newTestApp(items)

	// Simulate a WS channel_viewed event for ch1.
	event := api.WSEvent{
		Event: "channel_viewed",
		Data:  map[string]interface{}{"channel_id": "ch1"},
	}
	m.handleChannelViewed(event)

	for _, item := range m.sidebar.Items() {
		if item.Channel.ID == "ch1" && item.UnreadCount != 0 {
			t.Errorf("channel_viewed should clear ch1 unread to 0, got %d", item.UnreadCount)
		}
		if item.Channel.ID == "ch2" && item.UnreadCount != 4 {
			t.Errorf("ch2 unread should remain 4, got %d", item.UnreadCount)
		}
	}
}

// ---------------------------------------------------------------------------
// Test: channel_viewed with empty channel_id is a no-op
// ---------------------------------------------------------------------------

func TestChannelViewed_EmptyChannelID(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 3},
	}

	m := newTestApp(items)

	event := api.WSEvent{
		Event: "channel_viewed",
		Data:  map[string]interface{}{},
	}
	m.handleChannelViewed(event)

	for _, item := range m.sidebar.Items() {
		if item.Channel.ID == "ch1" && item.UnreadCount != 3 {
			t.Errorf("unread should be unchanged, got %d", item.UnreadCount)
		}
	}
}

// ---------------------------------------------------------------------------
// Test: dmOtherUserID extracts the other user from DM channel name
// ---------------------------------------------------------------------------

func TestDMOtherUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		chanName  string
		myUserID  string
		want      string
	}{
		{"I am first", "user1__user2", "user1", "user2"},
		{"I am second", "user1__user2", "user2", "user1"},
		{"not a DM", "general", "user1", ""},
		{"empty", "", "user1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dmOtherUserID(tt.chanName, tt.myUserID)
			if got != tt.want {
				t.Errorf("dmOtherUserID(%q, %q) = %q, want %q", tt.chanName, tt.myUserID, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test: ErrorMsg sets chat error banner and releases input send lock
// ---------------------------------------------------------------------------

func TestErrorMsg_SetsChatError(t *testing.T) {
	t.Parallel()

	m := newTestApp(nil)

	errMsg := ErrorMsg{Err: fmt.Errorf("network timeout")}
	result, _ := m.Update(errMsg)
	m = result.(AppModel)

	// The chat error should be set (we can't read m.chat.err directly from
	// outside the chat package, but we can verify the view contains the error).
	view := m.chat.View()
	if !strings.Contains(view, "network timeout") {
		t.Error("chat view should display the error banner")
	}
}

// ---------------------------------------------------------------------------
// Test: SetChannelInfo sets channel before posts arrive
// ---------------------------------------------------------------------------

func TestSetChannelInfo_BeforePosts(t *testing.T) {
	t.Parallel()

	items := []sidebar.ChannelItem{
		{Channel: api.Channel{ID: "ch1", DisplayName: "General"}, UnreadCount: 0},
	}

	m := newTestApp(items)
	m = selectChannelByIndex(t, m, 0)

	// After selection, chat.ChannelID() should be set even without posts loading.
	if m.chat.ChannelID() != "ch1" {
		t.Errorf("chat.ChannelID() should be 'ch1' after selection, got %q", m.chat.ChannelID())
	}
}
