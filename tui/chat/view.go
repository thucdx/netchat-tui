package chat

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// ansiEscape matches ANSI terminal escape sequences.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[mABCDEFGHJKSTfhil]`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// newMarkdownRenderer returns a function that renders a markdown string to ANSI
// using glamour's dark theme, word-wrapped to the given width.
// Falls back to plain text on error.
func newMarkdownRenderer(width int) func(string) string {
	opts := []glamour.TermRendererOption{
		glamour.WithStylePath("dark"),
		glamour.WithEmoji(),
	}
	if width > 2 {
		opts = append(opts, glamour.WithWordWrap(width-2))
	}
	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return func(s string) string { return s }
	}
	return func(s string) string {
		out, err := r.Render(s)
		if err != nil {
			return s
		}
		// Trim the blank lines glamour adds around the document.
		return strings.Trim(out, "\n")
	}
}

// RenderPosts converts a sorted []api.Post into a single string for the viewport.
// Groups consecutive posts from the same user (only first shows username+timestamp).
// myUserID is the logged-in user's ID; their messages show "You" in a distinct colour.
// Each unique user gets a rotating colour on the header bar for easy scanning.
// Message content is rendered with glamour (markdown + syntax highlighting).
// Shows "(edited)" for posts with EditAt > 0.
// cursor is the index of the highlighted post (-1 = none).
// lastViewedAt is the Unix-ms timestamp of the channel member's last read; posts
// with CreateAt > lastViewedAt after the first such post get an unread divider.
// visualStart and visualEnd define the inclusive range of the visual selection (-1,-1 = none).
func RenderPosts(posts []api.Post, userCache map[string]api.User, myUserID string, width int, imageCache map[string]string, fileInfoCache map[string]api.FileInfo, useContactName bool, cursor int, lastViewedAt int64, visualStart int, visualEnd int) string {
	if len(posts) == 0 {
		return ""
	}

	renderMD := newMarkdownRenderer(width)
	var sb strings.Builder
	lastUserID := ""
	var currentBg lipgloss.Color
	dividerInserted := false
	needLeadingNewline := false // tracks whether to emit a "\n" separator before the next block

	for i, post := range posts {
		// Insert the unread divider before the first unread post.
		if lastViewedAt > 0 && !dividerInserted && post.CreateAt > lastViewedAt {
			dividerInserted = true
			divider := styles.UnreadDivider.Render("──── unread ────")
			centred := lipgloss.PlaceHorizontal(width, lipgloss.Center, divider)
			if needLeadingNewline || i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(centred)
			// After the divider, the next post should be separated by a newline.
			needLeadingNewline = true
		}

		// System messages get special rendering (no user background).
		if post.Type != "" {
			content := stripANSI(post.Message)
			if content == "" {
				content = "— system message —"
			}
			if needLeadingNewline || i > 0 {
				sb.WriteString("\n")
			}
			block := styles.MessageSystem.Render(content)
			if i == cursor {
				block = styles.CursorBorder.Render(block)
			} else if visualStart >= 0 && i >= visualStart && i <= visualEnd {
				block = styles.VisualSelectionBorder.Render(block)
			}
			sb.WriteString(block)
			needLeadingNewline = true
			lastUserID = ""
			currentBg = lipgloss.Color("")
			continue
		}

		// Normal post — build the entire block in a local builder first so we
		// can optionally wrap it with the cursor border in one shot.
		isGrouped := post.UserID == lastUserID

		if needLeadingNewline || i > 0 {
			sb.WriteString("\n")
		}
		needLeadingNewline = true

		var block strings.Builder

		if !isGrouped {
			// Determine per-user palette entry.
			var usernameFg lipgloss.Color
			var usernameLabel string
			if myUserID != "" && post.UserID == myUserID {
				currentBg = styles.MessageSelfBg
				usernameFg = lipgloss.Color("#10B981")
				usernameLabel = "You ▶"
			} else {
				entry := styles.MessageUserPalette[styles.UserColorIndex(post.UserID)]
				currentBg = entry.Bg
				usernameFg = entry.Fg
				usernameLabel = resolveUsername(post.UserID, userCache, useContactName)
			}

			// Header: username + timestamp with user-colour background spanning full width.
			timestamp := FormatTimestamp(post.CreateAt)
			usernameStr := lipgloss.NewStyle().Foreground(usernameFg).Bold(true).Background(currentBg).Render(usernameLabel)
			timestampStr := styles.MessageTimestamp.Background(currentBg).Render(" " + timestamp)
			used := lipgloss.Width(usernameStr) + lipgloss.Width(timestampStr)
			if width > used {
				pad := lipgloss.NewStyle().Background(currentBg).Render(strings.Repeat(" ", width-used))
				block.WriteString(usernameStr + timestampStr + pad + "\n")
			} else {
				block.WriteString(usernameStr + timestampStr + "\n")
			}
		}

		// Render content with glamour (handles markdown, code blocks, images, etc.).
		// Strip raw ANSI from the source first so glamour sees clean markdown.
		rendered := renderMD(stripANSI(post.Message))
		block.WriteString(rendered)

		// Show "(edited)" if applicable.
		if post.EditAt > 0 {
			block.WriteString(" " + styles.MessageEdited.Render("(edited)"))
		}

		// Render file attachments: images as terminal art, other files as metadata lines.
		for _, fid := range post.FileIds {
			if imgRendered, ok := imageCache[fid]; ok && imgRendered != "" {
				// Image rendered as terminal art — already handled by imageCache.
				block.WriteString("\n")
				block.WriteString(imgRendered)
			} else if fi, ok := fileInfoCache[fid]; ok {
				// Non-image (or image that failed to render) — show metadata line.
				block.WriteString("\n")
				block.WriteString("📎 " + stripANSI(fi.Name) + "  (" + formatFileSize(fi.Size) + ")")
			}
			// If neither cache has the file, skip silently (metadata not yet loaded).
		}

		// Apply cursor border to the whole block when this post is selected.
		// Visual selection border is applied for posts in [visualStart, visualEnd],
		// but cursor border takes priority when i == cursor.
		blockStr := block.String()
		if i == cursor {
			blockStr = styles.CursorBorder.Render(blockStr)
		} else if visualStart >= 0 && i >= visualStart && i <= visualEnd {
			blockStr = styles.VisualSelectionBorder.Render(blockStr)
		}
		sb.WriteString(blockStr)

		lastUserID = post.UserID
	}

	return sb.String()
}

// formatFileSize returns a human-readable file size string.
func formatFileSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1048576:
		return fmt.Sprintf("%d KB", bytes/1024)
	default:
		return fmt.Sprintf("%d MB", bytes/1048576)
	}
}

// FormatTimestamp converts a Unix millisecond timestamp to a human-readable string.
// Today → "HH:MM", Yesterday → "Yesterday HH:MM", older → "02/01 HH:MM"
func FormatTimestamp(ms int64) string {
	t := time.UnixMilli(ms).Local()
	now := time.Now()

	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayMidnight := todayMidnight.Add(-24 * time.Hour)
	tMidnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	switch {
	case !tMidnight.Before(todayMidnight):
		return t.Format("15:04")
	case !tMidnight.Before(yesterdayMidnight):
		return "Yesterday " + t.Format("15:04")
	default:
		return t.Format("02/01 15:04")
	}
}

// resolveUsername returns the display name for a userID from cache.
// When useContact is true, prefers FirstName+LastName; falls back to Username.
// When useContact is false, returns Username directly.
// Falls back to "unknown" if not in cache — never panics.
func resolveUsername(userID string, userCache map[string]api.User, useContact bool) string {
	if userCache == nil {
		return "unknown"
	}
	u, ok := userCache[userID]
	if !ok {
		return "unknown"
	}
	if useContact {
		if full := strings.TrimSpace(u.FirstName + " " + u.LastName); full != "" {
			return full
		}
	}
	if u.Username != "" {
		return u.Username
	}
	return "unknown"
}
