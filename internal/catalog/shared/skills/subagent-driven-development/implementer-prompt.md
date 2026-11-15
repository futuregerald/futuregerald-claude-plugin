# Implementer Subagent Prompt Template

Use this template when dispatching a fresh-context TDD implementer subagent.

```
Agent tool:
  subagent_type: "implementer"
  model: "sonnet"
  description: "Implement Stream N: [stream name]"
  prompt: |
    Implement Stream N ([stream name]) from the approved plan.

    - Context bundle: `/tmp/subagent-context-<project>-<stream>.md`
    - Plan path: `<plan_path>` (Tasks: `<task numbers>`, Acceptance Criteria: `<AC IDs>`)
    - Fast gate command: `<combined fast test + typecheck/lint command>`
    - Working directory: `[directory]`

    ## Execution & Batching Rules (Mandatory)

    1. **Turn 1 Parallel Context Load:** Read `/tmp/subagent-context-<project>-<stream>.md` (whenever `>750` lines, issue parallel `Read` calls across every `StartLine-EndLine` window in `## Read Chunk Index`) and `<plan_path>` in parallel on Turn 1 before searching or editing. Do NOT re-search or re-read unchanged slices already covered in the context bundle.
    2. **Parallel `Read` Batching:** Batch additional file reads (`5–10` files per turn, `400–800` lines per read). Never inspect files in `30–50` line micro-slices or serial 1-file tail loops. Match the newest sibling file's conventions.
    3. **Distinct-File vs. Single-File `Edit` Batching:** Batch `Edit`/`Write` calls across **distinct files** in parallel (`4–8` files/turn). On any **single file**, apply `1` contiguous `Edit` block (or a full `Write` when `<300` lines) — never issue `3+` separate `Edit` calls against the same file in one turn, and never mutate source or test files with inline `node -e` or `python3 -c` regex one-liners.
    4. **Before You Begin:** If anything in the requirements, acceptance criteria, or contracts is unclear or contradicted by the codebase, pause and ask before guessing.
    5. **Strict TDD & Real Runtime Wiring (Zero Test-Only Theater):** Follow `RED -> GREEN -> REFACTOR` and satisfy every `AC-N` row in the plan's `## Acceptance Criteria & Success Metrics` table. Never export a dead helper solely to pass an isolated unit test without wiring it into the mounted component or active runtime entry point.
    6. **Zero Code Comments:** Write no code comments (`//`, `#`, `/* */`) in any source or test file unless explicitly instructed by the user (shebangs and required compiler directives like `//go:embed` excepted). Leave existing accurate comments untouched.
    7. **Single Combined Fast Gate & 2-Strike CLI Rule:** Verify using fast, scoped unit tests and lint/typecheck gates only (`cmd > /tmp/t.log 2>&1; rc=$?; tail -25 /tmp/t.log; echo "exit=$rc"`). **NEVER run unscoped full-repo suites, E2E suites (`playwright test` / `npm run e2e`), or `pretest` multi-bundle builds locally.** If a CLI or test runner fails twice consecutively, immediately read `Makefile`, `package.json`, `vitest.config.*`, or `tsconfig.json` instead of guessing flags.

    ## Self-Review Before Reporting

    Verify completeness against every `AC-N` row, YAGNI discipline, pattern parity with the newest sibling file, and real behavioral test coverage. Fix any self-review issue before returning.

    ## Mandatory 5-Section Handoff

    Return all 5 sections in your completion report:
    1. `### Changes Made` — concise summary of modified/created `file:line` paths and why.
    2. `### Acceptance Criteria & Success Metrics Met` — markdown table mapping every assigned `AC-N` from the plan to the exact `file:line` where it is wired in runtime code and the test covering it.
    3. `### Flawed Assumptions Surfaced` — any discrepancy between the plan's premises and actual code/tooling behavior (`None` if none).
    4. `### Carried Risks & Edge Cases` — residual edge cases, unverified external boundaries, or follow-ups (`None` if none).
    5. `### Verification Evidence` — exact fast-gate command(s) executed, exit codes, and concise test counts.
```
