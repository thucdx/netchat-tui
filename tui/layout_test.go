package tui

import "testing"

func TestNewLayout_Normal(t *testing.T) {
	l := NewLayout(120, 40)

	if l.SidebarWidth != 28 {
		t.Errorf("SidebarWidth: got %d, want 28", l.SidebarWidth)
	}
	if l.ChatWidth != 92 {
		t.Errorf("ChatWidth: got %d, want 92", l.ChatWidth)
	}
	if l.ChatHeight != 37 {
		t.Errorf("ChatHeight: got %d, want 37", l.ChatHeight)
	}
	if l.InputHeight != 3 {
		t.Errorf("InputHeight: got %d, want 3", l.InputHeight)
	}
	if l.TotalWidth != 120 {
		t.Errorf("TotalWidth: got %d, want 120", l.TotalWidth)
	}
	if l.TotalHeight != 40 {
		t.Errorf("TotalHeight: got %d, want 40", l.TotalHeight)
	}
}

func TestNewLayout_SmallTerminal(t *testing.T) {
	l := NewLayout(40, 5)

	if l.IsValid() {
		t.Error("IsValid() should be false for a 40x5 terminal")
	}
	if l.SidebarWidth < 1 {
		t.Errorf("SidebarWidth should be >= 1, got %d", l.SidebarWidth)
	}
	if l.ChatWidth < 1 {
		t.Errorf("ChatWidth should be >= 1, got %d", l.ChatWidth)
	}
	if l.InputHeight < 1 {
		t.Errorf("InputHeight should be >= 1, got %d", l.InputHeight)
	}
	if l.ChatHeight < 1 {
		t.Errorf("ChatHeight should be >= 1, got %d", l.ChatHeight)
	}
}

func TestNewLayout_Minimum(t *testing.T) {
	l := NewLayout(60, 10)

	if !l.IsValid() {
		t.Error("IsValid() should be true for a 60x10 terminal")
	}
}

func TestNewLayout_ExactMinimum(t *testing.T) {
	l59x10 := NewLayout(59, 10)
	if l59x10.IsValid() {
		t.Error("IsValid() should be false for a 59x10 terminal")
	}

	l60x9 := NewLayout(60, 9)
	if l60x9.IsValid() {
		t.Error("IsValid() should be false for a 60x9 terminal")
	}
}
