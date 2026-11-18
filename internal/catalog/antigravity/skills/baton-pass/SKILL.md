---
name: baton-pass
description: >-
  Pass the baton with a deterministic ACTIVE_CONTEXT.md snapshot so fresh
  context picks up the work — both with swarm (peer-pipeline-swarm / native
  swarm) and without swarm (invoke_subagent / inline). MANDATORY in Antigravity
  and Gemini when the breaker hook warns that the context estimate has reached
  ~310k tokens (it blocks at ~350k, sized for 512k compaction on Carbon, Argon,
  Hydrogen, and Flash 3.8), before auto-compaction hits, or when asked to
  "pass the baton", "baton pass", "save active context", "snapshot context", or
  "continue from active context". Not a replacement for regular handoffs.
---

# Baton Pass (Antigravity & Gemini)

Writes a deterministic, high-signal `ACTIVE_CONTEXT.md` snapshot before context compaction degrades reasoning and **auto-relays in the same chat via a fresh `TypeName: "self"` Child Orchestrator** (`350k` window) — whether the current orchestrator ran **without `swarm`** (`invoke_subagent` / inline) or **with `swarm`** (`peer-pipeline-swarm` / native `swarm`). Only print a manual `Ingest ...` prompt for a new chat tab when the Root Supervisor itself reaches `>=450k` tokens (approaching the `512k` compaction ceiling) or when the user explicitly asks to pass/snapshot the baton.

## When to Trigger

1. **Child Orchestrator (`TypeName: "self"`) hits `~350k` (`Subagent Context Cap`):** Write `ACTIVE_CONTEXT.md` (unlocks `+25k` cleanup tokens), `git fetch`/anchor any `share`-workspace SHAs, `swarm{action: "kill"}` all child swarm workers (`impl`, `code-rev`, `sec-rev`, `monitor`, `ops`), and return `BATON_CONTINUE: <abs-path-to-ACTIVE_CONTEXT.md>` via `send_message` to the Thin Root Supervisor, which **immediately spawns a fresh `TypeName: "self"` Child Orchestrator in the SAME chat** to ingest `ACTIVE_CONTEXT.md` and continue `Pending Steps`.
2. **Root Supervisor hits `~310k` (`Baton Warn`) or `~350k` (`Baton Block`, `<450k`) with work remaining:** Write `ACTIVE_CONTEXT.md` (unlocking `+50k` tokens), anchor any active `share` SHAs and `swarm{action: "kill"}` all Root-owned swarm workers, save Prism `session_save_handoff`, and **immediately spawn `invoke_subagent({ TypeName: "self", Model: "inherit" })` in the SAME chat** to ingest `ACTIVE_CONTEXT.md` and finish `Pending Steps` — **DO NOT** stop and ask the user to open a new chat!
3. **Swarm worker context cap (`@coordinator [blocked] context-cap`):** When a swarm worker hits `~350k` tokens, `git fetch`/verify its `SHARE_ROOT` `SHA` before `swarm{action: "kill"}`; spawn a `-2` continuation worker if the orchestrator is `<310k`, or run Trigger 1 / Trigger 2 above if the orchestrator is `>=310k`.
4. **Explicit user request OR Root `>=450k` (near `512k` compaction ceiling):** `"pass the baton"`, `"baton pass"`, `"save active context"`, `"snapshot context"`, or `Root estimate >= 450000` — **ONLY** in these two cases stop and emit the manual `Ingest ~/.gemini/handoffs/<project-key>/<file> and continue from Pending Steps.` prompt for a fresh chat (`Session B`).
5. **Resuming in a Child Orchestrator or Session B:** `"Ingest ~/.gemini/handoffs/<project-key>/<file> and continue from Pending Steps."`

## Snapshot Location

Write the snapshot to:

```text
~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md
```

- `<project-key>`: `basename $(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null | sed 's/\/\.git$//')` lowercased and hyphen-normalized (or `"dev"` outside a git repo).
- Also mirror a copy to `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md` when in Antigravity so it is visible in the session artifact directory.

## Mandatory 4-Section Template

Keep the file under 200 lines of dense, verifiable facts (no conversational filler). See [references/template.md](references/template.md) for the expanded field-by-field reference:

```markdown
# ACTIVE_CONTEXT — <Project / Stream> (<YYYY-MM-DD HH:MM>)

- **Repo / Worktree:** `<abs-path>`
- **Branch & HEAD SHA:** `<branch>` (`<sha>`)
- **Execution Mode:** `swarm | subagent | inline`
- **Anchored WIP Ref / Swarm Wind-Down:** `<none | merged-to-branch | baton/<stream> (<sha>) — all swarm workers killed>`
- **Plan Path:** `<abs-path-to-PLAN.md>`
- **Active Phase:** `Phase N (<Name>)`

## 1. Active Files & Contracts
- `path/to/file.ext:start-end` — <exact signature / contract / state>

## 2. Decisions, Evidence & Test Results
- <confirmed decisions from user, plan review, or peer reviewers>
- <exact gate commands run and exit codes>

## 3. Pending Steps (Ordered)
1. [ ] <next concrete action with exact file path and gate command>
2. [ ] <subsequent step>

## 4. Negative Constraints (Do Not Repeat)
- <ruled-out hypotheses, forbidden commands, or repo constraints>
```

## Snapshot & Wind-Down Procedure

### Without Swarm (`subagent` / `inline`)

1. If a standalone persistent sleeping `ops` worker (`ops`, `ops-2`, ...) was spawned on `gemini-4-hydrogen-high` (`workspace: "inherit"`) for mechanical scouting/test/build/CI work, read any final `/tmp/*.md` pointer it wrote and terminate it with `swarm{action: "kill", name: "ops"}` (`swarm{action: "status"|"kill"}` and `/tmp/*.md` reads remain exempt at `>=350k` for both Root and Child Orchestrators).
2. Read `/tmp/subagent-context-<project>-<stream>.md` if an `invoke_subagent` helper wrote findings or diffs there (`read_file` on `/tmp/*.md` and `ACTIVE_CONTEXT.md` remains exempt even at `>=350k`).
3. Write `~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md` with `Execution Mode: subagent` (or `inline`) and mirror to `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md`. Writing `ACTIVE_CONTEXT.md` unlocks `+50k` tokens in Root (and `+25k` tokens in a `TypeName: "self"` Child Orchestrator) for final git status/commit cleanup and relay.
4. If Prism MCP is configured, call `session_save_handoff` with the current state and next steps.

### With Swarm (`peer-pipeline-swarm` / native `swarm`)

1. **Unlock if at `>=350k` (`Baton Block` / `Subagent Context Cap`):** At `>=350k`, `read_board`, `post_message`, `swarm{action: "status"|"kill"}`, and `/tmp/*.md` / `ACTIVE_CONTEXT.md` reads stay exempt. Write an initial `ACTIVE_CONTEXT.md` first to unlock `+50k` tokens in Root (or `+25k` tokens in a `TypeName: "self"` Child Orchestrator) for `git fetch` / branch anchoring.
2. **Collect `SHARE_ROOT` & `SHA`:** Read `read_board` and `/tmp/subagent-context-<project>-<stream>.md` (`impl` records `SHARE_ROOT=$(pwd)` on startup; if a worker hit `350k`, it commits WIP using its `+25k` `/tmp/*.md` unlock and posts `@coordinator [blocked] context-cap SHARE_ROOT=<path> SHA=<sha>`).
3. **Fetch & verify `share` commits BEFORE `swarm{action: "kill"}`:** `kill` permanently deletes `workspace: "share"` directories. If `/tmp/subagent-context-<project>-<stream>.md` has `SHARE_ROOT` without a posted `<SHA>` yet, confirm `impl` is idle via `swarm{action: "status"}`, snapshot uncommitted edits in `<SHARE_ROOT>` if dirty (`git -C "<SHARE_ROOT>" status --porcelain`; if non-empty, `git -C "<SHARE_ROOT>" add -A && git -C "<SHARE_ROOT>" commit -m "wip: baton snapshot"`), and read `SHA=$(git -C "<SHARE_ROOT>" rev-parse HEAD)`. Always anchor commits into the main repo first:
   ```bash
   git fetch "<SHARE_ROOT>" "<SHA>" && git rev-parse --verify "<SHA>^{commit}"
   ```
   Merge onto your feature branch if `[green]`, or anchor WIP with `git branch -f "baton/<stream>" "<SHA>"` if mid-phase.
4. **Kill all workers (including `monitor` & `ops`) & finalize snapshot:** Call `swarm{action: "kill"}` on **every** active or sleeping worker — **including `monitor` and any persistent sleeping `ops` (`ops-N`) worker** — update `ACTIVE_CONTEXT.md` (`Execution Mode: swarm`, `Anchored WIP Ref / Swarm Wind-Down` with the verified SHA and worker termination status), and call Prism `session_save_handoff` if configured.

### Relay Action: Default Same-Chat Child Orchestrator Auto-Chaining vs. Manual New Chat

- **Default A — Inside a `TypeName: "self"` Child Orchestrator (`>=350k`):** Return `BATON_CONTINUE: ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md` (plus 3–5 lines of status) via `send_message` to the Thin Root Supervisor. Upon receiving `BATON_CONTINUE: <path>`, Root **immediately calls `invoke_subagent({ TypeName: "self", Model: "inherit" })` in the SAME chat** instructing the fresh Child Orchestrator (`350k` budget) to ingest `<path>` and finish `Pending Steps`.
- **Default B — Inside the Root Supervisor (`310k <= estimate < 450k`, NOT an explicit user baton request):** Do **NOT** stop and ask the user to open a new chat! Using the `+50k` `ACTIVE_CONTEXT.md` unlock window, **immediately call `invoke_subagent({ TypeName: "self", Model: "inherit", Role: "Continuation Child Orchestrator", Prompt: "Ingest <abs-path-to-ACTIVE_CONTEXT.md> and execute all remaining Pending Steps, returning <=15 lines (or BATON_CONTINUE: <path> if 350k is reached)." })` in this SAME chat** and end your turn to wait for wakeup.
- **Fallback ONLY when Root `estimate >= 450000` (near `512k` ceiling) OR User Explicitly Asked (`"pass the baton"`, `"save active context"`, etc.):** Stop calling tools and print the exact one-line prompt for a fresh chat (`Session B`):
  ```text
  Ingest ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md and continue from Pending Steps.
  ```

## Resuming in a Fresh Child Orchestrator or Session B (Turn 1 Rule)

When a spawned `TypeName: "self"` Child Orchestrator or a new Session B starts with `Ingest ~/.gemini/handoffs/...-ACTIVE_CONTEXT.md and continue from Pending Steps`:

1. Read `ACTIVE_CONTEXT.md` (and `PLAN.md` if referenced) on Turn 1. Trust `Active Files & Contracts` and `Decisions, Evidence & Test Results` — **do not** re-read codebase files already summarized there unless you are about to edit them.
2. Update or create `task.md` from `Pending Steps`.
3. **Without Swarm (`subagent` / `inline`):** Seed `/tmp/subagent-context-<project>-<stream>.md` from `ACTIVE_CONTEXT.md` if delegating, and immediately execute Pending Step 1 inline or via `invoke_subagent` (`Model: "inherit"`).
4. **With Swarm (`peer-pipeline-swarm` / native `swarm`):** Verify the anchored branch or `baton/<stream>` SHA (`git rev-parse --verify`), write a fresh `/tmp/subagent-context-<project>-<stream>.md` (and `/tmp/plan-<project>-<stream>.md` if `PLAN.md` is untracked), and spawn the pipeline in one turn. Because each fresh Child Orchestrator or Session B is its own conversation, worker names start fresh (`impl`, `code-rev`, `sec-rev`, `monitor`, and optional sleeping `ops` — no `-2` suffix needed on Turn 1).
