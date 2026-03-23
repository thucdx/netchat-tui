package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

type teamLoadedMsg struct{ teamID string }

type channelsLoadedMsg struct{ items []sidebar.ChannelItem }

type postsReadyMsg struct {
	channelID   string
	channelName string
	posts       api.PostList
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
	layout  Layout
	focus   FocusPane
	keys    keymap.KeyMap
	api     *api.Client  // nil when running without a real backend
	ws      *api.WSClient // nil until WS connects
	wsCancel context.CancelFunc // cancels WS reconnect loop
	userID  string
	teamID  string
	ready   bool

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
		focus:   FocusSidebar,
		keys:    keys,
		api:     apiClient,
		userID:  userID,
		sidebar: sidebar.NewModel(keys, userID),
		chat:    chat.NewModel(keys),
		input:   input.NewModel(keys),
	}
}

// WithSidebarLimit applies a custom channel list limit to the model.
// Call this after NewAppModel before starting the Bubbletea program.
func (m AppModel) WithSidebarLimit(n int) AppModel {
	m.sidebar.SetLimit(n)
	return m
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.chat.Init(),  // starts spinner tick
		m.input.Init(), // starts textarea blink cursor
	}
	if m.api != nil {
		cmds = append(cmds, m.cmdLoadTeam())
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Layout ────────────────────────────────────────────────────────────────

	case tea.WindowSizeMsg:
		m.layout = NewLayout(msg.Width, msg.Height)
		m.ready = true
		m.sidebar.SetHeight(m.layout.TotalHeight)
		m.chat.SetSize(m.layout.ChatWidth, m.layout.ChatHeight)
		m.input.SetSize(m.layout.ChatWidth, m.layout.InputHeight)
		return m, nil

	// ── Global key handling ───────────────────────────────────────────────────

	case tea.KeyMsg:
		// Ctrl+C always quits.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// q quits only when sidebar is focused (not while typing).
		if m.focus == FocusSidebar && key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		// Focus cycling.
		switch {
		case key.Matches(msg, m.keys.NextPanel):
			switch m.focus {
			case FocusSidebar:
				m.focus = FocusChat
			case FocusChat:
				m.focus = FocusInput
			case FocusInput:
				m.focus = FocusSidebar
			}
			m.syncFocus()
			return m, nil

		case key.Matches(msg, m.keys.FocusInput):
			if m.focus == FocusSidebar || m.focus == FocusChat {
				m.focus = FocusInput
				m.syncFocus()
				return m, nil
			}

		case key.Matches(msg, m.keys.FocusSidebar):
			if m.focus == FocusInput || m.focus == FocusChat {
				m.focus = FocusSidebar
				m.syncFocus()
				return m, nil
			}
		}

		// Route key to the focused pane.
		return m.routeKey(msg)

	// ── Data loading ──────────────────────────────────────────────────────────

	case teamLoadedMsg:
		m.teamID = msg.teamID
		if m.api != nil {
			return m, tea.Batch(m.cmdLoadChannels(), m.cmdStartWS())
		}
		return m, nil

	case channelsLoadedMsg:
		m.sidebar.SetItems(msg.items)
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
		// Clear sidebar unread badge.
		m.sidebar.ClearUnread(msg.ChannelID)
		if m.api != nil {
			return m, m.cmdFetchPosts(msg.ChannelID, channelName)
		}
		return m, nil

	case postsReadyMsg:
		m.chat.LoadPosts(msg.channelID, msg.channelName, msg.posts, make(map[string]api.User))
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
		return m, markRead

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
	sidebarView := sidebarStyle.Height(m.layout.TotalHeight).Render(m.sidebar.View())

	chatView := styles.ChatStyle.
		Width(m.layout.ChatWidth).
		Height(m.layout.ChatHeight).
		Render(m.chat.View())

	inputView := m.input.View()

	chatAndInput := lipgloss.JoinVertical(lipgloss.Left, chatView, inputView)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, chatAndInput)
}

// ── API commands ──────────────────────────────────────────────────────────────

func (m AppModel) cmdLoadTeam() tea.Cmd {
	client := m.api
	userID := m.userID
	return func() tea.Msg {
		team, err := client.GetFirstTeam(userID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return teamLoadedMsg{teamID: team.ID}
	}
}

func (m AppModel) cmdLoadChannels() tea.Cmd {
	userID := m.userID
	teamID := m.teamID
	apiClient := m.api
	return func() tea.Msg {
		channels, err := apiClient.GetChannelsForUser(userID, teamID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		members, err := apiClient.GetChannelMembersForUser(userID, teamID)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Build member lookup map.
		memberByChannelID := make(map[string]api.ChannelMember, len(members))
		for _, mb := range members {
			memberByChannelID[mb.ChannelID] = mb
		}

		// Collect user IDs from DM channels for batch resolution.
		dmUserIDs := make(map[string]struct{})
		for _, ch := range channels {
			if ch.Type == "D" {
				if otherID := dmOtherUserID(ch.Name, userID); otherID != "" {
					dmUserIDs[otherID] = struct{}{}
				}
			}
		}

		// Batch fetch DM users for display name resolution.
		dmUserCache := make(map[string]api.User)
		if len(dmUserIDs) > 0 {
			ids := make([]string, 0, len(dmUserIDs))
			for id := range dmUserIDs {
				ids = append(ids, id)
			}
			users, err := apiClient.GetUsersByIDs(ids)
			if err == nil {
				for _, u := range users {
					dmUserCache[u.ID] = u
				}
			}
			// On error, fall back to channel DisplayName (no crash).
		}

		items := make([]sidebar.ChannelItem, 0, len(channels))
		for _, ch := range channels {
			mb := memberByChannelID[ch.ID]
			displayName := ch.DisplayName
			if ch.Type == "D" {
				if otherID := dmOtherUserID(ch.Name, userID); otherID != "" {
					if u, ok := dmUserCache[otherID]; ok && u.Username != "" {
						displayName = u.Username
					}
				}
			}
			item := sidebar.ChannelItem{
				Channel:     ch,
				Member:      mb,
				DisplayName: displayName,
				UnreadCount: mb.UnreadCount(ch),
				IsMuted:     mb.IsMuted(),
			}
			items = append(items, item)
		}
		return channelsLoadedMsg{items: items}
	}
}

func (m AppModel) cmdFetchPosts(channelID, channelName string) tea.Cmd {
	apiClient := m.api
	return func() tea.Msg {
		posts, err := apiClient.GetPostsForChannel(channelID, 0, 60)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return postsReadyMsg{
			channelID:   channelID,
			channelName: channelName,
			posts:       posts,
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
func (m AppModel) waitForWSEvent() tea.Cmd {
	ws := m.ws
	if ws == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-ws.Events
		if !ok {
			return nil // channel closed
		}
		return messages.WSEventMsg{Event: event}
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
		m.handleChannelViewed(event)
	}
	return nil
}

// handlePosted double-unmarshals the post from event.Data["post"] (JSON string)
// and returns a NewPostMsg Cmd. All routing logic lives in the NewPostMsg handler
// inside Update, so the test suite exercises the same code path as the WS flow.
func (m *AppModel) handlePosted(event api.WSEvent) tea.Cmd {
	post, ok := unmarshalPostFromEvent(event)
	if !ok {
		return nil
	}

	// Skip own posts — they're already shown via postSentMsg.
	if post.UserID == m.userID {
		return nil
	}

	return func() tea.Msg { return messages.NewPostMsg{Post: post} }
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
func (m *AppModel) handleChannelViewed(event api.WSEvent) {
	channelID, _ := event.Data["channel_id"].(string)
	if channelID != "" {
		m.sidebar.ClearUnread(channelID)
	}
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
