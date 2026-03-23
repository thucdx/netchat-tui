package input

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
	"github.com/thucdx/netchat-tui/tui/styles"
)

const maxMessageLength = 4000 // Mattermost limit (security: S9)

// Model wraps bubbles/textarea for the message input box.
type Model struct {
	textarea  textarea.Model
	channelID string      // which channel this message will be sent to
	sending   bool        // true while CreatePost is in flight (disable send)
	keys      keymap.KeyMap
	width     int
	focused   bool
}

// NewModel returns an input model ready to use.
func NewModel(keys keymap.KeyMap) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.CharLimit = maxMessageLength
	ta.ShowLineNumbers = false
	ta.SetWidth(80)  // updated later by SetSize
	ta.SetHeight(1)  // single-line; grows with Shift+Enter
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))
	return Model{textarea: ta, keys: keys}
}

// SetChannelID updates the target channel (called when sidebar selection changes).
func (m *Model) SetChannelID(channelID string) {
	m.channelID = channelID
}

// SetSize updates the textarea width and height.
// width: full chat panel width minus border (2 cols)
// height: InputHeight minus border (2 rows) = 1
func (m *Model) SetSize(width, height int) {
	m.width = width
	// The InputStyle has PaddingLeft(1) and a border (left+right = 2 cols).
	// So the textarea inner content width = width - 2 (border) - 1 (padding) = width - 3.
	// We use width - 4 to account for border (2) + padding (1) + small safety margin (1).
	taWidth := width - 4
	if taWidth < 1 {
		taWidth = 1
	}
	m.textarea.SetWidth(taWidth)
	m.textarea.SetHeight(height)
}

// SetFocused gives or removes focus from the textarea.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
}

// SetSending enables/disables the "sending" lock (prevents double-send).
func (m *Model) SetSending(sending bool) {
	m.sending = sending
}

// Init satisfies tea.Model; returns the textarea blink command.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Plain Enter (no modifiers) triggers send.
		if msg.Type == tea.KeyEnter && !msg.Alt {
			// If a send is already in flight, drop the key to prevent double-send.
			if m.sending {
				return m, nil
			}

			text := strings.TrimSpace(m.textarea.Value())
			if text != "" && m.channelID != "" {
				m.sending = true
				m.textarea.Reset()
				return m, func() tea.Msg {
					return messages.SendMessageMsg{
						ChannelID: m.channelID,
						Text:      text,
					}
				}
			}
			// Empty message or no channel — swallow the Enter key.
			return m, nil
		}

		// All other keys delegate to the textarea.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	// Non-key messages (e.g. blink tick) also propagate to the textarea.
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View satisfies tea.Model.
func (m Model) View() string {
	// Choose the appropriate border style based on focus.
	var st = styles.InputStyle
	if m.focused {
		st = styles.InputFocusedStyle
	}

	// InputStyle has PaddingLeft(1) and a NormalBorder (left+right = 2 cols).
	// Set the style's content width to (m.width - 2) so that with the border
	// the total rendered width equals m.width.
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	return st.Width(contentWidth).Render(m.textarea.View())
}
