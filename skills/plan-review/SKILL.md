---
name: plan-review
description: Use when the user asks to review a plan, sanity-check a plan, grill a plan, poke holes in a plan, or do an adversarial review of a plan. Dispatches one fresh staff-engineer reviewer against the plan file. Runs once before implementation - findings are fixed, and the plan is never sent back for a second review.
tags: [plan, review, adversarial, staff-engineer]
model: opus
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Plan Review

Dispatch one fresh reviewer to attack an implementation plan the way an experienced staff
engineer would, then fix what it finds before writing code.

**This is a gate, and it runs exactly once.** Blocking and Worth-fixing findings are fixed
before implementation starts. The plan is *not* sent back for a second review — you verify
your own fixes against the plan diff and proceed. One pass, then implement.

That single pass is the whole design. A gate that re-reviews until it is satisfied is
unbounded: the previous version of this gate ran to a median of ~5 rounds and a maximum of 9,
which cost more than the defects it caught were worth.

## When to use

- The user asks to review, sanity-check, grill, or poke holes in a plan
- A plan is written and it is worth a second pair of eyes before implementing

Judgment call, not an automatic step on every plan. A one-line fix with an obvious blast
radius does not need it; a change touching a contract other code depends on does.

## Preconditions

A plan file must exist. If the plan is still an idea in the conversation, write it down
first (`writing-plans`) — there is nothing to review otherwise.

## Dispatch

One Agent call:

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

**Do not steer it.** No suspicions, no "areas of concern", no "please check X". A reviewer
pointed at your worries inherits your blind spots, which defeats the point of dispatching a
fresh one. Do not summarise the plan for it either; let it form its own view. The agent
reports steering it detects.

If the reviewer will read a working tree shared with other sessions, add one line telling it
to investigate read-only and not to mutate git state. That is a safety constraint about the
checkout, not steering about the plan.

## Handling the findings

Relay every finding with its evidence, at the severity the reviewer assigned. **Never soften
a finding while relaying it, and never quietly drop one.**

- **Blocking** and **Worth fixing** — fix the plan before implementing.
- **Minor** — fix when trivial; otherwise surface it with the reviewer's evidence and let
  the author decide.
- A finding you believe is factually wrong gets explained to the author, with your evidence.
  Never argue one away silently.

Then **verify each fix against the plan diff yourself** and report the evidence per finding.
That verification is what replaces a second review — do not re-dispatch, and do not ask for
another round. Once the fixes are verified, implementation starts.

Reading the plan over yourself is not a substitute for the review itself. You wrote it; a
self-review is exactly what the fresh reviewer replaces.
