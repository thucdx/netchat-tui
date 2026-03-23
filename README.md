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

## First launch — getting your token

On first run you will see a token prompt.

1. Log in to netchat.viettel.vn in a browser.
2. Go to **Profile → Security → Personal Access Tokens** and create a new token.
   (Or open DevTools → Network, copy the `Authorization: Bearer <value>` from any request.)
3. Paste the token into the prompt and press **Enter**.

The token is saved to `~/.config/netchat-tui/config.json` (mode `0600`) and reused on subsequent launches. To switch accounts, delete this file and restart.

---

## Layout

```
┌─────────────┬──────────────────────────────┐
│  Sidebar    │  Chat pane                   │
│             │                              │
│  DIRECT     │  #general                    │
│  @ Alice    │  ──────────────────────────  │
│  @ Bob      │  Alice  10:30                │
│             │    Hello world               │
│  CHANNELS   │                              │
│  # general  │  Bob  10:31                  │
│  # random   │    Hi there!                 │
│             │                              │
│             ├──────────────────────────────┤
│             │  > type a message here       │
└─────────────┴──────────────────────────────┘
```

---

## Keybindings

### Focus / navigation

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus: Sidebar → Chat → Input → Sidebar |
| `i` or `a` | Jump directly to message input |
| `Esc` | Return focus to sidebar (also dismisses error banner) |

### Sidebar (channel list)

Channels are ordered by most recent activity within each section (DMs first, then channels).

| Key | Action |
|-----|--------|
| `j` or `↓` | Move cursor down |
| `k` or `↑` | Move cursor up |
| `G` | Jump to bottom of list |
| `Enter` | Open highlighted channel |

### Chat pane (message history)

| Key | Action |
|-----|--------|
| `Ctrl+U` | Scroll up half page |
| `Ctrl+D` | Scroll down half page |
| `Ctrl+B` | Page up |
| `Ctrl+F` | Page down |
| `k` or `↑` | Scroll up one line |
| `j` or `↓` | Scroll down one line |
| `G` | Jump to latest message |

Scrolling to the top automatically loads older messages.

### Message input

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | Insert newline |

### App

| Key | Action |
|-----|--------|
| `q` (sidebar focused) | Quit |
| `Ctrl+C` | Quit from anywhere |
| `?` | Show keybinding help |

---

## Configuration

Config is stored at `~/.config/netchat-tui/config.json` (created automatically on first launch).

```json
{
  "token": "your-mmauthtoken",
  "user_id": "your-user-id",
  "sidebar_limit": 50
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `token` | — | Bearer token (set via the auth prompt) |
| `user_id` | — | Mattermost user ID (set automatically) |
| `sidebar_limit` | `50` | Max channels shown in the sidebar |

---

## Running tests

```bash
go test ./...
```
