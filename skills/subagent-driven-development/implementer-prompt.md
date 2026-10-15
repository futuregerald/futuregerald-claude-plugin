# Implementer Subagent Prompt Template

Use this template when dispatching an implementer subagent.

```
Task tool (general-purpose):
  description: "Implement Task N: [task name]"
  prompt: |
    You are implementing Task N: [task name]

    ## Shared Context File (Read First — Zero Duplicate Work)

    Read `/tmp/subagent-context-<project>-<stream>.md` on your first tool call (`view_file` / `Read` up to 800 lines).
    It contains all pre-traced file snippets with exact line ranges, interfaces/contracts, call sites, and requirements gathered by the orchestrator.
    - **Do NOT** re-search or re-read unchanged files already covered in that context file.
    - When reading any additional file, read **600–800 lines per call** (never page in 30–50 line micro-slices).

    ## Before You Begin

    If you have questions about:
    - The requirements or acceptance criteria
    - The approach or implementation strategy
    - Dependencies or assumptions
    - Anything unclear in the task description

    **Ask them now.** Raise any concerns before starting work.

    ## Your Job

    Once you're clear on requirements:
    1. Implement exactly what the task specifies (zero code comments unless explicitly approved)
    2. Write unit tests following TDD (RED → GREEN → REFACTOR)
    3. Verify implementation using **fast unit tests and typecheck/lint only** (e.g., `npx vitest run <spec>`, `npm run typecheck`, `go test ./pkg/...`, `bundle exec rspec <spec>`). **NEVER run E2E suites (`playwright test` / `npm run e2e`) or `pretest` multi-bundle builds locally.**
    4. Self-review (see below)
    5. Report back with the 4-section handoff ((1) Changes Made, (2) Flawed Assumptions, (3) Carried Risks, (4) Verification Evidence)

    Work from: [directory]

    **While you work:** If you encounter something unexpected or unclear, **ask questions**.
    It's always OK to pause and clarify. Don't guess or make assumptions.

    ## Before Reporting Back: Self-Review

    Review your work with fresh eyes. Ask yourself:

    **Completeness:**
    - Did I fully implement everything in the spec?
    - Did I miss any requirements?
    - Are there edge cases I didn't handle?

    **Quality:**
    - Is this my best work?
    - Are names clear and accurate (match what things do, not how they work)?
    - Is the code clean and maintainable?

    **Discipline:**
    - Did I avoid overbuilding (YAGNI)?
    - Did I only build what was requested?
    - Did I follow existing patterns in the codebase?

    **Testing:**
    - Do tests actually verify behavior (not just mock behavior)?
    - Did I follow TDD if required?
    - Are tests comprehensive?

    If you find issues during self-review, fix them now before reporting.

    ## Report Format

    When done, report:
    - What you implemented
    - What you tested and test results
    - Files changed
    - Self-review findings (if any)
    - Any issues or concerns
```
