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

// setupTmux configures the current tmux window for reliable title updates.
// It disables automatic-rename (which would fight our custom title) and returns:
//   - a cleanup function that resets the window name and restores automatic-rename on exit
//   - a titleUpdater function that calls "tmux rename-window <title>" directly
//
// When not inside tmux both return values are no-ops / nil.
func setupTmux() (cleanup func(), titleUpdater func(string)) {
	if os.Getenv("TMUX") == "" {
		return func() {}, nil
	}
	// Turn off automatic-rename so tmux doesn't immediately override our title.
	exec.Command("tmux", "set-window-option", "automatic-rename", "off").Run() //nolint:errcheck

	return func() {
			// Restore the window name and re-enable automatic-rename on exit.
			exec.Command("tmux", "rename-window", "netchat-tui").Run()           //nolint:errcheck
			exec.Command("tmux", "set-window-option", "automatic-rename", "on"). //nolint:errcheck
													Run()
		}, func(title string) {
			exec.Command("tmux", "rename-window", title).Run() //nolint:errcheck
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

	// Configure tmux window-title integration and schedule cleanup on exit.
	tmuxCleanup, tmuxTitleFn := setupTmux()
	defer tmuxCleanup()

	// Launch the main chat UI.
	app := tui.NewAppModel(apiClient)
	if cfg.SidebarLimit > 0 {
		app = app.WithSidebarLimit(cfg.SidebarLimit)
	}
	if tmuxTitleFn != nil {
		app = app.WithTitleUpdater(tmuxTitleFn)
	}
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: app encountered an error: %v\n", err)
		os.Exit(1)
	}
}
