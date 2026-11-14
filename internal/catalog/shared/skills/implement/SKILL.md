---
name: implement
description: "Implement a piece of work based on a spec or set of tickets."
---

Implement the work described by the user in the spec or tickets.

Use /tdd where possible, at pre-agreed seams.

Run typechecking and scoped fast unit tests regularly (never run unscoped full-repo test suites or E2E as per-turn gates; before opening or merging a PR, check `.github/workflows/` and run any suite gated off on `pull_request`).

Once done, use `comprehensive-code-review` (or `/code-review`) to review the work.

Commit your work to the current branch.
