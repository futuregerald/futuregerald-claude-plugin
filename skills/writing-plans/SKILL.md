---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code. Also writes and keeps current the committed roadmap (docs/roadmap.md) when the work spans more than one plan - phases, an initiative, or a PRD whose stories ship over more than one plan.
tags: [workflow]
model: inherit
effort: medium
---

# Writing Plans

## Overview

**Thinking budget:** Use **Medium thinking** by default to stay grounded in reading real code, lockfiles, and schemas rather than speculating. Bump to **HIGH thinking** only when architecting net-new subsystems, complex distributed state machines, or zero-downtime database migrations.

Write comprehensive implementation plans assuming the engineer has zero context for our
codebase and questionable taste. Document everything they need: which files to touch for each
task, the signatures, the test cases, docs they might need to check, how to verify it. Give them the
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

## Roadmap: work that spans more than one plan

A plan is disposable; the strategy behind a run of plans is not. When the work spans more
than one plan — phases, an initiative, a PRD whose stories ship over more than one plan —
write the roadmap before the first phase plan. For a single-plan change, skip it: the plan's
Goal is the record.

**Save it to `docs/roadmap.md` and commit it** on the branch the work happens on, each status
change in that phase's pull request. This is the deliberate contrast with plans:
a plan is never committed, the roadmap always is, because it is the only durable record of
the strategy the user agreed to. Without it the next session re-derives phases the user
already argued through.

It contains:

- **Status** — a table: phase, one-line scope, status (not started / in progress / done), plan slug (not a path: plans stay local)
- **Strategy** — why this order: what each phase unblocks, proves or de-risks
- **Each phase** — the work; acceptance criteria someone can check; the PRD user stories it
  delivers, by number. Every story sits in exactly one phase
- **Stop rules** — what halts or re-plans the sequence: a phase's criteria fail, a premise proves wrong
- **Agreement record** — the user's words agreeing to the strategy, quoted, with the date

Get the user's explicit agreement to the strategy before the first phase plan. Without it,
stop and ask; never paraphrase or infer one.

| Don't | Do |
|---|---|
| "Phase 2: search works well" | "Phase 2: searching `pasta` over the 500-recipe fixture returns its 12 pasta recipes in under 200 ms" |
| "User agreed to the phases" | "2026-03-04 — user: 'yes, ship import before sharing, sharing needs real data'" |

Keep it current:

- Update the status when a phase starts and when it finishes.
- When the user changes the strategy, update the roadmap and add a new Agreement record
  entry; never rewrite an earlier one.
- Each phase plan's header links to it.

## Required sections

Goal · **Impact Analysis** · Approach · Step-by-step TDD tasks · Risks · Rollback · Out of scope.

**Every plan MUST contain an Impact Analysis** — the call chain, up and down, of every symbol
the change touches. See *System Thinking: Trace Before You Touch* in CLAUDE.md. BLAST-RADIUS
VERIFY walks this list before COMMIT, so a plan without one leaves that phase nothing to check.

State each factual claim about existing code with the `file:line` you actually read, and say
plainly what you could not verify rather than writing round it. A plan built on an unverified
premise fails during implementation, which is the expensive place to find out — settle it
while writing, not after.

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

**Roadmap:** [`docs/roadmap.md`, phase N — omit for a single-plan change]

---
```

## Task structure

Each task names exact files, gives the exact function signatures and the test cases (inputs
and expected outputs), and states the exact command with its expected output. **It does not
give full function bodies:** the code is written once, during implementation, where the tests
and the code review check it. Writing it into the plan too makes the plan as expensive as the
build, and a reviewer can then only check it by building it. Write literal code only where the
text itself is the decision: a regex, a config or lint rule, a migration, a data format.

- **Files:** Create / Modify (with line ranges) / Test
- **Steps:** failing test → watch it fail → implement → watch it pass → commit
- **Gate:** a command whose exit status actually reflects success

A cited line range must match the replacement text supplied for it. If the Files list says
`foo.rb:12–18` but the replacement covers only `:12–15`, the implementer silently deletes
three lines.

**Every gate must be able to fail, and you must have watched it fail.** An existence check that
a one-line stub would satisfy is not a gate. Write `set -euo pipefail`; never let `| head` or
`| grep` decide an exit status. A "write the failing test" step whose test would actually pass
at that point in the sequence is the same defect in test form.

In a multi-task plan, each task changes the tree the next one runs against. Say what each task
alters about the symbols it touches, and check that a later task's expected output and cited
line numbers still hold after the earlier ones have run.

## Remember

- Exact file paths always
- Exact signatures and test cases, not "add validation"; literal code only where the text is the decision
- Exact commands with expected output
- State what your test suite does **not** prove, so nobody cites a green run as evidence
- DRY, YAGNI, TDD, frequent commits

## Execution handoff

After saving the plan, offer the execution choice:

**"Plan complete, saved to `docs/plans/<filename>.md`. Two execution options:**

**1. Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between
tasks, fast iteration

**2. Parallel Session (separate)** — new session with executing-plans, batch execution with
checkpoints

**Which approach?"**

If Subagent-Driven: use superpowers:subagent-driven-development, stay in this session, fresh
subagent per task plus code review.

If Parallel Session: guide them to open a new session in the worktree, which uses
superpowers:executing-plans.
