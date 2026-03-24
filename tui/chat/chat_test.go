package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
)

// newTestModel returns a Model wired up for testing (no real terminal needed).
func newTestModel() Model {
	km := keymap.DefaultKeyMap()
	m := NewModel(km, "")
	// Give the viewport a non-zero size so AtBottom()/GotoBottom() work correctly.
	m.viewport = viewport.New(80, 20)
	return m
}

// makePostList builds a PostList map from a slice of posts, using post.ID as the key.
func makePostList(posts []api.Post) api.PostList {
	pl := api.PostList{
		Posts: make(map[string]api.Post, len(posts)),
	}
	for _, p := range posts {
		pl.Posts[p.ID] = p
	}
	return pl
}

// ────────────────────────────────────────────────────────────────────────────
// 1. LoadPosts – chronological ordering
// ────────────────────────────────────────────────────────────────────────────

func TestLoadPosts_ChronologicalOrder(t *testing.T) {
	posts := []api.Post{
		{ID: "c", UserID: "u1", Message: "third", CreateAt: 300},
		{ID: "a", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "b", UserID: "u1", Message: "second", CreateAt: 200},
	}
	pl := makePostList(posts)

	m := newTestModel()
	m.LoadPosts("ch1", "general", pl, nil)

	if len(m.posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(m.posts))
	}
	if m.posts[0].CreateAt != 100 {
		t.Errorf("post[0] CreateAt = %d, want 100", m.posts[0].CreateAt)
	}
	if m.posts[1].CreateAt != 200 {
		t.Errorf("post[1] CreateAt = %d, want 200", m.posts[1].CreateAt)
	}
	if m.posts[2].CreateAt != 300 {
		t.Errorf("post[2] CreateAt = %d, want 300", m.posts[2].CreateAt)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 2. LoadPosts – deleted posts are filtered out
// ────────────────────────────────────────────────────────────────────────────

func TestLoadPosts_DeletedFiltered(t *testing.T) {
	posts := []api.Post{
		{ID: "alive", UserID: "u1", Message: "hello", CreateAt: 100},
		{ID: "dead", UserID: "u1", Message: "deleted", CreateAt: 200, DeleteAt: 999},
	}
	pl := makePostList(posts)

	m := newTestModel()
	m.LoadPosts("ch1", "general", pl, nil)

	if len(m.posts) != 1 {
		t.Fatalf("expected 1 post after filtering deleted, got %d", len(m.posts))
	}
	if m.posts[0].ID != "alive" {
		t.Errorf("expected post ID 'alive', got '%s'", m.posts[0].ID)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 3. AppendPost – auto-scrolls when viewport was at bottom
// ────────────────────────────────────────────────────────────────────────────

func TestAppendPost_ScrollsToBottom(t *testing.T) {
	m := newTestModel()
	// Start with an empty channel; viewport is naturally at bottom.
	m.LoadPosts("ch1", "general", makePostList(nil), nil)

	// Confirm viewport is at bottom before append.
	if !m.viewport.AtBottom() {
		t.Fatal("precondition failed: viewport should be at bottom before AppendPost")
	}

	m.AppendPost(api.Post{ID: "p1", UserID: "u1", Message: "hello", CreateAt: 1})

	if !m.viewport.AtBottom() {
		t.Error("viewport should still be at bottom after AppendPost when it was already at bottom")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 4. AppendPost – does NOT auto-scroll when viewport is not at bottom
// ────────────────────────────────────────────────────────────────────────────

func TestAppendPost_NoScrollWhenNotAtBottom(t *testing.T) {
	m := newTestModel()

	// Load enough content to make the viewport scrollable.
	posts := make([]api.Post, 50)
	for i := range posts {
		posts[i] = api.Post{
			ID:       string(rune('a' + i%26)),
			UserID:   "u1",
			Message:  strings.Repeat("line content ", 3),
			CreateAt: int64(i + 1),
		}
		// Ensure unique IDs.
		posts[i].ID = posts[i].ID + string(rune('0'+i/26))
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	// Scroll all the way up so we are NOT at the bottom.
	m.viewport.GotoTop()

	if m.viewport.AtBottom() {
		t.Skip("viewport is too small to have a non-bottom position; skipping test")
	}

	// Capture scroll position before.
	yBefore := m.viewport.YOffset

	m.AppendPost(api.Post{ID: "new", UserID: "u1", Message: "new message", CreateAt: 9999})

	// The YOffset should not have jumped to the bottom.
	if m.viewport.AtBottom() {
		t.Error("viewport should NOT scroll to bottom when it was not already at bottom")
	}
	_ = yBefore // yBefore is informational; main assertion is AtBottom()
}

// ────────────────────────────────────────────────────────────────────────────
// 5. RenderPosts – consecutive posts from same user: only first shows username
// ────────────────────────────────────────────────────────────────────────────

func TestRenderPosts_MessageGrouping(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "alice", Message: "msg1", CreateAt: 100},
		{ID: "2", UserID: "alice", Message: "msg2", CreateAt: 200},
		{ID: "3", UserID: "alice", Message: "msg3", CreateAt: 300},
	}
	userCache := map[string]api.User{
		"alice": {ID: "alice", Username: "alice"},
	}

	rendered := RenderPosts(posts, userCache, "", 0, nil, nil, false, -1, 0, -1, -1)

	// Strip ANSI so we can count occurrences in plain text.
	plain := stripANSI(rendered)

	count := strings.Count(plain, "alice")
	if count != 1 {
		t.Errorf("username 'alice' should appear exactly once (first post only), got %d", count)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 6. RenderPosts – system message rendered with dimmed style
// ────────────────────────────────────────────────────────────────────────────

func TestRenderPosts_SystemMessage(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "system", Message: "Alice joined the channel", CreateAt: 100, Type: "system_join_channel"},
	}

	rendered := RenderPosts(posts, nil, "", 0, nil, nil, false, -1, 0, -1, -1)

	// System message content should appear in the rendered output
	if !strings.Contains(rendered, posts[0].Message) && !strings.Contains(rendered, "system") {
		t.Error("system message content not found in rendered output")
	}
	// System message should NOT show username+timestamp header (it's grouped differently)
	if strings.Contains(rendered, "username") {
		t.Error("system message should not show username header")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 7. RenderPosts – edited indicator
// ────────────────────────────────────────────────────────────────────────────

func TestRenderPosts_EditedIndicator(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "original text", CreateAt: 100, EditAt: 200},
	}

	rendered := RenderPosts(posts, nil, "", 0, nil, nil, false, -1, 0, -1, -1)
	plain := stripANSI(rendered)

	if !strings.Contains(plain, "(edited)") {
		t.Errorf("expected '(edited)' in rendered output, got: %q", plain)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 8. RenderPosts – ANSI escape sequences stripped from message content
// ────────────────────────────────────────────────────────────────────────────

func TestRenderPosts_ANSIStripped(t *testing.T) {
	ansiMsg := "\x1b[2Jhidden garbage\x1b[0m normal text"
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: ansiMsg, CreateAt: 100},
	}

	rendered := RenderPosts(posts, nil, "", 0, nil, nil, false, -1, 0, -1, -1)
	plain := stripANSI(rendered)

	// The raw ANSI clear-screen sequence \x1b[2J must not appear in the plain output.
	if strings.Contains(plain, "\x1b[2J") {
		t.Error("ANSI escape sequence \\x1b[2J should have been stripped from the message")
	}
	// The human-readable text after stripping should remain.
	if !strings.Contains(plain, "normal text") {
		t.Errorf("expected 'normal text' to survive stripping, got: %q", plain)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 9. FormatTimestamp – today renders as "HH:MM"
// ────────────────────────────────────────────────────────────────────────────

func TestFormatTimestamp_Today(t *testing.T) {
	// Use a time that is definitely today (right now minus 1 minute).
	ts := time.Now().Add(-1 * time.Minute)
	ms := ts.UnixMilli()

	result := FormatTimestamp(ms)

	// Expected format: "HH:MM" (exactly 5 characters, digits and colon).
	if len(result) != 5 {
		t.Errorf("FormatTimestamp for today: got %q (len %d), want HH:MM (len 5)", result, len(result))
	}
	if result[2] != ':' {
		t.Errorf("FormatTimestamp for today: got %q, expected colon at index 2", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 10. FormatTimestamp – yesterday renders as "Yesterday HH:MM"
// ────────────────────────────────────────────────────────────────────────────

func TestFormatTimestamp_Yesterday(t *testing.T) {
	// Use noon yesterday so the result is always "yesterday", regardless of current time-of-day.
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 12, 0, 0, 0, now.Location())
	ms := yesterday.UnixMilli()

	result := FormatTimestamp(ms)

	if !strings.HasPrefix(result, "Yesterday ") {
		t.Errorf("FormatTimestamp for yesterday: got %q, want prefix 'Yesterday '", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 11. FormatTimestamp – 2 days ago renders as "DD/MM HH:MM"
// ────────────────────────────────────────────────────────────────────────────

func TestFormatTimestamp_Older(t *testing.T) {
	ts := time.Now().Add(-48 * time.Hour)
	ms := ts.UnixMilli()

	result := FormatTimestamp(ms)

	// Should NOT start with "Yesterday" and should contain two slashes (DD/MM HH:MM).
	if strings.HasPrefix(result, "Yesterday") {
		t.Errorf("FormatTimestamp for 2 days ago: got %q, should not start with 'Yesterday'", result)
	}
	// Format is "02/01 15:04" — expect a '/' at index 2.
	if len(result) < 3 || result[2] != '/' {
		t.Errorf("FormatTimestamp for 2 days ago: got %q, expected DD/MM HH:MM format", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 12. resolveUsername – userID not in cache returns "unknown"
// ────────────────────────────────────────────────────────────────────────────

func TestResolveUsername_Fallback(t *testing.T) {
	cache := map[string]api.User{
		"other": {ID: "other", Username: "other_user"},
	}
	result := resolveUsername("missing_id", cache, false)
	if result != "unknown" {
		t.Errorf("resolveUsername with missing ID: got %q, want 'unknown'", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 13. resolveUsername – contact name vs account name modes
// ────────────────────────────────────────────────────────────────────────────

func TestResolveUsername_ContactName(t *testing.T) {
	cache := map[string]api.User{
		"u1": {ID: "u1", Username: "plain_username", FirstName: "Alice", LastName: "Smith"},
	}
	if got := resolveUsername("u1", cache, true); got != "Alice Smith" {
		t.Errorf("contact mode: got %q, want 'Alice Smith'", got)
	}
	if got := resolveUsername("u1", cache, false); got != "plain_username" {
		t.Errorf("account mode: got %q, want 'plain_username'", got)
	}
}

func TestResolveUsername_ContactFallback(t *testing.T) {
	// No first/last name → falls back to username even in contact mode.
	cache := map[string]api.User{
		"u1": {ID: "u1", Username: "plain_username"},
	}
	if got := resolveUsername("u1", cache, true); got != "plain_username" {
		t.Errorf("contact mode fallback: got %q, want 'plain_username'", got)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 14. PrependPosts – older posts inserted at top, chronological
// ────────────────────────────────────────────────────────────────────────────

func TestPrependPosts_InsertsAtTop(t *testing.T) {
	m := newTestModel()
	initial := []api.Post{
		{ID: "c", UserID: "u1", Message: "recent", CreateAt: 300},
	}
	m.LoadPosts("ch1", "general", makePostList(initial), nil)

	if len(m.posts) != 1 {
		t.Fatalf("precondition: expected 1 post, got %d", len(m.posts))
	}

	older := []api.Post{
		{ID: "a", UserID: "u1", Message: "oldest", CreateAt: 100},
		{ID: "b", UserID: "u1", Message: "middle", CreateAt: 200},
	}
	m.PrependPosts(makePostList(older), 1)

	if len(m.posts) != 3 {
		t.Fatalf("expected 3 posts after prepend, got %d", len(m.posts))
	}
	// Oldest should be first.
	if m.posts[0].ID != "a" {
		t.Errorf("post[0] should be 'a' (oldest), got %q", m.posts[0].ID)
	}
	if m.posts[1].ID != "b" {
		t.Errorf("post[1] should be 'b', got %q", m.posts[1].ID)
	}
	if m.posts[2].ID != "c" {
		t.Errorf("post[2] should be 'c' (newest), got %q", m.posts[2].ID)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 15. PrependPosts – duplicates are filtered out
// ────────────────────────────────────────────────────────────────────────────

func TestPrependPosts_Deduplicates(t *testing.T) {
	m := newTestModel()
	initial := []api.Post{
		{ID: "a", UserID: "u1", Message: "existing", CreateAt: 100},
	}
	m.LoadPosts("ch1", "general", makePostList(initial), nil)

	// Prepend the same post ID — should not create a duplicate.
	dupe := []api.Post{
		{ID: "a", UserID: "u1", Message: "existing", CreateAt: 100},
		{ID: "b", UserID: "u1", Message: "new older", CreateAt: 50},
	}
	m.PrependPosts(makePostList(dupe), 1)

	if len(m.posts) != 2 {
		t.Fatalf("expected 2 posts (1 new + 1 existing), got %d", len(m.posts))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 16. PrependPosts – deleted posts are filtered out
// ────────────────────────────────────────────────────────────────────────────

func TestPrependPosts_DeletedFiltered(t *testing.T) {
	m := newTestModel()
	m.LoadPosts("ch1", "general", makePostList(nil), nil)

	older := []api.Post{
		{ID: "alive", UserID: "u1", Message: "ok", CreateAt: 50},
		{ID: "dead", UserID: "u1", Message: "deleted", CreateAt: 40, DeleteAt: 999},
	}
	m.PrependPosts(makePostList(older), 1)

	if len(m.posts) != 1 {
		t.Fatalf("expected 1 post (deleted filtered), got %d", len(m.posts))
	}
	if m.posts[0].ID != "alive" {
		t.Errorf("expected post 'alive', got %q", m.posts[0].ID)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 17. PrependPosts – empty result is a no-op
// ────────────────────────────────────────────────────────────────────────────

func TestPrependPosts_EmptyNoOp(t *testing.T) {
	m := newTestModel()
	initial := []api.Post{
		{ID: "a", UserID: "u1", Message: "hello", CreateAt: 100},
	}
	m.LoadPosts("ch1", "general", makePostList(initial), nil)

	m.PrependPosts(makePostList(nil), 1)

	if len(m.posts) != 1 {
		t.Fatalf("expected 1 post unchanged, got %d", len(m.posts))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 18. SetChannelInfo — sets channel ID and name without clearing posts
// ────────────────────────────────────────────────────────────────────────────

func TestSetChannelInfo(t *testing.T) {
	m := newTestModel()
	initial := []api.Post{
		{ID: "a", UserID: "u1", Message: "hello", CreateAt: 100},
	}
	m.LoadPosts("ch1", "general", makePostList(initial), nil)

	m.SetChannelInfo("ch2", "Random")

	if m.ChannelID() != "ch2" {
		t.Errorf("ChannelID should be 'ch2', got %q", m.ChannelID())
	}
	if m.channelName != "Random" {
		t.Errorf("channelName should be 'Random', got %q", m.channelName)
	}
	// Posts should not be cleared.
	if len(m.posts) != 1 {
		t.Errorf("posts should not be cleared by SetChannelInfo, got %d", len(m.posts))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 19. SetError — sets and clears error banner
// ────────────────────────────────────────────────────────────────────────────

func TestSetError(t *testing.T) {
	m := newTestModel()

	m.SetError(nil)
	if m.err != nil {
		t.Error("SetError(nil) should clear error")
	}

	m.SetError(errForTest("timeout"))
	if m.err == nil {
		t.Fatal("SetError should set error")
	}
	if m.err.Error() != "timeout" {
		t.Errorf("expected error 'timeout', got %q", m.err.Error())
	}

	// View should include the error.
	view := m.View()
	if !strings.Contains(view, "timeout") {
		t.Error("View should display the error banner")
	}

	// Clear the error.
	m.SetError(nil)
	if m.err != nil {
		t.Error("SetError(nil) should clear error")
	}
}

// errForTest creates a simple error for testing.
type errForTest string

func (e errForTest) Error() string { return string(e) }

// ────────────────────────────────────────────────────────────────────────────
// 20. Error banner dismiss — Esc key clears the error
// ────────────────────────────────────────────────────────────────────────────

func TestErrorBannerDismiss(t *testing.T) {
	// Use DefaultKeyMap so that Esc (FocusSidebar) key matches.
	km := keymap.DefaultKeyMap()
	m := NewModel(km, "")
	m.viewport = viewport.New(80, 20)
	m.SetError(errForTest("some error"))
	m.width = 80
	m.height = 20

	if m.err == nil {
		t.Fatal("precondition: error should be set")
	}

	// Press Esc (FocusSidebar key) to dismiss the error banner.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(Model)

	if m.err != nil {
		t.Errorf("Esc should dismiss the error banner, but error is still: %v", m.err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 21. Pagination — scrolling to top emits LoadMorePostsMsg
// ────────────────────────────────────────────────────────────────────────────

func TestScrollToTop_EmitsLoadMorePostsMsg(t *testing.T) {
	m := newTestModel()

	// Load some posts so the condition len(m.posts) > 0 is satisfied.
	posts := []api.Post{
		{ID: "p1", UserID: "u1", Message: "hello", CreateAt: 1000},
		{ID: "p2", UserID: "u1", Message: "world", CreateAt: 2000},
	}
	pl := makePostList(posts)
	m.LoadPosts("ch1", "General", pl, nil)

	// Scroll viewport to the bottom so we can then scroll back up.
	m.viewport.GotoBottom()

	// Scroll up until at top. Each LineUp(1) moves one line.
	for i := 0; i < 30; i++ {
		m.viewport.LineUp(1)
	}

	// Now press 'k' (Up) while already at top — should emit LoadMorePostsMsg.
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = result.(Model)

	if cmd == nil {
		t.Fatal("expected a Cmd to be returned when scrolling to top with posts loaded")
	}

	msg := cmd()
	lmp, ok := msg.(messages.LoadMorePostsMsg)
	if !ok {
		t.Fatalf("expected LoadMorePostsMsg, got %T", msg)
	}
	if lmp.ChannelID != "ch1" {
		t.Errorf("LoadMorePostsMsg.ChannelID = %q, want %q", lmp.ChannelID, "ch1")
	}
	if lmp.Page != 1 {
		t.Errorf("LoadMorePostsMsg.Page = %d, want 1", lmp.Page)
	}
	// loadingMore should be set to prevent duplicate triggers.
	if !m.loadingMore {
		t.Error("m.loadingMore should be true after emitting LoadMorePostsMsg")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 22. FormatTimestamp — epoch 0 (edge case)
// ────────────────────────────────────────────────────────────────────────────

func TestFormatTimestamp_Epoch(t *testing.T) {
	result := FormatTimestamp(0)
	// Epoch (1970-01-01) should render as DD/MM HH:MM format (very old).
	if len(result) == 0 {
		t.Error("FormatTimestamp(0) should return a non-empty string")
	}
	if strings.HasPrefix(result, "Yesterday") {
		t.Error("epoch should not be 'Yesterday'")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 22. FormatTimestamp — future timestamp (edge case)
// ────────────────────────────────────────────────────────────────────────────

func TestFormatTimestamp_Future(t *testing.T) {
	// A timestamp 1 hour in the future should render as today's HH:MM.
	future := time.Now().Add(1 * time.Hour).UnixMilli()
	result := FormatTimestamp(future)

	if len(result) != 5 || result[2] != ':' {
		t.Errorf("future timestamp should render as HH:MM, got %q", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 23. gg double-press scrolls viewport to the top.
// ────────────────────────────────────────────────────────────────────────────

func chatPressKey(m Model, msg tea.KeyMsg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func TestGGScrollsToTop(t *testing.T) {
	m := newTestModel()
	posts := make([]api.Post, 30)
	for i := range posts {
		posts[i] = api.Post{ID: string(rune('a'+i%26)) + string(rune('0'+i/26)), UserID: "u1", Message: "msg", CreateAt: int64(i + 1)}
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	// Scroll to the bottom first.
	m.viewport.GotoBottom()
	if m.viewport.AtTop() {
		t.Skip("viewport too small to test scroll; skipping")
	}

	// First g: arms pendingG.
	m = chatPressKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if !m.pendingG {
		t.Error("after first g: expected pendingG=true")
	}

	// Second g: fires GotoTop.
	m = chatPressKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.pendingG {
		t.Error("after second g: expected pendingG=false")
	}
	if !m.viewport.AtTop() {
		t.Errorf("after gg: viewport should be at top (YOffset=%d)", m.viewport.YOffset)
	}
}

// 24. A non-g key resets pendingG in chat without scrolling to top.
func TestChatGGCancelledByOtherKey(t *testing.T) {
	m := newTestModel()
	posts := make([]api.Post, 30)
	for i := range posts {
		posts[i] = api.Post{ID: string(rune('a'+i%26)) + string(rune('0'+i/26)), UserID: "u1", Message: "msg", CreateAt: int64(i + 1)}
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)
	m.viewport.GotoBottom()

	// Arm pendingG.
	m = chatPressKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if !m.pendingG {
		t.Fatalf("expected pendingG=true after first g")
	}

	// Press k — should cancel pendingG but NOT jump to top.
	m = chatPressKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.pendingG {
		t.Error("pendingG should be reset after pressing k")
	}
	if m.viewport.AtTop() {
		t.Error("viewport should not have jumped to top after g then k")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Per-message cursor tests (Task #16)
// ────────────────────────────────────────────────────────────────────────────

// TestCursorUp moves cursor from the last to the first post.
func TestCursorUp(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "second", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "third", CreateAt: 300},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	// Start at the last post.
	if m.CursorIndex() != 2 {
		t.Fatalf("precondition: cursor should be at last post (index 2), got %d", m.CursorIndex())
	}

	// Move up twice.
	m.CursorUp()
	if m.CursorIndex() != 1 {
		t.Errorf("after CursorUp: expected index 1, got %d", m.CursorIndex())
	}

	m.CursorUp()
	if m.CursorIndex() != 0 {
		t.Errorf("after second CursorUp: expected index 0, got %d", m.CursorIndex())
	}

	// One more up should clamp at 0.
	m.CursorUp()
	if m.CursorIndex() != 0 {
		t.Errorf("after third CursorUp (at top): expected index 0 (clamped), got %d", m.CursorIndex())
	}
}

// TestCursorDown moves cursor from the first to the last post.
func TestCursorDown(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "second", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "third", CreateAt: 300},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	// Reset cursor to the first post.
	m.cursor = 0

	// Move down twice.
	m.CursorDown()
	if m.CursorIndex() != 1 {
		t.Errorf("after CursorDown: expected index 1, got %d", m.CursorIndex())
	}

	m.CursorDown()
	if m.CursorIndex() != 2 {
		t.Errorf("after second CursorDown: expected index 2, got %d", m.CursorIndex())
	}

	// One more down should clamp at the last index.
	m.CursorDown()
	if m.CursorIndex() != 2 {
		t.Errorf("after third CursorDown (at bottom): expected index 2 (clamped), got %d", m.CursorIndex())
	}
}

// TestCursorToTop jumps cursor to the first post.
func TestCursorToTop(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "second", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "third", CreateAt: 300},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	if m.CursorIndex() != 2 {
		t.Fatalf("precondition: cursor should start at last post (index 2), got %d", m.CursorIndex())
	}

	m.CursorToTop()

	if m.CursorIndex() != 0 {
		t.Errorf("after CursorToTop: expected index 0, got %d", m.CursorIndex())
	}
	if !m.viewport.AtTop() {
		t.Error("after CursorToTop: viewport should be at top")
	}
}

// TestCursorToBottom jumps cursor to the last post.
func TestCursorToBottom(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "second", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "third", CreateAt: 300},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	m.cursor = 0 // Move to first.

	m.CursorToBottom()

	if m.CursorIndex() != 2 {
		t.Errorf("after CursorToBottom: expected index 2, got %d", m.CursorIndex())
	}
	if !m.viewport.AtBottom() {
		t.Error("after CursorToBottom: viewport should be at bottom")
	}
}

// TestCursorIndex returns the current cursor position.
func TestCursorIndex(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "second", CreateAt: 200},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	if m.CursorIndex() != 1 {
		t.Errorf("expected initial cursor at index 1, got %d", m.CursorIndex())
	}

	m.cursor = 0
	if m.CursorIndex() != 0 {
		t.Errorf("expected cursor at index 0 after set, got %d", m.CursorIndex())
	}
}

// TestCursorPost returns the post at the cursor position.
func TestCursorPost(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "first", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "second", CreateAt: 200},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	post, ok := m.CursorPost()
	if !ok {
		t.Fatal("expected CursorPost to succeed")
	}
	if post.ID != "2" {
		t.Errorf("expected post ID '2', got %q", post.ID)
	}
	if post.Message != "second" {
		t.Errorf("expected message 'second', got %q", post.Message)
	}

	// Test when no posts.
	m.cursor = -1
	_, ok = m.CursorPost()
	if ok {
		t.Error("expected CursorPost to fail when no posts")
	}
}

// TestCursorUpEmptyPosts is a no-op when there are no posts.
func TestCursorUpEmptyPosts(t *testing.T) {
	m := newTestModel()
	m.LoadPosts("ch1", "general", makePostList(nil), nil)

	if m.CursorIndex() != -1 {
		t.Fatalf("precondition: expected cursor -1 for empty channel, got %d", m.CursorIndex())
	}

	// Should not panic or crash.
	m.CursorUp()
	if m.CursorIndex() != -1 {
		t.Errorf("CursorUp on empty posts should remain -1, got %d", m.CursorIndex())
	}
}

// TestCursorDownEmptyPosts is a no-op when there are no posts.
func TestCursorDownEmptyPosts(t *testing.T) {
	m := newTestModel()
	m.LoadPosts("ch1", "general", makePostList(nil), nil)

	if m.CursorIndex() != -1 {
		t.Fatalf("precondition: expected cursor -1 for empty channel, got %d", m.CursorIndex())
	}

	// Should not panic or crash.
	m.CursorDown()
	if m.CursorIndex() != -1 {
		t.Errorf("CursorDown on empty posts should remain -1, got %d", m.CursorIndex())
	}
}

// TestCursorToTopEmptyPosts is a no-op when there are no posts.
func TestCursorToTopEmptyPosts(t *testing.T) {
	m := newTestModel()
	m.LoadPosts("ch1", "general", makePostList(nil), nil)

	// Should not panic or crash.
	m.CursorToTop()
	if m.CursorIndex() != -1 {
		t.Errorf("CursorToTop on empty posts should remain -1, got %d", m.CursorIndex())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Unread divider tests (Task #16)
// ────────────────────────────────────────────────────────────────────────────

// TestUnreadDivider_Before inserts divider before the first unread post.
func TestUnreadDivider_Before(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "read1", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "read2", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "unread1", CreateAt: 400},
		{ID: "4", UserID: "u1", Message: "unread2", CreateAt: 500},
	}
	userCache := map[string]api.User{
		"u1": {ID: "u1", Username: "alice"},
	}

	// lastViewedAt = 300, so posts at 100 and 200 are read; 400 and 500 are unread.
	rendered := RenderPosts(posts, userCache, "", 80, nil, nil, false, -1, 300, -1, -1)
	plain := stripANSI(rendered)

	if !strings.Contains(plain, "unread") {
		t.Errorf("expected 'unread' divider in output, got: %q", plain)
	}

	// The divider should appear before unread1, so unread1 should come after the divider text.
	dividerIdx := strings.Index(plain, "unread")
	unread1Idx := strings.Index(plain, "unread1")
	if dividerIdx >= unread1Idx {
		t.Error("divider should appear before unread1 message")
	}
}

// TestUnreadDivider_NoInsertWhenNoLastViewedAt does not insert divider when lastViewedAt is 0.
func TestUnreadDivider_NoInsertWhenNoLastViewedAt(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg1", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "msg2", CreateAt: 200},
	}
	userCache := map[string]api.User{
		"u1": {ID: "u1", Username: "alice"},
	}

	// lastViewedAt = 0 (default) — no divider should be inserted.
	rendered := RenderPosts(posts, userCache, "", 80, nil, nil, false, -1, 0, -1, -1)
	plain := stripANSI(rendered)

	// Should not contain the "unread" text marker.
	if strings.Contains(plain, "unread") {
		t.Error("divider should not appear when lastViewedAt == 0")
	}
}

// TestUnreadDivider_OnlyOnce divider is inserted only once (before first unread).
func TestUnreadDivider_OnlyOnce(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "read", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "unread1", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "unread2", CreateAt: 300},
		{ID: "4", UserID: "u1", Message: "unread3", CreateAt: 400},
	}
	userCache := map[string]api.User{
		"u1": {ID: "u1", Username: "alice"},
	}

	// lastViewedAt = 150 → posts at 200+ are unread.
	rendered := RenderPosts(posts, userCache, "", 80, nil, nil, false, -1, 150, -1, -1)
	plain := stripANSI(rendered)

	// Count occurrences of the unread divider marker.
	dividerCount := strings.Count(plain, "unread ────")
	if dividerCount != 1 {
		t.Errorf("expected exactly 1 divider, found %d", dividerCount)
	}
}

// TestUnreadDivider_AllRead does not insert divider when all posts are read.
func TestUnreadDivider_AllRead(t *testing.T) {
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg1", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "msg2", CreateAt: 200},
	}
	userCache := map[string]api.User{
		"u1": {ID: "u1", Username: "alice"},
	}

	// lastViewedAt = 300 (after all posts) — no unread divider.
	rendered := RenderPosts(posts, userCache, "", 80, nil, nil, false, -1, 300, -1, -1)
	plain := stripANSI(rendered)

	if strings.Contains(plain, "unread") {
		t.Error("divider should not appear when all posts are read (lastViewedAt after all posts)")
	}
}

// TestCursorToUnread jumps cursor to the first unread post.
func TestCursorToUnread(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "read1", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "read2", CreateAt: 200},
		{ID: "3", UserID: "u1", Message: "unread1", CreateAt: 400},
		{ID: "4", UserID: "u1", Message: "unread2", CreateAt: 500},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)
	m.SetLastViewedAt(300) // Posts at 400+ are unread.

	m.CursorToUnread()

	if m.CursorIndex() != 2 {
		t.Errorf("expected cursor at index 2 (first unread), got %d", m.CursorIndex())
	}
	post, ok := m.CursorPost()
	if !ok {
		t.Fatal("expected CursorPost to succeed")
	}
	if post.ID != "3" {
		t.Errorf("expected post '3', got %q", post.ID)
	}
}

// TestCursorToUnread_NoLastViewedAt falls back to bottom when lastViewedAt == 0.
func TestCursorToUnread_NoLastViewedAt(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg1", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "msg2", CreateAt: 200},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)
	// m.lastViewedAt stays 0 (default)

	m.CursorToUnread()

	if m.CursorIndex() != 1 {
		t.Errorf("expected cursor at bottom (index 1) when no lastViewedAt, got %d", m.CursorIndex())
	}
}

// TestCursorToUnread_AllRead falls back to bottom when all posts are read.
func TestCursorToUnread_AllRead(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg1", CreateAt: 100},
		{ID: "2", UserID: "u1", Message: "msg2", CreateAt: 200},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)
	m.SetLastViewedAt(300) // After all posts.

	m.CursorToUnread()

	if m.CursorIndex() != 1 {
		t.Errorf("expected cursor at bottom (index 1) when all posts are read, got %d", m.CursorIndex())
	}
}

// TestSetLastViewedAt stores the timestamp.
func TestSetLastViewedAt(t *testing.T) {
	m := newTestModel()

	m.SetLastViewedAt(12345)
	if m.LastViewedAt() != 12345 {
		t.Errorf("expected LastViewedAt 12345, got %d", m.LastViewedAt())
	}

	m.SetLastViewedAt(0)
	if m.LastViewedAt() != 0 {
		t.Errorf("expected LastViewedAt 0, got %d", m.LastViewedAt())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Attachment picker tests (Task #16)
// ────────────────────────────────────────────────────────────────────────────

// TestActivatePicker opens the picker with files.
func TestActivatePicker(t *testing.T) {
	m := newTestModel()
	files := []api.FileInfo{
		{ID: "f1", Name: "doc.pdf", Size: 1024},
		{ID: "f2", Name: "image.png", Size: 2048},
	}

	m.ActivatePicker(files)

	if !m.IsPickerActive() {
		t.Error("expected picker to be active")
	}
	if len(m.PickerFiles()) != 2 {
		t.Errorf("expected 2 files in picker, got %d", len(m.PickerFiles()))
	}
	if m.PickerCursorIndex() != 0 {
		t.Errorf("expected picker cursor at 0, got %d", m.PickerCursorIndex())
	}
}

// TestClosePicker dismisses the picker.
func TestClosePicker(t *testing.T) {
	m := newTestModel()
	files := []api.FileInfo{
		{ID: "f1", Name: "doc.pdf", Size: 1024},
	}
	m.ActivatePicker(files)

	if !m.IsPickerActive() {
		t.Fatal("precondition: picker should be active")
	}

	m.ClosePicker()

	if m.IsPickerActive() {
		t.Error("expected picker to be inactive after close")
	}
	if len(m.PickerFiles()) != 0 {
		t.Errorf("expected picker files cleared, got %d", len(m.PickerFiles()))
	}
	if m.PickerCursorIndex() != 0 {
		t.Errorf("expected picker cursor reset to 0, got %d", m.PickerCursorIndex())
	}
}

// TestIsPickerActive reports picker state.
func TestIsPickerActive(t *testing.T) {
	m := newTestModel()

	if m.IsPickerActive() {
		t.Error("expected picker inactive initially")
	}

	files := []api.FileInfo{{ID: "f1", Name: "file.txt", Size: 512}}
	m.ActivatePicker(files)

	if !m.IsPickerActive() {
		t.Error("expected picker active after activation")
	}

	m.ClosePicker()

	if m.IsPickerActive() {
		t.Error("expected picker inactive after close")
	}
}

// TestPickerCursorUp moves cursor up in the picker.
func TestPickerCursorUp(t *testing.T) {
	m := newTestModel()
	files := []api.FileInfo{
		{ID: "f1", Name: "first.txt", Size: 100},
		{ID: "f2", Name: "second.txt", Size: 200},
		{ID: "f3", Name: "third.txt", Size: 300},
	}
	m.ActivatePicker(files)

	// Start at the last file.
	if m.PickerCursorIndex() != 0 {
		t.Fatalf("precondition: expected cursor at 0, got %d", m.PickerCursorIndex())
	}

	m.PickerCursorDown()
	m.PickerCursorDown()
	if m.PickerCursorIndex() != 2 {
		t.Fatalf("precondition: expected cursor at 2 after 2x down, got %d", m.PickerCursorIndex())
	}

	m.PickerCursorUp()
	if m.PickerCursorIndex() != 1 {
		t.Errorf("after PickerCursorUp: expected 1, got %d", m.PickerCursorIndex())
	}

	m.PickerCursorUp()
	if m.PickerCursorIndex() != 0 {
		t.Errorf("after second PickerCursorUp: expected 0, got %d", m.PickerCursorIndex())
	}

	// One more up should clamp at 0.
	m.PickerCursorUp()
	if m.PickerCursorIndex() != 0 {
		t.Errorf("at picker top, PickerCursorUp should clamp at 0, got %d", m.PickerCursorIndex())
	}
}

// TestPickerCursorDown moves cursor down in the picker.
func TestPickerCursorDown(t *testing.T) {
	m := newTestModel()
	files := []api.FileInfo{
		{ID: "f1", Name: "first.txt", Size: 100},
		{ID: "f2", Name: "second.txt", Size: 200},
		{ID: "f3", Name: "third.txt", Size: 300},
	}
	m.ActivatePicker(files)

	if m.PickerCursorIndex() != 0 {
		t.Fatalf("precondition: expected cursor at 0, got %d", m.PickerCursorIndex())
	}

	m.PickerCursorDown()
	if m.PickerCursorIndex() != 1 {
		t.Errorf("after PickerCursorDown: expected 1, got %d", m.PickerCursorIndex())
	}

	m.PickerCursorDown()
	if m.PickerCursorIndex() != 2 {
		t.Errorf("after second PickerCursorDown: expected 2, got %d", m.PickerCursorIndex())
	}

	// One more down should clamp at 2 (last index).
	m.PickerCursorDown()
	if m.PickerCursorIndex() != 2 {
		t.Errorf("at picker bottom, PickerCursorDown should clamp at 2, got %d", m.PickerCursorIndex())
	}
}

// TestPickerSelected returns the file at the picker cursor.
func TestPickerSelected(t *testing.T) {
	m := newTestModel()
	files := []api.FileInfo{
		{ID: "f1", Name: "first.txt", Size: 100},
		{ID: "f2", Name: "second.txt", Size: 200},
	}
	m.ActivatePicker(files)

	fi, ok := m.PickerSelected()
	if !ok {
		t.Fatal("expected PickerSelected to succeed")
	}
	if fi.ID != "f1" {
		t.Errorf("expected file ID 'f1', got %q", fi.ID)
	}

	m.PickerCursorDown()
	fi, ok = m.PickerSelected()
	if !ok {
		t.Fatal("expected PickerSelected to succeed")
	}
	if fi.ID != "f2" {
		t.Errorf("expected file ID 'f2', got %q", fi.ID)
	}
}

// TestPickerSelected_InactiveReturnsNil when picker is not active.
func TestPickerSelected_InactiveReturnsNil(t *testing.T) {
	m := newTestModel()

	if m.IsPickerActive() {
		t.Fatal("precondition: picker should not be active")
	}

	_, ok := m.PickerSelected()
	if ok {
		t.Error("expected PickerSelected to fail when picker is inactive")
	}
}

// TestPickerSelected_EmptyReturnsNil when picker has no files.
func TestPickerSelected_EmptyReturnsNil(t *testing.T) {
	m := newTestModel()
	m.ActivatePicker(nil)

	if !m.IsPickerActive() {
		t.Fatal("precondition: picker should be active")
	}
	if len(m.PickerFiles()) != 0 {
		t.Fatal("precondition: picker should have no files")
	}

	_, ok := m.PickerSelected()
	if ok {
		t.Error("expected PickerSelected to fail when picker has no files")
	}
}

// TestPickerFiles returns the file list.
func TestPickerFiles(t *testing.T) {
	m := newTestModel()
	files := []api.FileInfo{
		{ID: "f1", Name: "doc.txt", Size: 100},
		{ID: "f2", Name: "image.png", Size: 200},
	}
	m.ActivatePicker(files)

	result := m.PickerFiles()
	if len(result) != 2 {
		t.Errorf("expected 2 files, got %d", len(result))
	}
	if result[0].ID != "f1" {
		t.Errorf("expected first file 'f1', got %q", result[0].ID)
	}
	if result[1].ID != "f2" {
		t.Errorf("expected second file 'f2', got %q", result[1].ID)
	}
}

// TestPickerCursorDownEmptyFiles is a no-op on empty picker.
func TestPickerCursorDownEmptyFiles(t *testing.T) {
	m := newTestModel()
	m.ActivatePicker(nil)

	// Should not panic.
	m.PickerCursorDown()

	if m.PickerCursorIndex() != 0 {
		t.Errorf("cursor should remain 0 on empty picker, got %d", m.PickerCursorIndex())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// File open tests (Task #16)
// ────────────────────────────────────────────────────────────────────────────

// TestOpenAttachmentForCursor_Single opens single attachment and returns OpenFileMsg.
func TestOpenAttachmentForCursor_Single(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg with file", CreateAt: 100, FileIds: []string{"f1"}},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	fileInfo := api.FileInfo{ID: "f1", Name: "document.pdf", Size: 5000}
	m.SetFileInfoCache(map[string]api.FileInfo{"f1": fileInfo})

	cmd := m.OpenAttachmentForCursor()

	if cmd == nil {
		t.Fatal("expected a Cmd to be returned")
	}

	msg := cmd()
	openMsg, ok := msg.(messages.OpenFileMsg)
	if !ok {
		t.Fatalf("expected OpenFileMsg, got %T", msg)
	}
	if openMsg.File.ID != "f1" {
		t.Errorf("expected file ID 'f1', got %q", openMsg.File.ID)
	}
}

// TestOpenAttachmentForCursor_Multiple opens multi-attachment picker.
func TestOpenAttachmentForCursor_Multiple(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg with files", CreateAt: 100, FileIds: []string{"f1", "f2"}},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	files := map[string]api.FileInfo{
		"f1": {ID: "f1", Name: "doc.pdf", Size: 1000},
		"f2": {ID: "f2", Name: "image.png", Size: 2000},
	}
	m.SetFileInfoCache(files)

	cmd := m.OpenAttachmentForCursor()

	// Should activate picker instead of returning a command.
	if cmd != nil {
		t.Error("expected nil Cmd when opening multiple attachments (picker activated)")
	}
	if !m.IsPickerActive() {
		t.Error("expected picker to be active when message has multiple attachments")
	}
	if len(m.PickerFiles()) != 2 {
		t.Errorf("expected 2 files in picker, got %d", len(m.PickerFiles()))
	}
}

// TestOpenAttachmentForCursor_NoCachedInfo returns nil when file info not cached.
func TestOpenAttachmentForCursor_NoCachedInfo(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "msg with file", CreateAt: 100, FileIds: []string{"f1"}},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	// Don't set file info cache — it's empty.

	cmd := m.OpenAttachmentForCursor()

	if cmd != nil {
		t.Error("expected nil Cmd when file info not yet cached")
	}
	if m.IsPickerActive() {
		t.Error("expected picker to remain inactive when file info is missing")
	}
}

// TestOpenAttachmentForCursor_NoFiles returns nil when post has no attachments.
func TestOpenAttachmentForCursor_NoFiles(t *testing.T) {
	m := newTestModel()
	posts := []api.Post{
		{ID: "1", UserID: "u1", Message: "just text", CreateAt: 100},
	}
	m.LoadPosts("ch1", "general", makePostList(posts), nil)

	cmd := m.OpenAttachmentForCursor()

	if cmd != nil {
		t.Error("expected nil Cmd when post has no attachments")
	}
}

// TestOpenAttachmentForCursor_NoCursorPost returns nil when cursor out of range.
func TestOpenAttachmentForCursor_NoCursorPost(t *testing.T) {
	m := newTestModel()
	m.LoadPosts("ch1", "general", makePostList(nil), nil)

	if m.CursorIndex() != -1 {
		t.Fatalf("precondition: cursor should be -1 for empty channel, got %d", m.CursorIndex())
	}

	cmd := m.OpenAttachmentForCursor()

	if cmd != nil {
		t.Error("expected nil Cmd when cursor is out of range")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Attachment open routing tests
// ────────────────────────────────────────────────────────────────────────────

// TestOpenAttachment_ImageInCache_OpensFile verifies that a cached image file
// still returns an OpenFileMsg Cmd (opens in OS native viewer).
func TestOpenAttachment_ImageInCache_OpensFile(t *testing.T) {
	m := newTestModel()
	m.SetSize(80, 20)

	fileID := "img123"
	post := api.Post{
		ID:       "p1",
		UserID:   "u1",
		Message:  "check this",
		CreateAt: 100,
		FileIds:  []string{fileID},
	}
	m.LoadPosts("ch1", "general", makePostList([]api.Post{post}), nil)

	m.fileInfoCache = map[string]api.FileInfo{
		fileID: {ID: fileID, Name: "photo.png", Size: 1024, MimeType: "image/png"},
	}
	m.imageCache = map[string]string{
		fileID: "ANSI art here",
	}

	cmd := m.OpenAttachmentForCursor()

	if cmd == nil {
		t.Error("expected non-nil Cmd (OpenFileMsg) for image file")
	}
}

// TestOpenAttachment_NonImageFile_OpensFile verifies that a non-image file
// returns an OpenFileMsg Cmd.
func TestOpenAttachment_NonImageFile_OpensFile(t *testing.T) {
	m := newTestModel()
	m.SetSize(80, 20)

	fileID := "doc789"
	post := api.Post{
		ID:       "p1",
		UserID:   "u1",
		Message:  "check this",
		CreateAt: 100,
		FileIds:  []string{fileID},
	}
	m.LoadPosts("ch1", "general", makePostList([]api.Post{post}), nil)

	m.fileInfoCache = map[string]api.FileInfo{
		fileID: {ID: fileID, Name: "document.pdf", Size: 2048, MimeType: "application/pdf"},
	}
	m.imageCache = make(map[string]string)

	cmd := m.OpenAttachmentForCursor()

	if cmd == nil {
		t.Error("expected non-nil Cmd (OpenFileMsg) for non-image file")
	}
}
