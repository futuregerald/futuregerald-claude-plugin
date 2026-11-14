---
name: subagent-driven-development
description: Execute implementation plans with independent tasks using orchestrated subagents (invoke_subagent TypeName implementer or peer-pipeline-swarm) and adversarial review in Antigravity and Gemini.
---

# Subagent-Driven Development (Antigravity & Gemini)

Execute an implementation plan by dispatching fresh implementer subagents (`invoke_subagent` with `TypeName: "implementer"`, `Model: "inherit"`) or running the **Peer-Pipeline Swarm** (`peer-pipeline-swarm` under `/start` or `/begin`), keeping the Coordinator's context lean and verifying every phase gate.

**Core principle:** Fresh subagent per task or phase + scoped context file (`/tmp/subagent-context-<project>-<stream>.md`) + adversarial review = high quality, zero context pollution.

## When to Use

- You have an approved `PLAN.md` from Phase 3 (`writing-plans`) and Phase 3½ (`plan-review`).
- The implementation touches `≥3` files or requires `>3` tool calls (for trivial `<5`-turn, `≤2`-file changes where all traced files are already in context, implement inline in the Coordinator).
- **In Antigravity:** Prefer [`peer-pipeline-swarm`](../peer-pipeline-swarm/SKILL.md) (`/start`, `/begin`) when you want Phases 4–9 (TDD implementation, fast tests, lint, simplification, and parallel `code-rev` + `sec-rev` peer review) to resolve peer-to-peer before waking the Coordinator once on `@coordinator [green]`. Use direct `invoke_subagent` (`TypeName: "implementer"`, `Model: "inherit"`) below when orchestrating step-by-step from the Coordinator or when running in Gemini CLI.

## The Process

### 1. Prepare Scoped Context (`/tmp/subagent-context-<project>-<stream>.md`)

Before dispatching any subagent:
- Read `PLAN.md` once and extract the task specification, acceptance criteria, `Affected Callers`, and exact file ranges.
- Write `/tmp/subagent-context-<project>-<stream>.md` containing:
  - `## Known Facts`: entry-point `file:line` ranges, newest sibling exemplar `file:line`, exact scoped fast test command (`go test ./<pkg> -run <AffectedTests> -race`, `npx vitest run <spec>`, `bundle exec rspec <spec>`), and repo lint/wording gate commands.
  - `## Affected Callers`: from Phase 2/3.
  - `## Hypotheses to Falsify`: any unproven assumptions.
- Update `<appDataDir>/brain/<conversation-id>/task.md` with the tasks to execute.

### 2. Dispatch Implementer (`invoke_subagent`)

Dispatch the dedicated `implementer` agent (`~/.gemini/config/plugins/antigravity-subagents/agents/implementer.md`):

```text
invoke_subagent({
  TypeName: "implementer",
  Role: "Phase 4 TDD Implementer",
  Model: "inherit",
  Prompt: "Read /tmp/subagent-context-<project>-<stream>.md and PLAN.md first. Execute Task N via strict TDD (RED -> GREEN -> REFACTOR) with zero code comments. Run only the scoped fast test and lint commands listed in the context file (if a CLI fails twice consecutively, read Makefile/package.json/vitest.config.* immediately). Return the mandatory 4-section handoff: Changes Made, Flawed Assumptions, Carried Risks, Verification Evidence."
})
```

- **Model routing (`GEMINI.md` §0.9):** Always pass `Model: "inherit"` (runs **Argon Medium** `1×` by default, or Carbon Medium if the UI is on Carbon). Never pass `Model: "pro"` on Argon or Carbon sessions. Reserve `Model: "flash"` strictly for `<50` LOC single-file mechanical boilerplate/renames with zero error-handling, concurrency, or security judgment.
- **Cap warm reuse at 1 phase (+ its fix round):** Never reuse a warm implementer across more than 1 roadmap phase (`~80k–100k` context). Spawn a fresh subagent for the next phase rather than re-billing a `150k–200k` history across `30+` tool calls.
- **End your turn immediately** after `invoke_subagent` if waiting asynchronously; never poll `manage_subagents` in a loop.

### 3. Coordinator Fast Gates & Simplification (Phases 5–7)

When the implementer returns its 4-section handoff:
1. Run the scoped fast unit tests (Phase 5) and linter/typecheck (Phase 6) in the Coordinator's shell and verify exit code `0`.
2. Run `code-simplifier` (Phase 7) on the modified files if needed, re-verifying fast tests and lint.
3. Append the passing test/lint output and `git diff -U3` to `/tmp/subagent-context-<project>-<stream>.md`.

### 4. Adversarial Review (`comprehensive-code-review` & `fix-check` — Phases 8–9)

Follow [`comprehensive-code-review`](../comprehensive-code-review/SKILL.md):
- **On Argon UI sessions (pre-PR review):** Stage uncommitted edits and spawn parallel one-shot `gemini-4-carbon-high` `swarm` workers (`code-rev` + `sec-rev`, plus `sql-rev` if SQL/migrations changed; `workspace: "inherit"`, no `monitor`, read-only by brief forbidding git/tree mutations, immediate `kill`).
- **When UI is on Carbon, on Gemini CLI, or for small changes:** Dispatch parallel read-only subagents via `invoke_subagent` (`TypeName: "code-quality-reviewer"` and `TypeName: "security-reviewer"`, plus `TypeName: "sql-reviewer"` if SQL changed; `Model: "inherit"`, `tools: [read_file]`, **zero spikes / zero `run_command`**).
- If CRITICAL or IMPORTANT findings are raised, fix them and verify via [`fix-check`](../fix-check/SKILL.md) (`invoke_subagent` `TypeName: "code-quality-reviewer"`, `Model: "inherit"` — never re-run a full review).
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
