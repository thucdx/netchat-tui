// Package messages holds shared Bubbletea message types that are emitted by
// sub-models and consumed by the root AppModel.  Types are kept here (rather
// than in the tui package) to avoid import cycles:
//
//	tui/app.go  →  tui/sidebar  →  internal/messages  (ok)
//	tui/app.go  →  tui/input    →  internal/messages  (ok)
package messages

import "github.com/thucdx/netchat-tui/api"

// ChannelSelectedMsg is emitted by the sidebar when the user opens a channel.
type ChannelSelectedMsg struct{ ChannelID string }

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
	// Images maps Mattermost file IDs to rendered ANSI image strings.
	Images map[string]string
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
