---
name: subagent-driven-development
description: Execute implementation plans with independent tasks using orchestrated subagents (invoke_subagent TypeName implementer or peer-pipeline-swarm) and adversarial review in Antigravity and Gemini.
---

# Subagent-Driven Development (Antigravity & Gemini)

Execute an implementation plan by dispatching fresh implementer subagents (`invoke_subagent` with `TypeName: "implementer"`, `Model: "inherit"`) or running the **Peer-Pipeline Swarm** (`peer-pipeline-swarm` under `/start` or `/begin`), keeping the Coordinator's context lean and verifying every phase gate.

**Core principle:** Fresh subagent per task or phase + scoped context file (`/tmp/subagent-context-<project>-<stream>.md`) + adversarial review = high quality, zero context pollution.

## When to Use

- You have an approved `PLAN.md` from Phase 3 (`future-writing-plans`) and Phase 3½ (`plan-review`).
- The implementation touches `≥3` files or requires `>3` tool calls (for trivial `<5`-turn, `≤2`-file changes where all traced files are already in context, implement inline in the Coordinator).
- **In Antigravity:** Prefer [`peer-pipeline-swarm`](../peer-pipeline-swarm/SKILL.md) (`/start`, `/begin`) when you want Phases 4–9 (TDD implementation, fast tests, lint, simplification, and parallel `code-rev` + `sec-rev` peer review) to resolve peer-to-peer before waking the Coordinator once on `@coordinator [green]`. Use direct `invoke_subagent` (`TypeName: "implementer"`, `Model: "inherit"`) below when orchestrating step-by-step from the Coordinator or when running in Gemini CLI.

## The Process

### 1. Prepare Pre-Packed Context (`/tmp/subagent-context-<project>-<stream>.md`)

Before dispatching any subagent:
- Run `python3 ~/.gemini/config/skills/subagent-driven-development/scripts/build_subagent_context.py pre-code --output /tmp/subagent-context-<project>-<stream>.md --plan <plan_path> [--slice <file:start-end> ...] [--gate-cmd "<cmd>" ...]` to bundle `## Acceptance Criteria` and exact numbered `file:start-end` target slices cleanly without dumping 2,000-line files.
- Append or include concise context notes as needed:
  - `## Product & Design Sources`: paths and key targets from `docs/prd.md`, `docs/decisions/product-scope.md`, `docs/point-of-view/*.md`, and `design-demos/*/direction-approved.md`.
  - `## Acceptance Criteria & Success Metrics`: every `AC-N` row from `PLAN.md` with required runtime behavior, measurable target, and mounted entry point (`file:Symbol`).
  - `## Known Facts`: numbered entry-point `file:start-end` slices, newest sibling exemplar `file:line`, and 1 combined fast scoped verification gate command.
  - `## Affected Callers` & `## Hypotheses to Falsify`.
- Update `<appDataDir>/brain/<conversation-id>/task.md` with the tasks to execute.

### 2. Dispatch Implementer (`invoke_subagent`, `24.8k` Turn-1 Floor)

Dispatch the dedicated `implementer` agent (`~/.gemini/config/plugins/antigravity-subagents/agents/implementer.md`):

```text
invoke_subagent({
  TypeName: "implementer",
  Role: "Phase 4 TDD Implementer",
  Model: "inherit",
  Prompt: "On Turn 1, read /tmp/subagent-context-<project>-<stream>.md and PLAN.md in 1 parallel read_file batch, and trust the embedded file:start-end slices rather than reading untouched neighbor files (<185k peak-context guard; git grep -n -> +-40 lines only, 5-12 files/turn when extra reads are required). Execute Task N via strict TDD (RED -> GREEN -> REFACTOR) with zero code comments, batching edit_file across 4-8 distinct files per turn (1 contiguous edit_file or write_file on <300-line files; never run node -e or python3 -c regex source edits), and wire every Acceptance Criterion (AC-N) into the real mounted runtime entry point. Run 1 combined fast scoped verification gate at the end (if a CLI fails twice consecutively, read Makefile/package.json/vitest.config.* immediately). Return the mandatory 5-section handoff: Changes Made, Acceptance Criteria & Success Metrics Met, Flawed Assumptions, Carried Risks, Verification Evidence."
})
```

- **Model routing (`GEMINI.md` §0.9):** Always pass `Model: "inherit"` (`24.8k` Turn-1 floor; runs **Carbon Medium** by default, or **Argon Medium** if you switched the UI to Argon for a simple task). Never pass `Model: "pro"` on Argon or Carbon sessions. Reserve `Model: "flash"` strictly for `<50` LOC single-file mechanical boilerplate/renames with zero error-handling, concurrency, or security judgment.
- **Cap warm reuse at 1 phase (+ its fix round) & `<185k` compaction guard:** Never reuse a warm implementer across more than 1 roadmap phase (`~80k–100k` context) and keep peak context `<185k` to avoid the `256k` auto-compaction cliff (`25+` wasted re-discovery turns).
- **End your turn immediately** after `invoke_subagent` if waiting asynchronously; never poll `manage_subagents` in a loop.

### 3. Coordinator Fast Gates, Simplification & Post-Code Diff Chunking (Phases 5–7)

When the implementer returns its 5-section handoff:
1. Run the combined `<15s` fast scoped unit tests (Phase 5) and linter/typecheck (Phase 6) inline in the Coordinator's shell (recording output to `/tmp/gate-<project>-<stream>.log`) and verify exit code `0`.
2. Run `code-simplifier` (Phase 7) on the modified files if needed, re-verifying fast tests and lint.
3. Run `python3 ~/.gemini/config/skills/subagent-driven-development/scripts/build_subagent_context.py post-code --output /tmp/subagent-context-<project>-<stream>.md --base <base_ref> --gate-log /tmp/gate-<project>-<stream>.log` (omit `--head` to capture uncommitted working-tree edits, or pass `--head HEAD` on committed branches) so `git diff -U3` is split into `<=750`-line sections with an exact two-pass `## Read Chunk Index` (`StartLine-EndLine`, bounded by both lines and `<46,080` bytes via `--max-chunk-bytes 38000`).

### 4. Adversarial Review (`comprehensive-code-review` & `fix-check` — Phases 8–9)

Follow [`comprehensive-code-review`](../comprehensive-code-review/SKILL.md):
- **Turn-1 & follow-up read batching:** Instruct every reviewer to read all indexed chunks of `/tmp/subagent-context-<project>-<stream>.md` (`1-700`, `701-1400`, ... when `>750` lines) in **1 parallel `read_file` batch on Turn 1**, and batch follow-up verification reads **`5–10` `read_file` calls per turn** in `400–800` line chunks (`±40` lines around narrow call sites — never serial 1-file-per-turn tail reads).
- **When UI is not `Carbon High` (default `Carbon Medium` or `Argon` — pre-PR review):** Stage uncommitted edits and spawn parallel one-shot `gemini-4-carbon-high` `swarm` workers (`29.3k` Turn-1 floor each: `code-rev` + `sec-rev`, plus `sql-rev` if SQL/migrations changed; `workspace: "inherit"`, no `monitor`, read-only by brief forbidding git/tree mutations, immediate `kill`).
- **When UI is on `Carbon High`, on Gemini CLI, or for small changes:** Dispatch parallel read-only subagents via `invoke_subagent` (`22.7k` Turn-1 floor each: `TypeName: "code-quality-reviewer"` and `TypeName: "security-reviewer"`, plus `TypeName: "sql-reviewer"` if SQL changed; `Model: "inherit"`, `tools: [read_file]`, **zero spikes / zero `run_command`**).
- If CRITICAL or IMPORTANT findings are raised, fix them and verify via [`fix-check`](../fix-check/SKILL.md) (when installed; `invoke_subagent` `TypeName: "code-quality-reviewer"`, `Model: "inherit"` — never re-run a full review).
- Once approved, delete `/tmp/subagent-context-<project>-<stream>.md`.

### 5. Blast-Radius Verify, Commit & Handoff (Phases 10–12)

- Walk every row in `Affected Callers` plus callers of any symbol touched in review fixes (Phase 10).
- Commit on the feature branch (Phase 11) and complete handoff / PR checks (Phase 12), marking all items complete in `task.md`.

## Red Flags

**Never:**
- Pass `Model: "pro"` to `invoke_subagent` on an Argon or Carbon session (downgrades to Gemini 3.1 Pro).
- Run unscoped `go test ./...` or E2E suites inside an implementer subagent.
- Allow reviewers to run shell commands (`run_command`), spikes, or git mutations.
- Reuse a warm implementer across multiple roadmap phases.
- Skip `Blast-radius verify` (Phase 10) after a review fix round.
