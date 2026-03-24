package styles

import "github.com/charmbracelet/lipgloss"

// Dimensions
const (
	SidebarWidth       = 28 // default sidebar width (includes border)
	SidebarBorderWidth = 1  // right-border added by BorderRight(true)
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
// Note: SidebarStyle and channel row styles intentionally omit Width so that
// the caller can apply dynamic widths at render time (e.g. for resizable sidebar).
var (
	// SidebarStyle renders the sidebar panel. Width is applied by the caller
	// at render time: SidebarStyle.Width(w).Height(h).Render(...).
	SidebarStyle = lipgloss.NewStyle().
			Background(colorBg).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	// SidebarFocusedStyle is the sidebar style when the panel has focus (purple border).
	SidebarFocusedStyle = lipgloss.NewStyle().
				Background(colorBg).
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBorderFocus)

	SidebarHeader = lipgloss.NewStyle().
			Foreground(colorFgDim).
			Bold(true).
			PaddingLeft(1)

	// Channel row styles — Width is NOT set here; view.go applies it dynamically.
	ChannelNormal = lipgloss.NewStyle().
			Foreground(colorFg).
			PaddingLeft(1)

	ChannelSelected = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorBgSelected).
			Bold(true).
			PaddingLeft(1)

	ChannelMuted = lipgloss.NewStyle().
			Foreground(colorFgDim).
			PaddingLeft(1)

	ChannelMutedUnread = lipgloss.NewStyle().
			Foreground(colorFgDim).
			Bold(true).
			PaddingLeft(1)

	ChannelActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			PaddingLeft(1)

	ChannelUnread = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true).
			PaddingLeft(1)

	UnreadBadge = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorUnread).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	MutedBadge = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorFgDim).
			PaddingLeft(1).
			PaddingRight(1)
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

// MessageUserPalette provides 8 rotating fg/bg pairs for chat participants.
// Backgrounds are very dark tints so they remain easy on the eye; foregrounds
// are vibrant enough to be clearly distinct at a glance.
var MessageUserPalette = []struct {
	Fg lipgloss.Color
	Bg lipgloss.Color
}{
	{Fg: lipgloss.Color("#60A5FA"), Bg: lipgloss.Color("#141c28")}, // blue
	{Fg: lipgloss.Color("#F472B6"), Bg: lipgloss.Color("#221420")}, // pink
	{Fg: lipgloss.Color("#FBBF24"), Bg: lipgloss.Color("#241e10")}, // amber
	{Fg: lipgloss.Color("#38BDF8"), Bg: lipgloss.Color("#121e24")}, // sky
	{Fg: lipgloss.Color("#C084FC"), Bg: lipgloss.Color("#1c1428")}, // violet
	{Fg: lipgloss.Color("#FB923C"), Bg: lipgloss.Color("#241a12")}, // orange
	{Fg: lipgloss.Color("#4ADE80"), Bg: lipgloss.Color("#101f14")}, // green
	{Fg: lipgloss.Color("#34D399"), Bg: lipgloss.Color("#141f1a")}, // emerald
}

// MessageSelfBg is the subtle background tint for the current user's messages.
var MessageSelfBg = lipgloss.Color("#131f16")

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
				Foreground(colorFgDim).
				Italic(true).
				PaddingLeft(1)

	// SearchNewItemStyle renders "+ @user" / "+ #channel" rows for API results.
	SearchNewItemStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				PaddingLeft(1)

	// SearchNewItemCursorStyle is SearchNewItemStyle with cursor highlight.
	SearchNewItemCursorStyle = lipgloss.NewStyle().
					Foreground(colorSuccess).
					Background(colorBgSelected).
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
	BorderForeground(lipgloss.Color("33")). // bright blue
	PaddingLeft(1)

// VisualSelectionBorder highlights posts inside the visual selection range.
// Uses a yellow thick left border to distinguish from the blue cursor border.
var VisualSelectionBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderLeft(true).
	BorderForeground(lipgloss.Color("3")). // yellow
	PaddingLeft(1)

// UnreadDivider is the style for the "──── unread ────" line shown between read and unread messages.
var UnreadDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

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
