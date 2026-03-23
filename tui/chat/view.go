package chat

import (
	"regexp"
	"strings"
	"time"

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

// RenderPosts converts a sorted []api.Post into a single string for the viewport.
// Groups consecutive posts from the same user (only first shows username+timestamp).
// myUserID is the logged-in user's ID; their messages show "You" in a distinct colour.
// Strips ANSI from message content.
// Shows "(edited)" for posts with EditAt > 0.
func RenderPosts(posts []api.Post, userCache map[string]api.User, myUserID string, width int) string {
	if len(posts) == 0 {
		return ""
	}

	var sb strings.Builder
	lastUserID := ""

	for i, post := range posts {
		// System messages get special rendering.
		if post.Type != "" {
			content := stripANSI(post.Message)
			if content == "" {
				content = "— system message —"
			}
			line := styles.MessageSystem.Render(content)
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(line)
			lastUserID = "" // reset grouping after system message
			continue
		}

		// Normal post.
		isGrouped := post.UserID == lastUserID

		if i > 0 {
			sb.WriteString("\n")
		}

		if !isGrouped {
			// Header line: username + timestamp.
			// Current user gets a distinct "You ▶" label in green; others get their name in purple.
			timestamp := FormatTimestamp(post.CreateAt)
			var usernameStr string
			if myUserID != "" && post.UserID == myUserID {
				usernameStr = styles.MessageMyUsername.Render("You ▶")
			} else {
				username := resolveUsername(post.UserID, userCache)
				usernameStr = styles.MessageUsername.Render(username)
			}
			timestampStr := styles.MessageTimestamp.Render(" " + timestamp)
			sb.WriteString(usernameStr + timestampStr + "\n")
		}

		// Message content: strip ANSI, word-wrap.
		content := stripANSI(post.Message)
		if width > 0 {
			content = lipgloss.NewStyle().Width(width).Render(content)
		}
		msgLine := styles.MessageText.Render(content)
		sb.WriteString(msgLine)

		// Show "(edited)" if applicable.
		if post.EditAt > 0 {
			sb.WriteString(" " + styles.MessageEdited.Render("(edited)"))
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
// Falls back to "unknown" if not in cache — never panics.
func resolveUsername(userID string, userCache map[string]api.User) string {
	if userCache == nil {
		return "unknown"
	}
	u, ok := userCache[userID]
	if !ok {
		return "unknown"
	}
	if u.Nickname != "" {
		return u.Nickname
	}
	if u.Username != "" {
		return u.Username
	}
	return "unknown"
}
