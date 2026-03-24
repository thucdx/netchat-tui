package styles

import "github.com/charmbracelet/lipgloss"

// Dimensions
const (
	SidebarWidth       = 28 // default sidebar width (includes border)
	SidebarBorderWidth = 1  // right-border added by BorderRight(true)
	InputHeight        = 5
)

// GitHub Dark palette
var (
	ColorBg          = lipgloss.Color("#0d1117") // base background
	colorSurface     = lipgloss.Color("#161b22") // sidebar / panel surface
	ColorSelected    = lipgloss.Color("#1c2128") // selected row highlight (exported for cursor bg)
	colorBorder      = lipgloss.Color("#21262d") // subtle border
	ColorBorderHi    = lipgloss.Color("#30363d") // slightly more visible border (system msg borders)
	colorBorderFocus = lipgloss.Color("#58a6ff") // focused panel border (blue)
	colorFg          = lipgloss.Color("#e6edf3") // primary text
	colorFgMuted     = lipgloss.Color("#7d8590") // secondary / muted text
	colorFgDimmer    = lipgloss.Color("#484f58") // very dimmed (muted channels, etc.)
	colorPrimary     = lipgloss.Color("#58a6ff") // blue accent
	colorSuccess     = lipgloss.Color("#3fb950") // green (own messages, success)
	colorUnread      = lipgloss.Color("#e3b341") // yellow (unread badge)
	colorError       = lipgloss.Color("#f85149") // red
)

// Sidebar styles
// Note: SidebarStyle and channel row styles intentionally omit Width so that
// the caller can apply dynamic widths at render time (e.g. for resizable sidebar).
var (
	// SidebarStyle renders the sidebar panel. Width is applied by the caller
	// at render time: SidebarStyle.Width(w).Height(h).Render(...).
	SidebarStyle = lipgloss.NewStyle().
			Background(colorSurface).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	// SidebarFocusedStyle is the sidebar style when the panel has focus (blue border).
	SidebarFocusedStyle = lipgloss.NewStyle().
				Background(colorSurface).
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBorderFocus)

	SidebarHeader = lipgloss.NewStyle().
			Foreground(colorFgMuted).
			Bold(true).
			PaddingLeft(1)

	// Channel row styles — Width is NOT set here; view.go applies it dynamically.
	ChannelNormal = lipgloss.NewStyle().
			Foreground(colorFgMuted).
			Background(colorSurface).
			PaddingLeft(1)

	ChannelSelected = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Background(ColorSelected).
			Bold(true).
			PaddingLeft(1)

	ChannelMuted = lipgloss.NewStyle().
			Foreground(colorFgDimmer).
			Background(colorSurface).
			PaddingLeft(1)

	ChannelMutedUnread = lipgloss.NewStyle().
				Foreground(colorFgDimmer).
				Background(colorSurface).
				Bold(true).
				PaddingLeft(1)

	ChannelActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Background(colorSurface).
			Bold(true).
			PaddingLeft(1)

	ChannelUnread = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorSurface).
			Bold(true).
			PaddingLeft(1)

	UnreadBadge = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(colorUnread).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	MutedBadge = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(colorFgMuted).
			PaddingLeft(1).
			PaddingRight(1)
)

// Chat styles
var (
	ChatStyle = lipgloss.NewStyle().
			Background(ColorBg)

	MessageUsername = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// MessageMyUsername styles the "You" header for messages sent by the current user.
	MessageMyUsername = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	MessageTimestamp = lipgloss.NewStyle().
				Foreground(colorFgMuted)

	MessageText = lipgloss.NewStyle().
			Foreground(colorFg)

	MessageSystem = lipgloss.NewStyle().
			Foreground(colorFgMuted).
			Italic(true)

	MessageEdited = lipgloss.NewStyle().
			Foreground(colorFgMuted)

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

// MessageUserPalette provides 8 rotating accent colors for chat participants.
// Each color is used for the user's name text and their message left-border.
var MessageUserPalette = []lipgloss.Color{
	lipgloss.Color("#58a6ff"), // blue
	lipgloss.Color("#f78166"), // coral
	lipgloss.Color("#7ee787"), // green
	lipgloss.Color("#ffa657"), // orange
	lipgloss.Color("#d2a8ff"), // purple
	lipgloss.Color("#79c0ff"), // sky
	lipgloss.Color("#ff7b72"), // red
	lipgloss.Color("#56d364"), // mint
}

// UserColorIndex maps a userID to a deterministic index into MessageUserPalette.
func UserColorIndex(userID string) int {
	if userID == "" {
		return 0
	}
	var sum int
	for _, b := range []byte(userID) {
		sum += int(b)
	}
	return sum % len(MessageUserPalette)
}

// Search styles
var (
	// SearchQueryStyle renders the "/ query█" line at the top of search mode.
	SearchQueryStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Bold(true).
				PaddingLeft(1)

	// SearchHintStyle renders the "type ≥3 chars" hint line.
	SearchHintStyle = lipgloss.NewStyle().
				Foreground(colorFgMuted).
				Italic(true).
				PaddingLeft(1)

	// SearchNewItemStyle renders "+ @user" / "+ #channel" rows for API results.
	SearchNewItemStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				PaddingLeft(1)

	// SearchNewItemCursorStyle is SearchNewItemStyle with cursor highlight.
	SearchNewItemCursorStyle = lipgloss.NewStyle().
					Foreground(colorSuccess).
					Background(ColorSelected).
					Bold(true).
					PaddingLeft(1)

	// SearchConfirmStyle renders the "Join #ch? [y/N]" confirmation line.
	SearchConfirmStyle = lipgloss.NewStyle().
				Foreground(colorUnread).
				Bold(true).
				PaddingLeft(1)
)

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

// CursorBorder is the left-border accent applied to the cursor message in the chat pane.
var CursorBorder = lipgloss.NewStyle().
	BorderLeft(true).
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(colorBorderFocus).
	BorderBackground(ColorBg).
	PaddingLeft(1)

// VisualSelectionBorder highlights posts inside the visual selection range.
// Uses a yellow thick left border to distinguish from the blue cursor border.
var VisualSelectionBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderLeft(true).
	BorderForeground(colorUnread).
	BorderBackground(ColorBg).
	PaddingLeft(1)

// UnreadDivider is the style for the "──── unread ────" line shown between read and unread messages.
var UnreadDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("#484f58"))

// Utility styles
var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(colorFgMuted)

	TitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	// SpinnerStyle is used for loading indicators throughout the UI.
	SpinnerStyle = lipgloss.NewStyle().Foreground(colorPrimary)
)
