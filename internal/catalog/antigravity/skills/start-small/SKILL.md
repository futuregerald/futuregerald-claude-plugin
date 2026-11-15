---
name: start-small
description: >-
  Lightweight Antigravity 12-phase Development Lifecycle (/start-small) for
  smaller, well-bounded code changes and bug fixes. Works directly inside a
  Carbon Medium chat without switching the UI dropdown: runs Phases 0–3 inline,
  uses a fast 1-turn invoke_subagent plan review (or skips if trivial), executes
  Phases 4–9 via Peer-Pipeline Swarm with an Argon Medium (1x) share implementer
  + Gemini 4 Carbon High peer reviewers (preserving weekly Anthropic Opus quota
  for /start) + Hydrogen Low stall monitor (or inline fast-path for up to 2 files),
  and runs Phases 10–12 inline. Use whenever the user invokes /start-small.
---

# Lightweight Antigravity Development Lifecycle (`/start-small`)

Executes the 12-phase Development Lifecycle tuned for **smaller, well-bounded code changes, bug fixes, and routine single-surface enhancements** — without needing to change your chat model dropdown off `Carbon Medium` and without spending weekly Anthropic `Opus` quota.

- **When to use `/start-small` vs. `/start`:**
  - **`/start` (Default):** Multi-surface features, E2EE/auth/privacy boundaries, sync/concurrency state machines, DB migrations, and complex debugging (`impl: gemini-4-carbon-medium`, `code-rev: claude-opus-5-5-high`, `plan-rev: gemini-4-carbon-high` via `swarm`).
  - **`/start-small`:** Bounded bug fixes, single-package enhancements, straightforward UI/copy tweaks, and clean refactors (`impl: gemini-4-argon-medium` `1×`, `code-rev` + `sec-rev: gemini-4-carbon-high`, 1-turn `invoke_subagent` `plan-review` or trivial skip).

## Phase Sequence & Model Routing

Create and maintain `<appDataDir>/brain/<conversation-id>/task.md` with the phases below, checking off each phase only when its gate passes:

| # | Phase | Execution & Model | Action | Gate |
|---|---|---|---|---|
| **0–0½** | **Shape & UI** | **Inline Coordinator** | Skip Phase 0 for bounded technical fixes once confirmed (or run brief check if user-facing behavior is new); skip Phase 0½ unless adding a net-new screen. | Scope clear |
| **1** | **Receive** | **Inline Coordinator** | Load Prism `session_load_context` (turn 1). Check repo rules (`GEMINI.md`, `AGENTS.md`). Ensure a clean feature branch (`gh pr view --json state,mergedAt && git fetch origin main` if reusing). | `task.md` created; feature branch ready |
| **2** | **Trace** | **Inline Coordinator** | `git grep -n` callers upward and side effects downward (including cross-package E2E suites where applicable). | Callers & contracts identified |
| **3** | **Plan** | **Inline Coordinator** | Inspect repo lint/wording gates and sibling exemplars (`file:line`). Write concise uncommitted `PLAN.md` (`future-writing-plans`) with Goal, `Acceptance Criteria & Success Metrics` (wired `file:Symbol`), `Affected Callers`, TDD tasks, and Rollback. **MANDATORY STOP FOR PLAN APPROVAL:** Present Phase 2 findings, Acceptance Criteria, and proposed `PLAN.md` in chat and **STOP calling tools to wait for approval** (unless user explicitly requested autonomous execution). | `PLAN.md` written & approved by user |
| **3½** | **Plan Review** | **`invoke_subagent` (`Model: "inherit"`)** *(or skip if trivial)* | **Skip** if blast radius is obvious and self-contained. Otherwise run a fast 1-turn harness-enforced read-only `invoke_subagent` (`TypeName: "plan-reviewer"`, `Model: "inherit"`, `tools: [read_file]`, zero spikes / zero `run_command`) + `review-lens` where configured. Fix CRITICAL/IMPORTANT findings once in place. | Plan verified (or skipped as trivial) |
| **4–9** | **Build & Peer Review** | **Micro fast-path (`≤2` files, `<5` turns):** Inline + `invoke_subagent` review<br>**Standard small (`≥3` files):** **Peer-Pipeline Swarm (`small` profile):**<br>• `impl`: `gemini-4-argon-medium` (`workspace: "share"`)<br>• `code-rev`: `gemini-4-carbon-high`<br>• `sec-rev`: `gemini-4-carbon-high`<br>• `sql-rev`: `gemini-4-carbon-high` *(if SQL changed)*<br>• `monitor`: `gemini-4-hydrogen-low` | **If `≤2` files / `<5` turns already in context:** Implement via TDD inline, run fast unit/lint gates, and dispatch parallel read-only `invoke_subagent` (`code-quality-reviewer` + `security-reviewer`, `Model: "inherit"`), avoiding `~230k` 4-worker `swarm` boot overhead.<br>**Otherwise (`≥3` files):** Execute [`peer-pipeline-swarm`](../peer-pipeline-swarm/SKILL.md) using the **`/start-small` roster** above (`impl` pinned to `gemini-4-argon-medium`, `code-rev` + `sec-rev` pinned to `gemini-4-carbon-high` on 5-hour Google quota). `impl` runs TDD, fast tests, lint, simplification, commits on `share`, and resolves reviews peer-to-peer until `@coordinator [green]`. | Fast gates & reviews green; `share` branch merged before `kill`; temp files removed |
| **10** | **Blast-Radius Verify** | **Inline Coordinator** | Run repo version bump once if required (e.g., `node scripts/version.mjs bump ... && make version-check`). Run scoped fast unit tests + lint/typecheck in Coordinator shell (`exit 0`). Verify every `Affected Callers` row (plus rendered Light/Dark screenshots if UI changed). | Fast gates green in Coordinator; callers verified |
| **11** | **Commit** | **Inline Coordinator** | Conventional commit (zero code comments, ticket key per org rules) on feature branch. | Committed |
| **12** | **Handoff** | **Inline Coordinator** | If user-facing UI/copy changed, run lightweight `invoke_subagent` `editor-pass` (`TypeName: "editor-pass-reviewer"`, `Model: "inherit"`). On auto-PR repos, push, open PR (`pull-request-description`), offer `/copy-edit` while watching CI (`gh pr checks <pr> --watch`), and save Prism ledger/handoff. Never auto-merge. | User acknowledged (and CI green on auto-PR repos) |
