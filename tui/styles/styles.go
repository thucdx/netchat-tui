package styles

import "github.com/charmbracelet/lipgloss"

// Dimensions
const (
	SidebarWidth       = 28
	SidebarBorderWidth = 1 // right-border added by BorderRight(true)
	InputHeight        = 3
)

// Colors
var (
	colorPrimary    = lipgloss.Color("#7C3AED") // purple accent
	colorMuted      = lipgloss.Color("#6B7280") // gray
	colorUnread     = lipgloss.Color("#F59E0B") // amber — unread badge
	colorError      = lipgloss.Color("#EF4444") // red
	colorSuccess    = lipgloss.Color("#10B981") // green
	colorFg         = lipgloss.Color("#E5E7EB") // near-white text
	colorFgDim      = lipgloss.Color("#9CA3AF") // dimmed text
	colorBg         = lipgloss.Color("#111827") // dark bg
	colorBgSelected = lipgloss.Color("#2D3748") // selected row (bumped for tmux 256-color visibility)
	colorBorder      = lipgloss.Color("#374151") // subtle border
	colorBorderFocus = lipgloss.Color("#7C3AED") // purple — focused panel border
)

// Sidebar styles
var (
	// SidebarStyle renders the sidebar panel.
	// Width is set to (SidebarWidth - SidebarBorderWidth) so that content
	// occupies 27 columns and the right border adds the final column,
	// giving a total of SidebarWidth (28) columns consumed in the layout.
	SidebarStyle = lipgloss.NewStyle().
			Width(SidebarWidth - SidebarBorderWidth).
			Background(colorBg).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	// SidebarFocusedStyle is the sidebar style when the panel has focus (purple border).
	SidebarFocusedStyle = lipgloss.NewStyle().
				Width(SidebarWidth - SidebarBorderWidth).
				Background(colorBg).
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBorderFocus)

	SidebarHeader = lipgloss.NewStyle().
			Foreground(colorFgDim).
			Bold(true).
			PaddingLeft(1)

	ChannelNormal = lipgloss.NewStyle().
			Foreground(colorFg).
			PaddingLeft(1).
			Width(SidebarWidth - 2)

	ChannelSelected = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorBgSelected).
			Bold(true).
			PaddingLeft(1).
			Width(SidebarWidth - 2)

	ChannelMuted = lipgloss.NewStyle().
			Foreground(colorFgDim).
			PaddingLeft(1).
			Width(SidebarWidth - 2)

	ChannelActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			PaddingLeft(1).
			Width(SidebarWidth - 2)

	ChannelUnread = lipgloss.NewStyle().
			Foreground(colorUnread).
			Bold(true).
			PaddingLeft(1).
			Width(SidebarWidth - 2)

	UnreadBadge = lipgloss.NewStyle().
			Foreground(colorUnread).
			Bold(true)

	MutedBadge = lipgloss.NewStyle().
			Foreground(colorFgDim)
)

// Chat styles
var (
	ChatStyle = lipgloss.NewStyle().
			Background(colorBg)

	MessageUsername = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// MessageMyUsername styles the "You" header for messages sent by the current user.
	MessageMyUsername = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	MessageTimestamp = lipgloss.NewStyle().
			Foreground(colorFgDim)

	MessageText = lipgloss.NewStyle().
			Foreground(colorFg)

	MessageSystem = lipgloss.NewStyle().
			Foreground(colorFgDim).
			Italic(true)

	MessageEdited = lipgloss.NewStyle().
			Foreground(colorFgDim)

	// ChannelHeader is the base style for the channel name header bar.
	// Callers must set the width at render time: ChannelHeader.Width(chatWidth).Render(text)
	ChannelHeader = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)
)

// ChannelHeaderWithWidth returns a ChannelHeader style with the given width applied.
// Use this instead of calling .Width() directly to ensure consistent rendering.
func ChannelHeaderWithWidth(w int) lipgloss.Style {
	return ChannelHeader.Width(w)
}

// Input styles
var (
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			PaddingLeft(1)

	InputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBorderFocus).
				PaddingLeft(1)
)

// Utility styles
var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(colorFgDim)

	TitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	// SpinnerStyle is used for loading indicators throughout the UI.
	SpinnerStyle = lipgloss.NewStyle().Foreground(colorPrimary)
)
