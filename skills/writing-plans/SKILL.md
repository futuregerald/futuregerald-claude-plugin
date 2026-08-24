---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code
tags: [workflow]
---

# Writing Plans

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our
codebase and questionable taste. Document everything they need: which files to touch for each
task, the code, the tests, docs they might need to check, how to verify it. Give them the
whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer who knows almost nothing about our toolset or problem
domain, and does not know good test design well.

**Announce at start:** "I'm using the writing-plans skill to create the implementation plan."

**Context:** run this in a dedicated worktree (created by the brainstorming skill).

**Save plans to:** `docs/plans/<TICKET>-<slug>.md` when a ticket or issue key exists,
otherwise `docs/plans/YYYY-MM-DD-<slug>.md`. This is the single source for the plan path —
other documents refer here rather than restating it.

**Never commit a plan.** Add `docs/plans/` to `.git/info/exclude` if it is not already
ignored.

## Required sections

A plan is rejected at PLAN REVIEW without these.

| Section | What it is | Reference |
|---|---|---|
| **Claim Ledger** | Every factual assertion, with `file:line` and what you actually read | [references/claim-ledger.md](references/claim-ledger.md) |
| **Could Not Verify** | Uncertainty, recorded rather than laundered into prose | [references/claim-ledger.md](references/claim-ledger.md) |
| **Impact Analysis** | The call chain, up and down, of every symbol touched | [references/impact-analysis.md](references/impact-analysis.md) |
| Goal · Approach · Risks · Rollback · Out of scope | | |

For a plan with more than one task, also carry contract deltas and a moving baseline —
[references/multiphase.md](references/multiphase.md).

Before tracing anything, read [references/indexing.md](references/indexing.md): use the index
if there is one, build it once if there is not, and degrade to grep without blocking if the
tooling is absent.

### Why these three

Measured across 83 real adversarial reviews: **false premises were 43% of gating findings**
and the sole cause of 12 rejections; **verification that cannot fail was 13%** and is owned
by no later gate; **intra-plan interference was 8%**. The ledger, the gate-falsifiability
rules and the moving baseline target those three directly.

## Bite-sized task granularity

Each step is one action, 2–5 minutes:

- Write the failing test
- Run it and watch it fail — with the expected output written down
- Write the minimal implementation
- Run the tests and watch them pass
- Commit

## Plan document header

```markdown
# [Feature Name] Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

**Base SHA:** [git rev-parse HEAD]

---
```

## Task structure

Each task names exact files, gives complete code, and states the exact command with its
expected output.

- **Files:** Create / Modify (with line ranges) / Test
- **Steps:** failing test → watch it fail → implement → watch it pass → commit
- **Gate:** a command whose exit status actually reflects success

A cited line range must match the replacement text supplied for it. If the Files list says
`foo.rb:12–18` but the replacement covers only `:12–15`, the implementer silently deletes
three lines.

**Every gate must be able to fail, and you must have watched it fail.** An existence check
that a one-line stub would satisfy is not a gate. Write `set -euo pipefail`; never let
`| head` or `| grep` decide an exit status. The full list of failure modes is in
[references/multiphase.md](references/multiphase.md) under Gate falsifiability.

## Remember

- Exact file paths always
- Complete code in the plan, not "add validation"
- Exact commands with expected output
- State what your test suite does **not** prove, so nobody cites a green run as evidence
- DRY, YAGNI, TDD, frequent commits

## Execution handoff

**MANDATORY GATE — do not skip, and do not offer execution before it passes.**

After saving the plan, PLAN REVIEW runs. Invoke the `plan-review` skill, which dispatches
three concurrent reviewers — `adversarial-plan-reviewer`, `plan-blindspot-hunter` and
`plan-consistency-checker` — against the saved plan file. Never review the plan yourself; you
wrote it, and a self-review is exactly what this gate replaces.

- **REJECT** (any CRITICAL or IMPORTANT from any reviewer): fix the plan, re-review with
  fresh agents. Do not offer execution, and do not begin implementation.
- **APPROVE**: report the findings and Verified Recommendations, then offer execution.

Only after APPROVE, offer the execution choice:

**"Plan complete, saved to `docs/plans/<filename>.md`, and APPROVED by adversarial review.
Two execution options:**

**1. Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between
tasks, fast iteration

**2. Parallel Session (separate)** — new session with executing-plans, batch execution with
checkpoints

**Which approach?"**

If Subagent-Driven: use superpowers:subagent-driven-development, stay in this session, fresh
subagent per task plus code review.

If Parallel Session: guide them to open a new session in the worktree, which uses
superpowers:executing-plans.
