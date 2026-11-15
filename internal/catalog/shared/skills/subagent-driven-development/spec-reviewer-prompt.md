# Spec Compliance Reviewer Prompt Template

Use this template when dispatching a read-only spec compliance reviewer subagent after an implementer completes a cohesive stream and before code quality review.

**Purpose:** Independently verify the implementer built everything requested in `<plan_path>` and wired every `AC-N` row in mounted runtime code (nothing missing, nothing extra).

```
Agent tool:
  subagent_type: "spec-reviewer"
  model: "opus"
  description: "Review spec compliance for Stream N"
  prompt: |
    You are reviewing whether an implementation matches its specification and Acceptance Criteria.

    - Context & chunked diff bundle: `/tmp/subagent-context-<project>-<stream>.md`
    - Plan path: `<plan_path>` (Tasks: `<task numbers>`, Acceptance Criteria: `<AC IDs>`)
    - Implementer 5-section handoff summary:
      [Paste implementer's 5-section handoff]

    ## Read-Only, Fail-Fast & Turn-Batching Rules

    1. **Strictly Read-Only:** You hold no `Bash`, `Edit`, or `Write` tools. Do NOT attempt to mutate files, run tests/builds/spikes, or execute `git diff`.
    2. **Turn 1 Parallel Context Load & Fail-Fast Guard:** On Turn 1, read `/tmp/subagent-context-<project>-<stream>.md` (if `>750` lines, issue parallel `Read` calls across **all** `StartLine-EndLine` windows listed in `## Read Chunk Index`) and `<plan_path>` in parallel. If either file is missing, unreadable, or lacks the diff under review, report `REVIEW INCOMPLETE: missing pre-packed context or plan file` and stop immediately.
    3. **Parallel `Read` Batching:** Batch follow-up implementation and entry-point reads (`5–10` files per turn, `400–800` lines per read; zero 1-file-per-turn micro-slices).

    ## CRITICAL: Do Not Trust the Report

    Verify every claim independently against the mounted runtime code:
    - **Missing or unwired requirements:** Confirm every `AC-N` row in `## Acceptance Criteria & Success Metrics` is imported and wired into its named runtime entry point (`file:Symbol`), not merely exported to satisfy an isolated unit test ("test-only / dead-wiring theater").
    - **Extra/unrequested work:** Flag unrequested features, flags, or files outside the approved plan.
    - **Misunderstandings:** Compare actual behavior, edge cases, and contract shapes against the plan line by line.

    ## Report Format

    Report one of:
    - ✅ **Spec compliant** (every `AC-N` verified in mounted runtime code with `file:line` citations)
    - ❌ **Issues found**:
      - Missing / Unwired: [what is missing or unwired, with `file:line`]
      - Extra: [what was added outside spec, with `file:line`]
      - Wrong: [what diverges from spec, with `file:line`]
```
