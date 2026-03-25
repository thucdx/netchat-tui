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

	// Select opens the highlighted channel and focuses the chat pane.
	Select key.Binding
	// PreviewChannel opens the highlighted channel without moving focus away from the sidebar.
	PreviewChannel key.Binding // p

	// Panel focus switching
	FocusInput   key.Binding // i or a — focus message input
	FocusChat    key.Binding // ] — jump directly to chat dialogue
	FocusSidebar key.Binding // [ — jump directly to sidebar
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

	// Chat message cursor actions
	OpenAttachment        key.Binding // o or l — open attachment for cursor post
	CloseAttachmentPicker key.Binding // h — close attachment picker
	JumpToUnread          key.Binding // r — jump to first unread post

	// Visual mode (Features 3 & 4)
	VisualMode    key.Binding // V — enter/exit visual selection mode
	Yank          key.Binding // y — copy selected messages to clipboard
	ExpandCollapse key.Binding // z — expand/collapse a long message

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
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open channel"),
		),
		PreviewChannel: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "preview channel"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("i", "a"),
			key.WithHelp("i/a", "type message"),
		),
		FocusChat: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "go to chat"),
		),
		FocusSidebar: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "go to sidebar"),
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
		OpenAttachment: key.NewBinding(
			key.WithKeys("o", "l"),
			key.WithHelp("o/l", "open attachment"),
		),
		CloseAttachmentPicker: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "close picker"),
		),
		JumpToUnread: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "jump to unread"),
		),
		VisualMode: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "visual select"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yank/copy"),
		),
		ExpandCollapse: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "expand/collapse"),
		),
	}
}

// HelpSection groups related keybindings under a named heading for the help popup.
type HelpSection struct {
	Title    string
	Bindings []key.Binding
}

// HelpSections returns all keybindings organised into named sections, ordered
// from most-used to least-used. Editing this method is the only change needed
// when keybindings are added or reorganised — the help popup renders it automatically.
func (k KeyMap) HelpSections() []HelpSection {
	return []HelpSection{
		{
			Title:    "Panels",
			Bindings: []key.Binding{k.FocusSidebar, k.FocusChat, k.FocusInput, k.NextPanel},
		},
		{
			Title:    "Channel",
			Bindings: []key.Binding{k.Select, k.PreviewChannel, k.Search},
		},
		{
			Title:    "Chat — Navigate",
			Bindings: []key.Binding{k.Up, k.Down, k.JumpToBottom, k.JumpToTop, k.JumpToUnread, k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
		},
		{
			Title:    "Chat — Copy",
			Bindings: []key.Binding{k.VisualMode, k.Yank, k.ExpandCollapse},
		},
		{
			Title:    "Message Input",
			Bindings: []key.Binding{k.Send, k.Newline},
		},
		{
			Title:    "Files & Images",
			Bindings: []key.Binding{k.OpenAttachment, k.CloseAttachmentPicker},
		},
		{
			Title:    "Sidebar — Navigate",
			Bindings: []key.Binding{k.Up, k.Down, k.JumpToTop, k.JumpToBottom, k.ScrollUp, k.ScrollDown},
		},
		{
			Title:    "Global",
			Bindings: []key.Binding{k.Help, k.ToggleName, k.Quit},
		},
	}
}

// ShortHelp returns the bindings shown in the mini help bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Search, k.JumpToTop, k.JumpToBottom, k.ScrollUp, k.ScrollDown, k.FocusInput, k.Quit}
}

// FullHelp returns all bindings for the full help page.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.JumpToTop, k.JumpToBottom, k.Select, k.PreviewChannel, k.Search, k.ToggleName},
		{k.FocusInput, k.FocusChat, k.FocusSidebar, k.NextPanel},
		{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
		{k.Send, k.Newline, k.Quit, k.Help},
		{k.OpenAttachment, k.CloseAttachmentPicker, k.JumpToUnread},
		{k.VisualMode, k.Yank},
	}
}
