---
name: lifecycle
description: Run the full 12-phase Development Lifecycle (0. Shape → 1. Receive → 2. Trace → 3. Plan → 3½. Plan review → 4. Implement → 5. Test → 6. Lint → 7. Simplify → 8. Code review → 9. SQL review → 10. Blast-radius verify → 11. Commit → 12. Handoff) with every phase gate enforced. Use when the user invokes /lifecycle, /start, or /begin, or asks to run the full development workflow on a task.
---

# Development Lifecycle

Run all 12 phases in order on the current task. Every phase has a gate; do not start the next phase until the current gate passes. Only pure documentation updates and `git revert` skip this workflow. Even a one-line code fix runs every phase (only Phase 3½ `plan-review` may be skipped for a trivial change with an obvious, self-contained blast radius).

## Setup

Before starting Phase 0, create and maintain the session's `task.md` artifact (`<appDataDir>/brain/<conversation-id>/task.md`) with one checklist item per phase (Phases 0 through 12, marking 0½ and 9 as conditional). Check off each phase only when its gate passes.

## The 12 Phases

### Phase 0: Shape
- **New or user-facing work** (feature, product, page, design, PRD):
  1. Run `craftsmans-wisdom` (unless a point-of-view brief is already confirmed in this session or on disk).
  2. Offer `future-grilling` on the details (or `future-wayfinder` if the work is too large for one session), then `write-a-prd` or `brainstorming` if needed.
  3. Put 3+ user questions on one `grilling-page`; keep 1–2 questions in chat.
- **Internal technical work** (bug fix, refactor, backend/CLI/infra change with no new UX):
  - Ask **once** whether to skip Phase 0 or run `future-grilling` on any open decisions.
- **Gate:** Brief confirmed by user, or Phase 0 explicitly skipped for internal technical work.

### Phase 0½: UI Design *(Conditional)*
- **When required:** Net-new pages, new apps, major/non-trivial changes to existing UI pages, or whenever requested. Skip for backend-only or trivial UI tweaks.
- **Action:** Run `ui-design` (`huashu-design` 3-direction method) to generate 3 distinct high-fidelity interactive HTML prototypes in `design-demos/index.html`, open it in the browser, and **stop** for the user to pick a direction (`direction-approved.md`).
- **Gate:** Direction selected by user, or N/A recorded for backend/trivial UI.

### Phase 1: Receive
- **Action:** Read the request and relevant repo instructions (`GEMINI.md`, `AGENTS.md`, `CONTRIBUTING.md`, `docs/adr/`). Load Prism `session_load_context` on turn 1. Suggest a tracker ticket and create it only after the user agrees. Ensure you are on a feature branch (never `main`; if reusing a branch with an existing PR, run `gh pr view --json state,mergedAt && git fetch origin main` first).
- **Gate:** Task and constraints understood; `task.md` created; feature branch checked out.

### Phase 2: Trace
- **Action:** Map the call chain both ways before planning or editing:
  - **Upward:** Every caller out to the controller, job, CLI command, rake task, or API endpoint, and what each caller expects.
  - **Downward:** Side effects, transactions, external calls, and what raises or returns errors.
  - **Invisible edges:** Grep for dynamic dispatch (`send`, `constantize`, serializers, callbacks, string-named jobs) and cross-package E2E suites where applicable.
- **Gate:** Every caller named, with what it expects.

### Phase 3: Plan
- **Action:** Follow `writing-plans`. Write the plan (`PLAN.md`, never committed) covering all work, including:
  - **Goal**
  - **Affected Callers** (callers the change breaks with consequence, plus unbroken callers with no test)
  - **Approach** (reusing existing helpers and matching the newest sibling file's pattern, citing `file:line`)
  - **TDD tasks** with exact file paths, signatures, test cases, and gate commands — never function bodies
  - **Risks**, **Rollback**, **Out of scope**
- **Gate:** Plan file exists on disk with `Affected Callers` and repo lint/wording gate invariants.

### Phase 3½: Plan Review
- **Action:** Follow `plan-review`. On Antigravity Argon UI sessions, spawn a one-shot `gemini-4-carbon-high` `swarm` worker (`plan-rev`, `workspace: "inherit"`, no `monitor`, read-only by brief forbidding git/tree mutations, immediate `kill`); when UI is already on Carbon or on Gemini CLI / small plans, dispatch `invoke_subagent` (`TypeName: "plan-reviewer"`, `Model: "inherit"`, `tools: [read_file]`) for a single adversarial review pass (**zero spikes / zero shell commands**). Where configured for the repo, also run the `review-lens` corpus check (`--code` and `--concept`). Fix all CRITICAL and IMPORTANT findings in the plan; never send the plan back for a second review.
- **Exemption:** Skip only for a trivial change with an obvious, self-contained blast radius.
- **Gate:** All CRITICAL and IMPORTANT plan findings fixed (or skipped for trivial change).

### Phase 4: Implement
- **Action:** Follow `test-driven-development` (`RED → GREEN → REFACTOR`) and write zero code comments:
  - **In Antigravity (`/start`, `/begin`, `/peer-pipeline-swarm`):** For standard 1-stream or multi-stream changes (`≥3` files or `>3` tool calls), run [`peer-pipeline-swarm`](../peer-pipeline-swarm/SKILL.md) (`gemini-4-argon-medium` `impl` with `workspace: "share"`, `claude-opus-5-5-high` `code-rev`, `gemini-4-carbon-high` `sec-rev`/`sql-rev`, and `gemini-4-hydrogen-low` `monitor`) so Phases 4–9 execute hands-off and wake the Coordinator on `@coordinator [green]`.
  - **Via `subagent-driven-development` (`invoke_subagent`):** Or dispatch `invoke_subagent` (`TypeName: "implementer"`, `Model: "inherit"`) reading `/tmp/subagent-context-<project>-<stream>.md` and returning the 4-section handoff (`Changes Made`, `Flawed Assumptions`, `Carried Risks`, `Verification Evidence`). For quick (`<5`-turn, `≤2`-file) changes where all traced files are already in context, implement inline.
- **Gate:** Fast scoped unit tests pass; nothing duplicated; zero new comments added.

### Phase 5: Test
- **Action:** Run fast unit tests scoped to the affected specs in the shell (e.g., `go test ./<affected-pkg> -run <AffectedTests> -race`, `npx vitest run <spec>`, `bundle exec rspec <spec_file>`; `<15s` target). Never run unscoped `go test ./...`, slow DB integration suites, or E2E suites locally between changes.
- **Gate:** Zero test failures (`exit code 0` verified in this turn).

### Phase 6: Lint
- **Action:** Run the project's linter, formatter check, and typechecker in the shell (e.g., `test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run`, `npm run lint`, `bundle exec rubocop`). Append passing test/lint output and `git diff -U3` to `/tmp/subagent-context-<project>-<stream>.md`.
- **Gate:** Zero offenses (`exit code 0` verified in this turn).

### Phase 7: Simplify
- **Action:** Follow `code-simplifier` on the recently changed code to improve reuse, clarity, and efficiency without changing behavior. Re-run Phase 5 and Phase 6 checks if any file changed.
- **Gate:** Code simplified and fast tests + lint still pass.

### Phase 8: Code Review
- **Action:** Follow `comprehensive-code-review`. Under `peer-pipeline-swarm`, `code-rev` (`claude-opus-5-5-high`) and `sec-rev` (`gemini-4-carbon-high`) review the committed `share` branch peer-to-peer with `@impl`; on standalone Argon sessions, spawn one-shot `gemini-4-carbon-high` `swarm` workers (`code-rev` + `sec-rev`, no `monitor`, immediate `kill`); when UI is on Carbon or on Gemini CLI, dispatch `invoke_subagent` (`TypeName: "code-quality-reviewer"` and `TypeName: "security-reviewer"`, `Model: "inherit"`, `tools: [read_file]`) in parallel reading `/tmp/subagent-context-<project>-<stream>.md`. Run the `review-lens` corpus check on the diff where configured. Fix all CRITICAL and IMPORTANT findings; verify any fix round via `fix-check` (never a second full review). Delete the temp context file once approved.
- **Gate:** Approved with zero open CRITICAL or IMPORTANT findings; temp context file deleted.

### Phase 9: SQL Review *(Conditional)*
- **When required:** Only if SQL queries, migrations, scopes, or ORM mutations changed.
- **Action:** Dispatch a fresh read-only `sql-reviewer` (`gemini-4-carbon-high` `sql-rev` in `peer-pipeline-swarm` / Argon sessions, or `invoke_subagent` `TypeName: "sql-reviewer"`, `Model: "inherit"`) following `sql-optimization-patterns` to audit query plans, indexes, security, and defensive coding. Fix any CRITICAL or IMPORTANT findings.
- **Gate:** SQL reviewed and approved, or N/A if no queries/migrations changed.

### Phase 10: Blast-Radius Verify
- **Action:** **Runs last before commit.** Any code edit made during Phase 7 (Simplify) or Phase 8/9 fix rounds re-opens Phase 10. Run repo version bump once if applicable (`node scripts/version.mjs bump patch|minor && make version-check`), re-run scoped fast unit tests and lint in the Coordinator's shell, and walk every row in `Affected Callers` plus callers of any symbol touched in the final diff that the plan did not list, verifying each with concrete evidence.
- **Gate:** Every affected caller verified with evidence; fast tests and lint green on the final tree.

### Phase 11: Commit
- **Action:** Create a clean conventional commit on a feature branch (never `main`), following repo and org branch/ticket conventions.
- **Gate:** `git status` clean and commit recorded on the feature branch.

### Phase 12: Handoff
- **Action:**
  - For user-facing work, run `editor-pass` against the Phase 0 brief before shipping.
  - On repos configured for auto-PR, push the branch, open the PR using `pull-request-description`, immediately ask whether the user wants to run `/copy-edit` while CI is running (`gh pr checks <pr> --watch`) so any copy-edit changes land on the same PR before merge, and watch CI until every check is green. Never auto-merge.
  - Present a concise summary of what changed, verification evidence, and the PR/commit link.
- **Gate:** All `task.md` items checked off; user acknowledges (and CI is green on auto-PR repos).
