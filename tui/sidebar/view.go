package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// channelTypeOrder maps channel types to sort priority (lower = earlier).
func channelTypeOrder(t string) int {
	switch t {
	case "D":
		return 0
	case "O":
		return 1
	case "P":
		return 2
	default:
		return 3
	}
}

// sectionFor returns the section header label for a given channel type.
func sectionFor(channelType string) string {
	switch channelType {
	case "D":
		return "DIRECT MESSAGES"
	default:
		return "CHANNELS"
	}
}

// Render builds the sidebar string for the given model.
func Render(m Model) string {
	if len(m.items) == 0 {
		return styles.SidebarStyle.Render("")
	}

	// Items are pre-sorted by SetItems/IncrementUnread (DMs first, then by
	// LastPostAt descending). Compute the visible window directly.
	end := m.viewOffset + m.height
	if end > len(m.items) {
		end = len(m.items)
	}
	if m.viewOffset >= len(m.items) {
		return styles.SidebarStyle.Render("")
	}
	visible := m.items[m.viewOffset:end]

	// Resolve cursor index and selected index for highlighting.
	cursorIdx := m.cursor
	selectedIdx := m.selected

	var lines []string
	lastSection := ""

	for i, item := range visible {
		// Emit a section header only when the section changes within the visible window.
		section := sectionFor(item.Channel.Type)
		if section != lastSection {
			lines = append(lines, styles.SectionHeaderStyle.Render(section))
			lastSection = section
		}

		absIdx := m.viewOffset + i
		isCursor := absIdx == cursorIdx
		isSelected := absIdx == selectedIdx

		// Build the icon prefix.
		var icon string
		if item.IsMuted {
			icon = "🔇"
		}
		switch item.Channel.Type {
		case "D":
			icon += "@"
		case "O":
			icon += "#"
		case "P":
			icon += "🔒"
		}

		// Build the badge.
		var badge string
		if item.UnreadCount > 0 {
			badgeText := fmt.Sprintf("%d", item.UnreadCount)
			if item.IsMuted {
				badge = styles.MutedBadge.Render(badgeText)
			} else {
				badge = styles.UnreadBadge.Render(badgeText)
			}
		}

		// Compute available width for the name.
		// Total content width = SidebarWidth - 2 (PaddingLeft(1) on each side accounted for in style).
		// We must fit: icon + space + name + space + badge within the row width.
		contentWidth := styles.SidebarWidth - 2
		iconWidth := lipgloss.Width(icon)
		badgeWidth := lipgloss.Width(badge)

		// space between icon and name, space before badge
		nameWidth := contentWidth - iconWidth - 1
		if badgeWidth > 0 {
			nameWidth -= badgeWidth + 1
		}
		if nameWidth < 0 {
			nameWidth = 0
		}

		// Truncate or pad the display name.
		name := item.DisplayName
		nameRunes := []rune(name)
		nameDispWidth := lipgloss.Width(name)
		if nameDispWidth > nameWidth {
			// Truncate to fit.
			for lipgloss.Width(string(nameRunes)) > nameWidth && len(nameRunes) > 0 {
				nameRunes = nameRunes[:len(nameRunes)-1]
			}
			name = string(nameRunes)
		}
		// Pad name to fill available width.
		currentNameWidth := lipgloss.Width(name)
		if currentNameWidth < nameWidth {
			name += strings.Repeat(" ", nameWidth-currentNameWidth)
		}

		// Assemble the row content.
		var rowContent string
		if badgeWidth > 0 {
			rowContent = icon + " " + name + " " + badge
		} else {
			rowContent = icon + " " + name
		}

		// Apply row style.
		var rowStyle lipgloss.Style
		switch {
		case isCursor:
			rowStyle = styles.ChannelSelected
		case item.IsMuted:
			rowStyle = styles.ChannelMuted
		case item.UnreadCount > 0:
			rowStyle = styles.ChannelUnread
		default:
			rowStyle = styles.ChannelNormal
		}
		_ = isSelected // reserved: open channel could get a distinct style later

		lines = append(lines, rowStyle.Render(rowContent))
	}

	return strings.Join(lines, "\n")
}
