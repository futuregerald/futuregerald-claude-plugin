---
name: baton-pass
description: >-
  Pass the baton with a deterministic ACTIVE_CONTEXT.md snapshot so fresh
  context picks up the work — both with swarm (peer-pipeline-swarm / native
  swarm) and without swarm (invoke_subagent / inline). MANDATORY in Antigravity
  and Gemini when the breaker hook warns that the context estimate has reached
  ~205k tokens (it blocks at ~235k, sized for 256k compaction on Carbon, Argon,
  Hydrogen, and Flash 3.8), before auto-compaction hits, or when asked to
  "pass the baton", "baton pass", "save active context", "snapshot context", or
  "continue from active context". Not a replacement for regular handoffs.
---

# Baton Pass (Antigravity & Gemini)

Writes a deterministic, high-signal `ACTIVE_CONTEXT.md` snapshot before context compaction degrades reasoning, so the next session (`Session B`) resumes in Turn 1 without re-reading the codebase — whether Session A ran **without `swarm`** (`invoke_subagent` / inline) or **with `swarm`** (`peer-pipeline-swarm` / native `swarm`).

## When to Trigger

1. **Circuit-breaker `BATON` warn (`~205k` estimated tokens) or block (`~235k` tokens):** Hand off at the next natural break (or immediately on block).
2. **Swarm worker context cap (`@coordinator [blocked] context-cap`):** When a swarm worker hits `~235k` tokens and Coordinator is also near `205k+` (or between phases).
3. **Explicit user request:** `"pass the baton"`, `"baton pass"`, `"save active context"`, `"snapshot context"`.
4. **Resuming in Session B:** `"Ingest ~/.gemini/handoffs/<project-key>/<file> and continue from Pending Steps."`

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

## Finalizing Session A

### Without Swarm (`subagent` / `inline`)

1. Read `/tmp/subagent-context-<project>-<stream>.md` if an `invoke_subagent` helper wrote findings or diffs there (`read_file` on `/tmp/*.md` remains exempt even at `>=235k`).
2. Write `~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md` with `Execution Mode: subagent` (or `inline`) and mirror to `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md`. Writing `ACTIVE_CONTEXT.md` unlocks `+50k` tokens for any final git status/commit cleanup in Session A.
3. If Prism MCP is configured, call `session_save_handoff` with the current state and next steps.

### With Swarm (`peer-pipeline-swarm` / native `swarm`)

1. **Unlock if at `>=235k` (`Baton Block`):** At `>=235k`, `swarm_wait` is blocked while `read_board`, `post_message`, `swarm{action: "status"|"kill"}`, and `/tmp/*.md` reads stay exempt. Write an initial `ACTIVE_CONTEXT.md` first to unlock `+50k` tokens for `git fetch` / branch anchoring.
2. **Collect `SHARE_ROOT` & `SHA`:** Read `read_board` and `/tmp/subagent-context-<project>-<stream>.md` (`impl` records `SHARE_ROOT=$(pwd)` on startup; if a worker hit `235k`, it commits WIP using its `+25k` `/tmp/*.md` unlock and posts `@coordinator [blocked] context-cap SHARE_ROOT=<path> SHA=<sha>`).
3. **Fetch & verify `share` commits BEFORE `swarm{action: "kill"}`:** `kill` permanently deletes `workspace: "share"` directories. If `/tmp/subagent-context-<project>-<stream>.md` has `SHARE_ROOT` without a posted `<SHA>` yet, confirm `impl` is idle via `swarm{action: "status"}`, snapshot uncommitted edits in `<SHARE_ROOT>` if dirty (`git -C "<SHARE_ROOT>" status --porcelain`; if non-empty, `git -C "<SHARE_ROOT>" add -A && git -C "<SHARE_ROOT>" commit -m "wip: baton snapshot"`), and read `SHA=$(git -C "<SHARE_ROOT>" rev-parse HEAD)`. Always anchor commits into the Coordinator repo first:
   ```bash
   git fetch "<SHARE_ROOT>" "<SHA>" && git rev-parse --verify "<SHA>^{commit}"
   ```
   Merge onto your feature branch if `[green]`, or anchor WIP with `git branch -f "baton/<stream>" "<SHA>"` if mid-phase.
4. **Kill all workers & finalize snapshot:** Call `swarm{action: "kill"}` on every worker **including `monitor`**, update `ACTIVE_CONTEXT.md` (`Execution Mode: swarm`, `Anchored WIP Ref / Swarm Wind-Down` with the verified SHA and worker termination status), and call Prism `session_save_handoff` if configured.

### Emit the Session B Prompt (Both Modes)

Print the exact one-line prompt for Session B:

```text
Ingest ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md and continue from Pending Steps.
```

## Resuming in Session B (Turn 1 Rule)

When the user starts a session with `Ingest ~/.gemini/handoffs/...-ACTIVE_CONTEXT.md and continue from Pending Steps`:

1. Read `ACTIVE_CONTEXT.md` (and `PLAN.md` if referenced) on Turn 1. Trust `Active Files & Contracts` and `Decisions, Evidence & Test Results` — **do not** re-read codebase files already summarized there unless you are about to edit them.
2. Create `task.md` from `Pending Steps`.
3. **Without Swarm (`subagent` / `inline`):** Seed `/tmp/subagent-context-<project>-<stream>.md` from `ACTIVE_CONTEXT.md` if delegating, and immediately execute Pending Step 1 inline or via `invoke_subagent` (`Model: "inherit"`).
4. **With Swarm (`peer-pipeline-swarm` / native `swarm`):** Verify the anchored branch or `baton/<stream>` SHA (`git rev-parse --verify`), write a fresh `/tmp/subagent-context-<project>-<stream>.md` (and `/tmp/plan-<project>-<stream>.md` if `PLAN.md` is untracked), and spawn the pipeline in one turn. Because Session B is a new conversation, worker names start fresh (`impl`, `code-rev`, `sec-rev`, plus a fresh `monitor` — no `-2` suffix needed on Turn 1).
