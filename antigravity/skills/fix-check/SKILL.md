---
name: fix-check
description: >-
  Light verification after a fix round. Takes the findings a review already
  raised and the fix diff, and reports per finding whether the code now fixes
  it, where, and which test covers it. Use after an implementer addresses
  review findings, between fix rounds, or before merging a fix. Use it INSTEAD
  of a fresh comprehensive review or a re-review; it does not hunt for new
  issues.
---

# Fix Check (Antigravity & Gemini)

A light, bounded check that runs **after** a fix round and **instead of** a second full code review.

A full review hunts across the whole diff and will invent new nitpicks on every pass. `fix-check` has a smaller job: here is the list of findings the reviewer already raised, and here is the fix diff — for each finding, did the fix actually resolve it, and what test covers it?

- **What good looks like:** One row per original CRITICAL/IMPORTANT finding with a `FIXED` / `PARTIAL` / `NOT FIXED` verdict, exact `file:line` where the fix landed, and the test name covering it (or `missing`).
- **Scope boundary:** Never hunt for new issues outside the listed findings. Only flag a new bug if the fix diff itself directly introduced a regression or syntax/contract break.

## How to Run

### 1. Inside `/start` / `peer-pipeline-swarm`

In the Peer-Pipeline Swarm, `@impl` appends the fix diff and updated test results to `/tmp/subagent-context-<project>-<stream>.md` (or `/tmp/fix-handoff-<project>-<stream>.md`) and tags **only** the reviewer(s) (`@code-rev` and/or `@sec-rev`) who requested changes (`plus @sec-rev` if auth/input/SQL/network changed) for a 1-turn `fix-check` pass.

### 2. From the Coordinator (`invoke_subagent`)

Append the original review findings and the fix diff (`git diff -U3`) to `/tmp/subagent-context-<project>-<stream>.md`, then dispatch `code-quality-reviewer` with `Model: "inherit"` (`tools: [read_file]`):

```text
invoke_subagent({
  TypeName: "code-quality-reviewer",
  Role: "Fix-Check Verifier",
  Model: "inherit",
  Prompt: "Run fix-check mode using /tmp/subagent-context-<project>-<stream>.md. Do NOT hunt for new issues outside the listed findings unless the fix diff itself introduced a regression. For each listed finding, verify via read_file whether it is FIXED, PARTIAL, or NOT FIXED, cite the exact file:line of the fix and the covering test, and return a Verdict (ALL FIXED or INCOMPLETE)."
})
```

## Required Output Format

```markdown
## Fix-Check Report

| # | Finding | Status | Fix Location | Test Coverage | Notes |
|---|---------|--------|--------------|---------------|-------|
| 1 | <short summary> | FIXED / PARTIAL / NOT FIXED | `path/file.go:42` | `TestName` (`path/file_test.go:88`) | <1 sentence> |

### Regressions Introduced by Fix Diff
- None (or exact `file:line` description if the fix broke something directly)

### Verdict
`ALL FIXED` or `INCOMPLETE (N remaining)`
```
