# netchat-tui Requirements

A TUI (Terminal User Interface) chat client for netchat.viettel.vn (Mattermost v4), designed to run in any terminal (tmux recommended).

---

## Tech Stack

- **Language**: Go 1.22+
- **TUI Framework**: Bubbletea (Elm-architecture event loop, state management)
- **Styling**: Lipgloss (colors, borders, layout)
- **Components**: Bubbles (viewport, textinput, spinner)
- **HTTP Client**: Go standard `net/http`
- **WebSocket**: `gorilla/websocket`
- **Markdown rendering**: `glamour`
- **Image rendering**: `ansimage` (inline terminal art via sixel/block pixels)
- **Config/Token storage**: `encoding/json` (standard library)

---

## Authentication

- **Method**: Personal Access Token (PAT) or browser session token
- **Flow**:
  1. On first launch (or when token is missing), TUI shows an auth prompt
  2. User obtains a token via PAT page or copies `MMAUTHTOKEN` from browser DevTools
  3. User pastes the token into the TUI prompt; it is hidden as `•••`
  4. TUI validates the token by calling `GET /api/v4/users/me`
  5. On success, token is saved locally and the app proceeds to the chat view
- **Token Storage**: `~/.config/netchat-tui/config.json` (mode `0600`)
  - Stores: `token`, `user_id`, `sidebar_limit`
- **Subsequent launches**: TUI reads stored token and starts directly in chat view
- **Switch accounts**: Delete the config file and restart

---

## Layout

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

- **Sidebar** (left): scrollable channel list, ordered by most recent activity
- **Chat pane** (top-right): scrollable message history with author headers, timestamps, markdown
- **Input** (bottom-right): multi-line message composer
- **Resizable sidebar**: drag the right border left/right with the mouse

---

## Channel List (Sidebar)

- **Sources**: All teams the user belongs to — channels from every team are merged into a single flat list
- **Channel types shown**: Direct Messages (`D`), Group messages (`G`), Public channels (`O`), Private channels (`P`)
- **Sort order**: Most recent activity (`LastPostAt`) descending; alphabetical tiebreak
- **Limit**: Configurable via `sidebar_limit` in config (default 200)
- **Unread count**: Shown as a badge on the right side of each entry
- **Muted channels**: Distinct icon + dimmed style; unread count still shown; suppressed from tmux title
- **Icons**:

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

## Sidebar Search

- Triggered by `/` or `Ctrl+F`
- The channel list is replaced by a live search results pane
- Query typed character-by-character; search fires after ≥3 characters
- Two search backends:
  - **Local channels**: filter already-joined channels/DMs by name
  - **API search**: `GET /api/v4/users/search` and `GET /api/v4/channels/search` for remote results
- **Results**: users (to start a new DM) and channels (to join or open)
- **Enter on a DM user**: creates a direct channel if it doesn't exist, then opens it
- **Enter on a new public channel**: shows a confirmation prompt (`Join #channel-name? [y/N]`); `y` or `Enter` to join, any other key to cancel
- **Navigation in search**: `↑` / `↓` (arrow keys); `j`/`k` type as text (no conflict with vim bindings)
- **Exit**: `Esc` returns to the channel list

---

## Display Name Toggle

- Press `n` while the sidebar has focus to toggle display mode everywhere (sidebar labels + chat message headers)
- **Contact name mode** (default): `FirstName + " " + LastName` from the user's profile; falls back to username if no name is set
- **Account name mode**: raw username (e.g. `nguyenvan.a`)
- Applies to DMs and Group channels only; public/private channel names are unaffected

---

## Chat Pane

- Scrollable message history using `bubbles/viewport`
- Each message shows: author name (respects display name toggle), timestamp, message content
- **Markdown rendering**: bold, italics, code blocks, lists, blockquotes via `glamour`
- **Edited messages**: `(edited)` marker shown after edited posts
- **Image previews**: inline terminal art via ansimage (rendered at post-load time)
- **Non-image file attachments**: rendered as `📎 filename.ext  (size)` below the message body; metadata fetched via `GET /api/v4/files/{id}/info` at post-load time
- **Author display**: "You ▶" for own messages; contact name or username for others
- **Infinite scroll**: scrolling to the very top triggers `GET /api/v4/channels/{id}/posts?page=N` for older pages
- **Real-time updates**: new messages appended instantly via WebSocket; no polling

### Message cursor

- When the chat pane has focus, a **message cursor** is always visible (highlighted left border `▌` in accent colour on the cursor message)
- On channel open: cursor starts at the **newest message** (bottom)
- `j/k` move the cursor one message at a time; the viewport scrolls automatically to keep the cursor visible
- `Ctrl+U/D/B/F` scroll the viewport; the cursor clamps to the viewport edge if it goes off-screen
- `gg` / `G` jump cursor to oldest / newest loaded message
- Loading older pages (infinite scroll) is triggered when the cursor reaches the top of the loaded buffer

### Unread marker

- An `──── unread ────` divider line is inserted **above the first post whose `CreateAt > member.LastViewedAt`** when a channel is opened with unreads
- `r` hotkey jumps the cursor directly to the unread marker (first unread message); no-op if everything is read
- The divider is purely cosmetic — it does not persist after the channel is re-opened

### Attachment picker

- When cursor message has file attachments, `o` or `l` opens them:
  - **Single attachment**: skip picker, download and open immediately
  - **Multiple attachments**: show an inline overlay anchored below the cursor message:
    ```
    ┌─ Attachments ──────────────────────┐
    │ ▶ report.pdf          (1.2 MB)     │
    │   screenshot.png      (340 KB)     │
    │   notes.txt           (4 KB)       │
    └────────────────────────────────────┘
    ```
  - Navigate picker with `j/k`, `Enter`/`o` to open selected file, `Esc`/`h` to close without opening
- `h` with picker closed: no-op (reserved for future use)

---

## Navigation (Vim-style)

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

| Key | Action |
|-----|--------|
| _(type)_ | Build query (search fires after ≥3 characters) |
| `↑` / `↓` | Move result cursor |
| `Enter` | Open channel / start DM / join channel |
| `Backspace` | Delete last character |
| `Esc` | Exit search, return to channel list |

### Chat pane

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor to next (newer) message |
| `k` / `↑` | Move cursor to previous (older) message |
| `Ctrl+U` | Scroll viewport up half page; cursor clamps to visible area |
| `Ctrl+D` | Scroll viewport down half page; cursor clamps to visible area |
| `Ctrl+B` | Page up; cursor clamps |
| `Ctrl+F` | Page down; cursor clamps |
| `gg` | Cursor to oldest loaded message, scroll to top |
| `G` | Cursor to newest message, scroll to bottom |
| `r` | Jump cursor to first unread message (unread marker); no-op if all read |
| `o` or `l` | Open attachment(s) of cursor message; no-op if no files |
| `h` | Close attachment picker (if open) |

> Moving the cursor to the **top of the loaded buffer** automatically loads the previous page of messages.

### Message input

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | Insert newline (multi-line messages) |

### Focus

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus: Sidebar → Chat → Input |
| `i` or `a` | Jump to message input |
| `Esc` | Return focus to sidebar (also dismisses error banner) |

### Global

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit from anywhere |
| `?` | Show keybinding help overlay |

---

## Real-time Messaging

- WebSocket connection established after authentication (`wss://netchat.viettel.vn/api/v4/websocket`)
- Auth challenge sent as the first message
- **Auto-reconnect**: if the connection drops, exponential backoff reconnect (2s → 4s → … → 30s max)
- Events handled:
  - `posted`: new message — append to chat if current channel, else increment sidebar unread badge
  - `post_edited`: update existing message in chat
  - `channel_viewed`: clear unread badge for the viewed channel
- Own messages are skipped from WebSocket (already shown via send confirmation)

---

## Mute / Unmute

- Mute state synced from server `notify_props` per channel member
- Muted = `notify_props.mark_unread == "mention"` (Mattermost convention)
- Muted channels: distinct icon, dimmed style, unread count still shown, excluded from tmux title counter

---

## File Cache

- Downloaded attachments are stored at `$TMPDIR/netchat-tui/<file_id>.<ext>` (e.g. `/tmp/netchat-tui/abc123.pdf`)
- Before downloading, the app checks if the file already exists at that path; if so, it opens directly (no re-download)
- Files are **never deleted automatically** by the app in this version; cache management (size limits, eviction) is a future feature
- The OS default application is used to open files:
  - macOS: `open <path>`
  - Linux: `xdg-open <path>`
  - Windows: `explorer <path>` (or `start` via `cmd /c start`)
  - Detected at runtime via `runtime.GOOS`
- Opening happens in a background goroutine (non-blocking); the chat pane remains interactive

---

## tmux Window Title Integration

- When running inside tmux (`$TMUX` is set), the app:
  1. Disables `automatic-rename` for the current window at startup (prevents tmux from continuously overriding the custom title)
  2. Updates the tab title by calling `tmux rename-window <title>` directly on every unread change
  3. Title format: `netchat-tui [<total_msgs>/<channel_count>]` when there are unmuted unreads, else `netchat-tui`
  4. On exit: resets title to `netchat-tui` and restores `automatic-rename on` for the window
- When NOT inside tmux, falls back to `tea.SetWindowTitle` (OSC 2 escape sequence) for any OSC 2-capable terminal
- No tmux configuration required; all changes are per-window (non-global)

---

## Configuration

File: `~/.config/netchat-tui/config.json` (macOS: `~/Library/Application Support/netchat-tui/config.json`)

| Field | Default | Description |
|-------|---------|-------------|
| `token` | — | Bearer token (written by auth prompt) |
| `user_id` | — | Mattermost user ID (written automatically) |
| `sidebar_limit` | `200` | Maximum channels shown in the sidebar |

---

## API Reference (Mattermost v4)

| Purpose | Method | Endpoint |
|---------|--------|----------|
| Validate token / get self | GET | `/api/v4/users/me` |
| Get team list | GET | `/api/v4/users/me/teams` |
| Get channels (per team) | GET | `/api/v4/users/me/teams/{team_id}/channels` |
| Get channel member info | GET | `/api/v4/channels/{channel_id}/members/me` |
| Get posts | GET | `/api/v4/channels/{channel_id}/posts?page=N&per_page=60` |
| Send message | POST | `/api/v4/posts` |
| Mark channel viewed | POST | `/api/v4/channels/members/me/view` |
| Search users | GET | `/api/v4/users/search` |
| Search channels | GET | `/api/v4/channels/search` |
| Create DM channel | POST | `/api/v4/channels/direct` |
| Join channel | POST | `/api/v4/channels/{channel_id}/members` |
| Get user info | GET | `/api/v4/users/{user_id}` |
| Get file info | GET | `/api/v4/files/{file_id}/info` |
| Download file | GET | `/api/v4/files/{file_id}` |
| WebSocket | WS | `/api/v4/websocket` |

---

## Distribution

- **GitHub Releases** via GoReleaser v2 + GitHub Actions
- Triggered automatically on `v*` git tags
- Cross-compiled artifacts:
  - `linux/amd64`, `linux/arm64`
  - `darwin/amd64`, `darwin/arm64`
  - `windows/amd64`
- Archives: `.tar.gz` for unix, `.zip` for Windows; `checksums.txt` included
- Install options:
  1. Download pre-built binary from Releases page
  2. `go install github.com/thucdx/netchat-tui@latest`
  3. Build from source: `go build -o netchat-tui .`

---

## Agent Collaboration

Five agents work together (see `CLAUDE.md` for full protocol):
- **orchestrator** — breaks tasks, coordinates agents, escalates to user
- **ws-dev** — WebSocket layer, API client, real-time event handling
- **ui-dev** — TUI layout, Lipgloss styles, Bubbletea models
- **qa** — writes and runs tests, reports bugs
- **reviewer** — code review + security review, final gate before merge

---

## Decisions Log

> Decisions made by agents or by the user during escalation are recorded here.

### 2026-03-22 USER — Multi-agent collaboration model
**Context**: How to structure the development process for quality assurance.
**Decision**: Use five specialized agents (orchestrator, ws-dev, ui-dev, qa, reviewer) with a defined communication and escalation protocol. Unresolved disagreements escalate to the user. All decisions are documented here.
**Reason**: Separation of concerns — each agent focuses on one dimension of quality without stepping on others.

### 2026-03-23 USER — Remove jump-to-top in chat (reversed)
**Context**: Reviewer flagged that `gg` (jump to top) requires a state machine.
**Decision (original)**: Remove jump-to-top entirely from chat. Only `G` (jump to bottom) allowed.
**Reversal**: `gg` was later implemented in the chat pane as "jump to oldest loaded message" (does NOT load all history, just scrolls viewport to top of current buffer). `G` jumps to latest.
**Reason**: Vim muscle memory; useful even with partial history loaded.

### 2026-03-23 USER — Integration test target channels
**Context**: Integration tests that call the real netchat API must not accidentally post to real work channels.
**Decision**: DM tests may only target user `thucdx`. Group/channel tests may only post to "PT sieu UD netchat". All other channels are off-limits for automated test messages.
**Reason**: Prevent test noise from appearing in real team conversations.

### 2026-03-22 USER — Authentication approach
**Context**: SSO login requires a browser redirect that cannot be automated in a TUI.
**Decision**: User obtains token via Personal Access Tokens page or browser DevTools; pastes into TUI prompt. Token stored in `~/.config/netchat-tui/config.json`.
**Reason**: Reverse-engineering the SSO portal is out of scope and fragile.

### 2026-03-23 USER — Display name toggle
**Context**: Users have both a "contact name" (First + Last) and an "account name" (username).
**Decision**: Default to contact name. Press `n` to toggle to account name. Toggle applies everywhere: sidebar labels and chat message headers. Falls back to username if contact name is empty.
**Reason**: Contact name is more human-readable; power users may prefer username.

### 2026-03-23 USER — tmux window title unread indicator
**Context**: User runs the app inside tmux (Ghostty) and wants unread counts visible in the tab title.
**Decision**: Use `tmux rename-window` directly when inside tmux (detected via `$TMUX`). Disable `automatic-rename` for the current window at startup so tmux doesn't fight the custom title. Fall back to `tea.SetWindowTitle` (OSC 2) outside tmux. Title format: `netchat-tui [msgs/channels]`.
**Reason**: `automatic-rename` overrides OSC 2-based renames; calling `tmux rename-window` directly is reliable regardless of tmux config. Per-window changes avoid affecting other tmux windows.

### 2026-03-24 USER — Chat message cursor + attachment open
**Context**: User wants to open file attachments from the chat pane. Requires knowing which message is targeted.
**Decision**: Add a permanent per-message cursor to the chat pane (visible whenever chat has focus). `j/k` move the cursor; `Ctrl+U/D/B/F` scroll the viewport (cursor clamps). `o`/`l` open attachments of the cursor message; multi-file messages show an inline picker. Non-image files are now rendered as `📎 name (size)` lines. Files are downloaded to `$TMPDIR/netchat-tui/` and cached permanently (cache management deferred). Cursor defaults to newest message on channel open; `r` jumps to first unread (using `ChannelMember.LastViewedAt`). An `──── unread ────` divider is injected above the first unread post.
**Reason**: A per-message cursor is the prerequisite for all future chat-level features (image popup, text selection, copy). Doing it once correctly rather than per feature.

### 2026-03-24 USER — WebSocket reconnect on connection drop
**Context**: WebSocket connections were silently dropped (server timeout, network issue), causing `waitForWSEvent` to block forever on `ws.Events` since `readLoop` only closes `ws.done`, not `ws.Events`.
**Decision**: Add `Done() <-chan struct{}` to `WSClient`; update `waitForWSEvent` to `select` on both `ws.Events` and `ws.Done()`; handle `WSDisconnectedMsg` with exponential-backoff reconnect via `ConnectWithRetry`.
**Reason**: Without reconnect, the app silently stops receiving messages after the first connection drop.

---

## Integration Test Constraints

When running integration tests that make **real API calls** to netchat.viettel.vn:

- **Direct Messages**: only send test messages to `thucdx` (yourself)
- **Group/Channel messages**: only allowed to post to the channel named **"PT sieu UD netchat"**
- **All other channels and groups are strictly off-limits** for any test-generated messages
- Unit tests (using `httptest.NewServer`) are unaffected — they never touch the real server

---

## Out of Scope (current version)

- Emoji reactions
- Thread replies
- User status changes (online/away/offline indicators)
- Sending OTP / automating SSO browser flow
- File/image uploads (viewing only, not uploading)
