# Code Quality Reviewer Prompt Template

Use this template when dispatching a code quality reviewer subagent.

**Purpose:** Verify implementation is well-built (clean, tested, maintainable)

**Only dispatch after spec compliance review passes.**

```
Agent tool:
  subagent_type: "code-quality-reviewer"
  model: "opus"
  description: "Review code quality for Task N"
  prompt: |
    Review the implementation for Task N (`[task summary]`) against `/tmp/subagent-context-<project>-<stream>.md`.

    - What was implemented: [from implementer's report]
    - Plan / requirements: Task N from `[plan-file]`
    - Commit range: `[BASE_SHA]..[HEAD_SHA]`

    Verify correctness, architecture, edge cases, test depth, and maintainability strictly read-only (zero spikes, zero builds or test runs).
```

**Code reviewer returns:** Strengths, Issues (Critical/Important/Minor), Assessment
