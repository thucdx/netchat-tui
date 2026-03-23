package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thucdx/netchat-tui/internal/keymap"
	"github.com/thucdx/netchat-tui/internal/messages"
)

func newTestModel(channelID string) Model {
	m := NewModel(keymap.DefaultKeyMap())
	m.SetChannelID(channelID)
	m.SetSize(80, 3)
	m.SetFocused(true)
	return m
}

// sendKey sends a key message through the model and returns (updated model, emitted cmd).
func sendKey(m Model, keyType tea.KeyType) (Model, tea.Cmd) {
	result, cmd := m.Update(tea.KeyMsg{Type: keyType})
	return result.(Model), cmd
}

// typeText types each rune into the model.
func typeText(m Model, text string) Model {
	for _, ch := range text {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = result.(Model)
	}
	return m
}

// collectMsg runs a cmd and returns the produced message (nil if cmd is nil).
func collectMsg(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// --- Tests ---

func TestEnterSendsAndClearsTextarea(t *testing.T) {
	m := newTestModel("channel-abc")
	m = typeText(m, "hello world")

	updated, cmd := sendKey(m, tea.KeyEnter)

	// cmd must emit a SendMessageMsg
	msg := collectMsg(cmd)
	sendMsg, ok := msg.(messages.SendMessageMsg)
	if !ok {
		t.Fatalf("expected SendMessageMsg, got %T", msg)
	}
	if sendMsg.ChannelID != "channel-abc" {
		t.Errorf("ChannelID = %q, want %q", sendMsg.ChannelID, "channel-abc")
	}
	if sendMsg.Text != "hello world" {
		t.Errorf("Text = %q, want %q", sendMsg.Text, "hello world")
	}

	// textarea must be cleared after send
	if updated.textarea.Value() != "" {
		t.Errorf("textarea not cleared after send, got %q", updated.textarea.Value())
	}

	// sending flag must be set to true
	if !updated.sending {
		t.Error("sending flag should be true after send")
	}
}

func TestEnterWhileSendingIsDropped(t *testing.T) {
	m := newTestModel("channel-abc")
	m = typeText(m, "first message")
	m.sending = true // simulate in-flight

	_, cmd := sendKey(m, tea.KeyEnter)
	if cmd != nil {
		msg := collectMsg(cmd)
		if _, ok := msg.(messages.SendMessageMsg); ok {
			t.Error("should not emit SendMessageMsg while sending is in-flight")
		}
	}
}

func TestEnterWithEmptyTextareaDoesNotSend(t *testing.T) {
	m := newTestModel("channel-abc")
	// textarea is empty by default
	_, cmd := sendKey(m, tea.KeyEnter)

	if cmd != nil {
		msg := collectMsg(cmd)
		if _, ok := msg.(messages.SendMessageMsg); ok {
			t.Error("should not emit SendMessageMsg for empty message")
		}
	}
}

func TestEnterWithWhitespaceOnlyDoesNotSend(t *testing.T) {
	m := newTestModel("channel-abc")
	m = typeText(m, "   ")

	_, cmd := sendKey(m, tea.KeyEnter)
	if cmd != nil {
		msg := collectMsg(cmd)
		if _, ok := msg.(messages.SendMessageMsg); ok {
			t.Error("should not send whitespace-only message")
		}
	}
}

func TestEnterWithNoChannelIDDoesNotSend(t *testing.T) {
	m := newTestModel("") // no channel
	m = typeText(m, "hello")

	_, cmd := sendKey(m, tea.KeyEnter)
	if cmd != nil {
		msg := collectMsg(cmd)
		if _, ok := msg.(messages.SendMessageMsg); ok {
			t.Error("should not send when channelID is empty")
		}
	}
}

func TestShiftEnterInsertsNewline(t *testing.T) {
	m := newTestModel("channel-abc")
	m = typeText(m, "line1")

	// Shift+Enter should NOT send; it should insert a newline in the textarea.
	_, cmd := sendKey(m, tea.KeyEnter) // plain Enter sends
	_ = cmd

	// Re-create fresh model and test shift+enter
	m2 := newTestModel("channel-abc")
	m2 = typeText(m2, "line1")

	// Shift+Enter via the textarea keymap (we simulate it as Alt+Enter which
	// textarea treats as InsertNewline since we bound InsertNewline to shift+enter).
	// In the Update path: shift+enter does NOT match msg.Type == tea.KeyEnter && !msg.Alt,
	// so it falls through to the textarea handler.
	result, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m2 = result.(Model)

	// No SendMessageMsg should be emitted.
	if cmd2 != nil {
		msg := collectMsg(cmd2)
		if _, ok := msg.(messages.SendMessageMsg); ok {
			t.Error("Alt+Enter (Shift+Enter equivalent) should NOT send a message")
		}
	}
	// textarea value should still contain content (no clear happened)
	if m2.textarea.Value() == "" {
		t.Error("textarea should not be cleared on Shift+Enter")
	}
}

func TestSetSendingDisablesAndReenables(t *testing.T) {
	m := newTestModel("channel-abc")
	m.SetSending(true)
	if !m.sending {
		t.Error("SetSending(true) should set sending=true")
	}
	m.SetSending(false)
	if m.sending {
		t.Error("SetSending(false) should set sending=false")
	}
}

func TestCharLimitEnforced(t *testing.T) {
	m := newTestModel("channel-abc")
	if m.textarea.CharLimit != maxMessageLength {
		t.Errorf("CharLimit = %d, want %d", m.textarea.CharLimit, maxMessageLength)
	}
}

func TestMaxMessageLengthIs4000(t *testing.T) {
	if maxMessageLength != 4000 {
		t.Errorf("maxMessageLength = %d, want 4000", maxMessageLength)
	}
}

func TestSetFocusedFocusesTextarea(t *testing.T) {
	m := newTestModel("channel-abc")
	m.SetFocused(false)
	if m.focused {
		t.Error("focused should be false after SetFocused(false)")
	}
	m.SetFocused(true)
	if !m.focused {
		t.Error("focused should be true after SetFocused(true)")
	}
}

func TestViewDoesNotPanic(t *testing.T) {
	m := newTestModel("channel-abc")
	// Should not panic regardless of focus or content
	_ = m.View()
	m.SetFocused(false)
	_ = m.View()
}

func TestSendTrimsWhitespace(t *testing.T) {
	m := newTestModel("channel-abc")
	m = typeText(m, "  trimmed  ")

	_, cmd := sendKey(m, tea.KeyEnter)
	msg := collectMsg(cmd)
	sendMsg, ok := msg.(messages.SendMessageMsg)
	if !ok {
		t.Fatalf("expected SendMessageMsg, got %T", msg)
	}
	if sendMsg.Text != "trimmed" {
		t.Errorf("Text = %q, want %q", sendMsg.Text, "trimmed")
	}
}
