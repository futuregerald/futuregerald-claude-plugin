# {{PROJECT_NAME}} - Antigravity & Gemini Configuration

## Project Overview

{{PROJECT_DESCRIPTION}}

## Key Directories

{{KEY_DIRECTORIES}}

---

## Delegating Work & Model Routing

**Filter bulky output at the shell, never into context.** `cmd > /tmp/out.log 2>&1; rc=$?; tail -20 /tmp/out.log; echo $rc` — capture the exit code **before** the pipe, because `cmd | tail` exits 0 when `cmd` failed. For a count use a counting primitive (`grep -c`), never a match list tallied by eye.

**Triage before delegation (`Argon Medium` `1×` default + surgical `Carbon High` `2×` & `Opus 5.5 High` reviews):**
- **Cost floor & inline lookup ceiling:** A subagent costs ~57k tokens up front (~31k with restricted `tools: [read_file]` grants in `antigravity-subagents`). Inline lookup ceiling: `≤2` files and `≤3` tool calls (filtered via `git grep -n` with `WaitMsBeforeAsync: 5000`). Identify the entry-point `file:line` first before dispatching any helper.
- **Lead / Planner (Phases 0–3, 5–7, 10–12):** Runs **inline** on the active UI model (**Argon Medium** `1×` default; switch UI to **Carbon Medium / High** `2×` only for ~10% net-new multi-layer architecture, auth/crypto/security design, concurrency root-cause fixes, or zero-downtime DB migrations).
- **Phase 4–9 Execution (`/start`, `/begin`, `/peer-pipeline-swarm`, or `subagent-driven-development`):**
  - **In Antigravity (`peer-pipeline-swarm`):** Spawn an isolated `gemini-4-argon-medium` TDD implementer (`impl`, `workspace: "share"`), cross-family high-deliberation peer reviewers (`claude-opus-5-5-high` `code-rev` + `gemini-4-carbon-high` `sec-rev` / `sql-rev`, `workspace: "inherit"`, read-only by brief), and a `gemini-4-hydrogen-low` stall monitor (`monitor`, `workspace: "inherit"`) so build, test, review, and fix rounds resolve peer-to-peer before waking the Coordinator once on `@coordinator [green]`.
  - **Via `invoke_subagent` (`subagent-driven-development` / Gemini CLI):** Dispatch `invoke_subagent` with `TypeName: "implementer"` and **`Model: "inherit"`** (never pass `Model: "pro"` on Argon or Carbon sessions; reserve `Model: "flash"` strictly for `<50` LOC single-file mechanical boilerplate/renames). Cap warm implementer reuse at **1 roadmap phase (+ its fix round)**.
- **Adversarial Reviewers (Phases 3½, 8, 9, 12):** On Argon UI sessions, use one-shot `gemini-4-carbon-high` `swarm` workers (no `monitor`, read-only brief forbidding git/tree mutations, immediate `kill`); when UI is on Carbon, on Gemini CLI, or for small changes and `fix-check`, use `invoke_subagent` with `Model: "inherit"` and the harness-enforced read-only `TypeName` (`plan-reviewer`, `code-quality-reviewer`, `security-reviewer`, `sql-reviewer`, `editor-pass-reviewer` — `tools: [read_file]`, **zero spikes / zero `run_command`**).

**Never delegate:** synthesis whose input is this conversation · writing in the user's voice (tickets, PR bodies, messages) · the final verification gate in the Coordinator's shell.

---

## Development Lifecycle (MASTER WORKFLOW)

**Every code change follows this 12-phase workflow (`/lifecycle`, `/start`, or `/begin`). Every code change gets a written plan (Phase 3); only Phase 3½ (`plan-review`) may be skipped for a trivial change with an obvious, self-contained blast radius.**

| Phase | Action | Gate |
|-------|--------|------|
| 0. SHAPE | For new or user-facing work (feature, product, page, design, PRD): interview the user on what good means and write a confirmed brief (`craftsmans-wisdom` / `/shape`), optionally grill the details (`future-grilling`), then a PRD (`write-a-prd`) or `brainstorming`. For internal technical work: ask once per session whether to skip Phase 0 or run `future-grilling` on open decisions | Brief confirmed (or explicitly skipped for internal technical work) |
| 0½. UI DESIGN | For net-new pages, new apps, major/non-trivial changes to existing UI pages, or anytime requested: run `/ui-design` (`huashu-design`) to generate 3 distinct high-fidelity interactive prototypes on a single HTML comparison page (`design-demos/index.html`) and open it in the browser so the user can pick a direction (`direction-approved.md`) | Direction selected (or N/A for non-UI / trivial UI work) |
| 1. RECEIVE | Understand the task and repo rules; create and maintain `task.md` (`<appDataDir>/brain/<conversation-id>/task.md`) | Task understood & `task.md` created |
| 2. TRACE | Trace the call chain, up and down, of every symbol the change touches (`git grep -n`), including dynamic dispatch and cross-package E2E suites | Every caller named with contract expectations |
| 3. PLAN | Write the uncommitted plan (`PLAN.md`) with `writing-plans`, including `Affected Callers`, sibling exemplar `file:line` citations, signatures, test cases, and gate commands | Plan exists with `Affected Callers` & repo gates |
| 3½. PLAN REVIEW | `plan-review` in a fresh read-only reviewer (`gemini-4-carbon-high` one-shot `swarm` worker `plan-rev` on Argon UI, or `invoke_subagent` `TypeName: "plan-reviewer"`, `Model: "inherit"`), once. Fix every CRITICAL and IMPORTANT finding in `PLAN.md` | Review run; findings fixed (or skipped for trivial change) |
| 4. IMPLEMENT | TDD (`test-driven-development`): RED → GREEN → REFACTOR via `/start` (`peer-pipeline-swarm`) or `subagent-driven-development` (`invoke_subagent` `TypeName: "implementer"`, `Model: "inherit"`). Check existing helpers first (**Reuse Before You Build**) | Fast tests exist and pass; nothing duplicated; zero new comments |
| 5. TEST | Run fast unit tests scoped to the affected specs only (e.g. `go test ./<affected-pkg> -run <AffectedTests> -race`, `npx vitest run <spec>`, `bundle exec rspec <spec>`; `<15s` target). Never run unscoped full-repo suites or E2E suites between changes | Zero failures (`exit code 0`) |
| 6. LINT | Run the project's typecheck and linter (`{{TYPECHECK_COMMAND}}`) and append passing output + `git diff -U3` to `/tmp/subagent-context-<project>-<stream>.md` | Zero offenses (`exit code 0`) |
| 7. SIMPLIFY | `code-simplifier` pass on recently changed code; re-verify fast tests and lint | Simplified |
| 8. CODE REVIEW | `comprehensive-code-review` in parallel read-only reviewers (`code-rev` + `sec-rev` in `peer-pipeline-swarm` / `swarm`, or `invoke_subagent` `code-quality-reviewer` + `security-reviewer`, `Model: "inherit"`). Verify fix rounds with `fix-check` (never a second full review) | Approved; temp context file deleted |
| 9. SQL REVIEW | If database queries or migrations changed: `sql-reviewer` (`sql-optimization-patterns`) | SQL reviewed |
| 10. BLAST-RADIUS VERIFY | **Runs last, after every code-mutating phase.** Re-run fast unit/lint gates in the Coordinator's shell and walk every `Affected Callers` row plus callers of any symbol touched in review fixes | Every caller verified with evidence |
| 11. COMMIT | `git commit`, conventional format on a feature branch (never `main`) | Commit created |
| 12. HANDOFF | For user-facing work, run `editor-pass` against the brief before shipping. Push the branch and open the PR (`pull-request-description`), offer `/copy-edit` while watching CI (`gh pr checks <pr> --watch`), and never auto-merge | PR URL reported & user acknowledged |

**Exceptions that skip planning:** pure doc updates, `git revert`.

---

## Code Comments — Hard Rule

**Write no code comments. Ask the user before adding any, every time.** Prefer clear names, extracted helpers, and tests that state the case. Leave existing comments alone unless your change makes them wrong.

---

## Mandatory Skills

| Trigger | Skill / Slash Command |
|---------|-------|
| Full 12-phase development lifecycle (`0. SHAPE` → `12. HANDOFF`) | `/lifecycle`, `/start`, `/begin` (powered by `/peer-pipeline-swarm` in Antigravity) |
| Model routing & delegation budget questions | `future-model-router`, `antigravity-orchestration` |
| Before planning new or user-facing work | `/shape`, `/craftsmans-wisdom`, `/future-grilling`, `/write-a-prd`, `/brainstorming` |
| Net-new pages, new apps, or major UI changes (Phase 0½) | `/ui-design` (`huashu-design`) |
| Three or more questions for the user | `/grilling-page` (1–2 questions stay in chat) |
| Any code change (Phase 3 Plan & Phase 3½ Review) | `/writing-plans`, `/plan-review` |
| Feature implementation & TDD (Phase 4) | `/start`, `/peer-pipeline-swarm`, `/test-driven-development`, `/subagent-driven-development` |
| Simplifying recently changed code (Phase 7) | `/code-simplifier` |
| Code & SQL review (Phases 8–9) and fix verification | `/comprehensive-code-review`, `/sql-optimization-patterns`, `/fix-check` |
| Post-build quality check & PR creation (Phase 12) | `/editor-pass`, `/pull-request-description`, `/copy-edit` |
| Context estimate near 205k tokens | `/baton-pass` |

---

## Quick Reference (CI / Full-Suite & Build Commands)

*(During Phase 5 local iteration, run scoped unit tests on affected packages/specs rather than the full CI suite below.)*

```bash
{{TEST_COMMAND}}
{{TYPECHECK_COMMAND}}
{{BUILD_COMMAND}}
```

---

<!-- LANGUAGE_SPECIFIC -->
