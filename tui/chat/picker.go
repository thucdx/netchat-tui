package chat

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/api"
)

// Picker-local styles — not exported; only used in RenderPicker.
var (
	pickerCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("33")) // bright blue, matches CursorBorder

	pickerRowHighlight = lipgloss.NewStyle().
				Background(lipgloss.Color("236"))

	pickerBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

// RenderPicker renders the attachment picker box.
// files is the list of files to show; cursor is the highlighted row index;
// width is the available terminal width.
func RenderPicker(files []api.FileInfo, cursor int, width int) string {
	const maxWidth = 50
	boxWidth := width
	if boxWidth > maxWidth {
		boxWidth = maxWidth
	}
	if boxWidth < 10 {
		boxWidth = 10
	}

	// Inner content width: boxWidth minus "│ " (2) and " │" (2) = 4 chars of framing.
	innerWidth := boxWidth - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

	var sb strings.Builder

	// ── Top border with title ──────────────────────────────────
	// "┌─ Attachments ─...─┐"
	const titleText = "─ Attachments "
	titleLen := len([]rune(titleText)) // pure ASCII so len == rune count
	// remaining dashes: boxWidth - 1 (┌) - titleLen - 1 (┐)
	dashCount := boxWidth - 1 - titleLen - 1
	if dashCount < 0 {
		dashCount = 0
	}
	topBorder := "┌" + titleText + strings.Repeat("─", dashCount) + "┐"
	sb.WriteString(pickerBorderStyle.Render(topBorder))
	sb.WriteString("\n")

	// ── File rows ──────────────────────────────────────────────
	for i, fi := range files {
		// Cursor marker: "▶" for selected, " " for others.
		marker := " "
		if i == cursor {
			marker = pickerCursorStyle.Render("▶")
		}

		// Size string, up to 10 chars: "(1.2 MB)"
		sizeStr := "(" + formatFileSize(fi.Size) + ")"

		// Available width for the name:
		// innerWidth - marker(1) - space(1) - space(1) - sizeStr - space(1)
		// = innerWidth - 4 - len(sizeStr)
		nameAvail := innerWidth - 4 - len([]rune(sizeStr))
		if nameAvail < 1 {
			nameAvail = 1
		}

		name := fi.Name
		nameRunes := []rune(name)
		if len(nameRunes) > nameAvail {
			// Truncate and add ellipsis.
			if nameAvail > 1 {
				name = string(nameRunes[:nameAvail-1]) + "…"
			} else {
				name = string(nameRunes[:nameAvail])
			}
		}

		// Pad name with spaces so size is right-aligned within innerWidth.
		// Layout: marker(1) + " " + name + padding + sizeStr + " "
		//         total content = 1 + 1 + len(name) + padding + len(sizeStr) + 1 = innerWidth
		paddingLen := innerWidth - 1 - 1 - len([]rune(name)) - len([]rune(sizeStr)) - 1
		if paddingLen < 0 {
			paddingLen = 0
		}
		padding := strings.Repeat(" ", paddingLen)

		rowContent := marker + " " + name + padding + sizeStr + " "

		// Apply highlight background to the entire inner row for the selected entry.
		if i == cursor {
			rowContent = pickerRowHighlight.Render(rowContent)
		}

		sb.WriteString(pickerBorderStyle.Render("│"))
		sb.WriteString(" ")
		sb.WriteString(rowContent)
		sb.WriteString(pickerBorderStyle.Render("│"))
		sb.WriteString("\n")
	}

	// Handle empty picker.
	if len(files) == 0 {
		emptyMsg := " (no attachments) "
		padLen := innerWidth - len([]rune(emptyMsg))
		if padLen < 0 {
			padLen = 0
		}
		row := emptyMsg + strings.Repeat(" ", padLen)
		sb.WriteString(pickerBorderStyle.Render("│"))
		sb.WriteString(" ")
		sb.WriteString(row)
		sb.WriteString(pickerBorderStyle.Render("│"))
		sb.WriteString("\n")
	}

	// ── Bottom border ──────────────────────────────────────────
	// "└──...──┘"
	bottomBorder := "└" + strings.Repeat("─", boxWidth-2) + "┘"
	sb.WriteString(pickerBorderStyle.Render(bottomBorder))

	return sb.String()
}
