---
name: editor-pass
description: >-
  Post-build quality review that asks whether finished work is good — not just
  done, correct or free of slop — and returns a lettered crit sheet of pinned,
  specific fixes. Use as the last gate before a user-facing page, artifact,
  report, PRD, ticket set or skill goes out, or when asked "is this good?",
  "give it an editor pass", "edit this for quality", "ready to ship?". Runs
  after correctness and slop checks, in a fresh subagent.
---

# Editor Pass (Phase 12 — Antigravity & Gemini)

A post-build quality review that runs **after** correctness (`comprehensive-code-review`) and slop (`no-slop-ui`) checks, in a fresh read-only subagent, judging the finished work against the confirmed `craftsmans-wisdom` point-of-view brief (`~/.gemini/point-of-view/<slug>.md`).

- **What good looks like:** A lettered crit sheet (`A`, `B`, `C`, ...) of pinned, specific edits (`file:line` or exact section + before/after direction) that close the gap between "done" and "good", or `SHIP IT` when the work already meets the brief.

## Pre-Flight Context

Write `/tmp/subagent-context-<project>-editor.md` containing:
1. The confirmed point-of-view brief (`~/.gemini/point-of-view/<slug>.md` or Phase 0 brief from this session).
2. Exact paths to the finished files/artifacts to review (plus rendered text or screenshots if applicable).

## How to Dispatch

Never run `editor-pass` in the same context that built the artifact.

### Option A: Standard Dispatch (`invoke_subagent`)

Dispatch the harness-enforced read-only `editor-pass-reviewer` subagent (`tools: [read_file]`, `Model: "inherit"`):

```text
invoke_subagent({
  TypeName: "editor-pass-reviewer",
  Role: "Senior Editor (editor-pass)",
  Model: "inherit",
  Prompt: "Read /tmp/subagent-context-<project>-editor.md and inspect the target files using read_file only. Judge the work against the point-of-view brief across 5 dimensions: (1) Point of View & Conviction, (2) Specificity & Evidence, (3) Rhythm, Pacing & Hierarchy, (4) Craft Details & Edge States, (5) Subtraction (what to cut). Return a lettered crit sheet (A, B, C...) with exact file:line citations and concrete fixes, followed by Verdict: SHIP IT or REVISE."
})
```

### Option B: Major User-Facing Launch on an Argon UI Session

Spawn a one-shot `gemini-4-carbon-high` `swarm` worker (`name: "editor-rev"`, `workspace: "inherit"`, no `monitor`, read-only by brief reading `~/.gemini/config/plugins/antigravity-subagents/agents/editor-pass-reviewer.md` and `/tmp/subagent-context-<project>-editor.md`, posting once to `@coordinator`, then immediately killed via `swarm{action: "kill", name: "editor-rev"}`).

## Applying the Crit Sheet

1. Apply every high-leverage fix from the crit sheet directly.
2. If code or UI files changed, re-run fast unit/lint gates (and Phase 10 `Blast-radius verify`) before committing.
3. Delete `/tmp/subagent-context-<project>-editor.md`.
