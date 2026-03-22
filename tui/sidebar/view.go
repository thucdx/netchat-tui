package sidebar

import (
	"fmt"
	"sort"
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

	// Pass 1: sort a copy of items: DMs first, then Open, then Private.
	sorted := make([]ChannelItem, len(m.items))
	copy(sorted, m.items)
	sort.Slice(sorted, func(i, j int) bool {
		ti := channelTypeOrder(sorted[i].Channel.Type)
		tj := channelTypeOrder(sorted[j].Channel.Type)
		if ti != tj {
			return ti < tj
		}
		return sorted[i].DisplayName < sorted[j].DisplayName
	})

	// Pass 2: compute the visible window and iterate only visible items.
	end := m.viewOffset + m.height
	if end > len(sorted) {
		end = len(sorted)
	}
	if m.viewOffset >= len(sorted) {
		return styles.SidebarStyle.Render("")
	}
	visible := sorted[m.viewOffset:end]

	// Resolve cursor and selected channel IDs for highlighting.
	selectedID := ""
	cursorID := ""
	if m.selected >= 0 && m.selected < len(m.items) {
		selectedID = m.items[m.selected].Channel.ID
	}
	if m.cursor >= 0 && m.cursor < len(m.items) {
		cursorID = m.items[m.cursor].Channel.ID
	}

	var lines []string
	lastSection := ""

	for _, item := range visible {
		// Emit a section header only when the section changes within the visible window.
		section := sectionFor(item.Channel.Type)
		if section != lastSection {
			lines = append(lines, styles.SectionHeaderStyle.Render(section))
			lastSection = section
		}

		isCursor := item.Channel.ID == cursorID
		isSelected := item.Channel.ID == selectedID

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
		_ = isSelected // selected channel is highlighted differently only via cursor

		lines = append(lines, rowStyle.Render(rowContent))
	}

	return strings.Join(lines, "\n")
}
