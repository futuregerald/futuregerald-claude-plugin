---
name: pull-request-description
description: Use when creating a pull request, writing a PR description, or preparing changes for code review. Ensures PR descriptions include structured summary, background context, test plan, and rollback plan.
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Pull Request Description

## Overview

Write PR descriptions that give reviewers everything they need to understand **what** changed, **why** it changed, **how** it was tested, and **how to roll back** if something goes wrong. A good PR description is a decision document, not a changelog.

## Template

```markdown
## Summary

<1-3 bullet points: what changed and why, at a high level>

Refs [JIRA-KEY](https://your-site.atlassian.net/browse/JIRA-KEY)
<!-- or: Closes JIRA-KEY / Fixes JIRA-KEY -->

## Background

### Context
<What does the reader need to know to understand this change? Domain concepts, how the system works today, relevant data model relationships. Assume the reviewer is smart but unfamiliar with this specific area.>

### The problem
<What is broken, missing, or suboptimal? Be specific - include symptoms, affected users/flows, and severity. Reference logs, errors, or screenshots if available.>

### The fix
<What approach did you take and WHY this approach over alternatives? If there were trade-offs, state them. If you considered and rejected other approaches, briefly explain why.>

### Pre-existing fixes (if any)
<Document any opportunistic fixes to existing code found during this work - e.g., N+1 queries, dead code, minor bugs in adjacent code. Explain why the fix is safe.>

## Test plan

- [ ] <Automated tests: what was added/modified, count of new specs>
- [ ] <Key scenarios covered: happy path, edge cases, error cases>
- [ ] <Manual verification steps if applicable>

## Rollback plan

<How to undo this change if it causes problems in production.>

<!-- For most changes: -->
Revert this PR.

<!-- For changes with side effects, be specific: -->
<!-- Examples of when a simple revert is NOT sufficient: -->
<!-- - DB migrations: "Revert PR, then run `rails db:rollback STEP=1` to undo the migration" -->
<!-- - Data backfills: "Revert PR. Run `rake data:undo_backfill_xyz` to restore previous values" -->
<!-- - External integrations: "Revert PR. Disable webhook in [service] dashboard" -->
<!-- - Feature flags: "Disable flag `xyz` in LaunchDarkly before reverting" -->
<!-- - Cache/queue changes: "Flush cache key `xyz` after reverting" -->
```

## Rules

1. **Summary is not a diff.** Summarize the _intent_, not the file list. "Fixes carried-over finding review status not restoring on re-validation" beats "Adds RestoreCarriedOver interactor and specs."

2. **Background teaches.** A reviewer who has never seen this code should understand the domain after reading Background. Define terms. Explain relationships. Use short code snippets or diagrams if they clarify.

3. **Explain the "why" of your approach.** Don't just describe what you did - explain why you chose this solution. If you rejected alternatives, say so briefly.

4. **Link the issue.** Always include a reference to the Jira ticket (or GitHub issue) driving the work. Use `Refs JIRA-KEY` for related work, `Closes JIRA-KEY` for direct fixes.

5. **Test plan is a checklist.** Use `- [ ]` checkboxes. Separate automated tests (specs) from manual verification. Include count of new/modified specs.

6. **Rollback plan is mandatory.** Default is "Revert this PR." But if the change includes DB migrations, data backfills, external service changes, feature flags, or cache/queue modifications, document the full rollback procedure.

7. **Pre-existing fixes get their own section.** If you fixed something unrelated while working (N+1 query, dead code, minor bug), call it out explicitly so reviewers know it's intentional and can review it separately.

8. **Keep it scannable.** Use headers, bullets, code blocks, and tables. Avoid walls of text. A reviewer should be able to skim headers and understand the PR in 30 seconds, then dive deep where needed.

## Rollback Complexity Guide

| Change type | Rollback | Notes |
|---|---|---|
| Code-only (no side effects) | Revert PR | Default case |
| Additive DB migration (add column/table) | Revert PR + rollback migration | New column/table is unused after revert |
| Destructive DB migration (remove/rename column) | **Cannot simply revert** | Must write forward migration to restore; plan carefully before merging |
| Data backfill / data mutation | Revert PR + run undo script | Document the undo script or SQL in the rollback plan |
| External webhook / integration | Revert PR + disable in external service | Note the service and dashboard URL |
| Feature flag gated | Disable flag first, then revert | Flag must be disabled before code revert |
| Cache schema change | Revert PR + flush affected keys | Note specific cache keys or patterns |
| Queue/job format change | Drain queue first, then revert | Old code can't process new-format jobs |

## Common Mistakes

**Changelog-style summary:** "Updated file X, added method Y, modified test Z." Instead: explain the user-facing or system-level impact.

**Missing domain context:** Jumping straight into the fix without explaining how the system works. Reviewers shouldn't need to read 5 files to understand your PR.

**"Tests pass" as a test plan:** List _what_ is tested, not just that tests exist. "33 new specs covering happy path, edge cases, and no-op scenarios" is useful. "Tests pass" is not.

**No rollback plan:** Every PR should state how to undo it. "Revert this PR" is fine for most changes, but say it explicitly.

**Burying the approach rationale:** If you chose approach A over B, say why up front. Don't make reviewers guess or ask in comments.
