//go:build ignore

// Run with: go run theme_preview.go
// Shows a side-by-side before/after of the proposed GitHub Dark theme.
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Proposed palette ─────────────────────────────────────────────────────────

var (
	// Backgrounds
	pBg        = lipgloss.Color("#0d1117")
	pSurface   = lipgloss.Color("#161b22")
	pSelected  = lipgloss.Color("#1c2128")
	pBorder    = lipgloss.Color("#21262d")
	pBorderHi  = lipgloss.Color("#30363d")

	// Text
	pFg        = lipgloss.Color("#e6edf3")
	pFgMuted   = lipgloss.Color("#7d8590")
	pFgDimmer  = lipgloss.Color("#484f58")

	// Accents
	pBlue      = lipgloss.Color("#58a6ff")
	pGreen     = lipgloss.Color("#3fb950")
	pYellow    = lipgloss.Color("#e3b341")
	pRed       = lipgloss.Color("#f85149")

	// User palette (left-border only, no bg)
	userColors = []lipgloss.Color{
		lipgloss.Color("#58a6ff"), // blue
		lipgloss.Color("#f78166"), // coral
		lipgloss.Color("#7ee787"), // green
		lipgloss.Color("#ffa657"), // orange
		lipgloss.Color("#d2a8ff"), // purple
		lipgloss.Color("#79c0ff"), // sky
		lipgloss.Color("#ff7b72"), // red
		lipgloss.Color("#56d364"), // mint
	}
)

// ── Layout widths ─────────────────────────────────────────────────────────────

const (
	sidebarW = 26
	chatW    = 54
	totalW   = sidebarW + chatW + 3
)

// ── Sidebar rendering ─────────────────────────────────────────────────────────

func renderSidebar() string {
	type channel struct {
		icon, name string
		unread     int
		muted      bool
		active     bool
	}

	channels := []channel{
		{"@", "Alice Smith", 0, false, true},
		{"#", "general", 3, false, false},
		{"⊕", "Team Alpha", 0, false, false},
		{"#", "random", 0, false, false},
		{"■", "ops-team", 1, false, false},
		{"@", "Bob Nguyen", 7, false, false},
		{"#", "announcements", 0, false, false},
		{"ø", "quietchan", 0, true, false},
	}

	inner := sidebarW - 2 // subtract right border
	var rows []string

	// Header
	header := lipgloss.NewStyle().
		Foreground(pFgMuted).
		Bold(true).
		PaddingLeft(1).
		Width(inner).
		Render("CHANNELS")
	rows = append(rows, header)
	rows = append(rows, lipgloss.NewStyle().Foreground(pBorder).Width(inner).Render(strings.Repeat("─", inner-1)))

	for _, ch := range channels {
		label := ch.icon + " " + ch.name

		var badge string
		if ch.unread > 0 && !ch.muted {
			badge = lipgloss.NewStyle().
				Foreground(pBg).
				Background(pYellow).
				Bold(true).
				PaddingLeft(1).PaddingRight(1).
				Render(fmt.Sprintf("%d", ch.unread))
		}

		var row string
		switch {
		case ch.active:
			nameStyle := lipgloss.NewStyle().
				Foreground(pBlue).
				Bold(true).
				PaddingLeft(1)
			if badge != "" {
				avail := inner - 1 - lipgloss.Width(badge)
				row = nameStyle.Width(avail).Render(label) + badge
			} else {
				row = nameStyle.Width(inner).Render(label)
			}
		case ch.muted:
			row = lipgloss.NewStyle().
				Foreground(pFgDimmer).
				PaddingLeft(1).
				Width(inner).
				Render(ch.icon + " " + ch.name)
		case ch.unread > 0:
			avail := inner - 1 - lipgloss.Width(badge)
			row = lipgloss.NewStyle().
				Foreground(pFg).
				Bold(true).
				PaddingLeft(1).
				Width(avail).
				Render(label) + badge
		default:
			row = lipgloss.NewStyle().
				Foreground(pFgMuted).
				PaddingLeft(1).
				Width(inner).
				Render(label)
		}
		rows = append(rows, row)
	}

	// Footer
	rows = append(rows, "")
	rows = append(rows, lipgloss.NewStyle().
		Foreground(pFgDimmer).
		PaddingLeft(1).
		Width(inner).
		Render("↕ 8/42"))

	body := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Background(pSurface).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(pBorder).
		Width(sidebarW).
		Height(22).
		Render(body)
}

// ── Chat rendering ────────────────────────────────────────────────────────────

type msg struct {
	user      string
	userColor lipgloss.Color
	isSelf    bool
	time      string
	lines     []string
}

func renderMsg(m msg, showHeader bool) string {
	borderColor := m.userColor
	if m.isSelf {
		borderColor = pGreen
	}

	var b strings.Builder

	if showHeader {
		var nameStr string
		if m.isSelf {
			nameStr = lipgloss.NewStyle().Foreground(pGreen).Bold(true).Render("You ▶")
		} else {
			nameStr = lipgloss.NewStyle().Foreground(m.userColor).Bold(true).Render(m.user)
		}
		ts := lipgloss.NewStyle().Foreground(pFgMuted).Render("  " + m.time)
		b.WriteString(nameStr + ts + "\n")
	}

	for _, line := range m.lines {
		b.WriteString(lipgloss.NewStyle().Foreground(pFg).Render(line) + "\n")
	}

	content := strings.TrimRight(b.String(), "\n")

	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(borderColor).
		PaddingLeft(1).
		Width(chatW - 2).
		Render(content)
}

func renderChat() string {
	messages := []struct {
		msg
		showHeader bool
	}{
		{msg{"Alice Smith", userColors[0], false, "10:28", []string{"Hey team! Sprint review at 3pm today"}}, true},
		{msg{"Alice Smith", userColors[0], false, "10:28", []string{"Please update your tickets before then"}}, false},
		{msg{"Bob Nguyen", userColors[1], false, "10:29", []string{"Got it, will do 👍"}}, true},
		{msg{"You", pGreen, true, "10:31", []string{"I'll prepare the demo slides"}}, true},
		{msg{"You", pGreen, true, "10:31", []string{"Should take about 20 mins"}}, false},
		{msg{"Carol", userColors[4], false, "10:33", []string{"Can we push to 4pm? I have a conflict"}}, true},
		{msg{"Alice Smith", userColors[0], false, "10:34", []string{"Sure, 4pm works. Updating the invite"}}, true},
	}

	const unreadLabel = " unread "
	unreadInner := chatW - 2 // full inner width of the chat pane
	dashEach := (unreadInner - len(unreadLabel)) / 2
	if dashEach < 1 {
		dashEach = 1
	}
	unreadLine := strings.Repeat("─", dashEach) + unreadLabel + strings.Repeat("─", unreadInner-dashEach-len(unreadLabel))
	unreadDivider := lipgloss.NewStyle().
		Foreground(pFgDimmer).
		Width(unreadInner).
		Render(unreadLine)

	header := lipgloss.NewStyle().
		Foreground(pFg).
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(pBorder).
		Width(chatW).
		Render("@ Alice Smith")

	var rows []string
	rows = append(rows, header)

	for i, m := range messages {
		if i == 4 {
			rows = append(rows, unreadDivider)
		}
		rows = append(rows, renderMsg(m.msg, m.showHeader))
	}

	// Input box
	inputBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(pBlue).
		PaddingLeft(1).
		Width(chatW - 2).
		Height(4).
		Render(lipgloss.NewStyle().Foreground(pFg).Render("> type a message…") +
			"\n\n" +
			lipgloss.NewStyle().Foreground(pFgDimmer).Render("  Shift+Enter for newline · Enter to send"))

	rows = append(rows, inputBorder)

	return lipgloss.NewStyle().
		Background(pBg).
		Width(chatW).
		Render(strings.Join(rows, "\n"))
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	sidebar := renderSidebar()
	chat := renderChat()

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chat)

	title := lipgloss.NewStyle().
		Foreground(pBlue).
		Bold(true).
		MarginBottom(1).
		Render("netchat-tui  ·  Proposed GitHub Dark theme preview")

	fmt.Println()
	fmt.Println(title)
	fmt.Println(layout)
	fmt.Println()

	// Color swatch
	fmt.Println(lipgloss.NewStyle().Foreground(pFgMuted).Render("User accent palette:"))
	names := []string{"blue", "coral", "green", "orange", "purple", "sky", "red", "mint"}
	var swatches []string
	for i, c := range userColors {
		swatch := lipgloss.NewStyle().
			Foreground(pBg).
			Background(c).
			PaddingLeft(1).PaddingRight(1).
			Render(names[i])
		swatches = append(swatches, swatch)
	}
	fmt.Println(strings.Join(swatches, " "))
	fmt.Println()
}
