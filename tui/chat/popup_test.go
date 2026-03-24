package chat

import (
	"strings"
	"testing"
)

// ────────────────────────────────────────────────────────────────────────────
// RenderImagePopup unit tests (Task #17)
// ────────────────────────────────────────────────────────────────────────────

// TestRenderImagePopup_ContainsTitle verifies that the popup output contains
// the title string in the top border.
func TestRenderImagePopup_ContainsTitle(t *testing.T) {
	output := RenderImagePopup("image content", "myfile.png", 80, 20)

	if !strings.Contains(output, "myfile.png") {
		t.Errorf("popup output should contain title 'myfile.png', got: %s", output)
	}
}

// TestRenderImagePopup_ContainsBorders verifies that the popup output contains
// the expected border characters.
func TestRenderImagePopup_ContainsBorders(t *testing.T) {
	output := RenderImagePopup("image content", "test.png", 80, 20)

	// Check for top-left border.
	if !strings.Contains(output, "┌") {
		t.Error("popup output should contain top-left border '┌'")
	}
	// Check for bottom-left border.
	if !strings.Contains(output, "└") {
		t.Error("popup output should contain bottom-left border '└'")
	}
	// Check for side borders.
	if !strings.Contains(output, "│") {
		t.Error("popup output should contain side border '│'")
	}
}

// TestRenderImagePopup_ContainsCloseHint verifies that the bottom border
// contains the close hint text.
func TestRenderImagePopup_ContainsCloseHint(t *testing.T) {
	output := RenderImagePopup("image content", "test.png", 80, 20)

	if !strings.Contains(output, "h/Esc: close") {
		t.Errorf("popup output should contain close hint 'h/Esc: close', got: %s", output)
	}
}

// TestRenderImagePopup_EmptyImage verifies that rendering an empty image
// string does not panic.
func TestRenderImagePopup_EmptyImage(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderImagePopup with empty image should not panic, got: %v", r)
		}
	}()

	output := RenderImagePopup("", "file.png", 80, 20)

	// Should produce a valid popup with empty body.
	if !strings.Contains(output, "┌") || !strings.Contains(output, "└") {
		t.Error("popup with empty image should still have borders")
	}
}

// TestRenderImagePopup_NarrowWidth verifies that rendering with a very narrow
// width (6 pixels) does not panic and still produces valid output.
func TestRenderImagePopup_NarrowWidth(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderImagePopup with width=6 should not panic, got: %v", r)
		}
	}()

	output := RenderImagePopup("test image", "file.png", 6, 10)

	// Should produce valid output with minimal borders.
	if !strings.Contains(output, "┌") || !strings.Contains(output, "└") {
		t.Error("popup with narrow width should still have borders")
	}
	// Ensure there are lines (at least top, content, bottom).
	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		t.Errorf("popup should have at least 3 lines (top, body, bottom), got %d", len(lines))
	}
}
