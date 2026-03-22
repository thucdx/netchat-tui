package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TestAuthModelInitialState verifies that a freshly constructed AuthModel is in
// stateInput, has a focused textinput, and uses EchoPassword mode.
func TestAuthModelInitialState(t *testing.T) {
	m := NewAuthModel("https://netchat.viettel.vn")

	if m.state != stateInput {
		t.Errorf("expected state stateInput (%d), got %d", stateInput, m.state)
	}

	if !m.input.Focused() {
		t.Error("expected textinput to be focused, but it is not")
	}

	if m.input.EchoMode != textinput.EchoPassword {
		t.Errorf("expected EchoMode EchoPassword (%d), got %d", textinput.EchoPassword, m.input.EchoMode)
	}
}

// TestAuthModelEnterWithEmptyInput verifies that pressing Enter with an empty
// input transitions to stateError (with an error message) and returns no async cmd.
func TestAuthModelEnterWithEmptyInput(t *testing.T) {
	m := NewAuthModel("https://netchat.viettel.vn")

	// Ensure input is empty (it is by default, but be explicit).
	m.input.SetValue("")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.Update(enterMsg)

	am, ok := result.(AuthModel)
	if !ok {
		t.Fatalf("Update returned unexpected model type %T", result)
	}

	// The implementation sets stateError and an error message when the token is
	// empty, but returns nil cmd — so there should be no command.
	if cmd != nil {
		t.Error("expected nil cmd when pressing Enter with empty input, got non-nil cmd")
	}

	// State should move to stateError (token cannot be empty).
	if am.state != stateError {
		t.Errorf("expected state stateError (%d), got %d", stateError, am.state)
	}

	if am.errMsg == "" {
		t.Error("expected a non-empty error message after pressing Enter with empty input")
	}
}

// TestAuthModelTransitionsToValidating verifies that pressing Enter with a
// non-empty token transitions the model to stateValidating and returns a non-nil
// cmd (the batch of spinner.Tick + validateToken). The cmd is NOT executed so
// no real HTTP call is made.
func TestAuthModelTransitionsToValidating(t *testing.T) {
	m := NewAuthModel("https://netchat.viettel.vn")
	m.input.SetValue("some-token-value")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.Update(enterMsg)

	am, ok := result.(AuthModel)
	if !ok {
		t.Fatalf("Update returned unexpected model type %T", result)
	}

	if am.state != stateValidating {
		t.Errorf("expected state stateValidating (%d), got %d", stateValidating, am.state)
	}

	if cmd == nil {
		t.Error("expected a non-nil cmd when a token is submitted, got nil")
	}
}

// TestAuthModelClearsInputOnSuccess verifies that receiving AuthSuccessMsg
// causes the textinput value to be cleared (for security).
func TestAuthModelClearsInputOnSuccess(t *testing.T) {
	m := NewAuthModel("https://netchat.viettel.vn")
	m.input.SetValue("my-secret-token")
	m.state = stateValidating

	successMsg := AuthSuccessMsg{Token: "my-secret-token", UserID: "user-123"}
	result, _ := m.Update(successMsg)

	am, ok := result.(AuthModel)
	if !ok {
		t.Fatalf("Update returned unexpected model type %T", result)
	}

	if am.input.Value() != "" {
		t.Errorf("expected textinput value to be empty after AuthSuccessMsg, got %q", am.input.Value())
	}
}

// TestAuthModelResetsOnError verifies that receiving ErrorMsg transitions the
// model to stateError, stores the error message, and clears the input.
func TestAuthModelResetsOnError(t *testing.T) {
	m := NewAuthModel("https://netchat.viettel.vn")
	m.input.SetValue("bad-token")
	m.state = stateValidating

	errMsg := ErrorMsg{Err: errors.New("authentication failed: token rejected (401)")}
	result, cmd := m.Update(errMsg)

	am, ok := result.(AuthModel)
	if !ok {
		t.Fatalf("Update returned unexpected model type %T", result)
	}

	if am.state != stateError {
		t.Errorf("expected state stateError (%d), got %d", stateError, am.state)
	}

	if am.errMsg != errMsg.Err.Error() {
		t.Errorf("expected errMsg %q, got %q", errMsg.Err.Error(), am.errMsg)
	}

	if am.input.Value() != "" {
		t.Errorf("expected textinput to be cleared after ErrorMsg, got %q", am.input.Value())
	}

	if cmd != nil {
		t.Error("expected nil cmd after ErrorMsg, got non-nil cmd")
	}
}
