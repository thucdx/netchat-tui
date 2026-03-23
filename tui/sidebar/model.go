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
	"github.com/thucdx/netchat-tui/tui/styles"
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
	ContactName string            // first+last name; empty for public/private channels
	AccountName string            // username; empty for public/private channels
	UnreadCount int64
	IsMuted     bool
}

// Model is the sidebar Bubbletea sub-model.
type Model struct {
	allItems       []ChannelItem     // full sorted list of all channels
	items          []ChannelItem     // top-limit slice of allItems (displayed)
	limit          int               // max channels to show (default 200)
	cursor         int               // index into items
	selected       int               // index into items (-1 = none)
	viewOffset     int               // virtual scroll: first visible item index
	height         int               // visible rows available (set by AppModel on resize)
	width          int               // total sidebar width including border (set by AppModel)
	pendingG       bool              // true after first 'g' press (for gg jump-to-top)
	keys           keymap.KeyMap
	userID         string            // current user ID (to resolve DM names)
	userCache      map[string]api.User
	search         searchState       // search mode state (zero value = inactive)
	useContactName bool              // true = show first+last name; false = show username
}

// NewModel returns an empty sidebar model.
func NewModel(keys keymap.KeyMap, userID string) Model {
	return Model{
		limit:          defaultLimit,
		cursor:         0,
		selected:       -1,
		height:         20,
		width:          styles.SidebarWidth,
		keys:           keys,
		userID:         userID,
		userCache:      make(map[string]api.User),
		useContactName: true,
	}
}

// SetWidth updates the total sidebar width (including border).
// Called by AppModel on resize or drag.
func (m *Model) SetWidth(w int) {
	if w < 10 {
		w = 10
	}
	m.width = w
}

// visibleHeight returns the number of item rows the view can actually render.
// When the indicator row is shown (items exceed height), one row is reserved.
func (m Model) visibleHeight() int {
	if len(m.items) > m.height && m.height > 1 {
		return m.height - 1
	}
	return m.height
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

	// Clamp viewOffset to max (never increase it — auto-scrolling down would
	// hide newly-active channels that bubbled to the top on each IncrementUnread).
	vh := m.visibleHeight()
	maxOffset := len(m.items) - vh
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
		items[i].ContactName = stripANSI(items[i].ContactName)
		items[i].AccountName = stripANSI(items[i].AccountName)
	}
	m.allItems = items
	m.cursor = 0
	m.selected = -1
	m.viewOffset = 0
	m.applyDisplayNames()
}

// Items returns the currently displayed channel items (top-limit after sort).
func (m Model) Items() []ChannelItem {
	return m.items
}

// UpsertItem adds or updates a channel item in allItems, then re-sorts.
// Used when a new DM or joined channel should appear in the sidebar.
func (m *Model) UpsertItem(item ChannelItem) {
	item.DisplayName = stripANSI(item.DisplayName)
	item.ContactName = stripANSI(item.ContactName)
	item.AccountName = stripANSI(item.AccountName)
	for i := range m.allItems {
		if m.allItems[i].Channel.ID == item.Channel.ID {
			m.allItems[i] = item
			m.applyDisplayNames()
			return
		}
	}
	m.allItems = append(m.allItems, item)
	m.applyDisplayNames()
}

// ToggleDisplayName flips between contact name and account name display.
func (m *Model) ToggleDisplayName() {
	m.useContactName = !m.useContactName
	m.applyDisplayNames()
}

// UseContactName reports whether contact name mode is active.
func (m Model) UseContactName() bool {
	return m.useContactName
}

// applyDisplayNames updates DisplayName for all DM/group items based on the
// current mode, then re-sorts and rebuilds the displayed list.
func (m *Model) applyDisplayNames() {
	for i := range m.allItems {
		item := &m.allItems[i]
		if item.AccountName == "" {
			continue // public/private channel — DisplayName already set correctly
		}
		if m.useContactName && item.ContactName != "" {
			item.DisplayName = item.ContactName
		} else {
			item.DisplayName = item.AccountName
		}
	}
	m.sortAndRebuild()
}

// UnreadSummary returns total unread messages and number of channels with
// unreads, counting only unmuted channels. Uses allItems so channels outside
// the current top-N window are included.
func (m Model) UnreadSummary() (totalMsgs int64, channelCount int) {
	for _, item := range m.allItems {
		if item.IsMuted || item.UnreadCount == 0 {
			continue
		}
		totalMsgs += item.UnreadCount
		channelCount++
	}
	return
}

// ExitSearch exits search mode. Safe to call when not searching.
func (m *Model) ExitSearch() {
	m.exitSearch()
}

// IsSearching reports whether the sidebar is currently in search mode.
func (m Model) IsSearching() bool {
	return m.search.active
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
		// Search mode intercepts all key events when active.
		if m.search.active {
			return m.updateSearch(msg)
		}

		// Enter search mode.
		if key.Matches(msg, m.keys.Search) {
			m.enterSearch()
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.JumpToTop):
			if m.pendingG {
				// Second g: jump to top
				m.cursor = 0
				m.viewOffset = 0
				m.pendingG = false
			} else {
				// First g: arm pending
				m.pendingG = true
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			m.pendingG = false
			if len(m.items) > 0 {
				m.cursor++
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
				vh := m.visibleHeight()
				if m.cursor > m.viewOffset+vh-1 {
					m.viewOffset = m.cursor - vh + 1
				}
			}

		case key.Matches(msg, m.keys.Up):
			m.pendingG = false
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
			m.pendingG = false
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				offset := len(m.items) - m.visibleHeight()
				if offset < 0 {
					offset = 0
				}
				m.viewOffset = offset
			}

		case key.Matches(msg, m.keys.ScrollUp):
			m.pendingG = false
			vh := m.visibleHeight()
			m.viewOffset -= vh / 2
			if m.viewOffset < 0 {
				m.viewOffset = 0
			}
			// Keep cursor inside the visible window.
			if m.cursor > m.viewOffset+vh-1 {
				m.cursor = m.viewOffset + vh - 1
			}

		case key.Matches(msg, m.keys.ScrollDown):
			m.pendingG = false
			vh := m.visibleHeight()
			maxOffset := len(m.items) - vh
			if maxOffset < 0 {
				maxOffset = 0
			}
			m.viewOffset += vh / 2
			if m.viewOffset > maxOffset {
				m.viewOffset = maxOffset
			}
			// Keep cursor inside the visible window.
			if m.cursor < m.viewOffset {
				m.cursor = m.viewOffset
			}

		case key.Matches(msg, m.keys.Select):
			m.pendingG = false
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
