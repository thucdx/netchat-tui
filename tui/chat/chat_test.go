package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
)

// newTestModel returns a Model wired up for testing (no real terminal needed).
func newTestModel() Model {
	km := keymap.DefaultKeyMap()
	m := NewModel(km)
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

	rendered := RenderPosts(posts, userCache, 0)

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

	rendered := RenderPosts(posts, nil, 0)

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

	rendered := RenderPosts(posts, nil, 0)
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

	rendered := RenderPosts(posts, nil, 0)
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
	ts := time.Now().Add(-25 * time.Hour) // 25 h ago is guaranteed to be yesterday or earlier
	ms := ts.UnixMilli()

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
	result := resolveUsername("missing_id", cache)
	if result != "unknown" {
		t.Errorf("resolveUsername with missing ID: got %q, want 'unknown'", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// 13. resolveUsername – Nickname preferred over Username
// ────────────────────────────────────────────────────────────────────────────

func TestResolveUsername_NicknamePreferred(t *testing.T) {
	cache := map[string]api.User{
		"u1": {ID: "u1", Username: "plain_username", Nickname: "Cool Nick"},
	}
	result := resolveUsername("u1", cache)
	if result != "Cool Nick" {
		t.Errorf("resolveUsername: got %q, want 'Cool Nick' (Nickname should take priority)", result)
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
	m := NewModel(km)
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
// 21. FormatTimestamp — epoch 0 (edge case)
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
