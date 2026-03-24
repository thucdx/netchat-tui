package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/internal/keymap"
)

// helpPopupWidth is the total inner width (excluding border) of the help box.
const helpPopupWidth = 64

var (
	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	helpDismissStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("11"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 2)
)

// renderHelpOverlay builds the keybinding popup and centres it within
// the terminal area of (totalWidth × totalHeight).
func renderHelpOverlay(sections []keymap.HelpSection, totalWidth, totalHeight int) string {
	// Each column gets half of the inner width minus the gap between columns.
	colWidth := (helpPopupWidth - 2) / 2 // 2 = 1-char gap × 2 columns

	// Split sections evenly between left and right columns.
	half := (len(sections) + 1) / 2
	left := renderHelpColumn(sections[:half], colWidth)
	right := renderHelpColumn(sections[half:], colWidth)

	// Pad the shorter column so JoinHorizontal aligns correctly.
	for len(left) < len(right) {
		left = append(left, "")
	}
	for len(right) < len(left) {
		right = append(right, "")
	}

	leftStyle := lipgloss.NewStyle().Width(colWidth)
	rightStyle := lipgloss.NewStyle().Width(colWidth)

	var rows []string
	// Header row.
	header := helpTitleStyle.Render("Keybindings") +
		helpDismissStyle.Render("  — press ? or Esc to close")
	rows = append(rows, header, "")

	// Body rows: merge left + right side by side.
	for i := range left {
		l := leftStyle.Render(left[i])
		r := rightStyle.Render(right[i])
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, l, "  ", r))
	}

	inner := strings.Join(rows, "\n")
	box := helpBoxStyle.Width(helpPopupWidth).Render(inner)
	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}

// renderHelpColumn converts a slice of HelpSections into a flat list of
// display lines that fit within colWidth characters.
func renderHelpColumn(sections []keymap.HelpSection, colWidth int) []string {
	// Reserve space for key text; description fills the rest.
	const keyW = 13
	descW := max(colWidth-keyW-1, 8) // -1 for the space between key and desc

	var lines []string
	for _, s := range sections {
		lines = append(lines, helpSectionStyle.Render(s.Title))
		for _, b := range s.Bindings {
			h := b.Help()
			k := lipgloss.NewStyle().Width(keyW).Render(h.Key)
			d := helpDescStyle.MaxWidth(descW).Render(h.Desc)
			lines = append(lines, helpKeyStyle.Render(k)+d)
		}
		lines = append(lines, "") // blank separator between sections
	}
	return lines
}
