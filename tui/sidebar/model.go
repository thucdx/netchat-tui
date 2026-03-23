package sidebar

import (
	"regexp"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
)

// defaultLimit is the maximum number of channels shown in the sidebar.
const defaultLimit = 200

// ansiEscape matches ANSI terminal escape sequences.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[mABCDEFGHJKSTfhil]`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// ChannelItem wraps an api.Channel with computed display fields.
type ChannelItem struct {
	Channel     api.Channel
	Member      api.ChannelMember // unread counts, notify props
	DisplayName string            // resolved name (DM → "@username", else DisplayName)
	UnreadCount int64
	IsMuted     bool
}

// Model is the sidebar Bubbletea sub-model.
type Model struct {
	allItems   []ChannelItem     // full sorted list of all channels
	items      []ChannelItem     // top-limit slice of allItems (displayed)
	limit      int               // max channels to show (default 200)
	cursor     int               // index into items
	selected   int               // index into items (-1 = none)
	viewOffset int               // virtual scroll: first visible item index
	height     int               // visible rows available (set by AppModel on resize)
	keys       keymap.KeyMap
	userID     string            // current user ID (to resolve DM names)
	userCache  map[string]api.User
}

// NewModel returns an empty sidebar model.
func NewModel(keys keymap.KeyMap, userID string) Model {
	return Model{
		limit:     defaultLimit,
		cursor:    0,
		selected:  -1,
		height:    20,
		keys:      keys,
		userID:    userID,
		userCache: make(map[string]api.User),
	}
}

// SetLimit changes the maximum number of channels displayed.
func (m *Model) SetLimit(n int) {
	if n < 1 {
		n = 1
	}
	m.limit = n
	m.sortAndRebuild()
}

// sortAndRebuild sorts allItems in-place, slices it to the configured limit,
// and re-anchors cursor and selected to the same channel IDs. Call this after
// any mutation to allItems so that the displayed list and indices stay consistent.
func (m *Model) sortAndRebuild() {
	// Capture IDs before reordering.
	cursorID := ""
	if m.cursor >= 0 && m.cursor < len(m.items) {
		cursorID = m.items[m.cursor].Channel.ID
	}
	selectedID := ""
	if m.selected >= 0 && m.selected < len(m.items) {
		selectedID = m.items[m.selected].Channel.ID
	}

	// Sort allItems by LastPostAt descending; tiebreak by DisplayName ascending.
	sort.SliceStable(m.allItems, func(i, j int) bool {
		lpi := m.allItems[i].Channel.LastPostAt
		lpj := m.allItems[j].Channel.LastPostAt
		if lpi != lpj {
			return lpi > lpj // most recent first
		}
		return m.allItems[i].DisplayName < m.allItems[j].DisplayName
	})

	// Slice to limit.
	n := m.limit
	if n > len(m.allItems) {
		n = len(m.allItems)
	}
	m.items = m.allItems[:n]

	// Re-anchor cursor by ID.
	m.cursor = 0
	for i, item := range m.items {
		if item.Channel.ID == cursorID {
			m.cursor = i
			break
		}
	}

	// Re-anchor selected by ID (-1 if it fell out of the top-N window).
	m.selected = -1
	for i, item := range m.items {
		if item.Channel.ID == selectedID {
			m.selected = i
			break
		}
	}

	// Clamp viewOffset.
	maxOffset := len(m.items) - m.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}
}

// SetItems replaces the full channel list (called after API fetch).
func (m *Model) SetItems(items []ChannelItem) {
	for i := range items {
		items[i].DisplayName = stripANSI(items[i].DisplayName)
	}
	m.allItems = items
	m.cursor = 0
	m.selected = -1
	m.viewOffset = 0
	m.sortAndRebuild()
}

// Items returns the currently displayed channel items (top-limit after sort).
func (m Model) Items() []ChannelItem {
	return m.items
}

// SetHeight updates the visible row count (called on WindowSizeMsg).
func (m *Model) SetHeight(h int) {
	m.height = h
}

// SelectedChannel returns the currently open channel, or nil if none.
func (m Model) SelectedChannel() *ChannelItem {
	if m.selected < 0 || m.selected >= len(m.items) {
		return nil
	}
	item := m.items[m.selected]
	return &item
}

// IncrementUnread increments the unread count for the given channelID and
// bumps its LastPostAt to now so it rises to the top of its section.
// Searches allItems so channels outside the current top-N window are found too.
func (m *Model) IncrementUnread(channelID string) {
	for i := range m.allItems {
		if m.allItems[i].Channel.ID == channelID {
			m.allItems[i].UnreadCount++
			m.allItems[i].Channel.LastPostAt = time.Now().UnixMilli()
			m.sortAndRebuild()
			return
		}
	}
}

// ClearUnread sets the unread count to 0 for the given channelID.
// Searches allItems so channels outside the current top-N window are found.
func (m *Model) ClearUnread(channelID string) {
	for i := range m.allItems {
		if m.allItems[i].Channel.ID == channelID {
			m.allItems[i].UnreadCount = 0
			return
		}
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Down):
			if len(m.items) > 0 {
				m.cursor++
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
				if m.cursor > m.viewOffset+m.height-1 {
					m.viewOffset = m.cursor - m.height + 1
				}
			}

		case key.Matches(msg, m.keys.Up):
			if len(m.items) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 0
				}
				if m.cursor < m.viewOffset {
					m.viewOffset = m.cursor
				}
			}

		case key.Matches(msg, m.keys.JumpToBottom):
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				offset := len(m.items) - m.height
				if offset < 0 {
					offset = 0
				}
				m.viewOffset = offset
			}

		case key.Matches(msg, m.keys.ScrollUp):
			m.viewOffset -= m.height / 2
			if m.viewOffset < 0 {
				m.viewOffset = 0
			}
			// Keep cursor inside the visible window.
			if m.cursor > m.viewOffset+m.height-1 {
				m.cursor = m.viewOffset + m.height - 1
			}

		case key.Matches(msg, m.keys.ScrollDown):
			maxOffset := len(m.items) - m.height
			if maxOffset < 0 {
				maxOffset = 0
			}
			m.viewOffset += m.height / 2
			if m.viewOffset > maxOffset {
				m.viewOffset = maxOffset
			}
			// Keep cursor inside the visible window.
			if m.cursor < m.viewOffset {
				m.cursor = m.viewOffset
			}

		case key.Matches(msg, m.keys.Select):
			if len(m.items) > 0 && m.cursor >= 0 && m.cursor < len(m.items) {
				m.selected = m.cursor
				return m, func() tea.Msg {
					return messages.ChannelSelectedMsg{ChannelID: m.items[m.cursor].Channel.ID}
				}
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	return Render(m)
}
