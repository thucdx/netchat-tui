package sidebar

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/messages"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newSearchModel() Model {
	m := newModel()
	m.SetHeight(20)
	items := makeItems(5, "O")
	m.SetItems(items)
	return m
}

func typeRune(m Model, r rune) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return updated.(Model)
}

func typeString(m Model, s string) Model {
	for _, r := range s {
		m = typeRune(m, r)
	}
	return m
}

func pressBackspace(m Model) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	return updated.(Model)
}

func pressEsc(m Model) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	return updated.(Model)
}

func pressEnter(m Model) (Model, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return updated.(Model), cmd
}

func pressSlash(m Model) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	return updated.(Model)
}

// ── search activation ─────────────────────────────────────────────────────────

func TestSearchActivatedBySlash(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	if !m.search.active {
		t.Error("expected search to be active after '/'")
	}
	if m.search.query != "" {
		t.Errorf("expected empty query, got %q", m.search.query)
	}
}

func TestSearchActivatedByCtrlF(t *testing.T) {
	m := newSearchModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(Model)
	if !m.search.active {
		t.Error("expected search to be active after ctrl+f")
	}
}

// ── query building ────────────────────────────────────────────────────────────

func TestSearchQueryBuildsOnTyping(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "abc")
	if m.search.query != "abc" {
		t.Errorf("expected query 'abc', got %q", m.search.query)
	}
}

func TestSearchBackspaceRemovesLastChar(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "abc")
	m = pressBackspace(m)
	if m.search.query != "ab" {
		t.Errorf("expected query 'ab' after backspace, got %q", m.search.query)
	}
}

func TestSearchBackspaceOnEmptyQueryNoOp(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = pressBackspace(m) // should not panic
	if m.search.query != "" {
		t.Errorf("expected empty query, got %q", m.search.query)
	}
}

// ── API trigger ───────────────────────────────────────────────────────────────

func TestSearchTriggersMsgAtThreeChars(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)

	// Two chars → no cmd.
	m = typeRune(m, 'a')
	_, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = pressKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	// Third char → TriggerSearchMsg.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(Model)
	_ = cmd1

	if cmd == nil {
		t.Fatal("expected a cmd after 3rd char, got nil")
	}
	msg := cmd()
	trig, ok := msg.(messages.TriggerSearchMsg)
	if !ok {
		t.Fatalf("expected TriggerSearchMsg, got %T", msg)
	}
	if trig.Query != "abc" {
		t.Errorf("expected query 'abc', got %q", trig.Query)
	}
}

func TestSearchNoTriggerBelowThreeChars(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	_ = m
	if cmd != nil && cmd() != nil {
		msg := cmd()
		if _, ok := msg.(messages.TriggerSearchMsg); ok {
			t.Error("should not emit TriggerSearchMsg for single char")
		}
	}
}

// ── cursor navigation ─────────────────────────────────────────────────────────

func TestSearchCursorMovesWithArrows(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "cha") // matches all 5 items (DisplayName "Channel X")
	if len(m.search.results) == 0 {
		t.Fatal("expected results after typing 'cha'")
	}

	m = pressKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.search.cursor != 1 {
		t.Errorf("after down: expected cursor=1, got %d", m.search.cursor)
	}
	m = pressKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.search.cursor != 2 {
		t.Errorf("after down down: expected cursor=2, got %d", m.search.cursor)
	}
	m = pressKey(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.search.cursor != 1 {
		t.Errorf("after up: expected cursor=1, got %d", m.search.cursor)
	}
}

func TestSearchJKTypedAsText(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = pressKey(m, keyJ())
	m = pressKey(m, keyK())
	if m.search.query != "jk" {
		t.Errorf("expected query 'jk', got %q", m.search.query)
	}
	if m.search.cursor != 0 {
		t.Errorf("cursor should not move when typing j/k in search mode, got %d", m.search.cursor)
	}
}

func TestSearchCursorClampsAtBounds(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "cha")

	// Press up at top — should stay at 0.
	m = pressKey(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.search.cursor != 0 {
		t.Errorf("up at top: expected cursor=0, got %d", m.search.cursor)
	}

	// Press down past bottom.
	for i := 0; i < 10; i++ {
		m = pressKey(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.search.cursor >= len(m.search.results) {
		t.Errorf("cursor out of bounds: got %d, len=%d", m.search.cursor, len(m.search.results))
	}
}

// ── Esc exits search ──────────────────────────────────────────────────────────

func TestSearchEscExits(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "abc")
	m = pressEsc(m)
	if m.search.active {
		t.Error("expected search to be inactive after Esc")
	}
	if m.search.query != "" {
		t.Error("expected query cleared after Esc")
	}
}

// ── Enter on existing item ────────────────────────────────────────────────────

func TestSearchEnterOnExistingItemSelectsChannel(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "cha") // matches all items

	if len(m.search.results) == 0 {
		t.Fatal("need results")
	}

	m, cmd := pressEnter(m)
	if m.search.active {
		t.Error("expected search to exit after selecting existing item")
	}
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	sel, ok := msg.(messages.ChannelSelectedMsg)
	if !ok {
		t.Fatalf("expected ChannelSelectedMsg, got %T", msg)
	}
	if sel.ChannelID == "" {
		t.Error("ChannelID should not be empty")
	}
}

// ── Enter on new DM ───────────────────────────────────────────────────────────

func TestSearchEnterOnNewDMEmitsCreateDirectChannelMsg(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "xyz")

	// Inject API results with a new user.
	newUser := api.User{ID: "user-99", Username: "xyzuser"}
	m.SetSearchAPIResults("xyz", []api.User{newUser}, nil)

	if len(m.search.results) == 0 {
		t.Fatal("expected results with API user")
	}

	// Find the new DM result and navigate to it.
	idx := -1
	for i, r := range m.search.results {
		if r.kind == searchKindNewDM {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("no searchKindNewDM result found")
	}
	m.search.cursor = idx

	m, cmd := pressEnter(m)
	if m.search.active {
		t.Error("expected search to exit after selecting new DM")
	}
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	create, ok := msg.(messages.CreateDirectChannelMsg)
	if !ok {
		t.Fatalf("expected CreateDirectChannelMsg, got %T", msg)
	}
	if create.UserID != newUser.ID {
		t.Errorf("expected UserID=%q, got %q", newUser.ID, create.UserID)
	}
}

// ── Enter on new public channel → confirm state ───────────────────────────────

func TestSearchEnterOnNewChannelEntersConfirmState(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "pub")

	newCh := api.Channel{ID: "ch-pub", DisplayName: "Public Channel", Type: "O"}
	m.SetSearchAPIResults("pub", nil, []api.Channel{newCh})

	idx := -1
	for i, r := range m.search.results {
		if r.kind == searchKindNewChannel {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("no searchKindNewChannel result found")
	}
	m.search.cursor = idx

	m, cmd := pressEnter(m)
	if cmd != nil {
		t.Error("expected no cmd when entering confirm state")
	}
	if m.search.confirmTarget == nil {
		t.Fatal("expected confirmTarget to be set")
	}
	if m.search.confirmTarget.channel.ID != newCh.ID {
		t.Errorf("wrong confirmTarget channel ID: got %q", m.search.confirmTarget.channel.ID)
	}
	if !m.search.active {
		t.Error("search should still be active during confirm state")
	}
}

func TestSearchConfirmYEmitsJoinChannelMsg(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "pub")

	newCh := api.Channel{ID: "ch-pub-2", DisplayName: "Public Two", Type: "O"}
	m.SetSearchAPIResults("pub", nil, []api.Channel{newCh})
	for i, r := range m.search.results {
		if r.kind == searchKindNewChannel {
			m.search.cursor = i
			break
		}
	}
	m, _ = pressEnter(m) // enter confirm

	// Press y to confirm.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected JoinChannelMsg cmd")
	}
	msg := cmd()
	join, ok := msg.(messages.JoinChannelMsg)
	if !ok {
		t.Fatalf("expected JoinChannelMsg, got %T", msg)
	}
	if join.ChannelID != newCh.ID {
		t.Errorf("expected ChannelID=%q, got %q", newCh.ID, join.ChannelID)
	}
	if m.search.confirmTarget != nil {
		t.Error("confirmTarget should be cleared after confirm")
	}
}

func TestSearchConfirmEscCancels(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "pub")

	newCh := api.Channel{ID: "ch-pub-3", DisplayName: "Public Three", Type: "O"}
	m.SetSearchAPIResults("pub", nil, []api.Channel{newCh})
	for i, r := range m.search.results {
		if r.kind == searchKindNewChannel {
			m.search.cursor = i
			break
		}
	}
	m, _ = pressEnter(m) // enter confirm

	// Press Esc — should cancel confirm and remain in search.
	m = pressEsc(m)
	if m.search.confirmTarget != nil {
		t.Error("confirmTarget should be nil after cancel")
	}
	if !m.search.active {
		t.Error("should still be in search mode after cancel")
	}
}

// ── rebuildResults scoring ────────────────────────────────────────────────────

func TestRebuildResultsPrefixBeforeSubstring(t *testing.T) {
	m := newSearchModel()
	// Replace items with controlled names.
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-1", DisplayName: "abc-channel", Type: "O"}, DisplayName: "abc-channel"},
		{Channel: api.Channel{ID: "ch-2", DisplayName: "my-abc-channel", Type: "O"}, DisplayName: "my-abc-channel"},
	}
	m.SetItems(items)

	m = pressSlash(m)
	m = typeString(m, "abc")

	if len(m.search.results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(m.search.results))
	}
	if m.search.results[0].displayName != "abc-channel" {
		t.Errorf("prefix match should rank first, got %q", m.search.results[0].displayName)
	}
	if m.search.results[1].displayName != "my-abc-channel" {
		t.Errorf("substring match should rank second, got %q", m.search.results[1].displayName)
	}
}

func TestRebuildResultsAPIResultsRankLast(t *testing.T) {
	m := newSearchModel()
	items := []ChannelItem{
		{Channel: api.Channel{ID: "ch-1", DisplayName: "abc-local", Type: "O"}, DisplayName: "abc-local"},
	}
	m.SetItems(items)

	m = pressSlash(m)
	m = typeString(m, "abc")

	apiUser := api.User{ID: "user-api", Username: "abcuser"}
	m.SetSearchAPIResults("abc", []api.User{apiUser}, nil)

	if len(m.search.results) < 2 {
		t.Fatalf("expected ≥2 results, got %d", len(m.search.results))
	}
	// Local result should come first.
	if m.search.results[0].kind != searchKindExisting {
		t.Errorf("local result should rank first, got kind=%d", m.search.results[0].kind)
	}
	// API result last.
	last := m.search.results[len(m.search.results)-1]
	if last.kind != searchKindNewDM {
		t.Errorf("API result should rank last, got kind=%d", last.kind)
	}
}

// ── stale API results dropped ─────────────────────────────────────────────────

func TestSetSearchAPIResultsDropsStale(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "abc")

	// Inject results for a different (stale) query.
	m.SetSearchAPIResults("xyz", []api.User{{ID: "u1", Username: "xyz"}}, nil)

	// None of the results should be the API user.
	for _, r := range m.search.results {
		if r.kind == searchKindNewDM {
			t.Error("stale API result should have been dropped")
		}
	}
}

// ── view renders correctly in search mode ─────────────────────────────────────

func TestSearchViewShowsQueryLine(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "ab")

	view := m.View()
	if !strings.Contains(view, "/ ab") {
		t.Errorf("view should contain query line, got:\n%s", view)
	}
}

func TestSearchViewShowsHintWhenQueryShort(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "ab") // < 3 chars

	view := m.View()
	if !strings.Contains(view, "≥3") {
		t.Errorf("view should show hint for short query, got:\n%s", view)
	}
}

func TestSearchViewShowsResultsWhenQueryLong(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "cha") // matches "Channel X"

	view := m.View()
	// Hint should NOT appear.
	if strings.Contains(view, "≥3") {
		t.Error("hint should not appear when query ≥3 chars")
	}
	// At least one result.
	if !strings.Contains(view, "Channel") {
		t.Errorf("view should contain results, got:\n%s", view)
	}
}

func TestSearchViewConfirmLine(t *testing.T) {
	m := newSearchModel()
	m = pressSlash(m)
	m = typeString(m, "pub")

	newCh := api.Channel{ID: "ch-pub-v", DisplayName: "Public View", Type: "O"}
	m.SetSearchAPIResults("pub", nil, []api.Channel{newCh})
	for i, r := range m.search.results {
		if r.kind == searchKindNewChannel {
			m.search.cursor = i
			break
		}
	}
	m, _ = pressEnter(m)

	view := m.View()
	if !strings.Contains(view, "Join") {
		t.Errorf("confirm view should contain 'Join', got:\n%s", view)
	}
	if !strings.Contains(view, "[y/N]") {
		t.Errorf("confirm view should contain '[y/N]', got:\n%s", view)
	}
}
