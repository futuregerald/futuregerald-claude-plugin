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

- **What good looks like:** Every premise in the plan is checked against the actual codebase (`file:line`) AND against the Product/Business Goals, Point-of-View Brief, PRD/Decisions, and Approved UI Design (`direction-approved.md`); missing or unverified Acceptance Criteria & Success Metrics, unwired helper-only tests, missing callers, or unverified assumptions are caught before Phase 4 starts; and the Reviewer never runs shell commands, spikes, or git mutations.
- **When to skip:** Only for trivial changes with an obvious, self-contained blast radius.

## How to Dispatch

Never review a plan in the same context that wrote it. Choose the dispatch mode matching your active session (`GEMINI.md` §0.9):

### Option A: Antigravity Default (`Carbon Medium` or `Argon` UI — Non-Trivial Plans)

Whenever the UI is not already set to **`Carbon High`**, spawn a one-shot **`gemini-4-carbon-high`** `swarm` worker (`29.3k` Turn-1 floor, `workspace: "inherit"`, **no `monitor` or `narrator`**) so the plan gets full `Carbon High` deliberation rather than downgrading to `Medium` via `Model: "inherit"`:

```text
swarm({
  action: "add",
  name: "plan-rev",
  model: "gemini-4-carbon-high",
  workspace: "inherit",
  brief: "On Turn 1, read ~/.gemini/config/plugins/antigravity-subagents/agents/plan-reviewer.md, the plan at <plan-path>, referenced brief/PRD/design-demos files, and all indexed chunks of /tmp/subagent-context-<project>-<stream>.md (if present) in 1 parallel read_file batch; on follow-up turns, batch 5-10 read_file calls per turn in 400-800 line chunks (+-40 lines around narrow call sites; zero 1-file tail reads). STRICTLY READ-ONLY: use ONLY read_file and post_message. NEVER call run_command, edit_file, write_file, or swarm, and never run spikes or mutate git/files. Verify that every Product/Brief/Design goal and phase commitment is captured in the plan's Acceptance Criteria & Success Metrics table with a wired runtime entry point, and verify every premise, file path, contract, and Affected Callers entry against the repo using read_file. Post your structured CRITICAL / IMPORTANT / MINOR findings and Verdict once to @coordinator."
})
```

- **End your turn immediately** after `swarm{action: "add"}`.
- On wakeup, call `read_board`, then **immediately terminate** the worker with `swarm{action: "kill", name: "plan-rev"}` (use `plan-rev-2`, `plan-rev-3` on subsequent plans in the same conversation, because killed worker names cannot be reused).

### Option B: Antigravity `Carbon High` UI Session, Small/Routine Plan, or Gemini CLI

Dispatch the harness-enforced read-only `plan-reviewer` subagent (`tools: [read_file]`, `22.7k` Turn-1 floor, **zero `run_command`**):

```text
invoke_subagent({
  TypeName: "plan-reviewer",
  Role: "Adversarial Plan Reviewer",
  Model: "inherit",
  Prompt: "On Turn 1, read the implementation plan at <plan-path>, its referenced Point-of-View Brief, PRD/Decisions, and Approved UI Design files, and any /tmp/subagent-context-<project>-<stream>.md chunks in 1 parallel read_file batch; on follow-up turns, batch 5-10 read_file calls per turn in 400-800 line chunks (+-40 lines around narrow call sites; zero 1-file tail reads). Verify that every phase goal and design commitment has a measurable row in Acceptance Criteria & Success Metrics wired to a mounted runtime entry point, and check every premise, file path, signature, Affected Callers row, and test gate against the codebase using read_file only (zero spikes, zero shell commands). Return CRITICAL, IMPORTANT, and MINOR findings with exact file:line citations and a Verdict (APPROVED or REVISE PLAN)."
})
```

*(Never pass `Model: "pro"` on Argon or Carbon sessions.)*

## After the Reviewer Returns

1. **Optional `review-lens` check:** Where configured in `~/.gemini/config/skills/antigravity-orchestration/references/config.md`, run the `review-lens` corpus check (`--concept` and `--code`) in the Coordinator's shell after the reviewer returns.
2. **Fix in place (one pass only):** Update `PLAN.md` to resolve every CRITICAL and IMPORTANT finding (and trivial MINOR findings). Explain any factually wrong finding to the user with `file:line` evidence rather than silently ignoring it.
3. **Never re-review:** Do not send the updated plan back for a second `plan-review` pass. Proceed to Phase 4 (`Implement`).
