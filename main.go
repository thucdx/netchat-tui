package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thucdx/netchat-tui/api"
	"github.com/thucdx/netchat-tui/config"
	"github.com/thucdx/netchat-tui/tui"
)

// setupTmux enables window-title renaming for the current tmux window so that
// netchat-tui can update the tab with unread counts via OSC 2 escape sequences.
// Returns a cleanup function that resets the window title to a plain
// "netchat-tui" string when the app exits.
// No-op (and returns a no-op cleanup) when not running inside tmux.
func setupTmux() func() {
	if os.Getenv("TMUX") == "" {
		return func() {}
	}
	// Enable rename for this window only (not a global change).
	exec.Command("tmux", "set-window-option", "allow-rename", "on").Run() //nolint:errcheck
	return func() {
		// Clear the unread indicator from the tab title on exit.
		fmt.Fprint(os.Stdout, "\033]2;netchat-tui\007")
	}
}

// authWrapper is a thin root model that captures AuthSuccessMsg emitted by
// AuthModel so it can be inspected after the Bubbletea program exits.
type authWrapper struct {
	inner  tui.AuthModel
	result *tui.AuthSuccessMsg
}

func (w authWrapper) Init() tea.Cmd { return w.inner.Init() }

func (w authWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := w.inner.Update(msg)
	if am, ok := inner.(tui.AuthModel); ok {
		w.inner = am
	}
	if success, ok := msg.(tui.AuthSuccessMsg); ok {
		w.result = &success
		return w, tea.Quit
	}
	return w, cmd
}

func (w authWrapper) View() string { return w.inner.View() }

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Token == "" {
		wrapper := authWrapper{inner: tui.NewAuthModel(config.BaseURL)}
		p := tea.NewProgram(wrapper, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: auth screen encountered an error: %v\n", err)
			os.Exit(1)
		}

		final, ok := finalModel.(authWrapper)
		if !ok || final.result == nil {
			// User cancelled without completing auth.
			os.Exit(0)
		}

		cfg.Token = final.result.Token
		cfg.UserID = final.result.UserID

		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to save config: %v\n", err)
			os.Exit(1)
		}
	}

	// Build the API client with the stored credentials.
	apiClient, err := api.NewClient(config.BaseURL, cfg.Token, cfg.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create API client: %v\n", err)
		os.Exit(1)
	}

	// Enable tmux window-title renaming and schedule cleanup on exit.
	defer setupTmux()()

	// Launch the main chat UI.
	app := tui.NewAppModel(apiClient)
	if cfg.SidebarLimit > 0 {
		app = app.WithSidebarLimit(cfg.SidebarLimit)
	}
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: app encountered an error: %v\n", err)
		os.Exit(1)
	}
}
