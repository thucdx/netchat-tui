package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// channelIcon returns a single character that encodes both channel type and
// mute status, so no separate mute prefix is needed.
//
// Unmuted: # (public)  @ (DM)  ⊕ (group)  ■ (private)
// Muted:   ⊘ (public)  ø (DM)  ⊖ (group)  □ (private)
func channelIcon(chType string, muted bool) string {
	if muted {
		switch chType {
		case "D":
			return "ø"
		case "G":
			return "⊖"
		case "O":
			return "⊘"
		case "P":
			return "□"
		}
	}
	switch chType {
	case "D":
		return "@"
	case "G":
		return "⊕"
	case "O":
		return "#"
	case "P":
		return "■"
	}
	return "·"
}

// truncateName shortens name to fit within maxWidth display columns.
// If truncation is needed, the last visible character is replaced with "…"
// so the total display width is always exactly maxWidth.
// If maxWidth <= 0 the empty string is returned.
func truncateName(name string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(name) <= maxWidth {
		return name
	}
	// Reserve 1 column for the ellipsis.
	runes := []rune(name)
	for lipgloss.Width(string(runes))> maxWidth-1 && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// renderSearch renders the sidebar in search mode.
//
// When query length < 3: query bar (line 1) + hint (line 2) + normal item list.
// When query length ≥ 3: query bar (line 1) + result rows.
func renderSearch(m Model) string {
	rowWidth := m.width - 2 // text content width (same as Render)

	var lines []string

	// ── Line 1: query bar or confirm prompt ───────────────────────────────────
	if m.search.confirmTarget != nil {
		confirm := fmt.Sprintf("Join %s? [y/N]", truncateName(m.search.confirmTarget.displayName, rowWidth-13))
		lines = append(lines, styles.SearchConfirmStyle.Width(rowWidth+1).Render(confirm))
	} else {
		queryText := "/ " + m.search.query + "█"
		lines = append(lines, styles.SearchQueryStyle.Width(rowWidth+1).Render(queryText))
	}

	q := []rune(m.search.query)

	if len(q) < 3 {
		// ── Hint line + normal sidebar list ──────────────────────────────────
		hint := "type ≥3 chars to search"
		lines = append(lines, styles.SearchHintStyle.Width(rowWidth+1).Render(hint))

		// Render normal items in the remaining height.
		normalHeight := m.height - 2 // 1 query + 1 hint
		if normalHeight < 0 {
			normalHeight = 0
		}
		end := m.viewOffset + normalHeight
		if end > len(m.items) {
			end = len(m.items)
		}
		for i, item := range m.items[m.viewOffset:end] {
			absIdx := m.viewOffset + i
			isCursor := absIdx == m.cursor
			isSelected := absIdx == m.selected

			icon := channelIcon(item.Channel.Type, item.IsMuted)
			iconWidth := lipgloss.Width(icon)

			var badge string
			var badgeWidth int
			if item.UnreadCount > 0 {
				badgeText := fmt.Sprintf("%d", item.UnreadCount)
				if item.IsMuted {
					badge = styles.MutedBadge.Render(badgeText)
				} else {
					badge = styles.UnreadBadge.Render(badgeText)
				}
				badgeWidth = lipgloss.Width(badge)
			}

			nameWidth := rowWidth - iconWidth - 1
			if badgeWidth > 0 {
				nameWidth -= badgeWidth + 1
			}
			if nameWidth < 0 {
				nameWidth = 0
			}
			name := truncateName(item.DisplayName, nameWidth)
			if cur := lipgloss.Width(name); cur < nameWidth {
				name += strings.Repeat(" ", nameWidth-cur)
			}

			var rowContent string
			if badgeWidth > 0 {
				rowContent = icon + " " + name + " " + badge
			} else {
				rowContent = icon + " " + name
			}

			var rowStyle lipgloss.Style
			switch {
			case isCursor:
				rowStyle = styles.ChannelSelected
			case isSelected:
				rowStyle = styles.ChannelActive
			case item.IsMuted && item.UnreadCount > 0:
				rowStyle = styles.ChannelMutedUnread
			case item.IsMuted:
				rowStyle = styles.ChannelMuted
			case item.UnreadCount > 0:
				rowStyle = styles.ChannelUnread
			default:
				rowStyle = styles.ChannelNormal
			}
			lines = append(lines, rowStyle.Width(rowWidth+1).Render(rowContent))
		}
	} else {
		// ── Results list ──────────────────────────────────────────────────────
		maxResults := m.height - 1 // 1 line used by query bar
		if maxResults < 0 {
			maxResults = 0
		}
		for i, r := range m.search.results {
			if i >= maxResults {
				break
			}
			isCursor := i == m.search.cursor

			switch r.kind {
			case searchKindExisting:
				item := r.item
				icon := channelIcon(item.Channel.Type, item.IsMuted)
				iconWidth := lipgloss.Width(icon)

				var badge string
				var badgeWidth int
				if item.UnreadCount > 0 {
					badgeText := fmt.Sprintf("%d", item.UnreadCount)
					if item.IsMuted {
						badge = styles.MutedBadge.Render(badgeText)
					} else {
						badge = styles.UnreadBadge.Render(badgeText)
					}
					badgeWidth = lipgloss.Width(badge)
				}

				nameWidth := rowWidth - iconWidth - 1
				if badgeWidth > 0 {
					nameWidth -= badgeWidth + 1
				}
				if nameWidth < 0 {
					nameWidth = 0
				}
				name := truncateName(item.DisplayName, nameWidth)
				if cur := lipgloss.Width(name); cur < nameWidth {
					name += strings.Repeat(" ", nameWidth-cur)
				}

				var rowContent string
				if badgeWidth > 0 {
					rowContent = icon + " " + name + " " + badge
				} else {
					rowContent = icon + " " + name
				}

				var rowStyle lipgloss.Style
				if isCursor {
					rowStyle = styles.ChannelSelected
				} else if item.Channel.ID == func() string {
					if ch := m.SelectedChannel(); ch != nil {
						return ch.Channel.ID
					}
					return ""
				}() {
					rowStyle = styles.ChannelActive
				} else if item.IsMuted && item.UnreadCount > 0 {
					rowStyle = styles.ChannelMutedUnread
				} else if item.IsMuted {
					rowStyle = styles.ChannelMuted
				} else if item.UnreadCount > 0 {
					rowStyle = styles.ChannelUnread
				} else {
					rowStyle = styles.ChannelNormal
				}
				lines = append(lines, rowStyle.Width(rowWidth+1).Render(rowContent))

			case searchKindNewDM, searchKindNewChannel:
				var typeLabel string
				if r.kind == searchKindNewDM {
					typeLabel = "@"
				} else {
					typeLabel = "#"
				}
				nameWidth := rowWidth - 3 // "+ " + typeLabel
				if nameWidth < 0 {
					nameWidth = 0
				}
				name := truncateName(r.displayName, nameWidth)
				if cur := lipgloss.Width(name); cur < nameWidth {
					name += strings.Repeat(" ", nameWidth-cur)
				}
				rowContent := "+ " + typeLabel + name

				var rowStyle lipgloss.Style
				if isCursor {
					rowStyle = styles.SearchNewItemCursorStyle
				} else {
					rowStyle = styles.SearchNewItemStyle
				}
				lines = append(lines, rowStyle.Width(rowWidth+1).Render(rowContent))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// Render builds the sidebar string for the given model.
// Each item is guaranteed to occupy exactly one line because:
//   - rowContent is always a plain string (no embedded ANSI) so lipgloss
//     measures its width correctly.
//   - name is truncated (with "…") or padded to exactly nameWidth columns.
//   - The arithmetic icon+space+name+[space+badge] == rowWidth holds exactly.
//   - rowStyle.Width(rowWidth) therefore never wraps.
func Render(m Model) string {
	if m.search.active {
		return renderSearch(m)
	}
	if len(m.items) == 0 {
		return ""
	}

	visibleHeight := m.height
	if len(m.items) > m.height {
		visibleHeight = m.height - 1
	}
	end := m.viewOffset + visibleHeight
	if end > len(m.items) {
		end = len(m.items)
	}
	if m.viewOffset >= len(m.items) {
		return ""
	}
	visible := m.items[m.viewOffset:end]

	cursorIdx := m.cursor
	selectedIdx := m.selected

	// rowWidth is the text content width: what's available for icon + name + badge.
	// sidebarWidth = border(1) + paddingLeft(1) + rowWidth
	// The row style Width must be rowWidth + 1 (to include PaddingLeft) because
	// lipgloss.Width(w) wraps content at w - leftPadding.
	rowWidth := m.width - 2

	var lines []string

	for i, item := range visible {
		absIdx := m.viewOffset + i
		isCursor := absIdx == cursorIdx
		isSelected := absIdx == selectedIdx

		icon := channelIcon(item.Channel.Type, item.IsMuted)
		iconWidth := lipgloss.Width(icon)

		// Build badge (plain rendered string; width measured once).
		var badge string
		var badgeWidth int
		if item.UnreadCount > 0 {
			badgeText := fmt.Sprintf("%d", item.UnreadCount)
			if item.IsMuted {
				badge = styles.MutedBadge.Render(badgeText)
			} else {
				badge = styles.UnreadBadge.Render(badgeText)
			}
			badgeWidth = lipgloss.Width(badge)
		}

		// nameWidth: columns the name text must occupy.
		// Layout: icon(iconWidth) + sp(1) + name(nameWidth) [+ sp(1) + badge(badgeWidth)]
		nameWidth := rowWidth - iconWidth - 1
		if badgeWidth > 0 {
			nameWidth -= badgeWidth + 1
		}
		if nameWidth < 0 {
			nameWidth = 0
		}

		// Fit name into nameWidth: truncate with "…" or pad with spaces.
		// Result is always exactly nameWidth display columns.
		name := truncateName(item.DisplayName, nameWidth)
		currentW := lipgloss.Width(name)
		if currentW < nameWidth {
			name += strings.Repeat(" ", nameWidth-currentW)
		}

		// Assemble plain row content — no embedded ANSI.
		// Total visual width == rowWidth exactly.
		var rowContent string
		if badgeWidth > 0 {
			rowContent = icon + " " + name + " " + badge
		} else {
			rowContent = icon + " " + name
		}

		// Select row style.
		// Bold is applied by the style itself — never pre-rendered into rowContent —
		// so lipgloss always measures rowContent width correctly.
		var rowStyle lipgloss.Style
		switch {
		case isCursor:
			rowStyle = styles.ChannelSelected
		case isSelected:
			rowStyle = styles.ChannelActive
		case item.IsMuted && item.UnreadCount > 0:
			rowStyle = styles.ChannelMutedUnread
		case item.IsMuted:
			rowStyle = styles.ChannelMuted
		case item.UnreadCount > 0:
			rowStyle = styles.ChannelUnread
		default:
			rowStyle = styles.ChannelNormal
		}

		lines = append(lines, rowStyle.Width(rowWidth+1).Render(rowContent))
	}

	// Footer row: scroll indicator (when list overflows) + "? help" hint.
	// Both share one line to stay compact; the hint is right-aligned.
	const helpHint = "? help"
	footerWidth := rowWidth + 1 // matches other rows
	if len(m.items) > m.height {
		scroll := fmt.Sprintf("↕ %d/%d", m.cursor+1, len(m.items))
		gap := footerWidth - lipgloss.Width(scroll) - lipgloss.Width(helpHint) - 2 // 2 for leading "  "
		gap = max(gap, 1)
		indicator := "  " + scroll + strings.Repeat(" ", gap) + helpHint
		lines = append(lines, styles.SubtleStyle.Width(footerWidth).Render(indicator))
	} else if len(m.items) < m.height {
		// Spare rows available — show hint without consuming a channel slot.
		lines = append(lines, styles.SubtleStyle.Width(footerWidth).Render("  "+helpHint))
	}

	return strings.Join(lines, "\n")
}
