# netchat-tui Requirements

A TUI (Terminal User Interface) chat client for netchat.viettel.vn, designed to run inside a tmux session.

---

## Tech Stack

- **Language**: Go
- **TUI Framework**: Bubbletea (event loop, state management)
- **Styling**: Lipgloss (colors, borders, layout)
- **Components**: Bubbles (list, viewport, textinput, spinner)
- **HTTP Client**: Go standard `net/http`
- **WebSocket**: `gorilla/websocket` or `nhooyr.io/websocket`
- **Config/Token storage**: `encoding/json` (standard library)

---

## Authentication

- **Method**: SSO (Viettel corporate portal)
- **Flow**:
  1. On first launch (or when token is expired/missing), TUI instructs the user to open netchat.viettel.vn in their browser and complete SSO login
  2. User copies `MMAUTHTOKEN` cookie value from browser DevTools and pastes it into the TUI
  3. TUI validates the token by calling `GET /api/v4/users/me`
  4. On success, token is stored locally for future sessions
- **Token Storage**: `~/.config/netchat-tui/auth.json`
  - Stores: `MMAUTHTOKEN`, `MMUSERID`, `MMCSRF`, and user info
- **Subsequent launches**: TUI reads stored token and starts directly in the chat view

---

## Layout

```
┌─────────────────┬──────────────────────────────────────┐
│  Channel List   │  Chat Window                         │
│                 │                                      │
│ # general    3  │  [username] 10:30                    │
│ # random        │  Hello everyone!                     │
│ 🔇 announcem 2  │                                      │
│ @ john.doe   1  │  [you] 10:31                         │
│                 │  Hi!                                 │
│                 │                                      │
│                 │                                      │
│                 ├──────────────────────────────────────┤
│                 │ > type message here...               │
└─────────────────┴──────────────────────────────────────┘
```

- Left panel: channel list (fixed width ~25 chars)
- Right top: scrollable chat/message viewport
- Right bottom: message input box

---

## Channel List

- **Source**: Single team (no team switching needed)
- **Channel types shown**: All — Direct Messages (`D`), Open channels (`O`), Private groups (`P`)
- **Unread count**: Show number of unread messages per channel
- **Muted channels**:
  - Show muted icon (🔇) next to channel name
  - Show unread message count (so user knows messages exist)
  - Do NOT trigger notifications
- **Unmuted channels with new messages**: Show notification (visual highlight or badge)
- **Channel icons**:
  - `#` for Open channels
  - `🔒` for Private groups
  - `@` for Direct Messages
  - `🔇` prefix for muted channels

---

## Chat Window

- Scrollable message history (using `bubbles/viewport`)
- Messages show: username, timestamp, message content
- Load recent messages on channel switch
- Real-time new messages appear at the bottom via WebSocket

---

## Navigation (Vim-like)

**Sidebar (channel list):**
- `j` / `k` or arrow keys — move up/down in channel list
- `Enter` — open selected channel

**Chat window scrolling:**
- `k` / `↑` — scroll up one line
- `j` / `↓` — scroll down one line
- `Ctrl+u` — scroll up half page
- `Ctrl+d` — scroll down half page
- `Ctrl+b` — page up
- `Ctrl+f` — page down
- `G` — jump to bottom (latest messages)
- No jump-to-top: jumping to the very top of chat would skip too many messages

**Global:**
- `i` or `a` — focus message input (only when sidebar or chat is focused)
- `Esc` — return focus to channel list from input
- `q` — quit app (only when sidebar is focused, not during typing)
- `Tab` — switch focus between panels
- `?` — show keybinding help

---

## Real-time Messaging

- Connect to netchat WebSocket after authentication
- Listen for new message events
- Update channel unread counts in sidebar on new messages
- Append new messages to chat window if the channel is currently open
- Notify (highlight channel in sidebar) for unmuted channels with new messages

---

## Mute / Unmute

- Respect the mute state from netchat (synced from server `notify_props`)
- Muted = `notify_props.mark_unread == "mention"` (Mattermost convention)
- Muted channels: show 🔇 icon, show unread count, suppress notifications
- Unmuted channels: show normal icon, show unread count, trigger notification highlight

---

## API Reference (netchat.viettel.vn — Mattermost v4 API)

| Purpose | Method | Endpoint |
|---------|--------|----------|
| Validate token / get self | GET | `/api/v4/users/me` |
| Get team list | GET | `/api/v4/users/me/teams` |
| Get channel list | GET | `/api/v4/users/me/teams/{team_id}/channels` |
| Get unread counts | GET | `/api/v4/users/me/teams/unread` |
| Get messages | GET | `/api/v4/channels/{channel_id}/posts` |
| Mark channel viewed | POST | `/api/v4/channels/members/me/view` |
| Get channel member info | GET | `/api/v4/channels/{channel_id}/members/me` |
| WebSocket | WS | `/api/v4/websocket` |

---

## Agent Collaboration

Four agents work together to build this project (see `AGENTS.md` for full protocol):
- **Coder** — implements features
- **Tester** — writes and runs tests, reports to Coder
- **Reviewer** — reviews code quality and architecture
- **Security** — reviews against the security plan

A phase only advances when all four agents have signed off. Unresolved disagreements are escalated to the user.

---

## Decisions Log

> Decisions made by agents or by the user during escalation are recorded here.

### 2026-03-22 USER — Multi-agent collaboration model
**Context**: How to structure the development process for quality assurance.
**Decision**: Use four specialized agents (Coder, Tester, Reviewer, Security) with a defined communication and escalation protocol. Unresolved disagreements escalate to the user. All decisions are documented here.
**Reason**: Separation of concerns — each agent focuses on one dimension of quality without stepping on others.

### 2026-03-23 USER — Remove jump-to-top, keep vim scroll bindings
**Context**: Reviewer flagged that `gg` (jump to top) requires a state machine and jumping to top of chat skips too many messages.
**Decision**: Remove jump-to-top entirely. Chat scrolls with `k/j`, `Ctrl+U/D`, `Ctrl+B/F`. `G` jumps to bottom (latest messages) only. `q` to quit only fires when sidebar is focused, not during text input.
**Reason**: In a chat window, jumping to the very top is impractical. Vim-style incremental scroll is sufficient.

### 2026-03-23 USER — Integration test target channels
**Context**: Integration tests that call the real netchat API must not accidentally post to real work channels.
**Decision**: DM tests may only target user `thucdx`. Group/channel tests may only post to "PT sieu UD netchat". All other channels are off-limits for automated test messages.
**Reason**: Prevent test noise from appearing in real team conversations.

### 2026-03-22 USER — Authentication approach
**Context**: SSO login requires a browser redirect that cannot be automated in a TUI.
**Decision**: User completes SSO login in browser, copies `MMAUTHTOKEN` cookie value, pastes into TUI once. Token stored in `~/.config/netchat-tui/auth.json`.
**Reason**: Reverse-engineering the Viettel SSO portal is out of scope and fragile.

---

## Integration Test Constraints

When running integration tests that make **real API calls** to netchat.viettel.vn:

- **Direct Messages**: only send test messages to `thucdx` (yourself)
- **Group/Channel messages**: only allowed to post to the channel named **"PT sieu UD netchat"**
- **All other channels and groups are strictly off-limits** for any test-generated messages
- Unit tests (using `httptest.NewServer`) are unaffected — they never touch the real server

---

## Out of Scope (for now)

- File/image uploads or previews
- Emoji reactions
- Thread replies
- Multiple team support
- Search
- User status changes
- Sending OTP / automating SSO browser flow
