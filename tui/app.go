package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
	"github.com/thucdx/netchat-tui/tui/chat"
	"github.com/thucdx/netchat-tui/tui/input"
	"github.com/thucdx/netchat-tui/tui/sidebar"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// FocusPane identifies which panel currently has keyboard focus.
type FocusPane int

const (
	FocusSidebar FocusPane = iota
	FocusChat
	FocusInput
)

// app-internal message types (not exported; only used within this file)

type channelsLoadedMsg struct {
	items     []sidebar.ChannelItem
	userCache map[string]api.User
	teams     []api.Team
}

// directChannelReadyMsg carries a newly created or fetched DM channel.
type directChannelReadyMsg struct {
	channel  api.Channel
	userID   string // the other user's ID, for display-name resolution
	username string
}

// channelJoinedMsg carries the channel the user just joined.
type channelJoinedMsg struct{ channel api.Channel }

type postsReadyMsg struct {
	channelID   string
	channelName string
	posts       api.PostList
	userCache   map[string]api.User // authors fetched alongside posts
	fileIDs     []string            // file IDs from all posts (for async image loading)
}

type postSentMsg struct{ post api.Post }

type wsConnectedMsg struct {
	ws     *api.WSClient
	cancel context.CancelFunc
}

// morePostsReadyMsg carries older posts fetched for pagination.
type morePostsReadyMsg struct {
	channelID string
	posts     api.PostList
	page      int
}

// dmOtherUserID extracts the other user's ID from a DM channel name.
// DM channel names in Mattermost use the format "userid1__userid2".
func dmOtherUserID(channelName, myUserID string) string {
	parts := strings.SplitN(channelName, "__", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == myUserID {
		return parts[1]
	}
	return parts[0]
}

// AppModel is the root Bubbletea model.
type AppModel struct {
	layout       Layout
	focus        FocusPane
	keys         keymap.KeyMap
	api          *api.Client        // nil when running without a real backend
	ws           *api.WSClient      // nil until WS connects
	wsCancel     context.CancelFunc // cancels WS reconnect loop
	userID       string
	userCache    map[string]api.User // all known users (DM partners + post authors)
	imageCache map[string]string // inline half-block renders keyed by file ID
	ready        bool
	teams        []api.Team  // all teams the user belongs to (for channel search)
	sidebarWidth int  // total sidebar width including border; resizable via drag
	dragging     bool // true while the user is dragging the sidebar border
	titleUpdater func(string) // if non-nil, called instead of tea.SetWindowTitle

	sidebar sidebar.Model
	chat    chat.Model
	input   input.Model
}

// NewAppModel creates the root model. apiClient may be nil (e.g., tests).
func NewAppModel(apiClient *api.Client) AppModel {
	userID := ""
	if apiClient != nil {
		userID = apiClient.UserID()
	}
	keys := keymap.DefaultKeyMap()
	return AppModel{
		focus:        FocusSidebar,
		keys:         keys,
		api:          apiClient,
		userID:       userID,
		sidebarWidth: styles.SidebarWidth,
		sidebar:      sidebar.NewModel(keys, userID),
		chat:         chat.NewModel(keys, userID),
		input:        input.NewModel(keys),
	}
}

// WithSidebarLimit applies a custom channel list limit to the model.
// Call this after NewAppModel before starting the Bubbletea program.
func (m AppModel) WithSidebarLimit(n int) AppModel {
	m.sidebar.SetLimit(n)
	return m
}

// WithTitleUpdater sets a custom function that is called with the desired window
// title string whenever the unread state changes. When set, it replaces the
// default tea.SetWindowTitle behavior. Use this to drive tmux window renaming
// directly (e.g., exec.Command("tmux", "rename-window", title)).
func (m AppModel) WithTitleUpdater(fn func(string)) AppModel {
	m.titleUpdater = fn
	return m
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.chat.Init(),  // starts spinner tick
		m.input.Init(), // starts textarea blink cursor
	}
	if m.api != nil {
		cmds = append(cmds, m.cmdLoadAllChannels(), m.cmdStartWS())
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Layout ────────────────────────────────────────────────────────────────

	case tea.WindowSizeMsg:
		m.layout = NewLayout(msg.Width, msg.Height, m.sidebarWidth)
		m.ready = true
		m.sidebar.SetHeight(m.layout.TotalHeight)
		m.sidebar.SetWidth(m.sidebarWidth)
		m.chat.SetSize(m.layout.ChatWidth, m.layout.ChatHeight)
		m.input.SetSize(m.layout.ChatWidth, m.layout.InputHeight)
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg), nil

	// ── Global key handling ───────────────────────────────────────────────────

	case tea.KeyMsg:
		// Ctrl+C always quits.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Skip global hotkeys while the sidebar search input is active —
		// every key should reach the sidebar's own handler instead.
		sidebarSearching := m.focus == FocusSidebar && m.sidebar.IsSearching()

		if !sidebarSearching {
			// q quits only when sidebar is focused (not while typing).
			if m.focus == FocusSidebar && key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}

			// Focus cycling.
			switch {
			case key.Matches(msg, m.keys.NextPanel):
				// Tab only cycles between sidebar and chat; when in input the
				// key is passed through so the textarea can use it for indentation.
				if m.focus != FocusInput {
					switch m.focus {
					case FocusSidebar:
						m.focus = FocusChat
					case FocusChat:
						m.focus = FocusSidebar
					}
					m.syncFocus()
					return m, nil
				}

			case key.Matches(msg, m.keys.FocusInput):
				if m.focus == FocusSidebar || m.focus == FocusChat {
					m.focus = FocusInput
					m.syncFocus()
					return m, nil
				}

			// ] → chat; only active outside input so ] can be typed in messages.
			case key.Matches(msg, m.keys.FocusChat):
				if m.focus == FocusSidebar {
					m.focus = FocusChat
					m.syncFocus()
					return m, nil
				}

			// [ → sidebar; only active outside input so [ can be typed in messages.
			case key.Matches(msg, m.keys.FocusSidebar):
				if m.focus == FocusChat {
					m.focus = FocusSidebar
					m.syncFocus()
					return m, nil
				}

			// Esc from input returns to chat (not sidebar) so the flow
			// i → type → Esc lands back in the chat dialogue.
			case msg.Type == tea.KeyEsc && m.focus == FocusInput:
				m.focus = FocusChat
				m.syncFocus()
				return m, nil

			case key.Matches(msg, m.keys.ToggleName):
				if m.focus == FocusSidebar {
					m.sidebar.ToggleDisplayName()
					m.chat.SetUseContactName(m.sidebar.UseContactName())
					// Update chat header if a DM/group channel is currently open.
					if ch := m.sidebar.SelectedChannel(); ch != nil && ch.AccountName != "" {
						m.chat.SetChannelInfo(ch.Channel.ID, ch.DisplayName)
					}
					return m, nil
				}
			}
		}

		// Route key to the focused pane.
		return m.routeKey(msg)

	// ── Data loading ──────────────────────────────────────────────────────────

	case channelsLoadedMsg:
		m.sidebar.SetItems(msg.items)
		m.teams = msg.teams
		// Seed the app-wide user cache with DM partners / group members.
		if m.userCache == nil {
			m.userCache = make(map[string]api.User)
		}
		for id, u := range msg.userCache {
			m.userCache[id] = u
		}
		return m, m.titleCmd()

	// ── Search ─────────────────────────────────────────────────────────────

	case messages.TriggerSearchMsg:
		if m.api != nil && len([]rune(msg.Query)) >= 3 {
			return m, m.cmdSearch(msg.Query)
		}
		return m, nil

	case messages.SearchResultsMsg:
		m.sidebar.SetSearchAPIResults(msg.Query, msg.Users, msg.Channels)
		return m, nil

	case messages.CreateDirectChannelMsg:
		if m.api != nil {
			return m, m.cmdCreateDirectChannel(msg.UserID)
		}
		return m, nil

	case messages.JoinChannelMsg:
		if m.api != nil {
			return m, m.cmdJoinChannel(msg.ChannelID)
		}
		return m, nil

	case directChannelReadyMsg:
		item := sidebar.ChannelItem{
			Channel:     msg.channel,
			DisplayName: msg.username,
		}
		m.sidebar.UpsertItem(item)
		m.sidebar.ExitSearch()
		m.input.SetChannelID(msg.channel.ID)
		m.chat.SetChannelInfo(msg.channel.ID, msg.username)
		m.focus = FocusChat
		if m.api != nil {
			return m, m.cmdFetchPosts(msg.channel.ID, msg.username)
		}
		return m, nil

	case channelJoinedMsg:
		displayName := msg.channel.DisplayName
		item := sidebar.ChannelItem{
			Channel:     msg.channel,
			DisplayName: displayName,
		}
		m.sidebar.UpsertItem(item)
		m.sidebar.ExitSearch()
		m.input.SetChannelID(msg.channel.ID)
		m.chat.SetChannelInfo(msg.channel.ID, displayName)
		m.focus = FocusChat
		if m.api != nil {
			return m, m.cmdFetchPosts(msg.channel.ID, displayName)
		}
		return m, nil

	// ── Channel selection ─────────────────────────────────────────────────────

	case messages.ChannelSelectedMsg:
		// Update input so sends target the new channel.
		m.input.SetChannelID(msg.ChannelID)
		// Find channel name from sidebar for the chat header.
		channelName := msg.ChannelID
		if ch := m.sidebar.SelectedChannel(); ch != nil {
			channelName = ch.DisplayName
		}
		// Set channel on chat model immediately (before posts arrive).
		m.chat.SetChannelInfo(msg.ChannelID, channelName)
		// Pass the last-viewed timestamp so the unread marker and CursorToUnread work.
		if ch := m.sidebar.SelectedChannel(); ch != nil {
			m.chat.SetLastViewedAt(ch.Member.LastViewedAt)
		}
		// Clear sidebar unread badge.
		m.sidebar.ClearUnread(msg.ChannelID)
		// Shift focus to chat pane (Enter); p keeps focus in sidebar.
		if msg.FocusOnOpen {
			m.focus = FocusChat
		}
		if m.api != nil {
			return m, tea.Batch(m.cmdFetchPosts(msg.ChannelID, channelName), m.titleCmd())
		}
		return m, m.titleCmd()

	case postsReadyMsg:
		// Merge newly fetched post authors into the app-wide user cache.
		if m.userCache == nil {
			m.userCache = make(map[string]api.User)
		}
		for id, u := range msg.userCache {
			m.userCache[id] = u
		}
		m.chat.LoadPosts(msg.channelID, msg.channelName, msg.posts, m.userCache)
		// Mark channel as read (fire and forget via tea.Cmd — returns nil on completion).
		var markRead tea.Cmd
		if m.api != nil {
			client := m.api
			channelID := msg.channelID
			markRead = func() tea.Msg {
				_ = client.MarkChannelRead(channelID)
				return nil
			}
		}
		var imgCmd tea.Cmd
		if len(msg.fileIDs) > 0 {
			imgCmd = m.cmdFetchImages(msg.fileIDs)
		}
		return m, tea.Batch(markRead, imgCmd)

	case messages.ImagesReadyMsg:
		if m.imageCache == nil {
			m.imageCache = make(map[string]string)
		}
		for k, v := range msg.Images {
			m.imageCache[k] = v
		}
		m.chat = m.chat.SetImageCache(m.imageCache)
		if len(msg.FileInfos) > 0 {
			m.chat.SetFileInfoCache(msg.FileInfos)
		}
		return m, nil

	// ── Sending messages ──────────────────────────────────────────────────────

	case messages.SendMessageMsg:
		if m.api == nil {
			m.input.SetSending(false)
			return m, nil
		}
		return m, m.cmdSendPost(msg.ChannelID, msg.Text)

	case postSentMsg:
		m.chat.AppendPost(msg.post)
		m.input.SetSending(false)
		return m, nil

	case ErrorMsg:
		// Release the send lock so the user can retry.
		m.input.SetSending(false)
		// Show error in the chat pane banner.
		m.chat.SetError(msg.Err)
		return m, nil

	// ── File open ─────────────────────────────────────────────────────────────

	case messages.OpenFileMsg:
		file := msg.File
		apiClient := m.api
		return m, func() tea.Msg {
			// Sanitise API-supplied fields to prevent path traversal.
			// filepath.Base strips any directory component (e.g. "../../etc/passwd").
			safeID := filepath.Base(file.ID)
			safeExt := filepath.Base(file.Extension)
			if safeID == "" || safeID == "." || safeExt == "" || safeExt == "." {
				return ErrorMsg{Err: fmt.Errorf("open file: invalid file metadata")}
			}

			// No API client = nothing to download; bail early before touching disk.
			if apiClient == nil {
				return ErrorMsg{Err: fmt.Errorf("open file: no API client available")}
			}

			// Build a stable temp path keyed by file ID so repeated opens hit cache.
			dir := filepath.Join(os.TempDir(), "netchat-tui")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return ErrorMsg{Err: fmt.Errorf("open file: create temp dir: %w", err)}
			}
			path := filepath.Join(dir, safeID+"."+safeExt)

			// Download only if not already cached on disk.
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				data, err := apiClient.DownloadFile(file.ID)
				if err != nil {
					return ErrorMsg{Err: fmt.Errorf("open file: download: %w", err)}
				}
				if err := os.WriteFile(path, data, 0o600); err != nil {
					return ErrorMsg{Err: fmt.Errorf("open file: write: %w", err)}
				}
			}

			// Open with the OS default application (non-blocking).
			// go cmd.Wait() reaps the child process to avoid zombies.
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", path)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", "", path)
			default:
				cmd = exec.Command("xdg-open", path)
			}
			if err := cmd.Start(); err != nil {
				return ErrorMsg{Err: fmt.Errorf("open file: launch viewer: %w", err)}
			}
			go cmd.Wait() //nolint:errcheck — we only care about launching, not exit code
			return nil
		}

	// ── Clipboard yank ────────────────────────────────────────────────────────

	case messages.YankMsg:
		if err := copyToClipboard(msg.Text); err != nil {
			log.Printf("yank: clipboard write failed: %v", err)
		}
		return m, nil

	case messages.LoadMorePostsMsg:
		if m.api != nil {
			return m, m.cmdFetchMorePosts(msg.ChannelID, msg.Page)
		}
		return m, nil

	case morePostsReadyMsg:
		if msg.channelID == m.chat.ChannelID() {
			m.chat.PrependPosts(msg.posts, msg.page)
		}
		return m, nil

	// ── WebSocket real-time ───────────────────────────────────────────────────

	case messages.NewPostMsg:
		if msg.Post.ChannelID == m.chat.ChannelID() {
			m.chat.AppendPost(msg.Post)
		} else {
			m.sidebar.IncrementUnread(msg.Post.ChannelID)
			return m, m.titleCmd()
		}
		return m, nil

	// ── WebSocket connection ────────────────────────────────────────────────

	case wsConnectedMsg:
		m.ws = msg.ws
		m.wsCancel = msg.cancel
		return m, m.waitForWSEvent()

	// ── WebSocket event dispatch ─────────────────────────────────────────────

	case messages.WSEventMsg:
		cmd := m.handleWSEvent(msg.Event)
		// Keep listening for next event.
		return m, tea.Batch(cmd, m.waitForWSEvent())

	case messages.WSDisconnectedMsg:
		// Connection dropped — reconnect with retry in the background.
		ws := m.ws
		return m, func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := api.ConnectWithRetry(ctx, ws); err != nil {
				return ErrorMsg{Err: fmt.Errorf("WebSocket reconnect failed: %w", err)}
			}
			return messages.WSReconnectedMsg{}
		}

	case messages.WSReconnectedMsg:
		// Reconnected — resume listening for events.
		return m, m.waitForWSEvent()
	}

	// Propagate spinner ticks and other internal messages to sub-models.
	return m.tickSubModels(msg)
}

// syncFocus propagates the current focus state to the input sub-model.
func (m *AppModel) syncFocus() {
	m.input.SetFocused(m.focus == FocusInput)
}

// routeKey sends the key to the focused pane.
func (m AppModel) routeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusSidebar:
		result, cmd := m.sidebar.Update(msg)
		if sb, ok := result.(sidebar.Model); ok {
			m.sidebar = sb
		}
		return m, cmd

	case FocusChat:
		result, cmd := m.chat.Update(msg)
		if cm, ok := result.(chat.Model); ok {
			m.chat = cm
		}
		return m, cmd

	case FocusInput:
		result, cmd := m.input.Update(msg)
		if im, ok := result.(input.Model); ok {
			m.input = im
		}
		return m, cmd
	}
	return m, nil
}

// tickSubModels propagates non-key messages (spinner ticks, etc.) to all sub-models.
func (m AppModel) tickSubModels(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	chatResult, cmd := m.chat.Update(msg)
	if cm, ok := chatResult.(chat.Model); ok {
		m.chat = cm
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	inputResult, cmd := m.input.Update(msg)
	if im, ok := inputResult.(input.Model); ok {
		m.input = im
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m AppModel) View() string {
	if !m.ready {
		return ""
	}
	if !m.layout.IsValid() {
		return "Terminal too small — please resize to at least 60×10."
	}

	sidebarStyle := styles.SidebarStyle
	if m.focus == FocusSidebar {
		sidebarStyle = styles.SidebarFocusedStyle
	}
	contentWidth := m.sidebarWidth - styles.SidebarBorderWidth
	sidebarView := sidebarStyle.
		Width(contentWidth).
		Height(m.layout.TotalHeight).
		Render(m.sidebar.View())

	chatView := styles.ChatStyle.
		Width(m.layout.ChatWidth).
		Height(m.layout.ChatHeight).
		Render(m.chat.View())

	inputView := m.input.View()

	chatAndInput := lipgloss.JoinVertical(lipgloss.Left, chatView, inputView)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, chatAndInput)
}

// titleCmd returns a command that updates the window/tab title to reflect the
// current unread state. Format: "netchat-tui [msgs/channels]" when there are
// unmuted unreads, "netchat-tui" otherwise.
// When a custom titleUpdater is set (e.g., for tmux), it is called directly;
// otherwise tea.SetWindowTitle (OSC 2) is used.
func (m AppModel) titleCmd() tea.Cmd {
	msgs, chs := m.sidebar.UnreadSummary()
	var title string
	if chs == 0 {
		title = "netchat-tui"
	} else {
		title = fmt.Sprintf("netchat-tui [%d/%d]", msgs, chs)
	}
	if m.titleUpdater != nil {
		fn, t := m.titleUpdater, title
		return func() tea.Msg {
			fn(t)
			return nil
		}
	}
	return tea.SetWindowTitle(title)
}

// ── API commands ──────────────────────────────────────────────────────────────

// cmdLoadAllChannels fetches channels across ALL teams the user belongs to,
// deduplicates by channel ID, resolves DM display names, and returns the
// full list as a channelsLoadedMsg.
//
// Previously the app fetched only from teams[0], which meant channels in
// other teams (and sometimes all public/private channels) were never shown.
func (m AppModel) cmdLoadAllChannels() tea.Cmd {
	userID := m.userID
	apiClient := m.api
	return func() tea.Msg {
		teams, err := apiClient.GetTeamsForUser(userID)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Aggregate channels and members across all teams, dedup by channel ID.
		seenCh := make(map[string]struct{})
		seenMb := make(map[string]struct{})
		var allChannels []api.Channel
		memberByChannelID := make(map[string]api.ChannelMember)

		for _, team := range teams {
			channels, err := apiClient.GetChannelsForUser(userID, team.ID)
			if err != nil {
				continue // skip broken team, don't abort
			}
			for _, ch := range channels {
				if ch.IsDeleted() {
					continue
				}
				if _, seen := seenCh[ch.ID]; !seen {
					seenCh[ch.ID] = struct{}{}
					allChannels = append(allChannels, ch)
				}
			}

			members, err := apiClient.GetChannelMembersForUser(userID, team.ID)
			if err != nil {
				continue
			}
			for _, mb := range members {
				if _, seen := seenMb[mb.ChannelID]; !seen {
					seenMb[mb.ChannelID] = struct{}{}
					memberByChannelID[mb.ChannelID] = mb
				}
			}
		}

		// Collect user IDs from DM and Group channels for batch display-name resolution.
		// G channels encode all participant IDs in the channel name as uid1__uid2__uid3.
		userIDsToFetch := make(map[string]struct{})
		for _, ch := range allChannels {
			switch ch.Type {
			case "D":
				if otherID := dmOtherUserID(ch.Name, userID); otherID != "" {
					userIDsToFetch[otherID] = struct{}{}
				}
			case "G":
				for _, uid := range strings.Split(ch.Name, "__") {
					if uid != "" && uid != userID {
						userIDsToFetch[uid] = struct{}{}
					}
				}
			}
		}

		userCache := make(map[string]api.User)
		if len(userIDsToFetch) > 0 {
			ids := make([]string, 0, len(userIDsToFetch))
			for id := range userIDsToFetch {
				ids = append(ids, id)
			}
			if users, err := apiClient.GetUsersByIDs(ids); err == nil {
				for _, u := range users {
					userCache[u.ID] = u
				}
			}
			// On error, fall back to channel DisplayName (no crash).
		}

		items := make([]sidebar.ChannelItem, 0, len(allChannels))
		for _, ch := range allChannels {
			mb := memberByChannelID[ch.ID]
			displayName := ch.DisplayName
			var accountName, contactName string
			switch ch.Type {
			case "D":
				if otherID := dmOtherUserID(ch.Name, userID); otherID != "" {
					if u, ok := userCache[otherID]; ok {
						accountName = u.Username
						full := strings.TrimSpace(u.FirstName + " " + u.LastName)
						contactName = full
						if accountName == "" {
							accountName = otherID
						}
						displayName = accountName
					} else if displayName == "" {
						// User not in cache (e.g. deactivated account) — fall back to
						// their user ID so the sidebar row is never blank.
						displayName = otherID
						accountName = otherID
					}
				}
			case "G":
				// Build "user1, user2, ..." from participants excluding self.
				var accountNames, contactNames []string
				for _, uid := range strings.Split(ch.Name, "__") {
					if uid == "" || uid == userID {
						continue
					}
					if u, ok := userCache[uid]; ok && u.Username != "" {
						accountNames = append(accountNames, u.Username)
						full := strings.TrimSpace(u.FirstName + " " + u.LastName)
						if full != "" {
							contactNames = append(contactNames, full)
						} else {
							contactNames = append(contactNames, u.Username)
						}
					}
				}
				accountName = strings.Join(accountNames, ", ")
				contactName = strings.Join(contactNames, ", ")
				if accountName != "" {
					displayName = accountName
				}
			}
			item := sidebar.ChannelItem{
				Channel:     ch,
				Member:      mb,
				DisplayName: displayName,
				ContactName: contactName,
				AccountName: accountName,
				UnreadCount: mb.UnreadCount(ch),
				IsMuted:     mb.IsMuted(),
			}
			items = append(items, item)
		}
		return channelsLoadedMsg{items: items, userCache: userCache, teams: teams}
	}
}

func (m AppModel) cmdFetchPosts(channelID, channelName string) tea.Cmd {
	apiClient := m.api
	known := m.userCache // snapshot of already-known users (read-only in goroutine)
	return func() tea.Msg {
		posts, err := apiClient.GetPostsForChannel(channelID, 0, 60)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Collect unique author IDs not yet in the cache.
		missing := make(map[string]struct{})
		for _, p := range posts.Posts {
			if p.UserID != "" {
				if _, ok := known[p.UserID]; !ok {
					missing[p.UserID] = struct{}{}
				}
			}
		}

		userCache := make(map[string]api.User)
		if len(missing) > 0 {
			ids := make([]string, 0, len(missing))
			for id := range missing {
				ids = append(ids, id)
			}
			if users, err := apiClient.GetUsersByIDs(ids); err == nil {
				for _, u := range users {
					userCache[u.ID] = u
				}
			}
		}

		// Collect file IDs to be fetched asynchronously.
		var fileIDs []string
		for _, p := range posts.Posts {
			fileIDs = append(fileIDs, p.FileIds...)
		}

		return postsReadyMsg{
			channelID:   channelID,
			channelName: channelName,
			posts:       posts,
			userCache:   userCache,
			fileIDs:     fileIDs,
		}
	}
}

func (m AppModel) cmdFetchMorePosts(channelID string, page int) tea.Cmd {
	apiClient := m.api
	return func() tea.Msg {
		posts, err := apiClient.GetPostsForChannel(channelID, page, 60)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return morePostsReadyMsg{
			channelID: channelID,
			posts:     posts,
			page:      page,
		}
	}
}

// cmdFetchImages downloads thumbnails for the given file IDs and renders small
// inline half-block previews.
func (m AppModel) cmdFetchImages(fileIDs []string) tea.Cmd {
	if len(fileIDs) == 0 {
		return nil
	}
	apiClient := m.api
	return func() tea.Msg {
		rendered := make(map[string]string, len(fileIDs))
		fileInfos := make(map[string]api.FileInfo, len(fileIDs))
		for _, fid := range fileIDs {
			info, err := apiClient.GetFileInfo(fid)
			if err != nil {
				log.Printf("cmdFetchImages: GetFileInfo(%s): %v", fid, err)
				continue
			}
			fileInfos[fid] = info
			if !isImageMIME(info.MimeType) {
				continue
			}
			data, err := apiClient.DownloadFileThumbnail(fid)
			if err != nil {
				continue
			}
			// Inline thumbnail: small half-block render.
			r := chat.RenderImageHalfBlock(data, chat.InlineImageCols, chat.InlineImageRows)
			if r != "" {
				rendered[fid] = r
			}
		}
		return messages.ImagesReadyMsg{Images: rendered, FileInfos: fileInfos}
	}
}

func isImageMIME(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}

func (m AppModel) cmdSendPost(channelID, text string) tea.Cmd {
	apiClient := m.api
	return func() tea.Msg {
		post, err := apiClient.CreatePost(channelID, text)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return postSentMsg{post: post}
	}
}

// ── WebSocket commands ────────────────────────────────────────────────────────

// cmdStartWS creates a WSClient from the HTTP client and connects with retry.
// On success it returns a wsConnectedMsg so the live model stores the client.
func (m AppModel) cmdStartWS() tea.Cmd {
	apiClient := m.api
	return func() tea.Msg {
		ws, err := api.NewWSClientFromClient(apiClient)
		if err != nil {
			log.Printf("ws: failed to create client: %v", err)
			return ErrorMsg{Err: fmt.Errorf("WebSocket unavailable: %w", err)}
		}

		ctx, cancel := context.WithCancel(context.Background())

		if err := api.ConnectWithRetry(ctx, ws); err != nil {
			log.Printf("ws: connect failed: %v", err)
			cancel()
			return ErrorMsg{Err: fmt.Errorf("WebSocket connection failed: %w", err)}
		}

		return wsConnectedMsg{ws: ws, cancel: cancel}
	}
}

// waitForWSEvent blocks until the next event arrives on the WS channel.
// It also selects on ws.Done() so that a dropped connection is detected
// immediately rather than blocking forever on ws.Events.
func (m AppModel) waitForWSEvent() tea.Cmd {
	ws := m.ws
	if ws == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case event, ok := <-ws.Events:
			if !ok {
				return messages.WSDisconnectedMsg{}
			}
			return messages.WSEventMsg{Event: event}
		case <-ws.Done():
			return messages.WSDisconnectedMsg{}
		}
	}
}

// handleWSEvent processes a single WS event and returns a Cmd (if any).
// Pointer receiver: called from the value-receiver Update via `m.handleWSEvent(...)`.
// Go takes the address of Update's local copy, so mutations are visible on the
// returned copy. This is intentional — the updated AppModel is returned by Update.
func (m *AppModel) handleWSEvent(event api.WSEvent) tea.Cmd {
	switch event.Event {
	case "posted":
		return m.handlePosted(event)
	case "post_edited":
		return m.handlePostEdited(event)
	case "channel_viewed":
		return m.handleChannelViewed(event)
	}
	return nil
}

// handlePosted double-unmarshals the post from event.Data["post"] (JSON string)
// and applies the update directly to the model — no extra goroutine hop.
func (m *AppModel) handlePosted(event api.WSEvent) tea.Cmd {
	post, ok := unmarshalPostFromEvent(event)
	if !ok {
		return nil
	}

	// Skip own posts — they're already shown via postSentMsg.
	if post.UserID == m.userID {
		return nil
	}

	if post.ChannelID == m.chat.ChannelID() {
		m.chat.AppendPost(post)
	} else {
		m.sidebar.IncrementUnread(post.ChannelID)
	}
	return nil
}

// handlePostEdited double-unmarshals the post and updates the chat if active.
func (m *AppModel) handlePostEdited(event api.WSEvent) tea.Cmd {
	post, ok := unmarshalPostFromEvent(event)
	if !ok {
		return nil
	}

	if post.ChannelID == m.chat.ChannelID() {
		m.chat.UpdatePost(post)
	}
	return nil
}

// handleChannelViewed clears unread badge for the viewed channel.
func (m *AppModel) handleChannelViewed(event api.WSEvent) tea.Cmd {
	channelID, _ := event.Data["channel_id"].(string)
	if channelID != "" {
		m.sidebar.ClearUnread(channelID)
		return m.titleCmd()
	}
	return nil
}

// handleMouse processes mouse events for sidebar drag-to-resize.
// The sidebar border sits at column sidebarWidth-1 (0-indexed). Pressing on
// that column starts a drag; motion while dragging updates sidebarWidth.
func (m AppModel) handleMouse(msg tea.MouseMsg) AppModel {
	const minSidebarWidth = 10
	borderCol := m.sidebarWidth - 1

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft && msg.X == borderCol {
			m.dragging = true
		}
	case tea.MouseActionRelease:
		m.dragging = false
	case tea.MouseActionMotion:
		if m.dragging && msg.Button == tea.MouseButtonLeft {
			newWidth := msg.X + 1
			maxWidth := m.layout.TotalWidth - 20
			if maxWidth < minSidebarWidth {
				maxWidth = minSidebarWidth
			}
			if newWidth < minSidebarWidth {
				newWidth = minSidebarWidth
			}
			if newWidth > maxWidth {
				newWidth = maxWidth
			}
			m.sidebarWidth = newWidth
			m.layout = NewLayout(m.layout.TotalWidth, m.layout.TotalHeight, newWidth)
			m.sidebar.SetWidth(newWidth)
			m.chat.SetSize(m.layout.ChatWidth, m.layout.ChatHeight)
			m.input.SetSize(m.layout.ChatWidth, m.layout.InputHeight)
		}
	}
	return m
}

// cmdSearch fires parallel user + channel searches across all known teams.
func (m AppModel) cmdSearch(query string) tea.Cmd {
	apiClient := m.api
	teams := m.teams
	return func() tea.Msg {
		var users []api.User
		if u, err := apiClient.SearchUsers(query); err == nil {
			users = u
		}

		// Search channels across all teams, deduplicate by channel ID.
		seen := make(map[string]struct{})
		var channels []api.Channel
		for _, t := range teams {
			chs, err := apiClient.SearchChannels(query, t.ID)
			if err != nil {
				continue
			}
			for _, ch := range chs {
				if _, ok := seen[ch.ID]; ok {
					continue
				}
				seen[ch.ID] = struct{}{}
				channels = append(channels, ch)
			}
		}

		return messages.SearchResultsMsg{Query: query, Users: users, Channels: channels}
	}
}

// cmdCreateDirectChannel creates or opens a DM with the given user.
func (m AppModel) cmdCreateDirectChannel(otherUserID string) tea.Cmd {
	apiClient := m.api
	return func() tea.Msg {
		ch, err := apiClient.CreateDirectChannel(otherUserID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		// Fetch the other user's username for display.
		username := otherUserID
		if users, err := apiClient.GetUsersByIDs([]string{otherUserID}); err == nil && len(users) > 0 {
			username = users[0].Username
		}
		return directChannelReadyMsg{channel: ch, userID: otherUserID, username: username}
	}
}

// cmdJoinChannel joins a public channel and fetches the full channel object.
func (m AppModel) cmdJoinChannel(channelID string) tea.Cmd {
	apiClient := m.api
	return func() tea.Msg {
		if err := apiClient.JoinChannel(channelID); err != nil {
			return ErrorMsg{Err: err}
		}
		ch, err := apiClient.GetChannel(channelID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return channelJoinedMsg{channel: ch}
	}
}

// copyToClipboard writes text to the OS clipboard using the platform's native tool.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default: // linux / bsd
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// unmarshalPostFromEvent extracts and JSON-decodes a Post from event.Data["post"].
// Mattermost sends the post as a JSON string inside the Data map.
func unmarshalPostFromEvent(event api.WSEvent) (api.Post, bool) {
	raw, ok := event.Data["post"]
	if !ok {
		return api.Post{}, false
	}
	postStr, ok := raw.(string)
	if !ok {
		return api.Post{}, false
	}
	var post api.Post
	if err := json.Unmarshal([]byte(postStr), &post); err != nil {
		return api.Post{}, false
	}
	return post, true
}
