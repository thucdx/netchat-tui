package tui

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
