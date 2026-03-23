package chat

import (
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
func RenderPosts(posts []api.Post, userCache map[string]api.User, myUserID string, width int, imageCache map[string]string, useContactName bool) string {
	if len(posts) == 0 {
		return ""
	}

	renderMD := newMarkdownRenderer(width)
	var sb strings.Builder
	lastUserID := ""
	var currentBg lipgloss.Color

	for i, post := range posts {
		// System messages get special rendering (no user background).
		if post.Type != "" {
			content := stripANSI(post.Message)
			if content == "" {
				content = "— system message —"
			}
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(styles.MessageSystem.Render(content))
			lastUserID = ""
			currentBg = lipgloss.Color("")
			continue
		}

		// Normal post.
		isGrouped := post.UserID == lastUserID

		if i > 0 {
			sb.WriteString("\n")
		}

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
				sb.WriteString(usernameStr + timestampStr + pad + "\n")
			} else {
				sb.WriteString(usernameStr + timestampStr + "\n")
			}
		}

		// Render content with glamour (handles markdown, code blocks, images, etc.).
		// Strip raw ANSI from the source first so glamour sees clean markdown.
		rendered := renderMD(stripANSI(post.Message))
		sb.WriteString(rendered)

		// Show "(edited)" if applicable.
		if post.EditAt > 0 {
			sb.WriteString(" " + styles.MessageEdited.Render("(edited)"))
		}

		// Render file attachments (images as terminal art, other files as placeholder).
		for _, fid := range post.FileIds {
			if rendered, ok := imageCache[fid]; ok && rendered != "" {
				sb.WriteString("\n")
				sb.WriteString(rendered)
			}
		}

		lastUserID = post.UserID
	}

	return sb.String()
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
