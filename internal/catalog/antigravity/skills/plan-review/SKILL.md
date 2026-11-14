---
name: plan-review
description: >-
  Run automatically after writing any non-trivial implementation plan and before
  writing any code (skip only for trivial changes with an obvious, self-contained
  blast radius). Also use when the user asks to review, sanity-check, grill, or
  poke holes in a plan. For a multi-phase plan, re-run on the REMAINING tasks at
  each phase boundary, since tasks planned against a codebase that no longer
  exists drift and their gates still pass. Dispatches one fresh staff-engineer
  reviewer against the plan file. Runs once per pass - findings are fixed, and
  the plan is never sent back for a second review.
---

# Plan Review (Phase 3½ — Antigravity & Gemini)

One fresh-context adversarial pass over an implementation plan before any code is written, plus one pass over the remaining phases at each phase boundary of a multi-phase plan.

- **What good looks like:** Every premise in the plan is checked against the actual codebase (`file:line`), missing callers or unverified assumptions are caught before Phase 4 starts, and the Reviewer never runs shell commands, spikes, or git mutations.
- **When to skip:** Only for trivial changes with an obvious, self-contained blast radius.

## How to Dispatch

Never review a plan in the same context that wrote it. Choose the dispatch mode matching your active session (`GEMINI.md` §0.9):

### Option A: Antigravity Argon UI Session (Default for Non-Trivial Plans)

Spawn a one-shot **`gemini-4-carbon-high`** `swarm` worker (`workspace: "inherit"`, **no `monitor` or `narrator`**):

```text
swarm({
  action: "add",
  name: "plan-rev",
  model: "gemini-4-carbon-high",
  workspace: "inherit",
  brief: "Read ~/.gemini/config/plugins/antigravity-subagents/agents/plan-reviewer.md and the plan at <plan-path> (plus /tmp/subagent-context-<project>-<stream>.md if present). STRICTLY READ-ONLY: use ONLY read_file and post_message. NEVER call run_command, edit_file, write_file, or swarm, and never run spikes or mutate git/files. Verify every premise, file path, contract, and Affected Callers entry against the repo using read_file, then post your structured CRITICAL / IMPORTANT / MINOR findings and Verdict once to @coordinator."
})
```

- **End your turn immediately** after `swarm{action: "add"}`.
- On wakeup, call `read_board`, then **immediately terminate** the worker with `swarm{action: "kill", name: "plan-rev"}` (use `plan-rev-2`, `plan-rev-3` on subsequent plans in the same conversation, because killed worker names cannot be reused).

### Option B: Antigravity Carbon UI Session, Small/Routine Plan, or Gemini CLI

Dispatch the harness-enforced read-only `plan-reviewer` subagent (`tools: [read_file]`, ~31k schema floor, **zero `run_command`**):

```text
invoke_subagent({
  TypeName: "plan-reviewer",
  Role: "Adversarial Plan Reviewer",
  Model: "inherit",
  Prompt: "Review the implementation plan at <plan-path>. Verify every premise, file path, signature, Affected Callers row, and test gate against the codebase using read_file only (zero spikes, zero shell commands). Return CRITICAL, IMPORTANT, and MINOR findings with exact file:line citations and a Verdict (APPROVED or REVISE PLAN)."
})
```

*(Never pass `Model: "pro"` on Argon or Carbon sessions.)*

## After the Reviewer Returns

1. **Optional `review-lens` check:** Where configured in `~/.gemini/config/skills/antigravity-orchestration/references/config.md`, run the `review-lens` corpus check (`--concept` and `--code`) in the Coordinator's shell after the reviewer returns.
2. **Fix in place (one pass only):** Update `PLAN.md` to resolve every CRITICAL and IMPORTANT finding (and trivial MINOR findings). Explain any factually wrong finding to the user with `file:line` evidence rather than silently ignoring it.
3. **Never re-review:** Do not send the updated plan back for a second `plan-review` pass. Proceed to Phase 4 (`Implement`).
