---
name: comprehensive-code-review
description: Use when performing code review on a PR, reviewing code changes before merge, or when a code review is requested - orchestrates parallel read-only code-quality and security reviewers in Antigravity and Gemini.
---

# Comprehensive Code Review (Phase 8 — Antigravity & Gemini)

Orchestrates two fresh, read-only adversarial reviewers in parallel — **Code Quality (`code-quality-reviewer`)** and **Security (`security-reviewer`)** — plus the `review-lens` corpus check where configured, synthesizing their findings into one actionable report.

## Pre-Flight: Prepare Chunked `/tmp/subagent-context-<project>-<stream>.md`

Because reviewers are strictly read-only (`tools: [read_file]`) and never run `git diff` or shell commands, build `/tmp/subagent-context-<project>-<stream>.md` via `python3 ~/.gemini/config/skills/subagent-driven-development/scripts/build_subagent_context.py post-code --output /tmp/subagent-context-<project>-<stream>.md --base <base_ref> [--head <head_ref>] [--share-root <dir>] [--gate-log <path>]` (plus task metadata) before dispatching:
- **Product & Business Goals:** What user/product outcome this phase delivers (and links to `docs/prd.md`, `docs/decisions/product-scope.md`, `docs/roadmap.md`)
- **Point-of-View Brief & Approved UI Design:** Exact paths and non-negotiable commitments from `docs/point-of-view/*.md` and `design-demos/*/direction-approved.md` (including responsive breakpoints for phone `<600dp` vs. tablet/desktop, top header & bottom navigation bar layout rules, tactile animations/filtering, and icon/asset fidelity)
- **Acceptance Criteria & Success Metrics Table:** Copied verbatim from `PLAN.md` (`AC-1`, `AC-2`, ...) with the required runtime behavior, measurable target, and mounted entry point (`file:Symbol`)
- `Affected Callers` from `PLAN.md`
- Passing fast test and lint exit codes (plus emulator/browser visual verification status when UI is touched)
- Chunked unified diff (`<=750` lines per section with an exact two-pass `## Read Chunk Index` (`StartLine-EndLine`, bounded by both lines and `<46,080` bytes via `--max-chunk-bytes 38000`) so reviewers read all chunks in **1 parallel Turn-1 `read_file` batch**)
- If reviewing a `workspace: "share"` or isolated worktree branch, include `SHARE_ROOT=<abs-path>` so reviewers inspect `$SHARE_ROOT/<file>` when reading ±40 lines around changed callers.

## Dispatching the Parallel Reviewers

Choose the dispatch mode matching your session (`GEMINI.md` §0.9). **`gemini-4-carbon-high` is the SOLE default adversarial reviewer — NEVER spawn `claude-opus-5-5-high` (`claude-opus-5-5-*`) unless the user explicitly asks for an Opus review in chat:**

### 1. Inside `/start` / `/start-small` / `peer-pipeline-swarm`

`code-rev` (`gemini-4-carbon-high`, `29.3k` floor) and `sec-rev` (`gemini-4-carbon-high`, `29.3k` floor) are already spawned in the pipeline (`~117k` 4-worker boot floor) and review `@impl`'s committed `share` branch directly against the Acceptance Criteria, Brief, and Approved UI Design. See [`peer-pipeline-swarm`](../peer-pipeline-swarm/SKILL.md) (when installed).

### 2. Standalone Session Where UI Is Not `Carbon High` (Default `Carbon Medium` or `Argon` — Pre-PR Review)

Stage or checkpoint uncommitted edits first, then spawn two one-shot `gemini-4-carbon-high` `swarm` workers (`29.3k` Turn-1 floor each) in a **single turn** (no `monitor` or `narrator`) so reviewers run at `Carbon High` rather than downgrading to `Medium` via `Model: "inherit"`:

```text
swarm({
  action: "add",
  name: "code-rev",
  model: "gemini-4-carbon-high",
  workspace: "inherit",
  brief: "On Turn 1, read ~/.gemini/config/plugins/antigravity-subagents/agents/code-quality-reviewer.md and all indexed chunks of /tmp/subagent-context-<project>-<stream>.md (`1-700`, `701-1400`, ... when >750 lines) plus the referenced Point-of-View Brief, PRD/Decisions, and Approved UI Design files in 1 parallel read_file batch. On follow-up verification turns, batch 5-10 read_file calls per turn in 400-800 line chunks (+-40 lines around narrow call sites; zero serial 1-file tail reads). STRICTLY READ-ONLY: use ONLY read_file and post_message. NEVER call run_command, edit_file, write_file, or swarm, and never mutate git or files. Verify every row of the Acceptance Criteria & Success Metrics table against the actual mounted/wired runtime code (flagging unwired test-only helpers, missing phone/tablet header or nav parity, missing animations, or truncated SVG/assets as CRITICAL), then audit logic, callers, and conventions. Post your Acceptance Criteria Audit table, CRITICAL / IMPORTANT / MINOR findings, and Verdict once to @coordinator."
})
swarm({
  action: "add",
  name: "sec-rev",
  model: "gemini-4-carbon-high",
  workspace: "inherit",
  brief: "On Turn 1, read ~/.gemini/config/plugins/antigravity-subagents/agents/security-reviewer.md and all indexed chunks of /tmp/subagent-context-<project>-<stream>.md (`1-700`, `701-1400`, ... when >750 lines) in 1 parallel read_file batch. On follow-up verification turns, batch 5-10 read_file calls per turn in 400-800 line chunks (+-40 lines around narrow call sites; zero serial 1-file tail reads). STRICTLY READ-ONLY: use ONLY read_file and post_message. NEVER call run_command, edit_file, write_file, or swarm, and never mutate git or files. Audit the diff for OWASP Top 10, auth/authz, injection, SSRF, path/symlink traversal, secrets, and race conditions. Post your CRITICAL / IMPORTANT / MINOR findings and Verdict once to @coordinator."
})
```

End your turn immediately; once both have posted (`swarm_wait{since: N}` if needed), read the board and **immediately `swarm{action: "kill"}` both workers** (suffix `-2`, `-3` on later streams).

### 3. `Carbon High` UI Session, Small Change, or Gemini CLI

Dispatch both harness-enforced read-only subagents (`tools: [read_file]`, `22.7k` Turn-1 floor each) in parallel in one turn:

```text
invoke_subagent({
  TypeName: "code-quality-reviewer",
  Role: "Staff Code Quality Reviewer",
  Model: "inherit",
  Prompt: "On Turn 1, read all indexed chunks of /tmp/subagent-context-<project>-<stream>.md (`1-700`, `701-1400`, ... when >750 lines) and the referenced Point-of-View Brief, PRD/Decisions, and Approved UI Design files in 1 parallel read_file batch; on follow-up turns, batch 5-10 read_file calls per turn in 400-800 line chunks (+-40 lines around narrow call sites; zero 1-file tail reads). First audit every Acceptance Criterion & Success Metric against the actual mounted/wired runtime code (flagging unwired test-only helpers, missing responsive header/nav parity, missing animations, or broken assets as CRITICAL), then audit logic/edge-case bugs, caller contract breaks, error handling, test quality, and maintainability using read_file only. Return an Acceptance Criteria Audit table, CRITICAL / IMPORTANT / MINOR findings with file:line citations, and a Verdict."
})
invoke_subagent({
  TypeName: "security-reviewer",
  Role: "Principal Security Reviewer",
  Model: "inherit",
  Prompt: "On Turn 1, read all indexed chunks of /tmp/subagent-context-<project>-<stream>.md (`1-700`, `701-1400`, ... when >750 lines) in 1 parallel read_file batch; on follow-up turns, batch 5-10 read_file calls per turn in 400-800 line chunks (+-40 lines around narrow call sites; zero 1-file tail reads). Audit the embedded git diff for injection, auth/authz, SSRF, path/symlink traversal, secrets, and unsafe deserialization using read_file only. Return CRITICAL, IMPORTANT, and MINOR findings with file:line citations and a Verdict."
})
```

*(If SQL queries or migrations changed, also dispatch `TypeName: "sql-reviewer"`, `Model: "inherit"` in the same turn for Phase 9.)*

## Synthesis & Fix-Check Gate

1. **Run `review-lens` (where configured):** On repos listed in `~/.gemini/config/skills/antigravity-orchestration/references/config.md`, run `--code` and `--concept` queries against the diff.
2. **Fix findings:** Fix every CRITICAL and IMPORTANT finding (and trivial MINOR findings). Explain any factually wrong finding to the user with `file:line` evidence.
3. **Verify fixes via `fix-check` (never a second full review):** After fixing CRITICAL/IMPORTANT findings, run [`fix-check`](../fix-check/SKILL.md) (when installed; `invoke_subagent` with `TypeName: "code-quality-reviewer"`, `Model: "inherit"`) to verify that each finding is resolved and covered by a test.
4. **Clean up:** Delete `/tmp/subagent-context-<project>-<stream>.md` once approved, and re-run Phase 10 (`Blast-radius verify`) if any code changed during review fixes.
