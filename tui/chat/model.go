package chat

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// Model is the chat panel Bubbletea sub-model.
type Model struct {
	viewport    viewport.Model      // bubbles/viewport for scrollable content
	posts       []api.Post          // raw posts in chronological order (oldest first)
	channelID   string              // currently displayed channel
	channelName string              // for header display
	userCache   map[string]api.User // keyed by userID
	userID      string              // logged-in user's ID (to distinguish own messages)
	loading     bool
	loadingMore bool                // true while fetching older posts (pagination)
	page        int                 // current pagination page (0 = initial)
	spinner     spinner.Model
	err         error // last API error, displayed as banner
	pendingG    bool             // true after first 'g' press (for gg jump-to-top)
	keys        keymap.KeyMap
	width       int
	height      int // chat area height (excludes input)
	imageCache     map[string]string    // keyed by file ID, value is rendered ANSI image string
	fileInfoCache  map[string]api.FileInfo // metadata for all file attachments
	useContactName bool                 // true = show first+last name; false = show username

	// Per-message cursor fields (Task #11).
	cursor       int            // index into m.posts; -1 = no posts
	lastViewedAt int64          // Unix-ms from ChannelMember.LastViewedAt; set before posts load
	pickerActive bool           // true when multi-attachment picker is shown
	pickerFiles  []api.FileInfo // files shown in picker
	pickerCursor int            // highlighted row in picker

	// Image popup fields (Task #17)
	popupActive bool
	popupImage  string // pre-rendered ANSI art from imageCache
	popupTitle  string // filename

	// Visual mode fields (Features 3 & 4)
	visualMode   bool // true when V visual selection is active
	visualAnchor int  // post index where V was pressed
}

// NewModel creates a new chat Model with default spinner and loading state.
// userID is the logged-in user's ID, used to render own messages differently.
func NewModel(keys keymap.KeyMap, userID string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.SpinnerStyle

	return Model{
		loading:        true,
		spinner:        sp,
		keys:           keys,
		userID:         userID,
		userCache:      make(map[string]api.User),
		fileInfoCache:  make(map[string]api.FileInfo),
		useContactName: true,
		cursor:         -1,
	}
}

// LoadPosts is called when PostsLoadedMsg arrives.
// Sorts posts chronologically, renders content, sets viewport.
func (m *Model) LoadPosts(channelID, channelName string, postList api.PostList, userCache map[string]api.User) {
	m.channelID = channelID
	m.channelName = channelName
	m.userCache = userCache
	m.loading = false
	m.loadingMore = false
	m.page = 0
	m.err = nil

	// Collect posts from the map.
	posts := make([]api.Post, 0, len(postList.Posts))
	for _, p := range postList.Posts {
		// Skip deleted posts.
		if p.DeleteAt > 0 {
			continue
		}
		posts = append(posts, p)
	}

	// Sort chronologically (oldest first).
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreateAt < posts[j].CreateAt
	})
	m.posts = posts

	// Set cursor to the last post, or -1 if there are none.
	if len(m.posts) == 0 {
		m.cursor = -1
	} else {
		m.cursor = len(m.posts) - 1
	}

	// Render posts content into the viewport.
	vs, ve := m.VisualSelection()
	content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// AppendPost adds a new post at the bottom (from WebSocket).
// If the viewport was at the bottom before, auto-scroll after append.
// If the cursor was already at the last post before appending, it advances to the new last post.
func (m *Model) AppendPost(post api.Post) {
	if post.DeleteAt > 0 {
		return
	}
	atBottom := m.viewport.AtBottom()

	// Advance cursor if it was at the last post (i.e. user is following the tail).
	wasAtLast := m.cursor == len(m.posts)-1

	m.posts = append(m.posts, post)

	if wasAtLast {
		m.cursor = len(m.posts) - 1
	}

	vs, ve := m.VisualSelection()
	content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
	m.viewport.SetContent(content)

	if atBottom {
		m.viewport.GotoBottom()
	}
}

// UpdatePost replaces an existing post (from WebSocket post_edited event).
// If the post ID is not found in the current list, the call is a no-op.
func (m *Model) UpdatePost(post api.Post) {
	for i, p := range m.posts {
		if p.ID == post.ID {
			atBottom := m.viewport.AtBottom()
			m.posts[i] = post
			vs, ve := m.VisualSelection()
			content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
			m.viewport.SetContent(content)
			if atBottom {
				m.viewport.GotoBottom()
			}
			return
		}
	}
}

// ChannelID returns the ID of the currently displayed channel.
func (m Model) ChannelID() string {
	return m.channelID
}

// SetChannelInfo sets the active channel ID and name without loading posts.
// Used when selecting a channel before post data arrives.
func (m *Model) SetChannelInfo(channelID, channelName string) {
	m.channelID = channelID
	m.channelName = channelName
}

// SetError sets or clears the error banner.
func (m *Model) SetError(err error) {
	m.err = err
}

// PrependPosts adds older posts at the top (from pagination).
// Preserves the user's scroll position relative to existing content.
func (m *Model) PrependPosts(postList api.PostList, page int) {
	m.loadingMore = false
	m.page = page

	// Collect new posts from the map.
	newPosts := make([]api.Post, 0, len(postList.Posts))
	existing := make(map[string]struct{}, len(m.posts))
	for _, p := range m.posts {
		existing[p.ID] = struct{}{}
	}
	for _, p := range postList.Posts {
		if p.DeleteAt > 0 {
			continue
		}
		if _, dup := existing[p.ID]; dup {
			continue
		}
		newPosts = append(newPosts, p)
	}

	if len(newPosts) == 0 {
		return
	}

	// Sort new posts chronologically.
	sort.Slice(newPosts, func(i, j int) bool {
		return newPosts[i].CreateAt < newPosts[j].CreateAt
	})

	// Snapshot YOffset before prepending so we can re-anchor it after SetContent.
	prevYOffset := m.viewport.YOffset

	// Prepend to existing posts.
	m.posts = append(newPosts, m.posts...)
	// Shift cursor so it still points at the same post after prepend.
	if m.cursor >= 0 {
		m.cursor += len(newPosts)
	}

	// Measure how many lines the prepended posts add.
	newContent := RenderPosts(newPosts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, -1, 0, -1, -1)
	addedLines := strings.Count(newContent, "\n") + 1

	vs, ve := m.VisualSelection()
	content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
	m.viewport.SetContent(content)

	// Re-anchor: keep the user's reading position stable.
	m.viewport.YOffset = prevYOffset + addedLines
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Header takes 2 lines (text + border), error banner takes 1 line when present.
	headerHeight := 2
	vpHeight := height - headerHeight
	if m.err != nil {
		vpHeight--
	}
	if vpHeight < 1 {
		vpHeight = 1
	}

	atBottom := m.viewport.AtBottom()
	m.viewport.Width = width
	m.viewport.Height = vpHeight

	if len(m.posts) > 0 {
		vs, ve := m.VisualSelection()
		content := RenderPosts(m.posts, m.userCache, m.userID, width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
		m.viewport.SetContent(content)
		if atBottom {
			m.viewport.GotoBottom()
		}
	}
}

// SetUseContactName switches between contact name (first+last) and account name
// (username) display for message authors, re-rendering the viewport immediately.
func (m *Model) SetUseContactName(useContact bool) {
	if m.useContactName == useContact {
		return
	}
	m.useContactName = useContact
	if len(m.posts) > 0 {
		atBottom := m.viewport.AtBottom()
		vs, ve := m.VisualSelection()
		content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
		m.viewport.SetContent(content)
		if atBottom {
			m.viewport.GotoBottom()
		}
	}
}

// SetFileInfoCache merges new file metadata entries into the model's fileInfoCache.
// Call this when file metadata has been fetched asynchronously (e.g. on ImagesReadyMsg).
func (m *Model) SetFileInfoCache(cache map[string]api.FileInfo) {
	if m.fileInfoCache == nil {
		m.fileInfoCache = make(map[string]api.FileInfo)
	}
	for k, v := range cache {
		m.fileInfoCache[k] = v
	}
	if len(m.posts) > 0 {
		vs, ve := m.VisualSelection()
		content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
		m.viewport.SetContent(content)
	}
}

// SetImageCache updates the image cache and re-renders the viewport content.
// Call this when new images have been fetched asynchronously.
func (m Model) SetImageCache(cache map[string]string) Model {
	// Merge new entries into existing cache.
	if m.imageCache == nil {
		m.imageCache = make(map[string]string)
	}
	for k, v := range cache {
		m.imageCache[k] = v
	}
	if len(m.posts) > 0 {
		vs, ve := m.VisualSelection()
		content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
		m.viewport.SetContent(content)
	}
	return m
}

// AtTop returns true if the viewport is scrolled to the very top (for pagination trigger).
func (m Model) AtTop() bool {
	return m.viewport.AtTop()
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		// Image popup: any key dismisses it.
		if m.IsPopupActive() {
			m.CloseImagePopup()
			return m, nil
		}

		// Visual mode: Esc exits, y yanks.
		// These must be checked before the error-banner guard.
		if m.IsVisualMode() {
			switch {
			case msg.Type == tea.KeyEsc:
				m.ExitVisualMode()
				m.refreshContent()
				return m, nil
			case key.Matches(msg, m.keys.Yank): // y
				text := m.SelectedPostsText()
				m.ExitVisualMode()
				m.refreshContent()
				if text != "" {
					return m, func() tea.Msg { return messages.YankMsg{Text: text} }
				}
				return m, nil
			}
			// j/k/gg/G still run normally (they move cursor, extending selection).
			// Fall through to the existing switch below.
		}

		// Esc dismisses the error banner when one is shown.
		if msg.Type == tea.KeyEsc && m.err != nil {
			m.err = nil
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.JumpToTop):
			if m.pendingG {
				m.CursorToTop()
				m.pendingG = false
			} else {
				m.pendingG = true
			}
		case key.Matches(msg, m.keys.ScrollUp):
			m.pendingG = false
			m.viewport.HalfViewUp()
			m.clampCursorToViewport()
			m.refreshContent()
		case key.Matches(msg, m.keys.ScrollDown):
			m.pendingG = false
			m.viewport.HalfViewDown()
			m.clampCursorToViewport()
			m.refreshContent()
		case key.Matches(msg, m.keys.PageUp):
			m.pendingG = false
			m.viewport.ViewUp()
			m.clampCursorToViewport()
			m.refreshContent()
		case key.Matches(msg, m.keys.PageDown):
			m.pendingG = false
			m.viewport.ViewDown()
			m.clampCursorToViewport()
			m.refreshContent()
		case key.Matches(msg, m.keys.JumpToBottom):
			m.pendingG = false
			m.CursorToBottom()
		case key.Matches(msg, m.keys.Up):
			m.pendingG = false
			if m.IsPickerActive() {
				m.PickerCursorUp()
			} else {
				cmd := m.CursorUp()
				m.refreshContent()
				m.scrollViewportToCursor()
				cmds = append(cmds, cmd)
			}
		case key.Matches(msg, m.keys.Down):
			m.pendingG = false
			if m.IsPickerActive() {
				m.PickerCursorDown()
			} else {
				cmd := m.CursorDown()
				m.refreshContent()
				m.scrollViewportToCursor()
				cmds = append(cmds, cmd)
			}
		case key.Matches(msg, m.keys.Select):
			if m.IsPickerActive() {
				if fi, ok := m.PickerSelected(); ok {
					m.ClosePicker()
					// If it's an image with a cached render, show popup.
					if rendered, ok := m.imageCache[fi.ID]; ok && rendered != "" {
						m.ActivateImagePopup(rendered, stripANSI(fi.Name))
						return m, nil
					}
					return m, func() tea.Msg { return messages.OpenFileMsg{File: fi} }
				}
				return m, nil
			}
		case key.Matches(msg, m.keys.OpenAttachment):
			m.pendingG = false
			cmd := m.OpenAttachmentForCursor()
			cmds = append(cmds, cmd)
		case key.Matches(msg, m.keys.CloseAttachmentPicker):
			m.pendingG = false
			if m.IsPickerActive() {
				m.ClosePicker()
			}
		case key.Matches(msg, m.keys.JumpToUnread):
			m.pendingG = false
			m.CursorToUnread()
		case key.Matches(msg, m.keys.VisualMode):
			m.pendingG = false
			if m.IsVisualMode() {
				m.ExitVisualMode()
			} else {
				m.EnterVisualMode()
			}
			m.refreshContent()
		}

		// After scrolling, check if we're at the top for pagination.
		if m.viewport.AtTop() && !m.loading && !m.loadingMore && m.channelID != "" && len(m.posts) > 0 {
			m.loadingMore = true
			nextPage := m.page + 1
			channelID := m.channelID
			cmds = append(cmds, func() tea.Msg {
				return messages.LoadMorePostsMsg{
					ChannelID: channelID,
					Page:      nextPage,
				}
			})
		}

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	header := styles.ChannelHeaderWithWidth(m.width).Render(stripANSI(m.channelName))

	var body string
	if m.loading {
		body = m.spinner.View() + " Loading…"
	} else if m.IsPopupActive() {
		body = RenderImagePopup(m.popupImage, m.popupTitle, m.width, m.viewport.Height)
	} else {
		body = m.viewport.View()
	}

	view := header + "\n" + body

	if m.err != nil {
		errLine := styles.ErrorStyle.Render("Error: " + m.err.Error())
		view += "\n" + errLine
	}

	// Overlay the attachment picker at the bottom of the chat area when active.
	if m.IsPickerActive() {
		picker := RenderPicker(m.pickerFiles, m.pickerCursor, m.width)
		view += "\n" + picker
	}

	return view
}

// --- Per-message cursor API (Task #11) ---

// SetLastViewedAt stores the Unix-ms timestamp from ChannelMember.LastViewedAt.
// Call this before loading posts so CursorToUnread can find the first unread post.
func (m *Model) SetLastViewedAt(ts int64) {
	m.lastViewedAt = ts
}

// LastViewedAt returns the stored last-viewed-at timestamp (Unix ms).
func (m Model) LastViewedAt() int64 {
	return m.lastViewedAt
}

// CursorUp moves the cursor one post up (toward older messages) and clamps to 0.
// If the cursor reaches index 0 and there are more pages to load, it returns a
// LoadMorePostsCmd to trigger pagination — the same way the viewport-scroll path does.
func (m *Model) CursorUp() tea.Cmd {
	if len(m.posts) == 0 {
		return nil
	}
	if m.cursor > 0 {
		m.cursor--
	}
	// If we've reached the top and pagination is possible, trigger a load.
	if m.cursor == 0 && !m.loading && !m.loadingMore && m.channelID != "" {
		m.loadingMore = true
		nextPage := m.page + 1
		channelID := m.channelID
		return func() tea.Msg {
			return messages.LoadMorePostsMsg{
				ChannelID: channelID,
				Page:      nextPage,
			}
		}
	}
	return nil
}

// CursorDown moves the cursor one post down (toward newer messages) and clamps
// to len(posts)-1.
func (m *Model) CursorDown() tea.Cmd {
	if len(m.posts) == 0 {
		return nil
	}
	if m.cursor < len(m.posts)-1 {
		m.cursor++
	}
	return nil
}

// CursorToTop sets the cursor to the first post and scrolls the viewport to the top.
func (m *Model) CursorToTop() {
	if len(m.posts) == 0 {
		return
	}
	m.cursor = 0
	m.viewport.GotoTop()
}

// CursorToBottom sets the cursor to the last post and scrolls the viewport to the bottom.
func (m *Model) CursorToBottom() {
	if len(m.posts) == 0 {
		return
	}
	m.cursor = len(m.posts) - 1
	m.viewport.GotoBottom()
}

// CursorToUnread sets the cursor to the first post whose CreateAt is strictly
// greater than m.lastViewedAt, then scrolls into view.
// Falls back to CursorToBottom when lastViewedAt == 0 or all posts are read.
func (m *Model) CursorToUnread() {
	if m.lastViewedAt == 0 || len(m.posts) == 0 {
		m.CursorToBottom()
		return
	}
	for i, p := range m.posts {
		if p.CreateAt > m.lastViewedAt {
			m.cursor = i
			return
		}
	}
	// All posts are read — go to bottom.
	m.CursorToBottom()
}

// CursorPost returns the post at the current cursor position.
// Returns (zero value, false) when cursor < 0 or out of range.
func (m Model) CursorPost() (api.Post, bool) {
	if m.cursor < 0 || m.cursor >= len(m.posts) {
		return api.Post{}, false
	}
	return m.posts[m.cursor], true
}

// CursorIndex returns the current cursor index into m.posts.
// Returns -1 when there are no posts.
func (m Model) CursorIndex() int {
	return m.cursor
}

// --- Attachment picker API (Task #11) ---

// ActivatePicker opens the multi-attachment picker with the given files.
func (m *Model) ActivatePicker(files []api.FileInfo) {
	m.pickerActive = true
	m.pickerFiles = files
	m.pickerCursor = 0
}

// ClosePicker dismisses the attachment picker.
func (m *Model) ClosePicker() {
	m.pickerActive = false
	m.pickerFiles = nil
	m.pickerCursor = 0
}

// IsPickerActive reports whether the attachment picker is currently shown.
func (m Model) IsPickerActive() bool {
	return m.pickerActive
}

// PickerCursorUp moves the picker cursor one row up; clamps to 0.
func (m *Model) PickerCursorUp() {
	if m.pickerCursor > 0 {
		m.pickerCursor--
	}
}

// PickerCursorDown moves the picker cursor one row down; clamps to len(pickerFiles)-1.
func (m *Model) PickerCursorDown() {
	if len(m.pickerFiles) > 0 && m.pickerCursor < len(m.pickerFiles)-1 {
		m.pickerCursor++
	}
}

// PickerSelected returns the FileInfo at the current picker cursor.
// Returns (zero value, false) when the picker is not active or has no files.
func (m Model) PickerSelected() (api.FileInfo, bool) {
	if !m.pickerActive || len(m.pickerFiles) == 0 {
		return api.FileInfo{}, false
	}
	return m.pickerFiles[m.pickerCursor], true
}

// PickerFiles returns the slice of files currently shown in the picker.
func (m Model) PickerFiles() []api.FileInfo {
	return m.pickerFiles
}

// PickerCursorIndex returns the current picker cursor position.
func (m Model) PickerCursorIndex() int {
	return m.pickerCursor
}

// --- Image popup API (Task #17) ---

// ActivateImagePopup shows the image popup overlay.
func (m *Model) ActivateImagePopup(image, title string) {
	m.popupActive = true
	m.popupImage = image
	m.popupTitle = title
}

// CloseImagePopup dismisses the image popup.
func (m *Model) CloseImagePopup() {
	m.popupActive = false
	m.popupImage = ""
	m.popupTitle = ""
}

// IsPopupActive reports whether the image popup is currently shown.
func (m Model) IsPopupActive() bool {
	return m.popupActive
}

// --- Visual mode API (Features 3 & 4) ---

// EnterVisualMode activates visual selection anchored at the current cursor.
func (m *Model) EnterVisualMode() {
	m.visualMode = true
	m.visualAnchor = m.cursor
}

// ExitVisualMode deactivates visual selection.
func (m *Model) ExitVisualMode() {
	m.visualMode = false
}

// IsVisualMode reports whether visual selection is active.
func (m Model) IsVisualMode() bool {
	return m.visualMode
}

// VisualSelection returns the inclusive [start, end] post-index range of the
// current visual selection. Returns (-1, -1) when not in visual mode.
func (m Model) VisualSelection() (start, end int) {
	if !m.visualMode {
		return -1, -1
	}
	a, c := m.visualAnchor, m.cursor
	if a <= c {
		return a, c
	}
	return c, a
}

// SelectedPostsText returns the plain-text of selected posts joined by "\n---\n",
// each prefixed with "Author  Timestamp\n". Returns "" when not in visual mode.
func (m Model) SelectedPostsText() string {
	if !m.visualMode {
		return ""
	}
	start, end := m.VisualSelection()
	if start < 0 || start >= len(m.posts) {
		return ""
	}
	if end >= len(m.posts) {
		end = len(m.posts) - 1
	}
	var parts []string
	for i := start; i <= end; i++ {
		post := m.posts[i]
		var author string
		if post.UserID == m.userID {
			author = "You"
		} else {
			author = resolveUsername(post.UserID, m.userCache, m.useContactName)
		}
		ts := FormatTimestamp(post.CreateAt)
		text := stripANSI(post.Message)
		parts = append(parts, author+"  "+ts+"\n"+text)
	}
	return strings.Join(parts, "\n---\n")
}

// --- Attachment open / cursor clamping helpers (Task #13) ---

// clampCursorToViewport adjusts m.cursor so it stays within the approximate
// visible range after a viewport-scroll operation.  The heuristic uses line
// counts and viewport height; per-line-per-post precision comes in a later
// task.  The cursor is clamped to [0, len(posts)-1] as a safe lower bound.
func (m *Model) clampCursorToViewport() {
	if len(m.posts) == 0 || m.cursor < 0 {
		return
	}
	// Approximate: assume each post renders to roughly the same number of lines.
	// Use total viewport lines and post count to estimate first/last visible index.
	totalLines := m.viewport.TotalLineCount()
	if totalLines == 0 {
		return
	}
	n := len(m.posts)
	// Fraction of content visible: offset / total tells us first visible post.
	firstVisiblePost := int(float64(m.viewport.YOffset) / float64(totalLines) * float64(n))
	lastVisiblePost := int(float64(m.viewport.YOffset+m.viewport.Height) / float64(totalLines) * float64(n))
	// Clamp estimates to valid range.
	if firstVisiblePost < 0 {
		firstVisiblePost = 0
	}
	if lastVisiblePost >= n {
		lastVisiblePost = n - 1
	}
	if m.cursor < firstVisiblePost {
		m.cursor = firstVisiblePost
	} else if m.cursor > lastVisiblePost {
		m.cursor = lastVisiblePost
	}
}

// OpenAttachmentForCursor handles the open-attachment action for the cursor message.
// If the message has one file whose info is cached, it emits OpenFileMsg.
// If it has multiple, it activates the inline picker.
// No-op if the cursor post has no files or file info is not yet cached.
func (m *Model) OpenAttachmentForCursor() tea.Cmd {
	post, ok := m.CursorPost()
	if !ok || len(post.FileIds) == 0 {
		return nil
	}
	// Gather FileInfo for all attached files from the cache.
	var infos []api.FileInfo
	for _, fid := range post.FileIds {
		fi, ok := m.fileInfoCache[fid]
		if !ok {
			// Info not yet loaded; graceful no-op until Task #12 populates the cache.
			return nil
		}
		infos = append(infos, fi)
	}
	if len(infos) == 1 {
		fi := infos[0]
		// If it's an image with a cached render, show the popup instead of opening externally.
		if rendered, ok := m.imageCache[fi.ID]; ok && rendered != "" {
			m.ActivateImagePopup(rendered, stripANSI(fi.Name))
			return nil
		}
		return func() tea.Msg { return messages.OpenFileMsg{File: fi} }
	}
	m.ActivatePicker(infos)
	return nil
}

// refreshContent re-renders all posts into the viewport with the current cursor,
// last-viewed-at state, and visual selection range.
// Call after any operation that changes m.cursor or visual mode state.
func (m *Model) refreshContent() {
	if len(m.posts) == 0 {
		return
	}
	vs, ve := m.VisualSelection()
	content := RenderPosts(m.posts, m.userCache, m.userID, m.width, m.imageCache, m.fileInfoCache, m.useContactName, m.cursor, m.lastViewedAt, vs, ve)
	m.viewport.SetContent(content)
}

// scrollViewportToCursor adjusts the viewport YOffset so the cursor post is
// visible.  Uses the same line-count heuristic as clampCursorToViewport.
func (m *Model) scrollViewportToCursor() {
	n := len(m.posts)
	if n == 0 || m.cursor < 0 {
		return
	}
	totalLines := m.viewport.TotalLineCount()
	if totalLines == 0 {
		return
	}
	// Estimate the start line of the cursor post.
	cursorLine := int(float64(m.cursor) / float64(n) * float64(totalLines))
	// Scroll up if cursor is above the visible window.
	if cursorLine < m.viewport.YOffset {
		m.viewport.YOffset = cursorLine
	}
	// Scroll down if cursor is below the visible window.
	if cursorLine >= m.viewport.YOffset+m.viewport.Height {
		offset := cursorLine - m.viewport.Height + 1
		if offset < 0 {
			offset = 0
		}
		m.viewport.YOffset = offset
	}
}
