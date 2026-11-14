---
name: peer-pipeline-swarm
description: >-
  Antigravity-only Phases 4–9 execution and peer-review swarm. Spawns an isolated
  Argon Medium implementer (workspace: share), cross-family high-deliberation
  peer reviewers (Claude Opus 5.5 High code-quality + Gemini 4 Carbon High
  security/SQL), and a cheap Hydrogen Low stall monitor so build, test, review,
  and fix rounds resolve peer-to-peer before waking the Coordinator once when
  green. Use when running Phases 4–9 under /start, /begin, or /peer-pipeline-swarm,
  or when executing 1 or more implementation streams hands-off in Antigravity.
---

# Peer-Pipeline Swarm (Phases 4–9)

Runs TDD implementation, fast checks, simplification, and adversarial code/security/SQL review as a self-verifying peer pipeline in Antigravity's native `swarm`, keeping the Coordinator's context lean and cutting middleman round-trips while preserving every lifecycle safety gate.

- **What good looks like:** The Coordinator writes a grounded `PLAN.md` and `/tmp/subagent-context-<project>-<stream>.md`, spawns the pipeline in one turn, sleeps at zero cost while `impl` builds and resolves any review findings directly with `code-rev` and `sec-rev`, wakes once on `@coordinator [green]`, merges the `share` branch before killing workers, and verifies final gates + blast radius inline.
- **Generic default to prevent:** Out-of-the-box `/swarm` behavior — copying Carbon across every worker and `monitor`, pasting diffs/logs onto the message board, letting `inherit` reviewers read stale pre-edit files instead of the `share` checkout, or running version bumps inside parallel branches.
- **Detail a careless run misses:** `impl` edits inside its own `workspace: "share"` directory while `code-rev` and `sec-rev` sit in `workspace: "inherit"`. `impl` **must** record `SHARE_ROOT=$(pwd)` and `git diff -U5` in `/tmp/subagent-context-<project>-<stream>.md` so reviewers inspect `$SHARE_ROOT/<file>` rather than the untouched Coordinator tree. When `<active-workspace>` differs from `<target-repo>` (because `workspace: "share"` clones `<active-workspace>`), create an isolated git worktree inside `<target-repo>/.worktrees/<stream>` as `$SHARE_ROOT` and spawn `impl` with `workspace: "inherit"` pointed strictly at that worktree.

## When to Use vs. Inline Fast-Path

| Task Shape | Route |
|---|---|
| Trivial change (`<5` turns, `≤2` files already in context, obvious blast radius) | Implement inline in Coordinator + run lightweight `invoke_subagent` review (`Model: "inherit"`, `tools: [read_file]`) |
| Standard 1-stream or multi-stream code change (`≥3` files or `>3` tool calls, or `/start`) | **Peer-Pipeline Swarm** (below) |

## Pinned Role Roster

Never omit `model` on `swarm{action: "add"}` — omitting `model` copies the UI dropdown onto every worker and cron tick.

| Worker Name (suffix `-2`, `-3` on later phases) | `model` | `workspace` | Role |
|---|---|---|---|
| `impl` *(or `impl-<stream>`)* | `gemini-4-argon-medium` | `share` *(or `inherit` inside `<target-repo>/.worktrees/<stream>` when `<active-workspace>` != `<target-repo>`)* | TDD (`RED → GREEN → REFACTOR`), scoped fast unit/lint gates, `/code-simplifier` pass, `git commit` on `share` branch, join point for peer reviews |
| `code-rev` *(or `code-rev-<stream>`)* | `claude-opus-5-5-high` *(fallback: `gemini-4-carbon-high` if Anthropic quota is 0%)* | `inherit` | Read-only Staff Code Quality Reviewer (`agents/code-quality-reviewer.md`) |
| `sec-rev` *(or `sec-rev-<stream>`)* | `gemini-4-carbon-high` *(fallback: `claude-opus-5-5-high` if Google quota is 0%)* | `inherit` | Read-only Principal Security Reviewer (`agents/security-reviewer.md`) |
| `sql-rev` *(only when SQL/ORM/migrations change)* | `gemini-4-carbon-high` | `inherit` | Read-only SQL Reviewer (`agents/sql-reviewer.md`) |
| `monitor` *(one per conversation)* | `gemini-4-hydrogen-low` *(fallback: `gemini-3.8-flash-low`)* | `inherit` | `swarm_monitor` `*/3 * * * *` stall & quota watchdog |

## Step-by-Step Execution

### 1. Pre-Flight Context File (Coordinator)

Before calling `swarm`, write `/tmp/subagent-context-<project>-<stream>.md` containing:
- Exact path to `PLAN.md` (copy `PLAN.md` into `/tmp/plan-<project>-<stream>.md` as well if `PLAN.md` is untracked in the repo, because `workspace: "share"` clones committed repo state)
- `## Known Facts` (entry-point `file:line` ranges, sibling exemplar `file:line`, exact scoped fast test & lint commands, repo gate commands)
- `## Affected Callers` (from Phase 2/3)
- `## Hypotheses to Falsify` (if any)

If `2+` independent streams run in parallel and `swarm` supports `action: "channel"`, create one channel per stream (`swarm{action: "channel", op: "create", channel: "<stream>", topic: "..."}`) before adding workers so each `impl + reviewers` group stays silent in `main`.

### 2. Spawn the Pipeline in One Turn (Coordinator)

Board tools (`read_board`, `post_message`, `swarm_wait`) appear one turn *after* the first `swarm{action: "add"}`. Spawn all workers for the phase in a **single turn**, then **end your turn immediately** (show the Swarm board link if returned by `add`):

1. **`impl` (`gemini-4-argon-medium`, `workspace: "share"`):**
   ```text
   Read ~/.gemini/config/plugins/antigravity-subagents/agents/implementer.md, /tmp/plan-<project>-<stream>.md, and /tmp/subagent-context-<project>-<stream>.md first.
   1. Execute all TDD tasks (RED -> GREEN -> REFACTOR) with zero code comments. Do NOT run version-bump scripts or E2E suites.
   2. Run scoped fast unit tests and lint/typecheck/wording gates (filtering output to /tmp/t.log; if a CLI fails twice, read Makefile/package.json/vitest.config.* immediately). Perform a self-simplification pass while keeping tests green.
   3. Stage and commit your changes on your share branch (`git rev-parse --abbrev-ref HEAD` and `git rev-parse HEAD`).
   4. Append `SHARE_ROOT=$(pwd)`, `BRANCH`, `COMMIT_SHA`, your 4-section handoff, passing gate exit codes, and `git diff -U5 <base>...HEAD` to /tmp/subagent-context-<project>-<stream>.md.
   5. Tag `@code-rev @sec-rev` (and `@sql-rev` if present) with a <=5-line pointer (SHARE_ROOT, SHA, context file path — NEVER paste diffs or logs on the board), then `swarm_wait` for ALL assigned reviewers to reply before acting.
   6. If any reviewer replies `[changes-requested]`, fix all findings in one pass, re-run fast gates, commit a new SHA, append the fix diff to the context file, and tag ONLY the reviewer(s) who requested changes (plus `@sec-rev` if auth/input/SQL/network changed) for a 1-turn fix-check. Cap at 2 fix rounds; escalate `@coordinator [blocked]` if unresolved.
   7. Once EVERY assigned reviewer has replied `[approved]` on the current SHA, post ONE <=5-line message: `@coordinator [green] SHARE_ROOT=<path> BRANCH=<branch> SHA=<sha> Context=/tmp/subagent-context-<project>-<stream>.md`.
   ```

2. **`code-rev` (`claude-opus-5-5-high`, `workspace: "inherit"`) & `sec-rev` (`gemini-4-carbon-high`, `workspace: "inherit"`):**
   ```text
   Startup: Read ~/.gemini/config/plugins/antigravity-subagents/agents/<code-quality-reviewer|security-reviewer>.md, then IMMEDIATELY call `swarm_wait{since: 0, mentions_only: true}` to sleep until `@impl` tags you.
   When woken by `@impl`:
   1. STRICTLY READ-ONLY: Use ONLY `read_file`, `read_board`, `post_message`, and `swarm_wait`. NEVER call `run_command`, `edit_file`, `write_file`, or `swarm`, and never mutate git or files.
   2. Read /tmp/subagent-context-<project>-<stream>.md. When reading surrounding +-40 lines of modified files, read from `$SHARE_ROOT/<relpath>` (NOT the coordinator workspace).
   3. On first pass, perform the full adversarial audit. On a follow-up `fix-check` ping, verify ONLY whether the prior CRITICAL/IMPORTANT findings were fixed cleanly without regression.
   4. Reply ONLY to `@impl` (NEVER `@coordinator` or `@all`) in <=10 lines: either `@impl [approved] (<your-name>) SHA=<sha>` (if 0 CRITICAL and 0 IMPORTANT) or `@impl [changes-requested] (<your-name>) SHA=<sha>` with concise `file:line` findings (write detailed notes to `/tmp/review-<your-name>-<stream>.md` if >10 lines). Then `swarm_wait` with `mentions_only: true`.
   ```

3. **`monitor` (`gemini-4-hydrogen-low`, `workspace: "inherit"` — add once per conversation if not already live):**
   ```text
   You are this swarm's stall monitor. Load the swarm_monitor skill and follow it. Post only to break a stall.
   ```

### 3. Coordinator Park & Stall Handling

- On wakeup before `[green]`: call `read_board{since: LAST}`.
  - If `[stall]` from `monitor` or quota exhaustion on a worker: check `swarm{action: "status"}` and `swarm{action: "models"}`; switch a quota-blocked worker's model with `swarm{action: "model", name, model}` (e.g., `code-rev` → `gemini-4-carbon-high` or `sec-rev` → `claude-opus-5-5-high`) or re-ping the dropped `@target`, then `swarm_wait{since: LAST}`.
  - If `[blocked]` from `impl`: diagnose in one turn using `/tmp/subagent-context-<project>-<stream>.md` and direct `impl` (or ask user if product ambiguity).

### 4. Land, Kill, and Verify Inline (Coordinator — Phases 10–12)

Once `@coordinator [green]` arrives for every stream in the phase:

1. **Read evidence:** Read `/tmp/subagent-context-<project>-<stream>.md` (and any `/tmp/review-*.md`). Run the `review-lens` corpus check on the diff where configured in `antigravity-orchestration/references/config.md`.
2. **Land `share` branch BEFORE `kill`:** `swarm{action: "kill"}` permanently deletes a `share` workspace. Fetch/merge each stream's commit into your Coordinator feature branch first:
   ```bash
   git fetch "<SHARE_ROOT>" "<SHA>" && git merge --no-ff FETCH_HEAD -m "chore: merge <stream> implementation"
   ```
   *(For a single clean commit history, `git cherry-pick <SHA>` or `git merge --squash FETCH_HEAD` onto your feature branch.)*
3. **Silence `monitor` & clean up workers:**
   - Post `[done] <phase/stream> landed` (with **no `@`**) if more roadmap phases remain in this session so `monitor` goes dormant, or `swarm{action: "kill", name: "monitor"}` when all swarm work for the session is finished.
   - Immediately `swarm{action: "kill"}` `impl`, `code-rev`, `sec-rev` (and `sql-rev`). Suffix `-2`, `-3` on the next roadmap phase (killed names cannot be reused).
   - Remove `/tmp/subagent-context-<project>-<stream>.md`, `/tmp/plan-<project>-<stream>.md`, and `/tmp/review-*-<stream>.md`.
4. **Coordinator Inline Completion Gates (Never Delegated):**
   - Run repo version bump **once** if applicable (e.g., `node scripts/version.mjs bump patch|minor && make version-check`).
   - Run scoped fast unit tests and linter/typecheck directly in the Coordinator's shell and inspect exit codes (`0`).
   - Execute **Phase 10 (Blast-radius verify)** across all `Affected Callers` plus any symbols touched during review fixes, then proceed to **Phase 11 (Commit)** and **Phase 12 (Handoff)**.

## Do / Don't

| Do | Don't |
|---|---|
| Pin `impl` to `gemini-4-argon-medium` with `workspace: "share"` | Leave `model` blank on `swarm{action: "add"}` (copies Carbon across workers and `monitor`) |
| Copy untracked `PLAN.md` to `/tmp/plan-<project>-<stream>.md` and write `SHARE_ROOT` into the context file | Point `inherit` reviewers at relative repo paths (they will read pre-edit Coordinator files) |
| Have `code-rev` and `sec-rev` reply to `@impl`, waking `@coordinator` once on `[green]` | Have reviewers tag `@coordinator` individually (wakes Coordinator mid-fix and inflates context) |
| Merge/cherry-pick from `SHARE_ROOT` **before** calling `swarm{action: "kill"}` | Call `kill` before landing the `share` branch (`kill` deletes the `share` repo immediately) |
| Run version bumps and final unit/lint gates once in the Coordinator after merge | Run `version.mjs bump` inside parallel `share` branches (guarantees `package-lock.json` conflicts) |
