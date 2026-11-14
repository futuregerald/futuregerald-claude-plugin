---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code. Also writes and keeps current the committed roadmap (docs/roadmap.md) when the work spans more than one plan - phases, an initiative, or a PRD whose stories ship over more than one plan.
---

# Writing Plans

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, the function signatures, the test cases, and the verification commands they will run, with file-path and line-number references to existing patterns. Give them the whole plan as bite-sized tasks, without writing the implementation code into the plan. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm using the writing-plans skill to create the implementation plan."

**Context:** This should be run in a dedicated worktree (created by brainstorming skill) or on a feature branch before any source file is edited.

**Save plans to:** `docs/plans/YYYY-MM-DD-<feature-name>.md` or `PLAN.md` (uncommitted, per `GEMINI.md` Phase 3).

## The Roadmap (`docs/roadmap.md`)

A plan in `docs/plans/` is tactical — one slice of work, written against the code as it exists today. When the work is larger than one plan (a multi-phase project, an initiative, or a PRD whose user stories cannot ship in a single PR), a single plan either bloats or goes stale before Phase 2 starts.

For multi-plan work, also create or update **`docs/roadmap.md`** — the committed, living index of phases and the strategy we agreed on.

**Single-plan work does not need a roadmap.** If everything ships in one plan and one PR, skip straight to writing the plan below.

### When to create or update `docs/roadmap.md`

- **Creating a PRD (`write-a-prd`) or finishing `future-grilling` / `brainstorming`** on work that will take more than one plan -> create `docs/roadmap.md` (or add a section to it) before writing the Phase 1 plan.
- **Starting the next phase** -> read `docs/roadmap.md` first, check that the remaining phases still make sense against what earlier phases actually built, update the roadmap if the shape changed, then write the tactical plan for this phase only.
- **Finishing a phase** -> mark the phase `done` in `docs/roadmap.md` with a link to its plan and PR, and note any departures that affect later phases.

### What belongs in `docs/roadmap.md`

Keep it short — a screenful or two. It is a map, not a second plan.

```markdown
# Roadmap: <Initiative or Product Area>

**PRD / Spec:** #<issue-number> (or link to spec/brief)
**Point-of-view brief:** `~/.gemini/point-of-view/<slug>.md` (if one exists)

## Agreed Strategy

The 3-6 load-bearing choices from grilling/PRD that every phase must respect
(and the paths we explicitly rejected, in one line each, so a future session
does not re-propose them). Detail lives in `docs/adr/` if an ADR was written.

## Phases

| # | Phase | Stories / Scope | Depends on | Status | Plan |
|---|-------|-----------------|------------|--------|------|
| 1 | <name> | US-1, US-2: <one-line deliverable> | — | in-progress | `docs/plans/YYYY-MM-DD-<phase-1>.md` |
| 2 | <name> | US-3, US-4: <one-line deliverable> | Phase 1 | planned | — |
| 3 | <name> | US-5: <one-line deliverable> | Phase 1, 2 | planned | — |

## Open / Deferred

Anything deliberately pushed out of scope or waiting on a future phase.
```

**Coupling to the PRD:** Every user story in the PRD appears in the `Stories / Scope` column of at least one phase. If a story is cut or moved between phases, update both the PRD issue and `docs/roadmap.md` so they never disagree on what is shipping when. Only write the detailed `docs/plans/YYYY-MM-DD-*.md` for the phase you are about to build — later phases stay as one-line rows in the roadmap until their turn comes, because planning them in detail against code that does not exist yet guarantees drift.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

## Plan Document Header

**Every plan MUST start with this header:**

```markdown
# [Feature Name] Implementation Plan

> **For Antigravity / Gemini:** REQUIRED SUB-SKILL: Use `/start` (`peer-pipeline-swarm`) or `subagent-driven-development` (`invoke_subagent` with `TypeName: "implementer"`, `Model: "inherit"`) to implement this plan.

**Goal:** [One sentence describing what this builds]

**Roadmap:** `docs/roadmap.md` — Phase N of M *(include when part of multi-plan work; omit for single-plan work)*

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

**Affected Callers:**
- Breaking callers (with consequence): ...
- Untested unbroken callers: ...

---
```

## Task Structure

Plans specify **interfaces, cases, and gates — never function bodies**. A plan whose tasks contain pasted implementation code has already done Phase 4 inside Phase 3 without tests first, and the code will drift the moment Task 1 touches a real file.

```markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145` (follow pattern in `sibling/file.py:40-65`)
- Test: `tests/exact/path/to/test.py`

**Contracts & cases:**
- Signature: `function_name(input: Type) -> ReturnType` (raises `ErrorType` when ...)
- Test cases to cover (RED -> GREEN, one at a time):
  1. happy path: `<input>` -> `<expected output>`
  2. edge / boundary: `<input>` -> `<expected behavior>`
  3. error / unauthorized / invalid: `<input>` -> `<expected error>`
- Gate command: `pytest tests/path/test.py::test_name -v` (expected FAIL on RED, PASS on GREEN)
- Commit: `feat: add <component>`
```

## Remember
- Exact file paths always, with `file:line` references to existing patterns to follow
- Signatures, test cases, and gate commands — never pasted function or test bodies
- Exact commands with expected output
- Reference relevant skills
- DRY, YAGNI, TDD, frequent commits
- Include `Risks`, `Rollback`, and `Out of scope` sections

## Plan Review Gate (Mandatory Before Execution)

Before handing the plan off for implementation — and before asking the user how they want to execute it — run the **`plan-review`** skill (`gemini-4-carbon-high` one-shot `swarm` worker `plan-rev` on Argon UI sessions, or `invoke_subagent` `TypeName: "plan-reviewer"`, `Model: "inherit"` when UI is on Carbon or on Gemini CLI).

1. Run `plan-review` against the plan file you just wrote.
2. Address every CRITICAL and IMPORTANT finding in the plan file directly. Do **not** re-run `plan-review` after fixing (one pass only).
3. Only skip this gate for trivial changes with an obvious, self-contained blast radius.

For a multi-phase plan, re-run `plan-review` on the **remaining** phases at each phase boundary before starting the next phase.

## Execution Handoff

After saving the plan and completing the `plan-review` gate, offer execution choice:

**"Plan complete, reviewed, and saved to `docs/plans/<filename>.md`. Two execution options:**

**1. Peer-Pipeline Swarm (`/start` / `peer-pipeline-swarm`)** - Spawns an isolated `gemini-4-argon-medium` share-workspace implementer + cross-family high-deliberation peer reviewers (`claude-opus-5-5-high` `code-rev` + `gemini-4-carbon-high` `sec-rev`) + `gemini-4-hydrogen-low` stall monitor for hands-off Phases 4–9

**2. Subagent-Driven (`subagent-driven-development`)** - Dispatches fresh `invoke_subagent` (`TypeName: "implementer"`, `Model: "inherit"`) per task/phase with Coordinator gates between tasks

**Which approach?"**
