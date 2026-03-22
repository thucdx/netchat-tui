# Agent Collaboration Protocol

This document defines the roles, responsibilities, and communication protocol for the four agents building netchat-tui.

---

## Agents

| Agent | Name | Responsibility |
|-------|------|----------------|
| **Coder** | `coder` | Implements features phase by phase, fixes bugs reported by other agents |
| **Tester** | `tester` | Writes tests, runs them, reports results and failures back to Coder |
| **Reviewer** | `reviewer` | Reviews code for quality, architecture, and best practices |
| **Security** | `security` | Reviews code for security issues against the security plan in PLAN.md |

---

## Workflow per Phase

```
Orchestrator starts phase
       │
       ▼
  [Coder] implements the phase
       │
       ├──────────────────────────────────────┐
       ▼                                      ▼
  [Tester] writes & runs tests          [Security] reviews for security issues
       │                                      │
       ▼                                      ▼
  reports pass/fail to Coder            reports findings to Coder
       │                                      │
       └──────────────┬───────────────────────┘
                      ▼
               [Reviewer] reviews final code
                      │
                      ▼
               reports to Orchestrator
                      │
              ┌───────┴────────┐
              ▼                ▼
           All clear       Issues found
              │                │
              ▼                ▼
        next phase      Coder fixes → re-test loop
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

1. **Coder → Tester**: notify when a phase or feature is ready for testing. Include which files were changed.
2. **Tester → Coder**: report each failing test with: test name, error message, and a suggested fix if obvious. If the fix requires a design change, escalate.
3. **Security → Coder**: report each finding with: severity (High/Medium/Low), location (file:line), description, and recommended fix.
4. **Reviewer → Coder**: report issues with: category (architecture/readability/performance), location, description, and recommendation.
5. **Coder → all**: after fixing, notify which agents need to re-check.
6. **Any agent → Orchestrator**: escalate when two agents disagree, when a fix would require changing requirements, or when blocked for more than one retry.

---

## Escalation Criteria

Escalate to the user when:
- Tester and Coder cannot agree on the correct behavior after 2 iterations
- Security finding conflicts with a planned feature (e.g. "masked input breaks copy-paste UX")
- Reviewer recommends a structural change that affects multiple phases
- A bug is discovered that requires changing `requirements.md`

---

## Decision Documentation

Every decision made by agents OR by the user during escalation must be appended to `requirements.md` under a `## Decisions Log` section with:

```
### [DATE] [AGENT or USER] — short title
**Context**: what the disagreement or question was
**Decision**: what was decided
**Reason**: why
```

---

## Commit Rule

After **every completed TODO item**, the Coder agent must create a git commit:
- Stage only the files changed for that item
- Commit message format: `<type>(<scope>): <short description>` (e.g. `feat(config): add AuthConfig load/save`)
- Never batch multiple TODO items into one commit
- The Orchestrator updates `TODO.md` (`[ ]` → `[x]`) and includes it in the same commit

---

## Per-Phase Checklist (gate before moving to next phase)

A phase is complete only when ALL of the following are true:

- [ ] Coder: all planned tasks for the phase are implemented
- [ ] Tester: all tests for the phase pass (`go test ./...`)
- [ ] Security: no High severity findings open
- [ ] Reviewer: no blocking issues open
- [ ] Orchestrator: updated TODO.md with `[x]` for completed items

---

## Agent Startup Instructions

When the Orchestrator starts a phase, each agent receives:
- The phase number and goal
- The list of files to work on (from PLAN.md)
- A reference to this file (AGENTS.md) for protocol
- A reference to PLAN.md, TODO.md, and requirements.md for context

Agents must not modify `requirements.md` directly — only the Orchestrator appends to the Decisions Log after user confirmation.
