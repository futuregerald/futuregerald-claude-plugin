---
name: plan-review
description: MANDATORY before writing any code. Dispatches three concurrent reviewers - adversarial-plan-reviewer, plan-blindspot-hunter and plan-consistency-checker - to independently attack an implementation plan and verify its Impact Analysis against the real code. Use whenever the user says "review the plan", "review this plan", "plan review", "check the plan", "is this plan right", "adversarial review", "grill this plan", or whenever a plan has been written and implementation is about to start. This is a separate gate from code review and cannot be satisfied by reviewing the plan yourself.
tags: [plan, review, adversarial, gate, lifecycle, impact-analysis]
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Plan Review

Phase 4 of the development lifecycle. The plan is attacked by three fresh reviewers,
running concurrently, before any code is written.

One reviewer samples rather than drains. Measured against a plan with 22 known defects,
a single reviewer recalled 11; three recalled 16 between them, with barely any overlap —
each lens finds what the others structurally cannot.

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

Three Agent calls **in one message** so they run concurrently. Wall clock is one review,
not three. Each gets the identical neutral input block; none is told what the others do.

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

```
Agent tool:
  subagent_type: plan-blindspot-hunter
  description: "Plan blind-spot hunt"
  prompt: |
    <the identical block above, byte for byte>
```

```
Agent tool:
  subagent_type: plan-consistency-checker
  description: "Plan self-consistency check"
  prompt: |
    <the identical block above, byte for byte>
```

That is the entire prompt for each. Every agent carries its own methodology — do not
restate it, do not tailor the input per agent, and do not tell any of them that the
others exist.

The three lenses are deliberately different and barely overlap:

| Agent | Finds |
|---|---|
| `adversarial-plan-reviewer` | Wrong premises, and runtime semantics the plan asserts but never proves |
| `plan-blindspot-hunter` | Callers, consumers and invisible edges the plan never names |
| `plan-consistency-checker` | Gates that cannot fail, and task N contradicting task M |

## Merge

When all three return, merge before reporting:

- Concatenate every finding. Drop exact duplicates only.
- Where two agents describe the same defect, keep the more specific failure scenario and
  record that two found it independently — that is corroboration, and it is worth keeping.
- **Never soften a finding while merging, and never drop one because another agent missed
  it.** Disagreement between reviewers is not a tie to be broken; it is the point.

### Do not steer it

**Never include your own suspicions, uncertainties, "areas of concern", or "please
check X".** A reviewer pointed at your worries inherits your blind spots, which is the
exact failure this phase exists to prevent. The agent reports any steering it detects,
and a steered review does not count.

Do not summarize the plan for it, pre-empt its findings, or tell it which parts you
think are risky. Give it the path and the goal; it forms its own view.

**The Claim Ledger is the one exception, and it is not steering.** A plan may carry a
ledger — the author's own factual assertions with `file:line` citations. Steering is
"check X, I am worried about it", which points a reviewer at the author's frame. A ledger
is a list of liabilities the author has signed for, and every row is a target to falsify.
It lives *in the plan file*, not in the dispatch prompt, so nothing about the dispatch
changes.

## Handling the verdict

**Any CRITICAL or IMPORTANT from any of the three ⇒ REJECT.** No agent's verdict overrides
another's, and a clean report from two does not offset a finding from the third. All three
clean ⇒ APPROVE.

**REJECT** — Fix the plan, re-review with three fresh agents.
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
