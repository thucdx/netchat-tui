package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thucdx/netchat-tui/api"
)

// authState enumerates the stages of the auth screen.
type authState int

const (
	stateInput      authState = iota // waiting for the user to paste a token
	stateValidating                  // token submitted; HTTP call in flight
	stateError                       // previous attempt failed; user can retry
)

// Styles ─────────────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)

// AuthModel ───────────────────────────────────────────────────────────────────

// AuthModel is the Bubbletea model for the authentication screen.
// It is shown when no stored credentials exist.
type AuthModel struct {
	state   authState
	input   textinput.Model
	spinner spinner.Model
	errMsg  string
	baseURL string
}

// NewAuthModel constructs an AuthModel ready for use.
func NewAuthModel(baseURL string) AuthModel {
	ti := textinput.New()
	ti.Placeholder = "Paste MMAUTHTOKEN here…"
	ti.EchoMode = textinput.EchoPassword // token never echoed
	ti.EchoCharacter = '•'
	ti.CharLimit = 512
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return AuthModel{
		state:   stateInput,
		input:   ti,
		spinner: sp,
		baseURL: baseURL,
	}
}

// Init implements tea.Model.
func (m AuthModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.state != stateInput && m.state != stateError {
				return m, nil
			}
			token := strings.TrimSpace(m.input.Value())
			if token == "" {
				m.errMsg = "Token cannot be empty."
				m.state = stateError
				return m, nil
			}
			m.state = stateValidating
			m.errMsg = ""
			return m, tea.Batch(
				m.spinner.Tick,
				validateToken(m.baseURL, token),
			)
		}

	case spinner.TickMsg:
		if m.state == stateValidating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case AuthSuccessMsg:
		// Clear the token from the widget for security before returning the
		// success message to the parent.
		m.input.SetValue("")
		return m, func() tea.Msg { return msg }

	case ErrorMsg:
		m.state = stateError
		m.errMsg = msg.Err.Error()
		// Reset input so the user can re-paste without clearing manually.
		m.input.SetValue("")
		return m, nil
	}

	// Forward key events to the textinput only while accepting input.
	if m.state == stateInput || m.state == stateError {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m AuthModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("netchat-tui"))
	b.WriteString("\n\n")

	b.WriteString(subtleStyle.Render("Open https://netchat.viettel.vn in your browser, log in via SSO,"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("then open DevTools → Application → Cookies → copy MMAUTHTOKEN value."))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Token: "))
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	switch m.state {
	case stateValidating:
		b.WriteString(m.spinner.View())
		b.WriteString(" Validating token…")
	case stateError:
		b.WriteString(errorStyle.Render("Error: " + m.errMsg))
		b.WriteString("\n")
		b.WriteString(subtleStyle.Render("Press Enter to retry."))
	}

	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("Press Ctrl+C to quit"))

	return b.String()
}

// validateToken returns a tea.Cmd that calls GET /api/v4/users/me with the
// supplied token and emits either AuthSuccessMsg or ErrorMsg.
func validateToken(baseURL, token string) tea.Cmd {
	return func() tea.Msg {
		client, err := api.NewClient(baseURL, token, "")
		if err != nil {
			// Should not happen given a hard-coded https URL.
			return ErrorMsg{Err: fmt.Errorf("failed to create API client: %w", err)}
		}

		body, err := client.Get("/api/v4/users/me")
		if err != nil {
			// Distinguish auth failures from network errors without leaking the token.
			if isAuthError(err) {
				return ErrorMsg{Err: errors.New("authentication failed: token rejected (401)")}
			}
			return ErrorMsg{Err: fmt.Errorf("network error while validating token: %w", err)}
		}

		var user struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &user); err != nil {
			return ErrorMsg{Err: fmt.Errorf("unexpected response from server: %w", err)}
		}
		if user.ID == "" {
			return ErrorMsg{Err: errors.New("server returned a user record without an id field")}
		}

		return AuthSuccessMsg{Token: token, UserID: user.ID}
	}
}

// isAuthError reports whether err indicates an HTTP 401 response.
func isAuthError(err error) bool {
	return errors.Is(err, api.ErrUnauthorized)
}
