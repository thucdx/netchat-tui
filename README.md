# netchat-tui

A keyboard-driven terminal UI client for **netchat.viettel.vn** (Mattermost v4), built in Go with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

```
┌─────────────────┬──────────────────────────────────────────┐
│ @ Alice Smith   │ @ Alice Smith                            │
│ # general    3  │ ──────────────────────────────────────── │
│ ⊕ Team Alpha   │ Alice Smith  10:30                       │
│ # random        │   Hello everyone! How's it going?        │
│ ■ ops-team      │                                          │
│ @ Bob Nguyen    │ You ▶  10:31                             │
│ # announcements │   Doing great, thanks!                   │
│ ø quietchan     │                                          │
│   ↕ 8/42        ├──────────────────────────────────────────┤
└─────────────────┘ > type a message and press Enter         │
                   └──────────────────────────────────────────┘
```

---

## Features

- **Real-time messaging** via WebSocket — new messages appear instantly without polling; auto-reconnects if the connection drops
- **Open attachments** — press `o` on any message with files to download and open them with your default app; inline picker for multi-file messages
- **Unread marker** — `──── unread ────` divider marks your last read position; `r` jumps straight to it
- **All channel types** in one unified sidebar: DMs, Group messages, Public, and Private channels
- **Unread badges** per channel; automatically cleared when you open a channel
- **Muted channel** indicators — distinct icon and dimmed style
- **Markdown rendering** powered by [glamour](https://github.com/charmbracelet/glamour) — code blocks, bold, italics, lists, and more
- **Image & file previews** — inline terminal art for images; press `o` on an image message to view it in a full-pane popup overlay
- **Message editing** — `(edited)` marker on server-edited posts
- **Infinite scroll** — scroll to the top of any channel to page in older messages
- **Sidebar search** — fuzzy-search joined channels/DMs and discover new ones via the API; open a new DM or join a public channel directly from the search results
- **Display name toggle** — switch between contact name (first + last name) and account name (username) for all authors and channel labels at once
- **Resizable sidebar** — drag the right border left/right with the mouse
- **Vim-style navigation** — `j/k`, `gg`, `G`, `Ctrl+U/D`, `Ctrl+B/F` throughout
- **Visual selection & copy** — `V` enters visual mode; `j`/`k` extends the selection; `y` copies selected messages to the clipboard

---

## Installation

### Option 1 — Download a pre-built binary (no Go required)

Go to the [Releases page](https://github.com/thucdx/netchat-tui/releases/latest) and download the archive for your platform:

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `netchat-tui_*_darwin_arm64.tar.gz` |
| macOS (Intel) | `netchat-tui_*_darwin_amd64.tar.gz` |
| Linux (x86-64) | `netchat-tui_*_linux_amd64.tar.gz` |
| Linux (ARM64) | `netchat-tui_*_linux_arm64.tar.gz` |
| Windows (x86-64) | `netchat-tui_*_windows_amd64.zip` |

Extract and run:

```bash
# macOS / Linux
tar -xzf netchat-tui_*_linux_amd64.tar.gz
./netchat-tui

# move to PATH (optional)
sudo mv netchat-tui /usr/local/bin/
```

### Option 2 — `go install` (requires Go 1.22+)

```bash
go install github.com/thucdx/netchat-tui@latest
netchat-tui
```

### Option 3 — Build from source

```bash
git clone https://github.com/thucdx/netchat-tui
cd netchat-tui
go build -o netchat-tui .
./netchat-tui
```

---

## Requirements

- A valid account on **netchat.viettel.vn**
- A terminal with true-color support (iTerm2, Alacritty, kitty, Windows Terminal, etc.)
- Go 1.22+ (only for Options 2 and 3 above)

---

## First run

On first launch you will be taken to the **authentication screen** (see below). Subsequent launches go straight to the chat UI using the saved token.

---

## Authentication — getting your token

netchat-tui authenticates with a **Personal Access Token** (a long-lived bearer token). There are two ways to obtain one:

### Option A — Personal Access Tokens page (recommended)

1. Log in to [netchat.viettel.vn](https://netchat.viettel.vn) in your browser.
2. Click your avatar → **Profile** → **Security** → **Personal Access Tokens**.
3. Click **Create Token**, give it any name (e.g. `netchat-tui`), and copy the token value.

> **Note:** If the Personal Access Tokens page is missing, the feature may be disabled on your server — use Option B instead.

### Option B — Copy from browser DevTools

1. Log in to [netchat.viettel.vn](https://netchat.viettel.vn) in your browser.
2. Open DevTools (`F12`) → **Network** tab.
3. Reload the page or send any message.
4. Click any API request, look at its **Request Headers**, and copy the value after `Authorization: Bearer `.

### Pasting the token

When netchat-tui starts without a saved token it shows a prompt:

```
Paste MMAUTHTOKEN here…
```

Paste your token (it is hidden as `•••`) and press **Enter**. The app validates it against the server; on success the token is saved to `~/.config/netchat-tui/config.json` (mode `0600`) and you will not be asked again.

To **switch accounts**, delete the config file and restart:

```bash
rm ~/.config/netchat-tui/config.json
```

---

## Layout

The UI is three panels:

| Panel | Description |
|-------|-------------|
| **Sidebar** (left) | Scrollable channel list, ordered by most recent activity. Unread badge on the right. |
| **Chat pane** (top-right) | Message history with author headers, timestamps, and markdown rendering. |
| **Input** (bottom-right) | Multi-line message composer. |

Focus cycles with `Tab` or jump directly with `i`/`a` (input) and `Esc` (sidebar).

### Sidebar channel icons

| Icon | Meaning |
|------|---------|
| `#` | Public channel |
| `■` | Private channel |
| `@` | Direct message |
| `⊕` | Group message |
| `⊘` | Muted public channel |
| `□` | Muted private channel |
| `ø` | Muted DM |
| `⊖` | Muted group |

---

## Keybindings

### Focus

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus: Sidebar → Chat → Input |
| `i` or `a` | Jump to message input |
| `Esc` | Return focus to sidebar (also dismisses error banner) |

### Sidebar — channel list

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `gg` | Jump to top of list |
| `G` | Jump to bottom of list |
| `Ctrl+U` | Scroll up half page |
| `Ctrl+D` | Scroll down half page |
| `Enter` | Open highlighted channel |
| `/` or `Ctrl+F` | Open search bar |
| `n` | Toggle display name (contact name ↔ username) |
| `q` | Quit |

### Sidebar — search mode

Triggered by `/` or `Ctrl+F`. The sidebar is replaced by a live results list.

| Key | Action |
|-----|--------|
| _(type)_ | Build query (results appear after 3 characters) |
| `↑` / `↓` | Move result cursor |
| `Enter` | Open existing channel / start new DM / join channel |
| `Backspace` | Delete last character |
| `Esc` | Exit search, return to channel list |

When selecting a **new public channel**, a confirmation line appears:
```
Join #channel-name? [y/N]
```
Press `y` or `Enter` to confirm, any other key to cancel.

### Chat pane

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor to next (newer) message |
| `k` / `↑` | Move cursor to previous (older) message |
| `Ctrl+U` | Scroll viewport up half page |
| `Ctrl+D` | Scroll viewport down half page |
| `Ctrl+B` | Page up |
| `Ctrl+F` | Page down |
| `gg` | Cursor to oldest loaded message |
| `G` | Cursor to newest message |
| `r` | Jump to first unread message |
| `o` or `l` | Open attachment(s) of cursor message; images open in popup overlay |
| `h` | Close attachment picker / image popup |
| `V` | Enter visual selection mode (anchored at cursor) |
| `y` | Yank (copy) selected messages to clipboard; exit visual mode |
| `Esc` | Exit visual mode |

> Moving the cursor to the **top of the loaded buffer** automatically loads the previous page of messages.

### Message input

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | Insert newline |

### Global

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit from anywhere |
| `?` | Show keybinding help overlay |

---

## Display name toggle

Press `n` while the sidebar has focus to switch how user names are shown **everywhere** (sidebar labels, chat message headers):

| Mode | Display |
|------|---------|
| **Contact name** (default) | First + Last name from the user's profile. Falls back to username if no name is set. |
| **Account name** | Raw username (e.g. `nguyenvan.a`) |

The toggle applies to DMs and Group channels only; public/private channel names are unaffected.

---

## tmux integration

When running inside tmux, netchat-tui automatically updates the **tab title** with your unread count:

```
netchat-tui [12/3]   ← 12 unread messages across 3 channels
netchat-tui          ← everything read
```

Only **unmuted** channels are counted.

### How it works

At startup (when `$TMUX` is set) the app:
1. Disables `automatic-rename` for the current window (prevents tmux from fighting our custom title)
2. Calls `tmux rename-window` directly on every unread change

On exit it resets the window name to `netchat-tui` and restores `automatic-rename`.

No tmux configuration required — everything is set automatically per-window.

---

## Configuration

Config file: `~/.config/netchat-tui/config.json`
_(on macOS: `~/Library/Application Support/netchat-tui/config.json`)_

```json
{
  "token": "your-mmauthtoken",
  "user_id": "your-user-id",
  "sidebar_limit": 50
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `token` | — | Bearer token (written by the auth prompt) |
| `user_id` | — | Your Mattermost user ID (written automatically) |
| `sidebar_limit` | `200` | Maximum channels shown in the sidebar |

---

## Running tests

```bash
go test ./...
```

---

## Tech stack

| Library | Role |
|---------|------|
| [Bubbletea](https://github.com/charmbracelet/bubbletea) | Elm-architecture TUI framework |
| [Lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and layout |
| [Glamour](https://github.com/charmbracelet/glamour) | Markdown rendering |
| [Bubbles](https://github.com/charmbracelet/bubbles) | Viewport, text input, spinner components |
