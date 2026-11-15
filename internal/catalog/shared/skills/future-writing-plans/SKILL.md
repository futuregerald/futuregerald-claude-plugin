---
name: future-writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code. Writes a structured implementation plan with required Acceptance Criteria & Success Metrics traced to product/brief/design sources, Affected Callers, and TDD gates, and keeps current the committed roadmap (docs/roadmap.md) when the work spans more than one plan.
author: Jesse Vincent, Gerald Onyango
tags: [planning, workflow]
license: MIT, see LICENSE.txt
model: inherit
effort: medium
---

# Future Writing Plans

## Overview

**Thinking budget & Planner vs. Reviewer split:** Use **Medium thinking** by default (~90% of plans) to stay grounded in reading real code, lockfiles, and schemas rather than speculating, and pair it with a single read-only **High thinking** pass in Phase 3½ (`plan-review`). Bump the **Planner itself** to **HIGH thinking** only on the ~10% of work that designs a net-new cross-boundary subsystem (e.g., multi-context IPC bridges, E2EE key-wrapping protocols), complex distributed state machines, concurrency root-cause fixes, or zero-downtime database migrations.

**Pre-Plan Repo Gate & Contract Verification (Mandatory — prevents planner-to-implementer handoff drop-off):**
Because downstream implementer sub-agents treat `PLAN.md` as authoritative and do not re-audit repo policy scripts, before finalizing `PLAN.md` you MUST:
1. Inspect `package.json`, `Makefile`, and `scripts/` for repo-specific policy or wording checks (e.g., `scripts/check-wording.mjs`, custom lint gates) and list any banned terms or required invariants explicitly in `PLAN.md`.
2. Read the exact shared contract/type definitions (`contracts.ts`, `types.ts`, `models.go`) for every input/output type and specify both top-level and nested field shapes (e.g., `link.fields?.historyOnly`) directly in `PLAN.md`.

Write comprehensive implementation plans assuming the engineer has zero context for our
codebase and questionable taste. Document everything they need: which files to touch for each
task, the signatures, the test cases, docs they might need to check, how to verify it. Give them the
whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer who knows almost nothing about our toolset or problem
domain, and does not know good test design well.

**Announce at start:** "I'm using the future-writing-plans skill to create the implementation plan."

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

Goal · **Product & Design Sources** · **Acceptance Criteria & Success Metrics** · **Impact Analysis (Affected Callers)** · Approach · Step-by-step TDD tasks · Risks · Rollback · Out of scope.

**Every plan MUST contain an `Acceptance Criteria & Success Metrics` table** before any tasks are written. Every commitment made for the phase — from the Product/Business Goals, Point-of-View Brief (`docs/point-of-view/*.md`), PRD/Decisions (`docs/prd.md`, `docs/decisions/*.md`), and Approved UI Design (`design-demos/*/direction-approved.md`, including responsive breakpoints, header/navigation parity, interactive animations/filtering, and visual assets) — must map to a concrete `AC-N` row naming the **mounted/wired runtime entry point** (`file:Symbol`) and a measurable verification gate. A unit test on an isolated, unwired helper function never satisfies a user-facing or runtime acceptance criterion.

**Every plan MUST contain an Impact Analysis (Affected Callers)** — the call chain, up and down, of every symbol
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

## Plan document header & Acceptance Criteria

```markdown
# [Feature Name] Implementation Plan

> **For Claude / Antigravity:** REQUIRED SUB-SKILL: Use `subagent-driven-development` or `/start` to implement this plan.

**Goal:** [One sentence describing what this builds and why it matters to the user/product]

**Product & Design Sources:**
- **PRD / Decisions:** `docs/prd.md`, `docs/decisions/product-scope.md` *(or tracker ticket)*
- **Point-of-View Brief:** `docs/point-of-view/<slug>.md` *(or `~/.claude/point-of-view/<slug>.md`, or N/A if internal technical work)*
- **Approved UI Design:** `design-demos/<slug>/direction-approved.md` *(or N/A if no UI touched)*
- **Roadmap:** [`docs/roadmap.md`, phase N — omit for a single-plan change]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

**Base SHA:** [git rev-parse HEAD]

## Acceptance Criteria & Success Metrics (Required Gate)

| ID | Source (Goal / Brief / PRD / Design) | Required Runtime Behavior | Success Metric / Measurable Target | Mounted / Wired Entry Point (`file:Symbol`) | Verification (Unit Test + Runtime/Emulator Check) |
|----|--------------------------------------|---------------------------|------------------------------------|---------------------------------------------|---------------------------------------------------|
| AC-1 | Brief §1 / Design (`direction-approved.md`) | Phone (`<600dp`) and Tablet (`>=600dp`) both render the top brand header and non-wrapping navigation | 100% of breakpoints render `top-brand-header`; bottom nav stays single-row (`flex: 1` per tab, `0` wrapped rows at `360dp–412dp`) | `src/App.tsx:AppShell` | `screens.test.ts` + live browser/emulator screenshot check |
| AC-2 | Phase Goal / Extension Parity | Search filtering animates rank transitions and new-item stagger in the mounted list | Existing cards glide via FLIP delta (`computeFilterTransitionDelta` wired into `<Animated.View>`); new matches stagger at `18ms/row` | `src/components/TactileCard.tsx:TactileCard`, `src/screens/SearchScreen.tsx:SearchScreen` | `screens.test.ts` + live filter interaction |

## Affected Callers (Impact Analysis)
- Breaking callers (with consequence): ...
- Untested unbroken callers: ...

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
- Every task must trace to one or more `AC-N` rows in `## Acceptance Criteria & Success Metrics`, and every `AC-N` row must be wired into the real mounted/runtime entry point (never satisfied by exporting an unused helper just for a unit test)
- Exact signatures and test cases, not "add validation"; literal code only where the text is the decision
- Exact commands with expected output (plus live browser/emulator verification gates whenever UI, layout, or animations are touched)
- State what your test suite does **not** prove, so nobody cites a green run as evidence
- DRY, YAGNI, TDD, frequent commits

## Execution handoff (Mandatory Stop for User Approval)

After saving the plan and completing the `plan-review` gate, you **MUST** present the proposed plan (including the `Acceptance Criteria & Success Metrics` table and a summary of the tasks) to the user in chat and **STOP calling tools to wait for the user's approval** before continuing with development — **unless** the user explicitly told you to *"just work"*, *"work autonomously"*, *"keep going"*, or invoked `/goal`.

**"Plan complete, reviewed, and saved to `docs/plans/<filename>.md`. Ready to execute using `subagent-driven-development` (or `/start`) once you approve — or let me know if you want any adjustments."**

---

Forked by Gerald Onyango from `writing-plans` in Jesse Vincent's `superpowers` (MIT, see LICENSE.txt) — https://github.com/obra/superpowers
