package chat

import (
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
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
