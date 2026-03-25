package chat

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/tui/styles"
)

// ansiEscape matches ANSI terminal escape sequences.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[mABCDEFGHJKSTfhil]`)

// emojiShortcode matches :name: patterns (custom emoji shortcodes).
var emojiShortcode = regexp.MustCompile(`:([a-z0-9_+\-]+):`)

// substituteCustomEmoji replaces :name: patterns in s with rendered terminal art
// from cache when available.  Patterns already converted by glamour (standard
// emoji) will not appear as :name: in the input, so only custom ones remain.
//
// glamour's reflow library injects ANSI reset+colour sequences at word
// boundaries (including underscores), which splits a name like
// ":company_logo:" into ":company_\x1b[...]logo:" in the ANSI output.
// To work around this, the function strips ANSI from s first, applies the
// regex on the clean text, and returns the substituted plain text when any
// replacement was made.  The surrounding glamour colour is lost for those
// messages, but the emoji art itself is already colourful.
func substituteCustomEmoji(s string, cache map[string]string) string {
	if len(cache) == 0 {
		return s
	}
	plain := stripANSI(s)
	result := emojiShortcode.ReplaceAllStringFunc(plain, func(match string) string {
		name := match[1 : len(match)-1]
		if art, ok := cache[name]; ok && art != "" {
			return art
		}
		return match
	})
	if result == plain {
		return s // nothing substituted — preserve original ANSI styling
	}
	return result
}

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// cursorBgSeq is the ANSI true-color background escape sequence for
// styles.ColorSelected.  It is computed once at package init so it can be
// injected into glamour-rendered content to keep the selected-message
// background visible across the ANSI reset codes that glamour emits.
var cursorBgSeq = func() string {
	hex := strings.TrimPrefix(string(styles.ColorSelected), "#")
	if len(hex) != 6 {
		return ""
	}
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}()

// injectBgAfterResets prepends bg and re-applies it after every ANSI
// full-reset sequence (\x1b[0m / \x1b[m) in s.  Glamour emits these resets
// aggressively, which clears any outer background we set; this function
// ensures the background colour is reinstated after each reset so that it
// fills the entire rendered block uniformly.
func injectBgAfterResets(s, bg string) string {
	if bg == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bg)
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bg)
	return bg + s
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
// Each unique user gets a rotating colour on the left border for easy scanning.
// Message content is rendered with glamour (markdown + syntax highlighting).
// Shows "(edited)" for posts with EditAt > 0.
// cursor is the index of the highlighted post (-1 = none).
// lastViewedAt is the Unix-ms timestamp of the channel member's last read; posts
// with CreateAt > lastViewedAt after the first such post get an unread divider.
// visualStart and visualEnd define the inclusive range of the visual selection (-1,-1 = none).
// collapseThreshold is the number of rendered lines above which a message is
// auto-collapsed. collapsePreviewLines is how many lines are shown in collapsed state.
const collapseThreshold = 10
const collapsePreviewLines = 5

func RenderPosts(posts []api.Post, userCache map[string]api.User, myUserID string, width int, imageCache map[string]string, fileInfoCache map[string]api.FileInfo, useContactName bool, cursor int, lastViewedAt int64, visualStart int, visualEnd int, expandedPosts map[string]bool, customEmojiCache map[string]string) string {
	if len(posts) == 0 {
		return ""
	}

	renderMD := newMarkdownRenderer(width)
	var sb strings.Builder
	lastUserID := ""
	var currentAccent lipgloss.Color
	dividerInserted := false
	needLeadingNewline := false // tracks whether to emit a "\n" separator before the next block

	// Build an ID→Post index so reply quotes can look up the root post cheaply.
	postsById := make(map[string]*api.Post, len(posts))
	for i := range posts {
		postsById[posts[i].ID] = &posts[i]
	}

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
				block = styles.CursorBorder.Background(styles.ColorSelected).Width(width - 2).Render(
					injectBgAfterResets(block, cursorBgSeq))
			} else if visualStart >= 0 && i >= visualStart && i <= visualEnd {
				block = styles.VisualSelectionBorder.Render(block)
			} else {
				sysBorder := lipgloss.NewStyle().
					BorderLeft(true).
					BorderStyle(lipgloss.NormalBorder()).
					BorderForeground(styles.ColorBorderHi).
					BorderBackground(styles.ColorBg).
					PaddingLeft(1)
				block = sysBorder.Render(block)
			}
			sb.WriteString(block)
			needLeadingNewline = true
			lastUserID = ""
			currentAccent = lipgloss.Color("")
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
			// Determine per-user accent color.
			var usernameFg lipgloss.Color
			var usernameLabel string
			if myUserID != "" && post.UserID == myUserID {
				currentAccent = lipgloss.Color("#3fb950")
				usernameFg = currentAccent
				usernameLabel = "You ▶"
			} else {
				currentAccent = styles.MessageUserPalette[styles.UserColorIndex(post.UserID)]
				usernameFg = currentAccent
				usernameLabel = resolveUsername(post.UserID, userCache, useContactName)
			}

			// Header: username + timestamp (plain text, no background).
			timestamp := FormatTimestamp(post.CreateAt)
			usernameStr := lipgloss.NewStyle().Foreground(usernameFg).Bold(true).Render(usernameLabel)
			timestampStr := styles.MessageTimestamp.Render("  " + timestamp)
			block.WriteString(usernameStr + timestampStr + "\n")
		}

		// Reply quote: if this post replies to another, show a compact preview of the root.
		if post.RootID != "" {
			if root, ok := postsById[post.RootID]; ok {
				authorName := resolveUsername(root.UserID, userCache, useContactName)
				preview := strings.ReplaceAll(stripANSI(root.Message), "\n", " ")
				runes := []rune(preview)
				if len(runes) > 60 {
					preview = string(runes[:57]) + "…"
				}
				quoteStyle := lipgloss.NewStyle().
					Foreground(styles.ColorFgMuted).
					BorderLeft(true).
					BorderStyle(lipgloss.NormalBorder()).
					BorderForeground(styles.ColorFgDimmer).
					PaddingLeft(1).
					Italic(true)
				block.WriteString(quoteStyle.Render("↩ " + authorName + ": " + preview))
				block.WriteString("\n")
			}
		}

		// Render content with glamour (handles markdown, code blocks, images, etc.).
		// Strip raw ANSI from the source first so glamour sees clean markdown.
		rendered := renderMD(stripANSI(post.Message))

		// Substitute custom emoji :name: patterns with rendered terminal art.
		rendered = substituteCustomEmoji(rendered, customEmojiCache)

		// Auto-collapse messages that render beyond the threshold, unless the
		// user has explicitly expanded this post.
		if !expandedPosts[post.ID] {
			lines := strings.Split(rendered, "\n")
			if len(lines) > collapseThreshold {
				rendered = strings.Join(lines[:collapsePreviewLines], "\n") +
					"\n" + styles.SubtleStyle.Render("  … z to expand")
			}
		}

		block.WriteString(rendered)

		// Show "(edited)" if applicable.
		if post.EditAt > 0 {
			block.WriteString(" " + styles.MessageEdited.Render("(edited)"))
		}

		// Render file attachments: images as terminal art, other files as metadata lines.
		for _, fid := range post.FileIds {
			if imgRendered, ok := imageCache[fid]; ok && imgRendered != "" {
				// Image rendered as terminal art — show badge on the line above the art.
				block.WriteString("\n")
				if fi, ok := fileInfoCache[fid]; ok {
					block.WriteString(networkZoneBadge(fi) + "\n")
				}
				block.WriteString(imgRendered)
			} else if fi, ok := fileInfoCache[fid]; ok {
				// Non-image (or image that failed to render) — show metadata line with badge.
				block.WriteString("\n")
				block.WriteString("📎 " + stripANSI(fi.Name) + "  (" + formatFileSize(fi.Size) + ")  " + networkZoneBadge(fi))
			}
			// If neither cache has the file, skip silently (metadata not yet loaded).
		}

		// Reactions row: group by emoji, sorted by count desc.
		if len(post.Metadata.Reactions) > 0 {
			block.WriteString("\n")
			block.WriteString(renderReactions(post.Metadata.Reactions, customEmojiCache))
		}

		// Apply cursor border to the whole block when this post is selected.
		// Visual selection border is applied for posts in [visualStart, visualEnd],
		// but cursor border takes priority when i == cursor.
		// Default: per-user accent left border.
		blockStr := block.String()
		if i == cursor {
			blockStr = styles.CursorBorder.Background(styles.ColorSelected).Width(width - 2).Render(
				injectBgAfterResets(blockStr, cursorBgSeq))
		} else if visualStart >= 0 && i >= visualStart && i <= visualEnd {
			blockStr = styles.VisualSelectionBorder.Render(blockStr)
		} else {
			userBorder := lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(currentAccent).
				BorderBackground(styles.ColorBg).
				PaddingLeft(1)
			blockStr = userBorder.Render(blockStr)
		}
		sb.WriteString(blockStr)

		lastUserID = post.UserID
	}

	return sb.String()
}

// renderReactions formats a slice of reactions as a compact grouped row.
// Reactions are grouped by emoji, sorted by count descending, then name ascending.
// Example output: "👍 3  ❤️ 2  😂 1"
func renderReactions(reactions []api.Reaction, customEmojiCache map[string]string) string {
	counts := make(map[string]int, len(reactions))
	for _, r := range reactions {
		counts[r.EmojiName]++
	}

	type entry struct {
		name  string
		count int
	}
	entries := make([]entry, 0, len(counts))
	for name, n := range counts {
		entries = append(entries, entry{name, n})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].name < entries[j].name
	})

	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		ch := emojiChar(e.name)
		// If not a known standard emoji, try the custom emoji art cache.
		if ch == ":"+e.name+":" {
			if art, ok := customEmojiCache[e.name]; ok && art != "" {
				ch = art
			}
		}
		if e.count > 1 {
			parts = append(parts, ch+" "+fmt.Sprintf("%d", e.count))
		} else {
			parts = append(parts, ch)
		}
	}
	return styles.SubtleStyle.Render(strings.Join(parts, "  "))
}

// emojiChar converts a Mattermost/Slack emoji name to its Unicode character.
// Covers the most common reactions; falls back to ":name:" for unknown names.
func emojiChar(name string) string {
	m := map[string]string{
		"thumbsup": "👍", "+1": "👍",
		"thumbsdown": "👎", "-1": "👎",
		"heart": "❤️", "red_heart": "❤️",
		"laugh": "😄", "joy": "😂",
		"tada": "🎉", "hooray": "🎉", "party_popper": "🎉",
		"confused": "😕",
		"rocket":   "🚀",
		"eyes":     "👀",
		"sob":      "😭",
		"smile":    "😊", "slightly_smiling_face": "🙂",
		"fire":            "🔥",
		"100":             "💯",
		"white_check_mark": "✅",
		"x":               "❌",
		"pray":            "🙏",
		"clap":            "👏",
		"wave":            "👋",
		"ok_hand":         "👌",
		"raised_hands":    "🙌",
		"thinking_face":   "🤔",
		"face_palm":       "🤦",
		"shrug":           "🤷",
		"wink":            "😉",
		"sunglasses":      "😎",
		"star":            "⭐",
		"warning":         "⚠️",
		"zap":             "⚡",
		"bug":             "🐛",
		"memo":            "📝",
		"question":        "❓",
		"exclamation":     "❗",
		"construction":    "🚧",
		"speech_balloon":  "💬",
		"muscle":          "💪",
		"brain":           "🧠",
		"bulb":            "💡",
		"link":            "🔗",
		"lock":            "🔒",
		"key":             "🔑",
		"arrow_up":        "⬆️",
		"arrow_down":      "⬇️",
		"heavy_plus_sign": "➕",
		"heavy_minus_sign": "➖",
		"checkered_flag":  "🏁",
	}
	if ch, ok := m[name]; ok {
		return ch
	}
	return ":" + name + ":"
}

// networkZoneBadge returns a small coloured badge indicating whether a file is
// stored in the internal-network bucket ("chat") or the public bucket ("chat-public").
// Yellow [local] = internal only; dimmed [public] = accessible from anywhere.
func networkZoneBadge(fi api.FileInfo) string {
	if fi.IsPublic() {
		return styles.AttachmentPublicBadge.Render("[public]")
	}
	return styles.AttachmentLocalBadge.Render("[local]")
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
