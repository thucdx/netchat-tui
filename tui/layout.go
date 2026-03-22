package tui

import "github.com/thucdx/netchat-tui/tui/styles"

// Layout holds the computed dimensions for all panes.
// Recomputed on every tea.WindowSizeMsg.
type Layout struct {
	TotalWidth   int
	TotalHeight  int
	SidebarWidth int // always styles.SidebarWidth (28) — includes border
	ChatWidth    int // TotalWidth - SidebarWidth
	InputHeight  int // always styles.InputHeight (3) — includes border
	ChatHeight   int // TotalHeight - InputHeight
}

// NewLayout computes the Layout from the given terminal dimensions.
// Enforces minimum sizes so values are always positive.
func NewLayout(width, height int) Layout {
	chatWidth := width - styles.SidebarWidth
	if chatWidth < 1 {
		chatWidth = 1
	}

	chatHeight := height - styles.InputHeight
	if chatHeight < 1 {
		chatHeight = 1
	}

	return Layout{
		TotalWidth:   width,
		TotalHeight:  height,
		SidebarWidth: styles.SidebarWidth,
		ChatWidth:    chatWidth,
		InputHeight:  styles.InputHeight,
		ChatHeight:   chatHeight,
	}
}

// IsValid returns true if the terminal is large enough to render the UI.
// Minimum: 60 columns wide, 10 rows tall.
func (l Layout) IsValid() bool {
	return l.TotalWidth >= 60 && l.TotalHeight >= 10
}
