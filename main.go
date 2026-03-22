package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thucdx/netchat-tui/config"
	"github.com/thucdx/netchat-tui/tui"
)

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

	// Phase 3 will replace this with the real AppModel.
	fmt.Printf("Authenticated as %s. Main app coming soon...\n", cfg.UserID)
}
