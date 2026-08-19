---
name: plan-review
description: MANDATORY before writing any code. Dispatches the adversarial-plan-reviewer agent to independently attack an implementation plan and verify its Impact Analysis against the real code. Use whenever the user says "review the plan", "review this plan", "plan review", "check the plan", "is this plan right", "adversarial review", "grill this plan", or whenever a plan has been written and implementation is about to start. This is a separate gate from code review and cannot be satisfied by reviewing the plan yourself.
tags: [plan, review, adversarial, gate, lifecycle, impact-analysis]
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Plan Review

Phase 4 of the development lifecycle. The plan is attacked by a fresh adversarial
sub-agent before any code is written.

**You never review the plan yourself.** You wrote it — you cannot objectively review
it, and a reviewer that watched it being built rubber-stamps it. Reading it over again
is not this phase.

## When this runs

- Any time a plan exists and implementation is next — mandatory, including one-line fixes
- Any time the user asks to review, check, grill, or sanity-check a plan
- Before `ExitPlanMode`
- Again after every revision — a re-review always uses a fresh agent, never the same one

## Preconditions

The plan must be a file at `docs/plans/<TICKET>-<slug>.md` and must contain an
**Impact Analysis** section — the call chain, up and down, of every symbol the change
touches. Without one the reviewer auto-rejects, so write it before dispatching.

If no plan file exists, stop and write one (`writing-plans`). Do not dispatch a review
of an idea held in conversation.

## Dispatch

One Agent call. Give it neutral inputs only.

```
Agent tool:
  subagent_type: adversarial-plan-reviewer
  description: "Adversarial plan review"
  prompt: |
    Review the implementation plan at: <absolute path to plan file>

    Goal this plan serves: <ticket key + URL, or the task description verbatim>
    Repo root: <absolute path>
    Base SHA: <git rev-parse HEAD>
```

That is the entire prompt. The agent carries its own methodology — do not restate it.

### Do not steer it

**Never include your own suspicions, uncertainties, "areas of concern", or "please
check X".** A reviewer pointed at your worries inherits your blind spots, which is the
exact failure this phase exists to prevent. The agent reports any steering it detects,
and a steered review does not count.

Do not summarize the plan for it, pre-empt its findings, or tell it which parts you
think are risky. Give it the path and the goal; it forms its own view.

## Handling the verdict

**REJECT** — any CRITICAL or IMPORTANT. Fix the plan, re-review with a fresh agent.
Never argue a finding away; never proceed on a rejected plan.

**APPROVE** — implementation may begin. MINOR findings remain mandatory fixes; approval
is not permission to skip them. The only exception is a finding that is factually
incorrect, which is explained to the author rather than silently dropped.

**Verified Recommendations** are separate: emitted only for things established
empirically with a `file:line` citation at very high confidence — never style rules or
inference. They do not gate the plan. Surface each to the author with its evidence and
recommend adopting it; the call is theirs. An empty section is a good outcome.

## Report back

Verdict, then every finding with its evidence, then the Impact Analysis audit —
especially **any caller the reviewer found that the plan missed**, which is the signal
that the plan's system-level thinking was too shallow. Then Verified Recommendations,
then what you are changing and the re-dispatch.

Never paraphrase a finding into something softer than the reviewer wrote.
