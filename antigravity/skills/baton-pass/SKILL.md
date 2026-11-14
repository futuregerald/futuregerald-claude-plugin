---
name: baton-pass
description: >-
  Pass the baton with a deterministic ACTIVE_CONTEXT.md snapshot so fresh
  context picks up the work. MANDATORY in Antigravity and Gemini when the
  breaker hook warns that the context estimate has reached ~205k tokens (it
  blocks at ~235k, sized for 256k compaction on Carbon, Argon, Hydrogen, and
  Flash 3.8), before auto-compaction hits, or when asked to "pass the baton",
  "baton pass", "save active context", "snapshot context", or "continue from
  active context". Not a replacement for regular handoffs.
---

# Baton Pass (Antigravity & Gemini)

Writes a deterministic, high-signal `ACTIVE_CONTEXT.md` snapshot before context compaction degrades reasoning, so the next session (`Session B`) resumes in Turn 1 without re-reading the codebase.

## When to Trigger

1. **Circuit-breaker `BATON` warn (`~205k` estimated tokens) or block (`~235k` tokens):** Hand off at the next natural break (or immediately on block).
2. **Explicit user request:** `"pass the baton"`, `"baton pass"`, `"save active context"`, `"snapshot context"`.
3. **Resuming in Session B:** `"Ingest ~/.gemini/handoffs/<project-key>/<file> and continue from Pending Steps."`

## Snapshot Location

Write the snapshot to:

```text
~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md
```

- `<project-key>`: `basename $(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null | sed 's/\/\.git$//')` lowercased and hyphen-normalized (or `"dev"` outside a git repo).
- Also mirror a copy to `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md` when in Antigravity so it is visible in the session artifact directory.

## Mandatory 4-Section Template

Keep the file under 200 lines of dense, verifiable facts (no conversational filler):

```markdown
# ACTIVE_CONTEXT — <Project / Stream> (<YYYY-MM-DD HH:MM>)

- **Repo / Worktree:** `<abs-path>`
- **Branch & HEAD SHA:** `<branch>` (`<sha>`)
- **Plan Path:** `<abs-path-to-PLAN.md>`
- **Active Phase:** `Phase N (<Name>)`

## 1. Active Files & Contracts
- `path/to/file.ext:start-end` — <exact signature / contract / state>

## 2. Decisions, Evidence & Test Results
- <confirmed decisions from user or plan review>
- <exact commands run and exit codes>

## 3. Pending Steps (Ordered)
1. [ ] <next concrete action with exact file path and gate command>
2. [ ] <subsequent step>

## 4. Negative Constraints (Do Not Repeat)
- <ruled-out hypotheses, forbidden commands, or repo constraints>
```

## Finalizing Session A

1. Write `~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md` (this automatically unlocks the circuit breaker's `BATON` gate for another 50k tokens if finishing cleanup in Session A).
2. If Prism MCP is configured, call `session_save_handoff` with the current state and next steps.
3. Print the exact one-line prompt for Session B:
   ```text
   Ingest ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md and continue from Pending Steps.
   ```

## Resuming in Session B (Turn 1 Rule)

When the user starts a session with `Ingest ~/.gemini/handoffs/...-ACTIVE_CONTEXT.md and continue from Pending Steps`:
1. Read the `ACTIVE_CONTEXT.md` file (and `PLAN.md` if referenced) on Turn 1.
2. Trust the `Active Files & Contracts` and `Decisions, Evidence & Test Results` sections — **do not** re-read files already summarized there unless you are about to edit them.
3. Create `task.md` from `Pending Steps` and immediately execute Step 1.
