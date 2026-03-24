package sidebar

import (
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/messages"
)

// searchResultKind classifies each result row.
type searchResultKind int

const (
	searchKindExisting   searchResultKind = iota // already-joined channel / DM / group
	searchKindNewDM                              // new DM with a user found via API
	searchKindNewChannel                         // new public channel found via API
)

// searchResult is a single row in the search results list.
type searchResult struct {
	kind        searchResultKind
	item        *ChannelItem // non-nil for searchKindExisting
	user        *api.User    // non-nil for searchKindNewDM
	channel     *api.Channel // non-nil for searchKindNewChannel
	displayName string       // pre-computed label
	icon        string       // pre-computed icon character
	score       int          // higher = ranked first
}

// searchState holds all mutable search state; embedded into Model.
type searchState struct {
	active        bool
	query         string
	cursor        int
	results       []searchResult
	apiUsers      []api.User
	apiChannels   []api.Channel
	apiQuery      string        // query that produced the stored API results
	confirmTarget *searchResult // non-nil while waiting for y/N to join a channel
}

// ── Model helpers ─────────────────────────────────────────────────────────────

func (m *Model) enterSearch() {
	m.search = searchState{active: true}
	m.rebuildSearchResults()
}

func (m *Model) exitSearch() {
	m.search = searchState{}
}

// SetSearchAPIResults stores API results and rebuilds the result list.
// Stale results (query mismatch) are silently dropped.
func (m *Model) SetSearchAPIResults(query string, users []api.User, channels []api.Channel) {
	if !m.search.active || query != m.search.query {
		return
	}
	m.search.apiUsers = users
	m.search.apiChannels = channels
	m.search.apiQuery = query
	m.rebuildSearchResults()
}

// rebuildSearchResults refreshes m.search.results from allItems + stored API data.
func (m *Model) rebuildSearchResults() {
	q := strings.ToLower(m.search.query)

	// Build a set of channel IDs already in allItems for deduplication.
	existingIDs := make(map[string]struct{}, len(m.allItems))
	for i := range m.allItems {
		existingIDs[m.allItems[i].Channel.ID] = struct{}{}
	}

	// Build a set of user IDs already reachable as DMs.
	dmUserIDs := make(map[string]struct{})
	for i := range m.allItems {
		if m.allItems[i].Channel.Type == "D" {
			otherID := dmOtherUserIDFromChannel(m.allItems[i].Channel.Name, m.userID)
			if otherID != "" {
				dmUserIDs[otherID] = struct{}{}
			}
		}
	}

	var results []searchResult

	// ── Local: filter allItems ────────────────────────────────────────────────
	if q != "" {
		for i := range m.allItems {
			item := &m.allItems[i]
			name := strings.ToLower(item.DisplayName)
			score := 0
			if strings.HasPrefix(name, q) {
				score = 3
			} else if strings.Contains(name, q) {
				score = 2
			}
			if score == 0 {
				continue
			}
			results = append(results, searchResult{
				kind:        searchKindExisting,
				item:        item,
				displayName: item.DisplayName,
				icon:        channelIcon(item.Channel.Type, item.IsMuted),
				score:       score,
			})
		}
	}

	// ── API: new users / channels (only when query matches stored API results) ──
	if q != "" && m.search.apiQuery == m.search.query {
		for i := range m.search.apiUsers {
			u := &m.search.apiUsers[i]
			if u.ID == m.userID {
				continue // skip self
			}
			if _, exists := dmUserIDs[u.ID]; exists {
				continue // DM already in sidebar
			}
			name := u.Username
			if u.FirstName != "" || u.LastName != "" {
				name = strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
			results = append(results, searchResult{
				kind:        searchKindNewDM,
				user:        u,
				displayName: u.Username,
				icon:        "+",
				score:       1,
			})
			_ = name
		}
		for i := range m.search.apiChannels {
			ch := &m.search.apiChannels[i]
			if _, exists := existingIDs[ch.ID]; exists {
				continue // already in sidebar
			}
			results = append(results, searchResult{
				kind:        searchKindNewChannel,
				channel:     ch,
				displayName: ch.DisplayName,
				icon:        "+",
				score:       1,
			})
		}
	}

	// Sort: score desc, then displayName asc.
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return strings.ToLower(results[i].displayName) < strings.ToLower(results[j].displayName)
	})

	m.search.results = results

	// Clamp cursor.
	if m.search.cursor >= len(results) {
		m.search.cursor = max(0, len(results)-1)
	}
}

// dmOtherUserIDFromChannel extracts the other user's ID from a DM channel name.
// Duplicates the helper in app.go but kept local to avoid export/import cycle.
func dmOtherUserIDFromChannel(channelName, myUserID string) string {
	parts := strings.SplitN(channelName, "__", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == myUserID {
		return parts[1]
	}
	return parts[0]
}

// ── updateSearch handles all key events while search is active ────────────────

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ── Confirm-join state ────────────────────────────────────────────────────
	if m.search.confirmTarget != nil {
		switch msg.String() {
		case "y", "enter":
			target := m.search.confirmTarget
			m.search.confirmTarget = nil
			return m, func() tea.Msg {
				return messages.JoinChannelMsg{ChannelID: target.channel.ID}
			}
		default:
			// Any other key cancels confirm and returns to search.
			m.search.confirmTarget = nil
		}
		return m, nil
	}

	// ── Normal search input ───────────────────────────────────────────────────
	switch {
	case msg.Type == tea.KeyEsc:
		m.exitSearch()
		return m, nil

	case key.Matches(msg, m.keys.Select): // Enter
		if len(m.search.results) == 0 {
			return m, nil
		}
		r := m.search.results[m.search.cursor]
		switch r.kind {
		case searchKindExisting:
			m.selected = -1
			for i, it := range m.items {
				if it.Channel.ID == r.item.Channel.ID {
					m.selected = i
					m.cursor = i
					break
				}
			}
			m.exitSearch()
			return m, func() tea.Msg {
				return messages.ChannelSelectedMsg{ChannelID: r.item.Channel.ID}
			}
		case searchKindNewDM:
			m.exitSearch()
			return m, func() tea.Msg {
				return messages.CreateDirectChannelMsg{UserID: r.user.ID}
			}
		case searchKindNewChannel:
			m.search.confirmTarget = &m.search.results[m.search.cursor]
			return m, nil
		}

	case msg.Type == tea.KeyDown:
		if m.search.cursor < len(m.search.results)-1 {
			m.search.cursor++
		}
		return m, nil

	case msg.Type == tea.KeyUp:
		if m.search.cursor > 0 {
			m.search.cursor--
		}
		return m, nil

	case msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete:
		if len(m.search.query) > 0 {
			runes := []rune(m.search.query)
			m.search.query = string(runes[:len(runes)-1])
			m.rebuildSearchResults()
		}
		return m, nil

	case msg.Type == tea.KeyRunes:
		// Only accept printable characters that aren't navigation keys already
		// matched above (j/k are matched by Down/Up when search is active).
		for _, r := range msg.Runes {
			if unicode.IsPrint(r) {
				m.search.query += string(r)
			}
		}
		m.rebuildSearchResults()
		// Fire API search when query reaches threshold.
		if len([]rune(m.search.query)) >= 3 {
			q := m.search.query
			return m, func() tea.Msg {
				return messages.TriggerSearchMsg{Query: q}
			}
		}
		return m, nil
	}

	return m, nil
}
