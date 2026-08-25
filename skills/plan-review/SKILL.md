---
name: plan-review
description: Use when the user asks to review a plan, sanity-check a plan, grill a plan, poke holes in a plan, or do an adversarial review of a plan. Dispatches one fresh staff-engineer reviewer against the plan file and reports what it found. Advisory, not a gate.
tags: [plan, review, adversarial, staff-engineer]
model: opus
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Plan Review

Dispatch one fresh reviewer to attack an implementation plan the way an experienced staff
engineer would, then report back what it found.

**Advisory, not a gate.** It does not approve or reject. It does not re-dispatch. A finding
does not block implementation. The author reads the findings and decides.

## When to use

- The user asks to review, sanity-check, grill, or poke holes in a plan
- A plan is written and a second pair of eyes is genuinely worth it before implementing

**On demand only.** Do not run it automatically on every plan, and do not treat it as a
phase that must pass before code gets written.

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

## Reporting back

Relay every finding with its evidence, at the severity the reviewer assigned. **Never soften
a finding while relaying it, and never quietly drop one.**

Then say what you are doing about each — fixing it, or leaving it and why. A finding you
believe is factually wrong gets explained to the author, not silently skipped.

Reading the plan over yourself is not a substitute for this. You wrote it; a self-review is
exactly what the fresh reviewer replaces.
