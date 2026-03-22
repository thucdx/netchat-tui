package tui

import "github.com/thucdx/netchat-tui/api"

// AuthSuccessMsg is emitted by AuthModel when a token has been validated
// successfully. The root app should use Token and UserID to persist credentials
// and transition to the main chat view.
type AuthSuccessMsg struct {
	Token  string
	UserID string
}

// ErrorMsg carries an asynchronous error back to the Bubbletea update loop.
type ErrorMsg struct {
	Err error
}

// ChannelSelectedMsg is emitted by the sidebar when the user opens a channel.
type ChannelSelectedMsg struct{ ChannelID string }

// PostsLoadedMsg is emitted after a channel's post history has been fetched.
type PostsLoadedMsg struct {
	ChannelID string
	Posts     api.PostList
}

// NewPostMsg carries a single new post received via WebSocket.
type NewPostMsg struct{ Post api.Post }

// SendMessageMsg is emitted by the input box to request sending a message.
type SendMessageMsg struct {
	ChannelID string
	Text      string
}

// WSEventMsg wraps a raw WebSocket event for the Bubbletea update loop.
type WSEventMsg struct{ Event api.WSEvent }
