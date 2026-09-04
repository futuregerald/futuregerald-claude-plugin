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

**Every gate must be able to fail, and you must have watched it fail.** An existence check that
a one-line stub would satisfy is not a gate. Write `set -euo pipefail`; never let `| head` or
`| grep` decide an exit status. A "write the failing test" step whose test would actually pass
at that point in the sequence is the same defect in test form.

In a multi-task plan, each task changes the tree the next one runs against. Say what each task
alters about the symbols it touches, and check that a later task's expected output and cited
line numbers still hold after the earlier ones have run.

## Remember

- Exact file paths always
- Complete code in the plan, not "add validation"
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
