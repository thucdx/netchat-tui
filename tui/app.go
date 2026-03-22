package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// FocusPane identifies which panel currently has keyboard focus.
type FocusPane int

const (
	FocusSidebar FocusPane = iota
	FocusChat
	FocusInput
)

// AppModel is the root Bubbletea model.
// It owns layout, focus state, keymap, and will own sub-models in later phases.
type AppModel struct {
	layout Layout
	focus  FocusPane
	keys   keymap.KeyMap
	api    *api.Client // nil in Phase 3; always nil-check before use
	ready  bool        // true after first WindowSizeMsg received
}

// NewAppModel creates the root model.
func NewAppModel(apiClient *api.Client) AppModel {
	return AppModel{
		focus: FocusSidebar,
		keys:  keymap.DefaultKeyMap(),
		api:   apiClient,
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.layout = NewLayout(msg.Width, msg.Height)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Quit: q or Ctrl+C only when sidebar is focused (never swallow 'q' from input).
		if m.focus == FocusSidebar {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
		} else if msg.String() == "ctrl+c" {
			// Ctrl+C always quits regardless of focus.
			return m, tea.Quit
		}

		switch {
		case key.Matches(msg, m.keys.NextPanel):
			// Cycle focus: Sidebar → Chat → Input → Sidebar
			switch m.focus {
			case FocusSidebar:
				m.focus = FocusChat
			case FocusChat:
				m.focus = FocusInput
			case FocusInput:
				m.focus = FocusSidebar
			}

		case key.Matches(msg, m.keys.FocusInput):
			// Focus input from sidebar or chat.
			if m.focus == FocusSidebar || m.focus == FocusChat {
				m.focus = FocusInput
			}

		case key.Matches(msg, m.keys.FocusSidebar):
			// Return focus to sidebar from input or chat.
			if m.focus == FocusInput || m.focus == FocusChat {
				m.focus = FocusSidebar
			}
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m AppModel) View() string {
	if !m.ready {
		return ""
	}

	if !m.layout.IsValid() {
		return "Terminal too small — please resize to at least 60×10."
	}

	sidebarView := m.renderSidebar()
	chatView := m.renderChat()
	inputView := m.renderInput()

	chatAndInput := lipgloss.JoinVertical(lipgloss.Left, chatView, inputView)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, chatAndInput)
}

// renderSidebar renders the left panel with a static channel list.
func (m AppModel) renderSidebar() string {
	lines := []string{
		"# general",
		"@ john.doe",
		"# random",
	}
	content := strings.Join(lines, "\n")

	sidebarStyle := styles.SidebarStyle
	if m.focus == FocusSidebar {
		sidebarStyle = styles.SidebarFocusedStyle
	}

	// Height fills the full terminal height.
	return sidebarStyle.Height(m.layout.TotalHeight).Render(content)
}

// renderChat renders the top-right chat panel with a placeholder message.
func (m AppModel) renderChat() string {
	placeholder := "Select a channel to start chatting"

	// Note: ChatStyle has no border, so Width = ChatWidth directly.
	// If a border is ever added, change to Width(m.layout.ChatWidth - 2).
	chatStyle := styles.ChatStyle.
		Width(m.layout.ChatWidth).
		Height(m.layout.ChatHeight).
		Align(lipgloss.Center, lipgloss.Center)

	return chatStyle.Render(placeholder)
}

// renderInput renders the bottom-right input panel with a placeholder prompt.
func (m AppModel) renderInput() string {
	placeholder := "> type here..."

	inputStyle := styles.InputStyle
	if m.focus == FocusInput {
		inputStyle = styles.InputFocusedStyle
	}

	// Width accounts for the 2-character border (left + right).
	return inputStyle.
		Width(m.layout.ChatWidth - 2).
		Height(m.layout.InputHeight - 2).
		Render(placeholder)
}
