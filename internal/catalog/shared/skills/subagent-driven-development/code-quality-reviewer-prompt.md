# Code Quality Reviewer Prompt Template

Use this template when dispatching a read-only code quality reviewer subagent.

**Purpose:** Verify the implementation is correct, well-structured, tested, maintainable, and satisfies every `AC-N` row in mounted runtime code.

**Only dispatch after spec compliance review passes and `build_subagent_context.py post-code` has updated `/tmp/subagent-context-<project>-<stream>.md`.**

```
Agent tool:
  subagent_type: "code-quality-reviewer"
  model: "opus"
  description: "Review code quality for Stream N"
  prompt: |
    Perform an ADVERSARIAL correctness and code quality review for Stream N (`[stream summary]`).

    - Context & chunked diff bundle: `/tmp/subagent-context-<project>-<stream>.md`
    - Plan path: `<plan_path>`
    - Product / Brief / UI Design sources: `<paths or N/A>`
    - Implementer 5-section handoff summary: `[concise summary]`

    ## Read-Only, Fail-Fast & Turn-Batching Rules

    1. **Strictly Read-Only:** You hold no `Bash`, `Edit`, or `Write` tools. Never mutate files, run tests/builds/spikes, or execute `git diff`.
    2. **Turn 1 Parallel Chunk Load & Fail-Fast Guard:** On Turn 1, read `/tmp/subagent-context-<project>-<stream>.md` (issuing parallel `Read` calls across **all** `StartLine-EndLine` windows in `## Read Chunk Index` when `>750` lines) plus `<plan_path>`. If the context file or diff is missing or unreadable, fail fast with `REVIEW INCOMPLETE: missing or unreadable /tmp/subagent-context-<project>-<stream>.md diff bundle` and stop immediately.
    3. **Parallel `Read` Batching:** Batch follow-up caller, sibling, and contract reads (`5–10` files per turn, `400–800` lines per read; zero 1-file-per-turn micro-slices).

    Audit every `AC-N` row against mounted runtime code (flagging dead/unwired test-only helpers as Critical), verify failure paths, check pattern parity against the newest sibling file (`file:line`), and verify edge-case test depth.
```

**Code reviewer returns:** `Acceptance Criteria & Design Target Audit`, Issues (`Critical` / `Important` / `Minor` with `file:line`), and overall Assessment (`APPROVED` or `CHANGES REQUIRED`).
