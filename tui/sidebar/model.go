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
	items      []ChannelItem
	cursor     int              // highlighted row index
	selected   int              // currently open channel index (-1 = none)
	viewOffset int              // virtual scroll: first visible item index
	height     int              // visible rows available (set by AppModel on resize)
	keys       keymap.KeyMap
	userID     string           // current user ID (to resolve DM names)
	userCache  map[string]api.User // keyed by userID for DM name resolution
}

// NewModel returns an empty sidebar model.
func NewModel(keys keymap.KeyMap, userID string) Model {
	return Model{
		items:     nil,
		cursor:    0,
		selected:  -1,
		height:    20,
		keys:      keys,
		userID:    userID,
		userCache: make(map[string]api.User),
	}
}

// sortItems sorts m.items in-place: DMs first, then channels, each section
// ordered by LastPostAt descending (most recently active first), then by
// DisplayName for ties. This must be called whenever m.items changes so that
// cursor indices stay aligned with the displayed order.
func (m *Model) sortItems() {
	sort.SliceStable(m.items, func(i, j int) bool {
		ti := channelTypeOrder(m.items[i].Channel.Type)
		tj := channelTypeOrder(m.items[j].Channel.Type)
		if ti != tj {
			return ti < tj
		}
		lpi := m.items[i].Channel.LastPostAt
		lpj := m.items[j].Channel.LastPostAt
		if lpi != lpj {
			return lpi > lpj // most recent first
		}
		return m.items[i].DisplayName < m.items[j].DisplayName
	})
}

// SetItems replaces the channel list (called after API fetch).
// Preserves cursor/selected positions where possible.
func (m *Model) SetItems(items []ChannelItem) {
	// Strip ANSI escape sequences from server-supplied display names.
	for i := range items {
		items[i].DisplayName = stripANSI(items[i].DisplayName)
	}
	m.items = items
	m.sortItems()
	// Clamp cursor to valid range.
	if len(items) == 0 {
		m.cursor = 0
		m.selected = -1
		m.viewOffset = 0
		return
	}
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
	if m.selected >= len(items) {
		m.selected = -1
	}
	// Clamp viewOffset.
	maxOffset := len(items) - m.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}
}

// Items returns the current channel items (used by tests to verify state).
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
func (m *Model) IncrementUnread(channelID string) {
	for i := range m.items {
		if m.items[i].Channel.ID == channelID {
			m.items[i].UnreadCount++
			m.items[i].Channel.LastPostAt = time.Now().UnixMilli()
			m.sortItems()
			return
		}
	}
}

// ClearUnread sets the unread count to 0 for the given channelID.
func (m *Model) ClearUnread(channelID string) {
	for i := range m.items {
		if m.items[i].Channel.ID == channelID {
			m.items[i].UnreadCount = 0
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
				// Advance viewOffset if cursor scrolled past visible area.
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
				// Retreat viewOffset if cursor scrolled before visible area.
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
