// Package messages holds shared Bubbletea message types that are emitted by
// sub-models and consumed by the root AppModel.  Types are kept here (rather
// than in the tui package) to avoid import cycles:
//
//	tui/app.go  →  tui/sidebar  →  internal/messages  (ok)
//	tui/app.go  →  tui/input    →  internal/messages  (ok)
package messages

import "github.com/thucdx/netchat-tui/api"

// ChannelSelectedMsg is emitted by the sidebar when the user opens a channel.
// FocusOnOpen signals whether the app should shift keyboard focus to the chat pane.
// Enter sets it true; preview (p) sets it false (sidebar retains focus).
type ChannelSelectedMsg struct {
	ChannelID   string
	FocusOnOpen bool
}

// NewPostMsg carries a single new post received via WebSocket.
type NewPostMsg struct{ Post api.Post }

// SendMessageMsg is emitted by the input box to request sending a message.
type SendMessageMsg struct {
	ChannelID string
	Text      string
}

// LoadMorePostsMsg is emitted by the chat model when the user scrolls to the
// top and pagination should load the next page of older posts.
type LoadMorePostsMsg struct {
	ChannelID string
	Page      int
}

// WSEventMsg wraps a raw WebSocket event for the Bubbletea update loop.
type WSEventMsg struct{ Event api.WSEvent }

// ImagesReadyMsg carries rendered terminal-image strings keyed by file ID.
// It is dispatched when background image downloads complete.
type ImagesReadyMsg struct {
	// Images maps Mattermost file IDs to rendered inline image strings.
	// For Kitty terminals these are U+10EEEE placeholder grids; otherwise half-block art.
	Images map[string]string
	// FileInfos maps Mattermost file IDs to their metadata.
	FileInfos map[string]api.FileInfo
	// KittyIDs holds Kitty Graphics Protocol image IDs that were uploaded to the
	// terminal GPU.  The app tracks these so it can delete them on channel switch.
	KittyIDs []uint32
}

// TriggerSearchMsg is emitted by the sidebar when the search query reaches
// ≥3 characters and an API search should be fired.
type TriggerSearchMsg struct{ Query string }

// SearchResultsMsg carries API search results back to the sidebar.
type SearchResultsMsg struct {
	Query    string
	Users    []api.User
	Channels []api.Channel
}

// CreateDirectChannelMsg is emitted by the sidebar when the user picks a
// user from search results to start a new DM.
type CreateDirectChannelMsg struct{ UserID string }

// JoinChannelMsg is emitted by the sidebar when the user confirms joining
// a public channel found via search.
type JoinChannelMsg struct{ ChannelID string }

// WSDisconnectedMsg is sent when the WebSocket connection drops unexpectedly,
// signalling the app to begin a reconnect attempt.
type WSDisconnectedMsg struct{}

// WSReconnectedMsg is sent after a successful WebSocket reconnect.
type WSReconnectedMsg struct{}

// OpenFileMsg is emitted when the user opens a file attachment from the chat pane.
// The handler downloads the file (if not cached) and opens it with the OS default app.
type OpenFileMsg struct {
	File api.FileInfo
}

// YankMsg is emitted by the chat model when the user yanks (copies) selected
// messages. The app handler writes Text to the OS clipboard.
type YankMsg struct {
	Text string
}

// CustomEmojiImagesReadyMsg carries rendered terminal-art strings for custom
// emoji, keyed by emoji name.  Dispatched when background emoji downloads complete.
type CustomEmojiImagesReadyMsg struct {
	Rendered map[string]string // emoji name → rendered half-block art
}
