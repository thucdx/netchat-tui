package chat

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// popupBorderStyle is the border color for the image popup overlay.
// Uses color "33" (bright blue), matching the cursor border and picker cursor.
var popupBorderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("33"))

// RenderImagePopup renders a bordered popup overlay displaying a terminal-art image.
// image is the pre-rendered ANSI string (from imageCache).
// title is the filename shown in the top border.
// width and height are the available terminal dimensions for the popup area.
func RenderImagePopup(image, title string, width, height int) string {
	if width < 6 {
		width = 6
	}
	if height < 3 {
		height = 3
	}

	// Inner content width: width minus "│ " (2) and " │" (2) = 4 chars of framing.
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Body rows available: height minus top border (1) and bottom border (1).
	bodyRows := height - 2
	if bodyRows < 1 {
		bodyRows = 1
	}

	var sb strings.Builder

	// ── Top border with title ──────────────────────────────────
	// "┌─ <title> ─...─┐"
	titleRunes := []rune(title)
	// Reserve space: "┌─ " (3) + " ─...─┐" — need at least "┌──┐" (4 chars).
	maxTitleLen := width - 6 // "┌─ " + " ─┐" = 6 chars framing around title
	if maxTitleLen < 1 {
		maxTitleLen = 1
	}
	if len(titleRunes) > maxTitleLen {
		if maxTitleLen > 1 {
			titleRunes = []rune(string(titleRunes[:maxTitleLen-1]) + "…")
		} else {
			titleRunes = titleRunes[:maxTitleLen]
		}
	}
	titlePart := "─ " + string(titleRunes) + " "
	titlePartLen := 2 + len(titleRunes) + 1 // "─ " + title + " "
	// remaining dashes after title: width - 1(┌) - titlePartLen - 1(┐)
	dashCount := width - 1 - titlePartLen - 1
	if dashCount < 0 {
		dashCount = 0
	}
	topBorder := "┌" + titlePart + strings.Repeat("─", dashCount) + "┐"
	sb.WriteString(popupBorderStyle.Render(topBorder))
	sb.WriteString("\n")

	// ── Image body rows ────────────────────────────────────────
	// Split the image into lines and limit to bodyRows.
	imageLines := strings.Split(image, "\n")
	// Remove trailing empty line if present.
	if len(imageLines) > 0 && imageLines[len(imageLines)-1] == "" {
		imageLines = imageLines[:len(imageLines)-1]
	}

	// Clamp image lines to bodyRows.
	if len(imageLines) > bodyRows {
		imageLines = imageLines[:bodyRows]
	}

	// Vertical centering: pad blank lines above and below.
	blankAbove := 0
	if len(imageLines) < bodyRows {
		blankAbove = (bodyRows - len(imageLines)) / 2
	}

	rendered := 0
	// Emit blank lines above.
	for ; rendered < blankAbove && rendered < bodyRows; rendered++ {
		sb.WriteString(popupBorderStyle.Render("│"))
		sb.WriteString(" " + strings.Repeat(" ", innerWidth) + " ")
		sb.WriteString(popupBorderStyle.Render("│"))
		sb.WriteString("\n")
	}
	// Emit image lines, clamping each to innerWidth so border alignment is preserved.
	clipStyle := lipgloss.NewStyle().MaxWidth(innerWidth)
	for _, line := range imageLines {
		if rendered >= bodyRows {
			break
		}
		sb.WriteString(popupBorderStyle.Render("│"))
		sb.WriteString(" ")
		sb.WriteString(clipStyle.Render(line))
		sb.WriteString(" ")
		sb.WriteString(popupBorderStyle.Render("│"))
		sb.WriteString("\n")
		rendered++
	}
	// Emit blank lines below.
	for ; rendered < bodyRows; rendered++ {
		sb.WriteString(popupBorderStyle.Render("│"))
		sb.WriteString(" " + strings.Repeat(" ", innerWidth) + " ")
		sb.WriteString(popupBorderStyle.Render("│"))
		sb.WriteString("\n")
	}

	// ── Bottom border with hint ────────────────────────────────
	// "└─── h/Esc: close ───...─┘"
	const hintText = "─── h/Esc: close "
	hintLen := len([]rune(hintText))
	bottomDash := width - 1 - hintLen - 1
	if bottomDash < 0 {
		bottomDash = 0
	}
	bottomBorder := "└" + hintText + strings.Repeat("─", bottomDash) + "┘"
	sb.WriteString(popupBorderStyle.Render(bottomBorder))

	return sb.String()
}
