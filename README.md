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
- **All channel types** in one unified sidebar: DMs, Group messages, Public, and Private channels
- **Unread badges** per channel; automatically cleared when you open a channel
- **Unread marker** — `──── unread ────` divider marks your last read position; `r` jumps straight to it
- **Muted channel** indicators — distinct icon and dimmed style
- **Markdown rendering** powered by [glamour](https://github.com/charmbracelet/glamour) — code blocks, bold, italics, lists, and more
- **Image & file attachments** — inline thumbnail art in the chat; press `o` to open any file or image in your OS default viewer at full resolution
- **Message editing** — `(edited)` marker on server-edited posts
- **Infinite scroll** — scroll to the top of any channel to page in older messages
- **Sidebar search** — fuzzy-search joined channels/DMs and discover new ones; open a new DM or join a public channel directly from results
- **Display name toggle** — switch between contact name and username for all authors and channel labels at once
- **Resizable sidebar** — drag the right border left/right with the mouse
- **Vim-style navigation** — `j/k`, `gg`, `G`, `Ctrl+U/D`, `Ctrl+B/F`, count prefixes (`5j`, `5gg`) throughout
- **Visual selection & copy** — `V` enters visual mode; `j`/`k` extends the selection; `y` copies to the clipboard
- **Preview channel** — `p` loads a channel in the chat pane without leaving the sidebar

---

## Requirements

- A valid account on **netchat.viettel.vn**
- A terminal with **true-color support**: [Ghostty](https://ghostty.org), iTerm2, Alacritty, kitty, WezTerm, Windows Terminal, etc.
- Go 1.22+ (only for build-from-source installs)

> **Recommended: Ghostty** — best rendering fidelity, true 24-bit color, and fast redraws.

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

## Authentication — getting your MMAUTHTOKEN

netchat-tui authenticates with your **MMAUTHTOKEN** browser session cookie. No personal access token setup is required.

### How to copy your MMAUTHTOKEN

1. Log in to [netchat.viettel.vn](https://netchat.viettel.vn) in your browser.
2. Open DevTools (`F12`) → **Application** tab (Chrome/Edge) or **Storage** tab (Firefox).
3. Navigate to **Cookies** → `https://netchat.viettel.vn`.
4. Find the cookie named **`MMAUTHTOKEN`** and copy its **Value**.

> **Tip (Chrome/Edge shortcut):** Open the **Network** tab, reload the page, click any API request, go to **Request Headers**, and copy the value after `Authorization: Bearer `.

### First launch

When netchat-tui starts without a saved token it shows:

```
Paste MMAUTHTOKEN here…
```

Paste your token (input is hidden as `•••`) and press **Enter**. The app validates it against the server; on success it is saved to `~/.config/netchat-tui/config.json` (mode `0600`) and you will not be asked again.

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

Jump between panels with `[` (sidebar), `]` (chat), `i`/`a` (input), or cycle with `Tab`.

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

Keys are ordered from most-used to least-used within each section.

### 1. Navigate between panels

| Key | Action |
|-----|--------|
| `]` | Jump to chat pane |
| `[` | Jump to sidebar |
| `i` or `a` | Jump to message input |
| `Esc` | Return to chat pane from input |
| `Tab` | Cycle focus: Sidebar → Chat → Input |

### 2. Open / preview a channel

| Key | Action |
|-----|--------|
| `Enter` | Open channel and move focus to chat pane |
| `p` | Preview channel (load chat without leaving sidebar) |

### 3. Search channels and DMs

Press `/` to open the search bar. The sidebar is replaced by a live results list.

| Key | Action |
|-----|--------|
| `/` | Open search bar |
| _(type)_ | Build query (results appear after 3 characters) |
| `↑` / `↓` | Move result cursor |
| `Enter` | Open existing channel / start new DM / join channel |
| `Backspace` | Delete last character |
| `Esc` | Exit search, return to channel list |

When selecting a **new public channel**, a confirmation prompt appears:
```
Join #channel-name? [y/N]
```
Press `y` or `Enter` to confirm, any other key to cancel.

### 4. Navigate the chat pane

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor to next (newer) message |
| `k` / `↑` | Move cursor to previous (older) message |
| `G` | Jump to newest message |
| `r` | Jump to first unread message |
| `gg` | Jump to oldest loaded message |
| `Ctrl+D` | Scroll viewport down half page |
| `Ctrl+U` | Scroll viewport up half page |
| `Ctrl+F` | Page down |
| `Ctrl+B` | Page up |

> Scrolling or moving the cursor to the **top of the loaded buffer** automatically pages in older messages.

### 5. Copy messages

| Key | Action |
|-----|--------|
| `V` | Enter visual selection mode (anchored at cursor) |
| `j` / `k` | Extend selection down / up |
| `y` | Copy selected messages to clipboard; exit visual mode |
| `Esc` | Cancel visual mode without copying |

**Quick copy a single message:** navigate to it with `j`/`k`, press `V`, then `y`.

### 6. Send a message

| Key | Action |
|-----|--------|
| `i` or `a` | Focus the input bar |
| _(type)_ | Compose your message |
| `Enter` | Send message |
| `Shift+Enter` | Insert newline |
| `Esc` | Discard focus, return to chat pane |

### 7. Open files and images

| Key | Action |
|-----|--------|
| `o` or `l` | Open attachment(s) of the cursor message in the OS default viewer |
| `h` | Close attachment picker (when multiple files) |

For messages with **multiple files**, pressing `o` opens an inline picker — navigate with `j`/`k`, press `Enter` to open, `h` to dismiss.

Images are displayed as **inline thumbnail art** in the chat. Press `o` to open the full-resolution image in your OS viewer (Preview on macOS, default image app on Linux/Windows).

### 8. Navigate the sidebar list

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `{N}j` / `{N}k` | Move down/up N items (e.g. `5j`) |
| `{N}gg` / `{N}G` | Jump to item N, 1-based (e.g. `5gg` or `5G`) |
| `gg` | Jump to top of list |
| `G` | Jump to bottom of list |
| `Ctrl+U` | Scroll up half page |
| `Ctrl+D` | Scroll down half page |

### 9. Global

| Key | Action |
|-----|--------|
| `?` | Show keybinding help overlay |
| `n` | Toggle display name (contact name ↔ username) |
| `q` | Quit (sidebar focused) |
| `Ctrl+C` | Quit from anywhere |

---

## Display name toggle

Press `n` while the sidebar has focus to switch how names are shown **everywhere** (sidebar labels, chat message headers):

| Mode | Display |
|------|---------|
| **Contact name** (default) | First + Last name from the user's profile. Falls back to username if empty. |
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

At startup (when `$TMUX` is set) the app disables `automatic-rename` for the current window and calls `tmux rename-window` on every unread change. On exit it resets the window name and restores `automatic-rename`. No tmux configuration required.

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
| `token` | — | MMAUTHTOKEN value (written by the auth prompt) |
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
