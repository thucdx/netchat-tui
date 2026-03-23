# Agent Collaboration Protocol

This document defines the roles, responsibilities, and communication protocol for the five agents building netchat-tui.

---

## Team Structure (Option B — Parallel Model)

| Agent | Name | Responsibility |
|-------|------|----------------|
| **Orchestrator** | `orchestrator` | Plans phases, assigns tasks, gates commits, updates TODO.md. Acts as team lead. |
| **Builder-WS** | `builder-ws` | Implements WebSocket / real-time features: `api/websocket.go`, WS pump in `tui/app.go`, reconnect logic. |
| **Builder-UX** | `builder-ux` | Implements UI polish and edge cases: DM name resolution, pagination, resize, error banner, mute detection. |
| **QA** | `qa` | Writes and runs all tests (`go test ./...`), security audit against S1–S11. Also performs **live end-to-end testing**: builds the app, launches it against the real server, and exercises it as a real user to catch runtime bugs that unit tests cannot. Combines Tester + Security + Manual QA roles. |
| **Reviewer** | `reviewer` | Reviews code for architecture, Go idioms, and correctness. Gates each phase before Orchestrator commits. |

---

## Why This Structure

- `builder-ws` (Phase 7) and `builder-ux` (Phase 8) touch independent files — they can work in parallel.
- `qa` combines Tester and Security to reduce handoff overhead; security findings at this stage are well-scoped (S1–S11 checklist).
- `reviewer` acts as a final gate before each commit, catching issues neither builder would self-review.
- `orchestrator` is the only agent that commits code and updates `TODO.md`.

---

## Workflow per Phase

```
Orchestrator assigns tasks
        │
        ├─────────────────────────┐
        ▼                         ▼
  [builder-ws]             [builder-ux]
  implements WS            implements UX
  features                 features
        │                         │
        └──────────┬──────────────┘
                   ▼
              [qa] writes tests, runs go test ./..., audits S1–S11
                   │
                   ▼
             [reviewer] reviews final code
                   │
                   ▼
           [orchestrator] commits, updates TODO.md
                   │
          ┌────────┴────────┐
          ▼                 ▼
       All clear        Issues found
          │                 │
          ▼                 ▼
     next phase      builder fixes → re-test loop
                            │
                     if cannot agree
                            │
                            ▼
                     Escalate to User
                            │
                            ▼
                    User decides → document in requirements.md
```

---

## Communication Rules

1. **Orchestrator → builders**: assign tasks with file list and phase goal. Include which builder owns which area.
2. **Builder → Orchestrator**: report when implementation is ready for QA. List files changed.
3. **Builder → Builder**: coordinate on shared files (e.g., `tui/app.go`) — one builder at a time; notify Orchestrator of conflicts.
4. **QA → Builder**: report each failing test: test name, error message, suggested fix. Report each security finding: severity, file:line, description, recommended fix.
5. **Reviewer → Orchestrator**: report issues with: category (architecture/readability/performance/Go-idiom), location, description, recommendation.
6. **Any agent → Orchestrator**: escalate when two agents disagree, when a fix requires changing requirements, or when blocked for more than one retry.

---

## File Ownership

| Area | Owner |
|------|-------|
| `api/websocket.go` | builder-ws |
| `tui/app.go` (WS pump, event handling) | builder-ws |
| `api/channels.go` (DM batch fetch) | builder-ux |
| `tui/chat/` (error banner) | builder-ux |
| `tui/sidebar/` (pagination, mute) | builder-ux |
| `tui/layout.go` (resize) | builder-ux |
| `*_test.go`, `security_test.go` | qa |
| `AGENTS.md`, `TODO.md` | orchestrator |
| Code review (all files) | reviewer |

---

## Escalation Criteria

Escalate to the user when:
- Builder-WS and Builder-UX conflict on a shared file (e.g. `tui/app.go`)
- QA and a Builder cannot agree on correct behavior after 2 iterations
- Reviewer recommends a structural change affecting multiple phases
- A bug requires changing `requirements.md`

---

## Decision Documentation

Every decision made by agents OR by the user during escalation must be appended to `requirements.md` under the `## Decisions Log` section:

```
### [DATE] [AGENT or USER] — short title
**Context**: what the disagreement or question was
**Decision**: what was decided
**Reason**: why
```

---

## Commit Rule

Only the **Orchestrator** creates git commits:
- Stage only the files changed for that item
- Commit message format: `<type>(<scope>): <short description>`
- Never batch multiple TODO items into one commit
- Update `TODO.md` (`[ ]` → `[x]`) in the same commit

---

## Per-Phase Checklist (gate before moving to next phase)

A phase is complete only when ALL of the following are true:

- [ ] builder-ws / builder-ux: all planned tasks implemented
- [ ] qa: all tests pass (`go test ./...`), no High security findings open
- [ ] reviewer: no blocking issues open
- [ ] orchestrator: TODO.md updated with `[x]` for completed items

---

## Agent Startup Instructions

When the Orchestrator starts a phase, each agent receives:
- The phase number and goal
- The list of files to work on (from PLAN.md and File Ownership table above)
- A reference to this file (AGENTS.md) for protocol
- A reference to PLAN.md, TODO.md, and requirements.md for context

Agents must not modify `requirements.md` directly — only the Orchestrator appends to the Decisions Log after user confirmation.
