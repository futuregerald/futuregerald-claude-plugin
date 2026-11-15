---
name: comprehensive-code-review
description: Use when performing code review on a PR, reviewing code changes before merge, or when a code review is requested - orchestrates parallel read-only code-quality and security reviewers in Antigravity and Gemini.
---

# Comprehensive Code Review (Phase 8 — Antigravity & Gemini)

Orchestrates two fresh, read-only adversarial reviewers in parallel — **Code Quality (`code-quality-reviewer`)** and **Security (`security-reviewer`)** — plus the `review-lens` corpus check where configured, synthesizing their findings into one actionable report.

## Pre-Flight: Prepare `/tmp/subagent-context-<project>-<stream>.md`

Because reviewers are strictly read-only (`tools: [read_file]`) and never run `git diff` or shell commands, write `/tmp/subagent-context-<project>-<stream>.md` before dispatching:
- **Product & Business Goals:** What user/product outcome this phase delivers (and links to `docs/prd.md`, `docs/decisions/product-scope.md`, `docs/roadmap.md`)
- **Point-of-View Brief & Approved UI Design:** Exact paths and non-negotiable commitments from `docs/point-of-view/*.md` and `design-demos/*/direction-approved.md` (including responsive breakpoints for phone `<600dp` vs. tablet/desktop, top header & bottom navigation bar layout rules, tactile animations/filtering, and icon/asset fidelity)
- **Acceptance Criteria & Success Metrics Table:** Copied verbatim from `PLAN.md` (`AC-1`, `AC-2`, ...) with the required runtime behavior, measurable target, and mounted entry point (`file:Symbol`)
- `Affected Callers` from `PLAN.md`
- Passing fast test and lint exit codes (plus emulator/browser visual verification status when UI is touched)
- Full unified diff (`git diff -U3` or `git diff <base>...HEAD -U3`) and `file:startLine-endLine` hit ranges
- If reviewing a `workspace: "share"` or isolated worktree branch, include `SHARE_ROOT=<abs-path>` so reviewers inspect `$SHARE_ROOT/<file>` when reading ±40 lines around changed callers.

## Dispatching the Parallel Reviewers

Choose the dispatch mode matching your session (`GEMINI.md` §0.9):

### 1. Inside `/start` / `peer-pipeline-swarm`

`code-rev` (`claude-opus-5-5-high`) and `sec-rev` (`gemini-4-carbon-high`) are already spawned in the pipeline and review `@impl`'s committed `share` branch directly against the Acceptance Criteria, Brief, and Approved UI Design. See [`peer-pipeline-swarm`](../peer-pipeline-swarm/SKILL.md) (when installed).

### 2. Standalone Session Where UI Is Not `Carbon High` (Default `Carbon Medium` or `Argon` — Pre-PR Review)

Stage or checkpoint uncommitted edits first, then spawn two one-shot `gemini-4-carbon-high` `swarm` workers in a **single turn** (no `monitor` or `narrator`) so reviewers run at `Carbon High` rather than downgrading to `Medium` via `Model: "inherit"`:

```text
swarm({
  action: "add",
  name: "code-rev",
  model: "gemini-4-carbon-high",
  workspace: "inherit",
  brief: "Read ~/.gemini/config/plugins/antigravity-subagents/agents/code-quality-reviewer.md and /tmp/subagent-context-<project>-<stream>.md (plus the referenced Point-of-View Brief, PRD/Decisions, and Approved UI Design files). STRICTLY READ-ONLY: use ONLY read_file and post_message. NEVER call run_command, edit_file, write_file, or swarm, and never mutate git or files. Verify every row of the Acceptance Criteria & Success Metrics table against the actual mounted/wired runtime code (flagging unwired test-only helpers, missing phone/tablet header or nav parity, missing animations, or truncated SVG/assets as CRITICAL), then audit logic, callers, and conventions. Post your Acceptance Criteria Audit table, CRITICAL / IMPORTANT / MINOR findings, and Verdict once to @coordinator."
})
swarm({
  action: "add",
  name: "sec-rev",
  model: "gemini-4-carbon-high",
  workspace: "inherit",
  brief: "Read ~/.gemini/config/plugins/antigravity-subagents/agents/security-reviewer.md and /tmp/subagent-context-<project>-<stream>.md. STRICTLY READ-ONLY: use ONLY read_file and post_message. NEVER call run_command, edit_file, write_file, or swarm, and never mutate git or files. Audit the diff for OWASP Top 10, auth/authz, injection, SSRF, path/symlink traversal, secrets, and race conditions. Post your CRITICAL / IMPORTANT / MINOR findings and Verdict once to @coordinator."
})
```

End your turn immediately; once both have posted (`swarm_wait{since: N}` if needed), read the board and **immediately `swarm{action: "kill"}` both workers** (suffix `-2`, `-3` on later streams).

### 3. `Carbon High` UI Session, Small Change, or Gemini CLI

Dispatch both harness-enforced read-only subagents (`tools: [read_file]`) in parallel in one turn:

```text
invoke_subagent({
  TypeName: "code-quality-reviewer",
  Role: "Staff Code Quality Reviewer",
  Model: "inherit",
  Prompt: "Read /tmp/subagent-context-<project>-<stream>.md (and the referenced Point-of-View Brief, PRD/Decisions, and Approved UI Design files). First audit every Acceptance Criterion & Success Metric against the actual mounted/wired runtime code (flagging unwired test-only helpers, missing responsive header/nav parity, missing animations, or broken assets as CRITICAL), then audit logic/edge-case bugs, caller contract breaks, error handling, test quality, and maintainability using read_file only. Return an Acceptance Criteria Audit table, CRITICAL / IMPORTANT / MINOR findings with file:line citations, and a Verdict."
})
invoke_subagent({
  TypeName: "security-reviewer",
  Role: "Principal Security Reviewer",
  Model: "inherit",
  Prompt: "Read /tmp/subagent-context-<project>-<stream>.md and audit the embedded git diff for injection, auth/authz, SSRF, path/symlink traversal, secrets, and unsafe deserialization using read_file only. Return CRITICAL, IMPORTANT, and MINOR findings with file:line citations and a Verdict."
})
```

*(If SQL queries or migrations changed, also dispatch `TypeName: "sql-reviewer"`, `Model: "inherit"` in the same turn for Phase 9.)*

## Synthesis & Fix-Check Gate

1. **Run `review-lens` (where configured):** On repos listed in `~/.gemini/config/skills/antigravity-orchestration/references/config.md`, run `--code` and `--concept` queries against the diff.
2. **Fix findings:** Fix every CRITICAL and IMPORTANT finding (and trivial MINOR findings). Explain any factually wrong finding to the user with `file:line` evidence.
3. **Verify fixes via `fix-check` (never a second full review):** After fixing CRITICAL/IMPORTANT findings, run [`fix-check`](../fix-check/SKILL.md) (when installed; `invoke_subagent` with `TypeName: "code-quality-reviewer"`, `Model: "inherit"`) to verify that each finding is resolved and covered by a test.
4. **Clean up:** Delete `/tmp/subagent-context-<project>-<stream>.md` once approved, and re-run Phase 10 (`Blast-radius verify`) if any code changed during review fixes.
