# netchat-tui TODO

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked/escalated

## Phase Gate Checklist (repeat for each phase before moving on)

- [ ] Coder: all phase tasks implemented
- [ ] Tester: all phase tests pass (`go test ./...`)
- [ ] Security: no High severity findings open
- [ ] Reviewer: no blocking issues open
- [ ] Orchestrator: TODO.md updated

---

## Security

- [ ] `config/config.go` — create config dir with `0700`, auth.json with `0600`
- [ ] `tui/auth.go` — use `EchoPassword` mode for token input, clear input string after save
- [ ] `api/client.go` — assert `https://` scheme in `NewClient()`, abort if not
- [ ] `api/websocket.go` — assert `wss://` scheme before dialing, abort if not
- [ ] `api/client.go` — ensure raw token never appears in wrapped error messages
- [ ] `tui/chat/view.go` — strip ANSI escape sequences from all incoming message text before rendering
- [ ] `tui/input/model.go` — enforce 4000 char max on outgoing messages
- [ ] `security_test.go` — file permissions (0600/0700), scheme rejection, token not in errors, ANSI strip, input length cap

---

## Phase 1 — Foundation and Auth

- [x] Initialize Go module and add dependencies (bubbletea, lipgloss, bubbles, gorilla/websocket)
- [x] `config/config.go` — AuthConfig struct, Load() / Save() to ~/.config/netchat-tui/auth.json
- [x] `api/client.go` — HTTP client wrapper with Bearer token injection
- [x] `tui/auth.go` — token paste screen, validate via GET /api/v4/users/me, save on success
- [x] `main.go` — entry point: load config → auth screen if no token → launch main app

## Phase 2 — API Layer

- [x] `api/models.go` — User, Team, Channel, ChannelMember, Post, PostList structs
- [ ] `api/teams.go` — GetTeamsForUser()
- [ ] `api/channels.go` — GetChannelsForUser(), GetChannelMembersForUser(), GetPreferences(), MarkChannelRead()
- [ ] `api/posts.go` — GetPostsForChannel(), CreatePost(), GetPostsSince()
- [ ] `api/websocket.go` — connect, auth challenge, event parsing, reconnect loop

## Phase 3 — Core TUI Shell

- [ ] `tui/styles/styles.go` — all Lipgloss color/border definitions
- [ ] `internal/keymap/keymap.go` — vim keybinding definitions
- [ ] `tui/layout.go` — pane size calculation from terminal dimensions
- [ ] `tui/app.go` — root AppModel, three-pane layout renders with static data

## Phase 4 — Sidebar

- [ ] `tui/sidebar/model.go` — ChannelItem, cursor, virtual scroll, ChannelSelectedMsg
- [ ] `tui/sidebar/view.go` — icons, unread badges, muted style, section headers

## Phase 5 — Chat Viewport

- [ ] `tui/chat/model.go` — viewport wrapper, fetch posts on channel select, userCache
- [ ] `tui/chat/view.go` — message grouping, timestamps, system messages, channel header
- [ ] Loading spinner while fetching messages

## Phase 6 — Input Box

- [ ] `tui/input/model.go` — textarea wrapper, Enter to send, Shift+Enter for newline
- [ ] Wire SendMessageMsg in AppModel — call CreatePost, append to chat, scroll to bottom
- [ ] Disable input while send is in-flight

## Phase 7 — WebSocket Real-Time

- [ ] WebSocket goroutine → buffered channel → tea.Cmd pump into Bubbletea loop
- [ ] Handle `posted` event — append to active chat or increment sidebar unread
- [ ] Handle `post_edited` event — update post in list
- [ ] Handle `channel_viewed` event — clear unread badge
- [ ] Muted channel rule — increment count, no highlight

## Testing — Phase 1 (Auth & Config)

- [x] `config_test.go` — Save/Load roundtrip, missing file returns empty config
- [x] `client_test.go` — Bearer header injection, non-2xx returns error
- [x] `auth_test.go` (integration) — valid token passes, invalid token shows error

## Testing — Phase 2 (API Layer)

- [x] `models_test.go` — JSON unmarshal for Channel types, double-unmarshal for WS posted event
- [ ] `teams_test.go` (integration) — GetTeamsForUser returns at least one team
- [ ] `channels_test.go` (integration) — GetChannelsForUser, GetChannelMembersForUser
- [ ] `channels_test.go` — mute detection logic (mark_unread == "mention")
- [ ] `posts_test.go` (integration) — GetPostsForChannel order, CreatePost
- [ ] `websocket_test.go` — auth challenge JSON, posted event parse, unknown event no panic

## Testing — Phase 3 (Layout)

- [ ] `layout_test.go` — sidebar/chat/input dimensions, minimum terminal size

## Testing — Phase 4 (Sidebar)

- [ ] `sidebar_model_test.go` — j/k cursor, gg/G jump, virtual scroll, ChannelSelectedMsg, DM name resolution
- [ ] `sidebar_view_test.go` — muted icon, unread badge styles, rendered width within bounds

## Testing — Phase 5 (Chat Viewport)

- [ ] `chat_model_test.go` — posts cleared on channel switch, chronological order, message grouping
- [ ] `chat_view_test.go` — timestamp formats, system message style, unknown user graceful

## Testing — Phase 6 (Input)

- [ ] `input_model_test.go` — Enter sends, clears textarea; Shift+Enter newline; disabled while in-flight

## Testing — Phase 7 (WebSocket)

- [ ] `app_test.go` — posted to active/inactive/muted channel, channel_viewed clears badge
- [ ] `websocket_test.go` — reconnect backoff caps at 30s

## Testing — Phase 8 (Polish)

- [ ] `sidebar_model_test.go` — pagination triggers load-more on scroll to top
- [ ] `app_test.go` — WindowSizeMsg propagates to all sub-models
- [ ] `utils_test.go` — time formatting edge cases

## Phase 8 — Polish and Edge Cases

- [ ] DM display name resolution — batch fetch via POST /api/v4/users/ids, cache
- [ ] Mute detection — parse preferences, mark IsMuted per channel
- [ ] Pagination — load more posts when viewport scrolled to top
- [ ] Mark-as-read — call view endpoint on channel switch / viewport at bottom
- [ ] Terminal resize — re-render on WindowSizeMsg, propagate to all sub-models
- [ ] Error banner — dismissable error display in chat pane
