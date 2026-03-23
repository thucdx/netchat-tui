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

func keyJ() tea.KeyMsg        { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")} }
func keyK() tea.KeyMsg        { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")} }
func keyG() tea.KeyMsg        { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")} }
func keyLowercaseG() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")} }
func keyEnter() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyEnter} }
func keyCtrlU() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyCtrlU} }
func keyCtrlD() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyCtrlD} }

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
	// visibleHeight = height-1 = 2 (indicator takes 1 row when items > height).
	// viewOffset = len(items) - visibleHeight = 6 - 2 = 4.
	if m.viewOffset != 4 {
		t.Errorf("G: expected viewOffset=4, got %d", m.viewOffset)
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
	// visibleHeight = height-1 = 2; cursor=4: viewOffset = 4 - 2 + 1 = 3.
	if m.viewOffset != 3 {
		t.Errorf("virtual scroll: expected viewOffset=3, got %d", m.viewOffset)
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

// 9. Items are sorted by LastPostAt descending (most recent first).
func TestSortedByRecency(t *testing.T) {
	m := newModel()
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-o", DisplayName: "Open", Type: "O", LastPostAt: 100}, DisplayName: "Open"},
		{Channel: api.Channel{ID: "ch-d", DisplayName: "Direct", Type: "D", LastPostAt: 300}, DisplayName: "Direct"},
		{Channel: api.Channel{ID: "ch-p", DisplayName: "Private", Type: "P", LastPostAt: 200}, DisplayName: "Private"},
	}
	m.SetItems(items)

	view := m.View()

	dIdx := strings.Index(view, "Direct")
	pIdx := strings.Index(view, "Private")
	oIdx := strings.Index(view, "Open")

	if dIdx < 0 || oIdx < 0 || pIdx < 0 {
		t.Fatalf("view missing expected names: view=%q", view)
	}
	if dIdx >= pIdx {
		t.Errorf("Direct (300) should appear before Private (200): dIdx=%d pIdx=%d", dIdx, pIdx)
	}
	if pIdx >= oIdx {
		t.Errorf("Private (200) should appear before Open (100): pIdx=%d oIdx=%d", pIdx, oIdx)
	}
}

// 9b. Ctrl+D scrolls viewOffset down by half-page; cursor clamps to new window top.
func TestCtrlDScrollsDown(t *testing.T) {
	m := newModel()
	m.SetHeight(4)
	items := makeItems(10, "O")
	m.SetItems(items)

	// viewOffset starts at 0, cursor at 0.
	m = pressKey(m, keyCtrlD())

	// visibleHeight = 4-1 = 3 (indicator row); half-page = 3/2 = 1; viewOffset moves to 1.
	if m.viewOffset != 1 {
		t.Errorf("after ctrl+d: expected viewOffset=1, got %d", m.viewOffset)
	}
	// cursor was at 0, which is now below viewOffset=1, so it clamps to 1.
	if m.cursor != 1 {
		t.Errorf("after ctrl+d: cursor should clamp to viewOffset=1, got %d", m.cursor)
	}
}

// 9c. Ctrl+D clamps at max offset.
func TestCtrlDClampsAtBottom(t *testing.T) {
	m := newModel()
	m.SetHeight(4)
	items := makeItems(6, "O")
	m.SetItems(items)

	// visibleHeight = 3; max offset = 6 - 3 = 3; should clamp at 3.
	m = pressKey(m, keyCtrlD())
	m = pressKey(m, keyCtrlD())
	m = pressKey(m, keyCtrlD())
	m = pressKey(m, keyCtrlD())

	if m.viewOffset != 3 {
		t.Errorf("ctrl+d clamped: expected viewOffset=3, got %d", m.viewOffset)
	}
}

// 9d. Ctrl+U scrolls viewOffset up by half-page; cursor clamps to new window bottom.
func TestCtrlUScrollsUp(t *testing.T) {
	m := newModel()
	m.SetHeight(4)
	items := makeItems(10, "O")
	m.SetItems(items)

	// visibleHeight=3, half=1. Press Ctrl+D 4 times to reach viewOffset=4.
	for i := 0; i < 4; i++ {
		m = pressKey(m, keyCtrlD())
	}
	if m.viewOffset != 4 {
		t.Fatalf("setup: expected viewOffset=4, got %d", m.viewOffset)
	}

	// Move cursor to bottom of visible window: viewOffset+visibleHeight-1 = 4+3-1 = 6.
	for m.cursor < m.viewOffset+m.visibleHeight()-1 {
		m = pressKey(m, keyJ())
	}
	if m.cursor != 6 {
		t.Fatalf("setup: expected cursor=6, got %d", m.cursor)
	}

	// Ctrl+U: viewOffset -= 1 → 3; cursor=6 > 3+3-1=5, clamps to 5.
	m = pressKey(m, keyCtrlU())
	if m.viewOffset != 3 {
		t.Errorf("after ctrl+u: expected viewOffset=3, got %d", m.viewOffset)
	}
	if m.cursor != 5 {
		t.Errorf("after ctrl+u: cursor should clamp to viewOffset+visibleHeight-1=5, got %d", m.cursor)
	}
}

// 9e. Ctrl+U clamps at 0.
func TestCtrlUClampsAtTop(t *testing.T) {
	m := newModel()
	m.SetHeight(4)
	items := makeItems(10, "O")
	m.SetItems(items)

	m = pressKey(m, keyCtrlU())
	m = pressKey(m, keyCtrlU())

	if m.viewOffset != 0 {
		t.Errorf("ctrl+u clamped: expected viewOffset=0, got %d", m.viewOffset)
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
	// Public muted channel uses ⊘ (the combined muted-public icon).
	if !strings.Contains(view, "⊘") {
		t.Errorf("expected muted icon ⊘ in view, got:\n%s", view)
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

// 12. View does NOT contain section headers (flat list).
func TestNoSectionHeaders(t *testing.T) {
	m := newModel()
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-d", DisplayName: "DM User", Type: "D"}, DisplayName: "DM User"},
		{Channel: api.Channel{ID: "ch-o", DisplayName: "General", Type: "O"}, DisplayName: "General"},
	}
	m.SetItems(items)

	view := m.View()
	if strings.Contains(view, "DIRECT MESSAGES") {
		t.Errorf("unexpected 'DIRECT MESSAGES' section header in view:\n%s", view)
	}
	if strings.Contains(view, "CHANNELS") {
		t.Errorf("unexpected 'CHANNELS' section header in view:\n%s", view)
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

// 13b. Rendered view has exactly 1 line per item (no word-wrap overflow).
func TestRenderedRowsAreOneLine(t *testing.T) {
	m := newModel()
	m.SetHeight(10)
	items := makeItems(20, "O") // more items than height → indicator row shown
	m.SetItems(items)

	view := m.View()
	lines := strings.Split(view, "\n")

	// visibleHeight = 10-1 = 9 items + 1 indicator = 10 lines.
	want := 10
	if len(lines) != want {
		t.Errorf("expected %d rendered lines, got %d\nview:\n%s", want, len(lines), view)
	}
}

// 14. gg double-press jumps cursor to top and resets viewOffset.
func TestGGJumpsToTop(t *testing.T) {
	m := newModel()
	m.SetHeight(3)
	items := makeItems(6, "O")
	m.SetItems(items)

	// Move to bottom first.
	m = pressKey(m, keyG())
	if m.cursor != 5 {
		t.Fatalf("setup: expected cursor=5, got %d", m.cursor)
	}

	// First g: arms pendingG.
	m = pressKey(m, keyLowercaseG())
	if !m.pendingG {
		t.Error("after first g: expected pendingG=true")
	}
	if m.cursor != 5 {
		t.Errorf("after first g: cursor should be unchanged, got %d", m.cursor)
	}

	// Second g: fires jump-to-top.
	m = pressKey(m, keyLowercaseG())
	if m.pendingG {
		t.Error("after second g: expected pendingG=false")
	}
	if m.cursor != 0 {
		t.Errorf("after gg: expected cursor=0, got %d", m.cursor)
	}
	if m.viewOffset != 0 {
		t.Errorf("after gg: expected viewOffset=0, got %d", m.viewOffset)
	}
}

// 15. A non-g key resets pendingG without jumping.
func TestGGCancelledByOtherKey(t *testing.T) {
	m := newModel()
	items := makeItems(6, "O")
	m.SetItems(items)

	// Move down so cursor is not at 0.
	m = pressKey(m, keyJ())
	m = pressKey(m, keyJ())

	// Arm pendingG.
	m = pressKey(m, keyLowercaseG())
	if !m.pendingG {
		t.Fatalf("expected pendingG=true after first g")
	}

	// Press j — should cancel pendingG and move cursor, not jump to top.
	m = pressKey(m, keyJ())
	if m.pendingG {
		t.Error("pendingG should be reset after pressing j")
	}
	if m.cursor == 0 {
		t.Error("cursor should not have jumped to 0")
	}
}
