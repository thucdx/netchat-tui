package keymap

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// Compile-time check that KeyMap satisfies help.KeyMap.
var _ help.KeyMap = KeyMap{}

// KeyMap defines all keybindings for netchat-tui.
type KeyMap struct {
	// Sidebar navigation
	Up           key.Binding // k or up-arrow
	Down         key.Binding // j or down-arrow
	JumpToBottom key.Binding // G — jump to latest messages in chat
	JumpToTop    key.Binding // gg — jump to top

	// Search opens the search bar in the sidebar.
	Search key.Binding

	// Select opens the highlighted channel. Only active when sidebar is focused.
	Select key.Binding

	// Panel focus switching
	FocusInput   key.Binding // i or a — focus message input
	FocusSidebar key.Binding // Esc — return focus to sidebar
	NextPanel    key.Binding // Tab — cycle focus between panels

	// Chat scroll
	ScrollUp   key.Binding // Ctrl+U
	ScrollDown key.Binding // Ctrl+D
	PageUp     key.Binding // Ctrl+B
	PageDown   key.Binding // Ctrl+F

	// Send submits the message. Only active when input is focused.
	Send    key.Binding // Enter (in input mode)
	Newline key.Binding // Shift+Enter (insert newline)

	// ToggleName switches between contact name (first+last) and account name (username)
	// for DM and group channels in the sidebar.
	ToggleName key.Binding // n — toggle contact name / account name

	// App
	// Quit must only be matched when the sidebar is focused.
	// The model must NOT check this binding when the input box has focus,
	// otherwise typing 'q' in a message will quit the app.
	Quit key.Binding // q (sidebar only) or Ctrl+C
	Help key.Binding // ? — show keybinding help
}

// DefaultKeyMap returns the default vim-like keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "move down"),
		),
		JumpToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "jump to latest"),
		),
		JumpToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "jump to top"),
		),
		Search: key.NewBinding(
			key.WithKeys("/", "ctrl+f"),
			key.WithHelp("/ ctrl+f", "search"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open channel"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("i", "a"),
			key.WithHelp("i/a", "type message"),
		),
		FocusSidebar: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to channels"),
		),
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "page down"),
		),
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		Newline: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "new line"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		ToggleName: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "toggle name display"),
		),
	}
}

// ShortHelp returns the bindings shown in the mini help bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Search, k.JumpToTop, k.JumpToBottom, k.ScrollUp, k.ScrollDown, k.FocusInput, k.Quit}
}

// FullHelp returns all bindings for the full help page.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.JumpToTop, k.JumpToBottom, k.Select, k.Search, k.ToggleName},
		{k.FocusInput, k.FocusSidebar, k.NextPanel},
		{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
		{k.Send, k.Newline, k.Quit, k.Help},
	}
}
