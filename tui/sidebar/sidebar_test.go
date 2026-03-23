package sidebar

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// helpers

func defaultKeys() keymap.KeyMap {
	return keymap.DefaultKeyMap()
}

func newModel() Model {
	return NewModel(defaultKeys(), "user1")
}

// makeItems builds n ChannelItems of the given type with sequential IDs.
func makeItems(n int, chType string) []ChannelItem {
	items := make([]ChannelItem, n)
	for i := range items {
		id := string(rune('a' + i))
		items[i] = ChannelItem{
			Channel:     api.Channel{ID: "ch-" + id, Name: "channel-" + id, DisplayName: "Channel " + id, Type: chType},
			DisplayName: "Channel " + id,
		}
	}
	return items
}

// pressKey sends a KeyMsg to the model and returns the updated Model.
func pressKey(m Model, msg tea.KeyMsg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func keyJ() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")} }
func keyK() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")} }
func keyG() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")} }
func keyEnter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }

// ─── Model tests ───────────────────────────────────────────────────────────

// 1. j key moves cursor down, clamps at bottom.
func TestCursorMovement_Down(t *testing.T) {
	m := newModel()
	items := makeItems(3, "O")
	m.SetItems(items)

	// Press j twice: cursor should move from 0 → 1 → 2.
	m = pressKey(m, keyJ())
	if m.cursor != 1 {
		t.Errorf("after 1 j press: expected cursor=1, got %d", m.cursor)
	}
	m = pressKey(m, keyJ())
	if m.cursor != 2 {
		t.Errorf("after 2 j presses: expected cursor=2, got %d", m.cursor)
	}

	// Press j again at the bottom: should clamp at 2.
	m = pressKey(m, keyJ())
	if m.cursor != 2 {
		t.Errorf("clamped at bottom: expected cursor=2, got %d", m.cursor)
	}
}

// 2. k key moves cursor up, clamps at top.
func TestCursorMovement_Up(t *testing.T) {
	m := newModel()
	items := makeItems(3, "O")
	m.SetItems(items)

	// Move cursor to index 2.
	m = pressKey(m, keyJ())
	m = pressKey(m, keyJ())
	if m.cursor != 2 {
		t.Fatalf("setup: expected cursor=2, got %d", m.cursor)
	}

	// Press k twice: 2 → 1 → 0.
	m = pressKey(m, keyK())
	if m.cursor != 1 {
		t.Errorf("after 1 k press: expected cursor=1, got %d", m.cursor)
	}
	m = pressKey(m, keyK())
	if m.cursor != 0 {
		t.Errorf("after 2 k presses: expected cursor=0, got %d", m.cursor)
	}

	// Press k again at the top: should clamp at 0.
	m = pressKey(m, keyK())
	if m.cursor != 0 {
		t.Errorf("clamped at top: expected cursor=0, got %d", m.cursor)
	}
}

// 3. G key sets cursor to last item, adjusts viewOffset.
func TestJumpToBottom(t *testing.T) {
	m := newModel()
	m.SetHeight(3)
	items := makeItems(6, "O")
	m.SetItems(items)

	m = pressKey(m, keyG())

	if m.cursor != 5 {
		t.Errorf("G: expected cursor=5, got %d", m.cursor)
	}
	// viewOffset should be len(items) - height = 6 - 3 = 3.
	if m.viewOffset != 3 {
		t.Errorf("G: expected viewOffset=3, got %d", m.viewOffset)
	}
}

// 4. Virtual scroll: height=3, 6 items, press j 4 times → viewOffset advances.
func TestVirtualScroll(t *testing.T) {
	m := newModel()
	m.SetHeight(3)
	items := makeItems(6, "O")
	m.SetItems(items)

	// Press j 4 times (cursor goes 0→1→2→3→4).
	for i := 0; i < 4; i++ {
		m = pressKey(m, keyJ())
	}

	if m.cursor != 4 {
		t.Errorf("after 4 j presses: expected cursor=4, got %d", m.cursor)
	}
	// cursor=4, height=3: viewOffset should be 4 - 3 + 1 = 2.
	if m.viewOffset != 2 {
		t.Errorf("virtual scroll: expected viewOffset=2, got %d", m.viewOffset)
	}
}

// 5. Enter on a channel emits ChannelSelectedMsg with correct ChannelID.
func TestEnterEmitsChannelSelectedMsg(t *testing.T) {
	m := newModel()
	items := makeItems(3, "O")
	m.SetItems(items)

	// Move cursor to index 1.
	m = pressKey(m, keyJ())

	_, cmd := m.Update(keyEnter())
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}

	msg := cmd()
	selectedMsg, ok := msg.(messages.ChannelSelectedMsg)
	if !ok {
		t.Fatalf("expected ChannelSelectedMsg, got %T", msg)
	}
	if selectedMsg.ChannelID != items[1].Channel.ID {
		t.Errorf("expected ChannelID=%q, got %q", items[1].Channel.ID, selectedMsg.ChannelID)
	}
}

// 6. IncrementUnread increments the count for the given channelID.
func TestIncrementUnread(t *testing.T) {
	m := newModel()
	items := makeItems(3, "O")
	m.SetItems(items)

	targetID := items[1].Channel.ID
	m.IncrementUnread(targetID)
	m.IncrementUnread(targetID)

	for _, it := range m.items {
		if it.Channel.ID == targetID {
			if it.UnreadCount != 2 {
				t.Errorf("expected UnreadCount=2 for %q, got %d", targetID, it.UnreadCount)
			}
			return
		}
	}
	t.Errorf("channel %q not found in items", targetID)
}

// 7. ClearUnread resets count to 0.
func TestClearUnread(t *testing.T) {
	m := newModel()
	items := makeItems(3, "O")
	m.SetItems(items)

	targetID := items[0].Channel.ID
	m.IncrementUnread(targetID)
	m.IncrementUnread(targetID)
	m.ClearUnread(targetID)

	for _, it := range m.items {
		if it.Channel.ID == targetID {
			if it.UnreadCount != 0 {
				t.Errorf("expected UnreadCount=0 after ClearUnread, got %d", it.UnreadCount)
			}
			return
		}
	}
	t.Errorf("channel %q not found in items", targetID)
}

// 8. Fresh model returns nil from SelectedChannel().
func TestSelectedChannel_None(t *testing.T) {
	m := newModel()
	if got := m.SelectedChannel(); got != nil {
		t.Errorf("expected nil SelectedChannel on fresh model, got %+v", got)
	}
}

// 9. Items with type "D" always appear before "O" and "P" in the view.
func TestDMSortedFirst(t *testing.T) {
	m := newModel()
	// Mix types in intentionally unsorted order: O, D, P.
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-o", DisplayName: "Open", Type: "O"}, DisplayName: "Open"},
		{Channel: api.Channel{ID: "ch-d", DisplayName: "Direct", Type: "D"}, DisplayName: "Direct"},
		{Channel: api.Channel{ID: "ch-p", DisplayName: "Private", Type: "P"}, DisplayName: "Private"},
	}
	m.SetItems(items)

	view := m.View()

	dIdx := strings.Index(view, "Direct")
	oIdx := strings.Index(view, "Open")
	pIdx := strings.Index(view, "Private")

	if dIdx < 0 || oIdx < 0 || pIdx < 0 {
		t.Fatalf("view missing expected names: view=%q", view)
	}
	if dIdx >= oIdx {
		t.Errorf("DM should appear before Open channel: dIdx=%d oIdx=%d", dIdx, oIdx)
	}
	if oIdx >= pIdx {
		t.Errorf("Open channel should appear before Private channel: oIdx=%d pIdx=%d", oIdx, pIdx)
	}
}

// ─── View tests ─────────────────────────────────────────────────────────────

// 10. Muted channel renders the muted icon in the row.
func TestMutedIconInView(t *testing.T) {
	m := newModel()
	items := []ChannelItem{
		{
			Channel:     api.Channel{ID: "ch-m", DisplayName: "Muted Channel", Type: "O"},
			DisplayName: "Muted Channel",
			IsMuted:     true,
		},
	}
	m.SetItems(items)

	view := m.View()
	if !strings.Contains(view, "🔇") {
		t.Errorf("expected muted icon 🔇 in view, got:\n%s", view)
	}
}

// 11. Channel with unread=3 renders "3" in the row.
func TestUnreadBadgeInView(t *testing.T) {
	m := newModel()
	items := []ChannelItem{
		{
			Channel:     api.Channel{ID: "ch-u", DisplayName: "Unread Channel", Type: "O"},
			DisplayName: "Unread Channel",
			UnreadCount: 3,
		},
	}
	m.SetItems(items)

	view := m.View()
	if !strings.Contains(view, "3") {
		t.Errorf("expected badge '3' in view, got:\n%s", view)
	}
}

// 12. View contains "DIRECT MESSAGES" and "CHANNELS" when both types present.
func TestSectionHeaders(t *testing.T) {
	m := newModel()
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-d", DisplayName: "DM User", Type: "D"}, DisplayName: "DM User"},
		{Channel: api.Channel{ID: "ch-o", DisplayName: "General", Type: "O"}, DisplayName: "General"},
	}
	m.SetItems(items)

	view := m.View()
	if !strings.Contains(view, "DIRECT MESSAGES") {
		t.Errorf("expected 'DIRECT MESSAGES' section header in view, got:\n%s", view)
	}
	if !strings.Contains(view, "CHANNELS") {
		t.Errorf("expected 'CHANNELS' section header in view, got:\n%s", view)
	}
}

// 13. Each row's rendered width ≤ styles.SidebarWidth.
func TestRenderedWidthWithinBounds(t *testing.T) {
	m := newModel()
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-d", DisplayName: "Alice Smith", Type: "D"}, DisplayName: "Alice Smith"},
		{Channel: api.Channel{ID: "ch-o", DisplayName: "General Channel", Type: "O"}, DisplayName: "General Channel", UnreadCount: 42},
		{Channel: api.Channel{ID: "ch-p", DisplayName: "Very Long Private Channel Name Here", Type: "P"}, DisplayName: "Very Long Private Channel Name Here"},
		{Channel: api.Channel{ID: "ch-m", DisplayName: "Muted One", Type: "O"}, DisplayName: "Muted One", IsMuted: true, UnreadCount: 5},
	}
	m.SetItems(items)

	view := m.View()
	for _, row := range strings.Split(view, "\n") {
		w := lipgloss.Width(row)
		if w > styles.SidebarWidth {
			t.Errorf("row exceeds SidebarWidth (%d): width=%d row=%q", styles.SidebarWidth, w, row)
		}
	}
}
