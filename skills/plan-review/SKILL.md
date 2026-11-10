---
name: plan-review
description: Run automatically after writing any non-trivial implementation plan and before writing any code (skip only for trivial changes with an obvious, self-contained blast radius). Also use when the user asks to review, sanity-check, grill, or poke holes in a plan. For a multi-phase plan, re-run on the REMAINING tasks at each phase boundary, since tasks planned against a codebase that no longer exists drift and their gates still pass. Dispatches one fresh staff-engineer reviewer against the plan file. Runs once per pass - findings are fixed, and the plan is never sent back for a second review.
tags: [plan, review, adversarial, staff-engineer]
model: inherit
effort: high
---

# Plan Review

**Run this at the strongest reasoning model available to you.** `model: inherit` takes the session's model, so this is a request rather than a guarantee — if the session is on a small or fast model, escalate before reviewing. A downgraded adversarial review is the failure mode that matters here: it returns generic approval and nothing signals that it was shallow.

Dispatch one fresh reviewer to attack an implementation plan the way an experienced staff
engineer would, then fix what it finds before writing code.

**This is a gate, and it runs exactly once.** Blocking and Worth-fixing findings are fixed
before implementation starts. The plan is *not* sent back for a second review — you verify
your own fixes against the plan diff and proceed. One pass, then implement.

That single pass is the whole design. A gate that re-reviews until it is satisfied is
unbounded: the previous version of this gate ran to a median of ~5 rounds and a maximum of 9,
which cost more than the defects it caught were worth.

## When to use

- After writing any non-trivial implementation plan (`writing-plans`) and before writing any code — invoke it yourself without waiting to be asked
- When the user asks to review, sanity-check, grill, or poke holes in a plan
- At each phase boundary of a multi-phase plan, on the remaining tasks

**When to skip:** only for a trivial change with an obvious, self-contained blast radius (e.g., a one-line fix or config tweak that does not alter any contract other callers depend on). Every change still gets a written plan (`writing-plans`); only the `plan-review` sub-agent dispatch is skipped for trivial changes.

## Preconditions

A plan file must exist. If the plan is still an idea in the conversation, write it down
first (`writing-plans`) — there is nothing to review otherwise.

## Dispatch

One sub-agent call, routed per the active harness's root configuration (`~/.gemini/GEMINI.md` in Antigravity, `~/.claude/CLAUDE.md` / repo `CLAUDE.md` in Claude Code):

**Claude Code:**
```
Agent tool:
  subagent_type: adversarial-plan-reviewer
  model: opus
  prompt: |
    Review the implementation plan at: <absolute path to plan file>

    Goal this plan serves: <ticket key + URL, or the task description verbatim>
    Repo root: <absolute path>
    Base SHA: <git rev-parse HEAD>
```

**Antigravity (use `TypeName: "plan-reviewer"` for the restricted ~31k read-only grant, and `Model: "inherit"` on Carbon/Argon sessions so it stays on Carbon/Argon rather than downgrading to Gemini 3.1 Pro `"pro"`):**
```
invoke_subagent:
  TypeName: "plan-reviewer"
  Role: "Adversarial Plan Reviewer"
  Model: "inherit"
  Prompt: |
    Review the implementation plan at: <absolute path to plan file>
    Goal this plan serves: <ticket key + URL, or the task description verbatim>
    Repo root: <absolute path>
    Base SHA: <git rev-parse HEAD>
```

That is the entire prompt. The agent carries its own methodology — do not restate it. That
includes its cost rule: it reviews strictly by reading the plan and cited files (**zero spikes, zero builds or test runs**), so a plan that
contains full function bodies is already too expensive to review (see `writing-plans`).

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
