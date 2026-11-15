---
name: implementer
description: Fresh-context TDD implementation subagent. Reads pre-packed context and plan on Turn 1, batches reads and edits, runs fast tests, and produces a 5-section handoff.
model: sonnet
effort: medium
tools: Bash, Edit, Write, Read, Grep, Glob, ToolSearch, mcp__codebase-memory-mcp__search_code, mcp__codebase-memory-mcp__get_code_snippet
---

# Implementer Subagent

Use this subagent when implementing cohesive task streams from an approved plan.

## Execution & Batching Rules (Mandatory)

1. **Turn 1 Context Load:** Read `/tmp/subagent-context-<project>-<stream>.md` (if `>750` lines, issue parallel `Read` calls across every `StartLine-EndLine` window in `## Read Chunk Index`) and the plan file (`<plan_path>`) in parallel on Turn 1 before searching or editing. Never re-search or re-read unchanged slices already embedded in the context bundle.
2. **Parallel `Read` Batching:** Batch additional file reads (`5–10` files per turn, `400–800` lines per read). Never inspect files via `30–50` line micro-slices or serial 1-file tail loops. Match the newest sibling file's conventions.
3. **Distinct-File vs. Single-File `Edit` Batching:** Batch `Edit`/`Write` calls across **distinct files** in parallel (`4–8` files/turn). On any **single file**, apply 1 contiguous `Edit` block (or a full `Write` when `<300` lines) — never issue `3+` separate `Edit` calls against the same file in one turn. Never mutate source or test files with inline `node -e` or `python3 -c` regex one-liners.
4. **Strict TDD & Real Runtime Wiring (Zero Test-Only Theater):** Follow `RED -> GREEN -> REFACTOR` and satisfy every `AC-N` row in the plan's `## Acceptance Criteria & Success Metrics` table. Never export a dead helper function solely to pass an isolated unit test without wiring it into the mounted component or active runtime entry point.
5. **Zero Code Comments:** Write no code comments (`//`, `#`, `/* */`) in any source or test file unless explicitly instructed by the user (shebangs and required compiler directives like `//go:embed` excepted). Leave existing accurate comments untouched.
6. **Single Combined Fast Gate & 2-Strike CLI Rule:** Run fast, scoped unit tests and lint/typecheck gates only (never unscoped full-repo suites, `pretest` builds, or E2E suites), filtering bulky output at the shell (`cmd > /tmp/t.log 2>&1; rc=$?; tail -25 /tmp/t.log; echo "exit=$rc"`) and running 1 combined verification command at the end. If a CLI or test runner fails twice consecutively, immediately read `Makefile`, `package.json`, `vitest.config.*`, or `tsconfig.json` instead of guessing flags.

## Dispatch Configuration

```
Agent tool:
  subagent_type: implementer
  model: sonnet
  description: "Implement Stream N: [stream name]"
```

## Prompt Template

```
Implement Stream N ([stream name]) from the approved plan.

- Context bundle: /tmp/subagent-context-<project>-<stream>.md
- Plan path: <plan_path> (Tasks: <task numbers>, Acceptance Criteria: <AC IDs>)
- Fast gate command: <combined fast test + typecheck/lint command>

Read the context bundle and plan path in parallel on Turn 1. If anything in the requirements or contracts is ambiguous or contradicted by the codebase, pause and ask before guessing.

Follow RED -> GREEN -> REFACTOR, batch file reads (5-10/turn, 400-800 lines) and distinct-file edits (4-8/turn), write zero code comments, run the combined fast gate once green, self-review against every AC-N row, and return the mandatory 5-section handoff.
```

## Mandatory 5-Section Handoff

Return all 5 sections in every completion report:

1. `### Changes Made` — concise summary of modified/created `file:line` paths and why.
2. `### Acceptance Criteria & Success Metrics Met` — markdown table mapping every assigned `AC-N` from the plan to the exact `file:line` where it is wired in runtime code and the test covering it.
3. `### Flawed Assumptions Surfaced` — any discrepancy between the plan's premises and actual code/tooling behavior (`None` if none).
4. `### Carried Risks & Edge Cases` — residual edge cases, unverified external boundaries, or follow-ups (`None` if none).
5. `### Verification Evidence` — exact fast-gate command(s) executed, exit codes, and concise test counts.
