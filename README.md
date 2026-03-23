# netchat-tui

A terminal UI client for netchat.viettel.vn (Mattermost v4), built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Requirements

- Go 1.21 or later
- A valid account on netchat.viettel.vn

## Build and run

```bash
git clone https://github.com/thucdx/netchat-tui
cd netchat-tui
go run .
```

Or build a binary:

```bash
go build -o netchat-tui .
./netchat-tui
```

## First launch

On first run you will see a token prompt. To get your token:

1. Open netchat.viettel.vn in a browser and log in.
2. Go to **Profile → Security → Personal Access Tokens** and create a new token.
   Alternatively, open browser DevTools → Network, send any request, and copy the `Authorization: Bearer <token>` value.
3. Paste the token into the prompt and press **Enter**.

The token is saved to `~/.config/netchat-tui/auth.json` (mode `0600`) and reused on subsequent launches.

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between sidebar and chat |
| `j` / `k` | Move cursor down / up in sidebar |
| `gg` | Jump to top of sidebar |
| `G` | Jump to bottom of sidebar |
| `Enter` | Open selected channel |
| `↑` / `↓` | Scroll chat one line |
| `Ctrl+U` / `Ctrl+D` | Scroll chat half page |
| `PgUp` / `PgDn` | Scroll chat full page |
| `Shift+Enter` | Insert newline in message |
| `Enter` | Send message |
| `Esc` | Focus sidebar / dismiss error banner |
| `Ctrl+C` | Quit |

## Configuration

The only required configuration is the Bearer token set during first launch. Config is stored at:

```
~/.config/netchat-tui/auth.json
```

To log in as a different user, delete this file and restart.

## Running tests

```bash
go test ./...
```
