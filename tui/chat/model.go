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
	loading     bool
	loadingMore bool                // true while fetching older posts (pagination)
	page        int                 // current pagination page (0 = initial)
	spinner     spinner.Model
	err         error // last API error, displayed as banner
	keys        keymap.KeyMap
	width       int
	height      int // chat area height (excludes input)
}

// NewModel creates a new chat Model with default spinner and loading state.
func NewModel(keys keymap.KeyMap) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.SpinnerStyle

	return Model{
		loading:   true,
		spinner:   sp,
		keys:      keys,
		userCache: make(map[string]api.User),
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

	// Render posts content into the viewport.
	content := RenderPosts(m.posts, m.userCache, m.width)
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// AppendPost adds a new post at the bottom (from WebSocket).
// If the viewport was at the bottom before, auto-scroll after append.
func (m *Model) AppendPost(post api.Post) {
	if post.DeleteAt > 0 {
		return
	}
	atBottom := m.viewport.AtBottom()
	m.posts = append(m.posts, post)

	content := RenderPosts(m.posts, m.userCache, m.width)
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
			content := RenderPosts(m.posts, m.userCache, m.width)
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

	// Measure how many lines the prepended posts add.
	newContent := RenderPosts(newPosts, m.userCache, m.width)
	addedLines := strings.Count(newContent, "\n") + 1

	content := RenderPosts(m.posts, m.userCache, m.width)
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
		content := RenderPosts(m.posts, m.userCache, width)
		m.viewport.SetContent(content)
		if atBottom {
			m.viewport.GotoBottom()
		}
	}
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
		// Esc dismisses the error banner when one is shown.
		if key.Matches(msg, m.keys.FocusSidebar) && m.err != nil {
			m.err = nil
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.ScrollUp):
			m.viewport.HalfViewUp()
		case key.Matches(msg, m.keys.ScrollDown):
			m.viewport.HalfViewDown()
		case key.Matches(msg, m.keys.PageUp):
			m.viewport.ViewUp()
		case key.Matches(msg, m.keys.PageDown):
			m.viewport.ViewDown()
		case key.Matches(msg, m.keys.JumpToBottom):
			m.viewport.GotoBottom()
		case key.Matches(msg, m.keys.Up):
			m.viewport.LineUp(1)
		case key.Matches(msg, m.keys.Down):
			m.viewport.LineDown(1)
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
	} else {
		body = m.viewport.View()
	}

	view := header + "\n" + body

	if m.err != nil {
		errLine := styles.ErrorStyle.Render("Error: " + m.err.Error())
		view += "\n" + errLine
	}

	return view
}
