# Search Feature Spec

## UX Summary

- **Trigger**: `/` or `Ctrl+F` when sidebar is focused
- **Layout**: Sidebar replaced entirely by search bar + results while active
- **Input**: Simple append/backspace (cursor always at end, no left/right arrow)
- **API threshold**: 3 characters minimum before firing API calls

## Layout

```
┌────────────────────────┐
│ / general_█            │  ← line 1: query (█ = block cursor)
├────────────────────────┤
│ # general              │  ← results (j/k to navigate)
│ # general-ops          │
│ @ alice                │
│ ⊕ alice, bob           │
│ + @dave  (new DM)      │  ← from API, not yet in your list
│ + #ops-public (join)   │  ← public channel from API
└────────────────────────┘

Confirm-join state (line 1 changes):
│ Join #ops-public? [y/N]│
```

## Keybindings

| Key | Action |
|---|---|
| Any printable char (except j/k) | Append to query |
| `Backspace` | Remove last char from query |
| `j` / `↓` | Move result cursor down |
| `k` / `↑` | Move result cursor up |
| `Enter` | Select result (see below) |
| `Esc` | Exit search, return to normal sidebar |

**Enter behaviour by result type:**
- Existing channel/DM/group → select it, exit search, open chat
- New DM (user from API) → create DM channel, open chat, exit search
- New public channel (from API) → enter confirm-join state

**Confirm-join state:**
- `y` or `Enter` → join channel + open chat + exit search
- `n`, `Esc`, or any other key → cancel, return to search

## Result Scoring & Ordering (interleaved)

| Score | Condition |
|---|---|
| 3 | Prefix match on existing item DisplayName (case-insensitive) |
| 2 | Substring match on existing item |
| 1 | API result (new user / new public channel) |

Within same score: alphabetical by display name.
Unread badge shown on existing items.
API results are filtered to exclude items already in `allItems`.

## API Calls

Both fire when `len(query) >= 3`. Stale results discarded (query at response time must match current query).

- `POST /api/v4/users/search` — body `{"term":"...", "allow_inactive":false}`
- `POST /api/v4/teams/{teamID}/channels/search` — body `{"term":"..."}`
  (searched across all teams the user belongs to, results deduplicated by channel ID)

New channel operations:
- Create DM: `POST /api/v4/channels/direct` — body `["myUserID","otherUserID"]`
- Join channel: `POST /api/v4/channels/{channelID}/members` — body `{"user_id":"myUserID"}`

## Files Changed

| File | What changes |
|---|---|
| `api/search.go` (new) | `SearchUsers(term)`, `SearchChannels(term, teamID)` |
| `api/channels.go` | Add `CreateDirectChannel(otherUserID)`, `JoinChannel(channelID)` |
| `internal/messages/messages.go` | Add `TriggerSearchMsg`, `SearchResultsMsg`, `CreateDirectChannelMsg`, `JoinChannelMsg` |
| `internal/keymap/keymap.go` | Add `Search` binding (`/`, `ctrl+f`) |
| `tui/sidebar/search.go` (new) | `searchResult` type, `rebuildResults()`, `enterSearch()`, `exitSearch()`, `updateSearch()` |
| `tui/sidebar/model.go` | Add search fields to `Model`, wire `updateSearch` into `Update` |
| `tui/sidebar/view.go` | Add `renderSearch()`, call it when `m.searching` |
| `tui/app.go` | Route `/`+`ctrl+f`, handle `TriggerSearchMsg`→`cmdSearch`, handle `SearchResultsMsg`, `CreateDirectChannelMsg`, `JoinChannelMsg`; store `teams []api.Team` |

## Internal Message Flow

```
User types "/"
  → sidebar.Update returns TriggerSearchMsg{} via Cmd (only when query ≥ 3)
  → app.Update sees TriggerSearchMsg → fires cmdSearch(query)
  → cmdSearch returns SearchResultsMsg{Query, Users, Channels}
  → app.Update sees SearchResultsMsg → calls sidebar.SetSearchAPIResults(query, users, channels)
  → sidebar rebuilds results (drops if query changed)

User selects new DM:
  → sidebar.Update returns CreateDirectChannelMsg{UserID}
  → app.Update fires cmdCreateDirectChannel(userID)
  → on success: adds channel to sidebar allItems, selects it, exits search

User confirms join public channel:
  → sidebar.Update returns JoinChannelMsg{ChannelID}
  → app.Update fires cmdJoinChannel(channelID)
  → on success: adds channel to sidebar allItems, selects it, exits search
```

## Out of Scope (v1)

- Left/right cursor movement inside the search query
- Debouncing API calls (stale-result discard is sufficient)
- Searching within message content (full-text search)
- Multi-select / batch open
