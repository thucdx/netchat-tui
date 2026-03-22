# netchat-tui Implementation Plan

## Project Structure

```
netchat-tui/
├── main.go                          # Entry point, init config, launch TUI
├── go.mod
├── go.sum
│
├── config/
│   └── config.go                    # Load/save ~/.config/netchat-tui/auth.json
│
├── api/
│   ├── client.go                    # HTTP client wrapper, auth header injection
│   ├── models.go                    # API response structs (User, Channel, Post, Team)
│   ├── channels.go                  # Channel list, members, preferences
│   ├── posts.go                     # Fetch posts, send message, mark read
│   ├── teams.go                     # Get user's team
│   └── websocket.go                 # WebSocket connection, event parsing
│
├── tui/
│   ├── app.go                       # Root Bubbletea model (AppModel), top-level Update/View
│   ├── auth.go                      # Auth screen model (token paste flow)
│   ├── layout.go                    # Layout calculation (sizes, pane split)
│   ├── sidebar/
│   │   ├── model.go                 # Sidebar model, channel list state
│   │   └── view.go                  # Sidebar rendering with Lipgloss
│   ├── chat/
│   │   ├── model.go                 # Chat viewport model, message state
│   │   └── view.go                  # Message rendering, timestamps, usernames
│   ├── input/
│   │   ├── model.go                 # Input box model (wraps bubbles/textarea)
│   │   └── view.go                  # Input rendering
│   └── styles/
│       └── styles.go                # All Lipgloss style definitions (single source of truth)
│
└── internal/
    ├── keymap/
    │   └── keymap.go                # Vim-like keybinding definitions
    └── utils/
        └── time.go                  # Time formatting helpers
```

---

## Implementation Phases

### Phase 1 — Foundation and Auth

Goal: Get auth working and a skeleton app that launches.

1. Initialize Go module, add dependencies:
   - `github.com/charmbracelet/bubbletea`
   - `github.com/charmbracelet/lipgloss`
   - `github.com/charmbracelet/bubbles`
   - `github.com/gorilla/websocket`
2. `config/config.go`: `AuthConfig` struct with `Token` and `UserID`, implement `Load()` / `Save()` using `os.UserConfigDir()`, store as JSON.
3. `api/client.go`: HTTP client wrapper, injects `Authorization: Bearer <token>` on every request.
4. `tui/auth.go`: screen shown when no token found — textinput bubble for pasting token, validates via `GET /api/v4/users/me`, saves config, transitions to main app.
5. `main.go`: load config → if no token, run auth screen → launch main app.

Deliverable: App launches, prompts for token, validates, saves, starts.

---

### Phase 2 — API Layer

Goal: All data-fetching logic in place before building UI.

1. `api/models.go` — define Go structs:
   - `User`: ID, Username, FirstName, LastName, Nickname
   - `Team`: ID, Name, DisplayName
   - `Channel`: ID, Name, DisplayName, Type (D/O/P), TotalMsgCount, LastPostAt
   - `ChannelMember`: UserID, ChannelID, MsgCount, MentionCount, NotifyProps
   - `Post`: ID, ChannelID, UserID, Message, CreateAt, Type
   - `PostList`: Order `[]string`, Posts `map[string]Post`

2. `api/teams.go`: `GetTeamsForUser(userID)` — returns first team.

3. `api/channels.go`:
   - `GetChannelsForUser(userID, teamID)`
   - `GetChannelMembersForUser(userID, teamID)` — batch, one call for all unread counts
   - `GetPreferences(userID)` — detect muted channels
   - `MarkChannelRead(channelID, userID)`

4. `api/posts.go`:
   - `GetPostsForChannel(channelID, page, perPage)`
   - `CreatePost(channelID, message)`
   - `GetPostsSince(channelID, timestamp)` — incremental refresh

5. `api/websocket.go`:
   - Connect to `wss://netchat.viettel.vn/api/v4/websocket`
   - Send auth challenge as first message: `{"seq":1,"action":"authentication_challenge","data":{"token":"<TOKEN>"}}`
   - Reconnect loop with exponential backoff (2s → 30s max)
   - Parse events into typed `WSEvent` struct

---

### Phase 3 — Core TUI Shell

Goal: Three-pane layout with static placeholder content.

1. `tui/styles/styles.go`: define all colors/borders in one place. Theme struct: `SidebarBg`, `ActiveChannel`, `UnreadChannel`, `MutedChannel`, `MessageUsername`, `InputBorder`, etc.
2. `internal/keymap/keymap.go`: vim keybindings — `j/k` navigate, `gg/G` top/bottom, `Enter` select, `i` focus input, `Esc` back to sidebar, `Ctrl+U/D` scroll chat, `q` quit.
3. `tui/layout.go`: compute pane sizes from `tea.WindowSizeMsg`. Sidebar = 28 chars fixed, input = 3 lines fixed, chat = remaining.
4. `tui/app.go`: root `AppModel` — holds `sidebar`, `chat`, `input` sub-models, `focus FocusPane` enum, `api *api.Client`, `wsEvents chan WSEvent`. Renders with `lipgloss.JoinHorizontal` + `lipgloss.JoinVertical`.

---

### Phase 4 — Sidebar

Goal: Channel list with icons, unread counts, mute state, vim navigation.

1. `tui/sidebar/model.go`:
   - `[]ChannelItem` — wraps `api.Channel` + `UnreadCount`, `IsMuted`, `DisplayName`
   - Virtual scrolling (track `viewOffset` when list > visible height)
   - `cursor int` (highlighted), `selected int` (active)
   - Emits `ChannelSelectedMsg` on Enter

2. `tui/sidebar/view.go`:
   - Section headers: "DIRECT MESSAGES", "CHANNELS"
   - Per channel: `[icon] [name] [unread badge]`
   - Icons: `🔇` muted, `#` open, `🔒` private, `@` direct message
   - Unread badge right-aligned; dimmed style for muted, highlighted for unmuted
   - Cursor row highlighted with `ActiveChannel` style

---

### Phase 5 — Chat Viewport

Goal: Fetch and display messages for selected channel.

1. `tui/chat/model.go`:
   - Wraps `bubbles/viewport`
   - On `ChannelSelectedMsg`: fire cmd to fetch posts
   - On `PostsLoadedMsg`: render posts, call `viewport.SetContent()`
   - `userCache map[string]User` — resolve usernames without re-fetching

2. `tui/chat/view.go`:
   - Group consecutive messages from same user (only first shows username+timestamp)
   - Timestamps: "Today HH:MM", "Yesterday HH:MM", "DD/MM HH:MM"
   - System messages (join/leave) rendered dimmed
   - Channel header at top (name + topic)
   - Spinner shown while loading

3. Auto-scroll to bottom on new message in active channel.

---

### Phase 6 — Input Box

Goal: Type and send messages.

1. `tui/input/model.go`:
   - Wraps `bubbles/textarea`
   - `Enter` (without Shift) → emit `SendMessageMsg`, clear textarea
   - `Shift+Enter` → newline
2. `AppModel` handles `SendMessageMsg`: calls `api.CreatePost()` as cmd, appends returned post to chat, scrolls to bottom.
3. Disable input while send is in-flight to prevent double-send.

---

### Phase 7 — WebSocket Real-Time Updates

Goal: Live messages and unread badge updates.

1. WebSocket goroutine writes events to a buffered `chan WSEvent`. A long-running `tea.Cmd` blocks on that channel and pumps events into the Bubbletea loop one at a time.
2. Event handling in `AppModel.Update()`:
   - `posted`: if active channel → append to chat + scroll; else → increment sidebar unread
   - `post_edited`: update post in list
   - `channel_viewed`: clear unread badge for that channel
3. Muted channel rule: increment unread count but do NOT highlight/flash sidebar entry.

---

### Phase 8 — Polish and Edge Cases

1. **DM display names**: split `userA__userB`, fetch other user via `POST /api/v4/users/ids` (batch on startup), cache results.
2. **Mute detection**: `GET /api/v4/users/<id>/preferences`, category `channel_notifications`, value `mark_unread: "mention"` = muted.
3. **Pagination**: when `viewport.AtTop()` → fetch next page of posts, prepend to list.
4. **Mark-as-read**: on channel switch (or viewport at bottom) → call `POST /api/v4/channels/<id>/members/<userID>/view`.
5. **Resize handling**: re-render all messages on `tea.WindowSizeMsg`, propagate to all sub-models.
6. **Error banner**: dismissable error display at top of chat pane.

---

## Key Data Models

```go
// Config
AuthConfig { Token, UserID string }

// App state
ChannelItem {
    Channel      api.Channel
    UnreadCount  int
    MentionCount int
    IsMuted      bool
    DisplayName  string   // resolved (DM → @username)
}

FocusPane  // enum: SidebarFocused | ChatFocused | InputFocused
```

### Bubbletea Messages (cross-component events)

```go
ChannelSelectedMsg  { ChannelID string }
PostsLoadedMsg      { ChannelID string, Posts []api.Post }
NewPostMsg          { Post api.Post }
SendMessageMsg      { ChannelID string, Text string }
WSEventMsg          { Event api.WSEvent }
AuthSuccessMsg      { Token string, UserID string }
ErrorMsg            { Err error }
```

---

## Component Communication Flow

```
AppModel.Update()
  │
  ├── tea.KeyMsg ──────────────► route to focused pane (sidebar / chat / input)
  ├── ChannelSelectedMsg ──────► update chat.activeChannelID, fire PostsLoadedCmd
  ├── PostsLoadedMsg ──────────► pass to chat.Model
  ├── NewPostMsg ──────────────► active channel → append to chat; else → sidebar unread++
  ├── WSEventMsg ──────────────► dispatch by event type
  ├── SendMessageMsg ──────────► fire CreatePostCmd, clear input
  └── tea.WindowSizeMsg ───────► propagate to all sub-models

AppModel.View()
  sidebar.View()  |  chat.View()
  ──────────── input.View() ────────────
```

Sub-models never call the API directly. They return `tea.Cmd` closures. The API client is owned by `AppModel`.

---

## Gotchas to Watch Out For

| # | Issue | Solution |
|---|-------|----------|
| 1 | `posted` WS event: `data.post` is a JSON **string** inside JSON | Double `json.Unmarshal` |
| 2 | WS auth challenge must be the **first** message sent | Send synchronously before reading any events |
| 3 | Never mutate model state from goroutines | All goroutines communicate via `tea.Cmd` → `tea.Msg` |
| 4 | `viewport` stores pre-rendered string — resize breaks layout | Keep raw `[]Post` as source of truth, re-render on `WindowSizeMsg` |
| 5 | Unread count = `Channel.TotalMsgCount - ChannelMember.MsgCount` | Fetch all member records in one batch call |
| 6 | Muted channel: show unread count but no highlight | Check `IsMuted` before applying unread style |
| 7 | Emoji width varies by terminal | Use `lipgloss.Width()` not `len()` for all alignment |
| 8 | Bearer token = raw `MMAUTHTOKEN` cookie value | Use `Authorization: Bearer <token>` header, no cookie handling needed |

---

## Security Plan

### Threat Model

The app handles a corporate authentication token that grants full access to the user's netchat account. The main risks are: token theft from disk, token leakage via logs or process inspection, terminal injection via malicious message content, and insecure transport.

---

### Security Requirements

| # | Area | Requirement |
|---|------|-------------|
| S1 | Token storage | `auth.json` must be created with file permissions `0600` (owner read/write only) |
| S2 | Config directory | `~/.config/netchat-tui/` created with permissions `0700` (owner only) |
| S3 | Transport | All HTTP calls use `https://` — never allow plain `http://` fallback |
| S4 | Transport | All WebSocket connections use `wss://` — never allow plain `ws://` |
| S5 | TLS verification | Never skip TLS certificate verification (`InsecureSkipVerify` must stay `false`) |
| S6 | Token in logs | Token must never appear in log output, error messages, or stack traces |
| S7 | Token in memory | Token is stored as a plain string (Go has no secure memory wipe, document this limitation) |
| S8 | Terminal injection | All message content must be stripped of ANSI escape sequences before rendering to prevent terminal injection attacks |
| S9 | Input length | Outgoing messages capped at 4000 chars (Mattermost limit) — enforced in the input model |
| S10 | Process args | Token is never passed as a command-line argument (visible in `ps aux`) — only read from file or stdin prompt |
| S11 | Clipboard | Pasted token is not echoed to terminal during input (use masked input or clear after validation) |

---

### Security Implementation Tasks (per phase)

**Phase 1 — Auth & Config**
- Set `0700` on config directory, `0600` on `auth.json` at creation time
- Use `bubbles/textinput` with `EchoPassword` mode when accepting the token paste so it is not visible on screen
- After token is validated and saved, clear the in-memory input string
- Verify HTTPS scheme before making any API call; abort with a clear error if not

**Phase 2 — API Layer**
- Assert `baseURL` scheme is `https` in `NewClient()`, return error if not
- Assert WebSocket URL scheme is `wss` before dialing, return error if not
- Ensure error messages from API never include the raw token (wrap errors, don't propagate raw request details)
- TLS config: use default `http.Transport` (TLS verification enabled by default in Go)

**Phase 8 — Polish**
- Strip ANSI escape sequences from all incoming message text before passing to `lipgloss` renderer
- Add a `security_test.go` to verify the above properties (see Testing Plan)

---

### Security Testing

| Test | Type | What to verify |
|------|------|----------------|
| `security_test.go` | Unit | `auth.json` is created with mode `0600` |
| `security_test.go` | Unit | Config dir is created with mode `0700` |
| `security_test.go` | Unit | `NewClient()` rejects `http://` base URL |
| `security_test.go` | Unit | WebSocket dialer rejects `ws://` URL |
| `security_test.go` | Unit | Error messages do not contain the token string |
| `security_test.go` | Unit | ANSI escape sequences in message text are stripped before render |
| `security_test.go` | Unit | Message longer than 4000 chars is rejected by input model |

---

## Testing Plan

### Strategy

- **Unit tests**: pure logic with no I/O — models, parsers, formatters
- **Integration tests**: real HTTP calls against netchat.viettel.vn (requires valid token in env)
- **Manual TUI tests**: run the app in a terminal and verify visually (no automated TUI testing)

Integration tests are skipped in CI if `NETCHAT_TOKEN` env var is not set.

---

### Phase 1 — Auth & Config

| Test | Type | What to verify |
|------|------|----------------|
| `config_test.go` | Unit | `Save()` writes valid JSON; `Load()` reads it back correctly |
| `config_test.go` | Unit | `Load()` on missing file returns empty `AuthConfig`, no error |
| `client_test.go` | Unit | Client injects `Authorization: Bearer <token>` header on every request |
| `client_test.go` | Unit | Client returns error on non-2xx HTTP status |
| `auth_test.go` | Integration | Valid token passes `GET /api/v4/users/me` and returns user info |
| `auth_test.go` | Integration | Invalid token returns 401 and shows error in auth screen |

---

### Phase 2 — API Layer

| Test | Type | What to verify |
|------|------|----------------|
| `models_test.go` | Unit | JSON unmarshal of Channel with type D/O/P parses correctly |
| `models_test.go` | Unit | Double-unmarshal of WS `posted` event (JSON string inside JSON) works |
| `teams_test.go` | Integration | `GetTeamsForUser()` returns at least one team |
| `channels_test.go` | Integration | `GetChannelsForUser()` returns channels of all types |
| `channels_test.go` | Integration | `GetChannelMembersForUser()` returns correct MsgCount and NotifyProps |
| `channels_test.go` | Unit | Mute detection: `mark_unread == "mention"` in preferences → `IsMuted = true` |
| `posts_test.go` | Integration | `GetPostsForChannel()` returns posts in correct order |
| `posts_test.go` | Integration | `CreatePost()` sends message and returns the created post |
| `websocket_test.go` | Unit | Auth challenge JSON is correctly formed |
| `websocket_test.go` | Unit | `posted` event parsed into `WSEvent` with correct fields |
| `websocket_test.go` | Unit | Unknown event type does not panic |

---

### Phase 3 — Layout & Styles

| Test | Type | What to verify |
|------|------|----------------|
| `layout_test.go` | Unit | Sidebar width is fixed at 28; chat width = total - 28 |
| `layout_test.go` | Unit | Input height is fixed at 3; chat height = total - 3 |
| `layout_test.go` | Unit | Minimum terminal size (e.g. 80x24) produces valid (positive) dimensions |

---

### Phase 4 — Sidebar

| Test | Type | What to verify |
|------|------|----------------|
| `sidebar_model_test.go` | Unit | `j/k` moves cursor correctly, clamps at top/bottom |
| `sidebar_model_test.go` | Unit | `gg` jumps cursor to 0; `G` jumps to last item |
| `sidebar_model_test.go` | Unit | Virtual scroll: `viewOffset` advances when cursor exceeds visible height |
| `sidebar_model_test.go` | Unit | `ChannelSelectedMsg` is emitted on Enter |
| `sidebar_model_test.go` | Unit | DM channel name resolved to `@username` from userCache |
| `sidebar_view_test.go` | Unit | Muted channel renders `🔇` icon |
| `sidebar_view_test.go` | Unit | Unread badge is rendered for muted channel (dimmed style) |
| `sidebar_view_test.go` | Unit | No unread badge rendered when count is 0 |
| `sidebar_view_test.go` | Unit | Rendered width does not exceed `SidebarWidth` (use `lipgloss.Width()`) |

---

### Phase 5 — Chat Viewport

| Test | Type | What to verify |
|------|------|----------------|
| `chat_model_test.go` | Unit | On `ChannelSelectedMsg`, previous posts are cleared before loading |
| `chat_model_test.go` | Unit | Posts are rendered in chronological order (oldest at top) |
| `chat_model_test.go` | Unit | Consecutive posts from same user are grouped (no repeated username) |
| `chat_view_test.go` | Unit | Timestamp formats: today → "HH:MM", yesterday → "Yesterday HH:MM", older → "DD/MM HH:MM" |
| `chat_view_test.go` | Unit | System message (type != "") renders with dimmed style |
| `chat_view_test.go` | Unit | Unknown userID renders as "unknown" gracefully, no panic |

---

### Phase 6 — Input Box

| Test | Type | What to verify |
|------|------|----------------|
| `input_model_test.go` | Unit | Enter key emits `SendMessageMsg` with correct text |
| `input_model_test.go` | Unit | Textarea is cleared after send |
| `input_model_test.go` | Unit | Shift+Enter inserts newline, does NOT emit `SendMessageMsg` |
| `input_model_test.go` | Unit | Input is disabled (ignores Enter) while send is in-flight |

---

### Phase 7 — WebSocket Real-Time

| Test | Type | What to verify |
|------|------|----------------|
| `app_test.go` | Unit | `posted` event for active channel → post appended to chat model |
| `app_test.go` | Unit | `posted` event for inactive channel → sidebar unread count incremented |
| `app_test.go` | Unit | `posted` event for muted channel → unread incremented, no highlight applied |
| `app_test.go` | Unit | `channel_viewed` event → unread count reset to 0 for that channel |
| `websocket_test.go` | Unit | Reconnect backoff: delay doubles each attempt, caps at 30s |

---

### Phase 8 — Polish

| Test | Type | What to verify |
|------|------|----------------|
| `sidebar_model_test.go` | Unit | Pagination: scroll to top triggers load-more cmd |
| `app_test.go` | Unit | `tea.WindowSizeMsg` propagates new dimensions to sidebar, chat, and input |
| `utils_test.go` | Unit | Time formatting edge cases: midnight, year boundary |

---

### Running Tests

```bash
# Unit tests only (no token required)
go test ./...

# Integration tests (requires valid token)
NETCHAT_TOKEN=<your_token> NETCHAT_USER_ID=<your_user_id> go test ./... -tags=integration

# Verbose output
go test ./... -v
```

---



| Phase | Milestone |
|-------|-----------|
| 1 | Auth flow works, token saved, skeleton launches |
| 2 | All API calls implemented |
| 3 | Three-pane layout renders (static data) |
| 4 | Sidebar with channel list + vim navigation |
| 5 | Chat viewport loads and displays messages |
| 6 | Input box sends messages |
| 7 | WebSocket live updates + unread badges |
| 8 | DM names, mute, pagination, resize, polish |
