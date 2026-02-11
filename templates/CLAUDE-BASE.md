# {{PROJECT_NAME}} - Claude Code Configuration

## Project Overview

{{PROJECT_DESCRIPTION}}

## Key Directories

{{KEY_DIRECTORIES}}

---

## Development Lifecycle (MASTER WORKFLOW)

**MANDATORY: Create a todo list using TaskCreate for every non-trivial task.**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DEVELOPMENT LIFECYCLE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │ 1. RECEIVE  │───▶│ 2. PLAN     │───▶│ 3. REVIEW   │───▶│ 4. IMPLEMENT│  │
│  │    TASK     │    │             │    │    PLAN     │    │             │  │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘  │
│        │                  │                  │                  │          │
│        ▼                  ▼                  ▼                  ▼          │
│   Create todo        Use writing-      Staff Engineer      Follow TDD:     │
│   list for task      plans skill       sub-agent reviews   RED→GREEN→      │
│                                        MUST APPROVE        REFACTOR        │
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │ 5. TEST     │───▶│ 6. SIMPLIFY │───▶│ 7. CODE     │───▶│ 8. COMMIT   │  │
│  │             │    │             │    │    REVIEW   │    │             │  │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘  │
│        │                  │                  │                  │          │
│        ▼                  ▼                  ▼                  ▼          │
│   {{TEST_COMMAND}}  code-simplifier    superpowers:        git commit      │
│   {{TYPECHECK_COMMAND}} agent + Staff  code-reviewer       (after all      │
│   MUST PASS         review             MUST APPROVE        checks pass)    │
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐                                        │
│  │ 9. PUSH     │───▶│ 10. VERIFY  │───▶ DONE (only after CI passes)        │
│  │             │    │     CI      │                                        │
│  └─────────────┘    └─────────────┘                                        │
│        │                  │                                                 │
│        ▼                  ▼                                                 │
│   git push          gh run list                                            │
│                     MUST PASS                                              │
│                     If fails: fix → re-push → re-verify                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Planning Is MANDATORY

**ALL code changes require a plan.** No exceptions. This includes:

- Bug fixes (even "simple" ones)
- New features
- Refactoring
- Adding tests
- Updating dependencies

The only actions that don't require a plan are:

- Pure documentation updates (README, comments)
- Reverting a specific commit with `git revert`

**Never skip planning.** "Simple" changes often have hidden complexity.

### Staff Engineer Plan Review Is MANDATORY

**Every plan MUST be reviewed by a staff engineer sub-agent BEFORE implementation begins. No exceptions.**

This is Phase 3 of the Development Lifecycle. You CANNOT proceed to Phase 4 (Implement) without staff engineer approval.

**How to do it:**

```
Task(subagent_type="superpowers:code-reviewer", prompt="
  Review this implementation plan for correctness, completeness, and feasibility.
  Plan file: <path to plan>
  Verify:
  - All file paths and line numbers are accurate
  - Claimed facts about the codebase are correct (grep/read to verify)
  - No missing edge cases or tasks
  - Response shapes match actual controller/API patterns
  - Nothing is already implemented that the plan claims is missing
")
```

**Rules:**
- MUST use `Task` tool (fresh sub-agent with no shared context)
- NEVER review the plan yourself in the main conversation — you wrote it, you cannot objectively review it
- If the reviewer finds CRITICAL or IMPORTANT issues: fix the plan, then re-review
- Only proceed to implementation after the reviewer explicitly approves
- When using `ExitPlanMode`, the plan MUST already have staff engineer approval

**Red flags you're skipping this:**
- Calling `ExitPlanMode` without having dispatched a `superpowers:code-reviewer` Task for the plan
- Thinking "this plan is simple, it doesn't need review"
- Thinking "I already know it's correct"
- Reviewing the plan yourself instead of dispatching a sub-agent

### Code Simplifier Is MANDATORY

**Every code change MUST be run through the code-simplifier agent BEFORE code review. No exceptions.**

This is Phase 6 of the Development Lifecycle. You CANNOT proceed to Phase 7 (Code Review) without running the simplifier.

**How to do it:**

```
Task(subagent_type="code-simplifier")
```

**Rules:**
- MUST run after tests pass (Phase 5) and BEFORE code review (Phase 7)
- MUST run even for "simple" or single-line changes — the step exists for process discipline
- Only implement APPROVED simplifications — do not blindly apply all suggestions
- After applying approved changes, re-run tests to confirm nothing broke

**Red flags you're skipping this:**
- Jumping from "tests pass" directly to code review or commit
- Thinking "this change is too small to simplify"
- Running code review without having dispatched a `code-simplifier` Task first

### Code Review Is MANDATORY

**Every code change MUST be reviewed by a code-reviewer sub-agent BEFORE commit. No exceptions.**

This is Phase 7 of the Development Lifecycle. You CANNOT proceed to Phase 8 (Commit) without reviewer approval.

**How to do it:**

```
Task(subagent_type="superpowers:code-reviewer")
```

**Rules:**
- MUST use `Task` tool (fresh sub-agent with no shared context)
- NEVER review code yourself in the main conversation — you wrote it, you cannot objectively review it
- If the reviewer finds CRITICAL or IMPORTANT issues: fix them, re-run tests, and re-review
- Only proceed to commit after the reviewer explicitly approves

**Pre-existing issues found during review:**
- When a code reviewer flags a "pre-existing" issue in code you're touching, **ALWAYS add it to the todo list and fix it**
- Pre-existing does NOT mean "someone else's problem" — if you're shipping that code path, you own it
- Dismissing known-broken functionality as "pre-existing" means shipping a broken feature on purpose
- The only exception: the pre-existing issue is in completely unrelated code that your changes don't touch

**Red flags you're skipping this:**
- Committing without having dispatched a `superpowers:code-reviewer` Task
- Thinking "this is a one-line fix, it doesn't need review"
- Reviewing the code yourself instead of dispatching a sub-agent
- Dismissing reviewer findings as "pre-existing" without adding them to the todo list

---

## Pre-Push Workflow (MANDATORY)

**Every push MUST follow this workflow. No exceptions.**

```
┌──────────────────────────────────────────────────────────────────────┐
│  1. TESTS        →  {{TEST_COMMAND}}                                 │
│  2. TYPECHECK    →  {{TYPECHECK_COMMAND}}                            │
│  3. SIMPLIFY     →  code-simplifier agent (MANDATORY)                │
│  4. CODE REVIEW  →  superpowers:code-reviewer (MANDATORY)            │
│  5. FIX ISSUES   →  Address anything found, re-run 1-4               │
│  6. COMMIT       →  git commit                                       │
│  7. PUSH         →  git push                                         │
│  8. VERIFY CI    →  gh run list --limit 1 (MANDATORY)                │
│  9. IF CI FAILS  →  gh run view <id> --log-failed, fix & push        │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Mandatory Skills

| Priority | Trigger | Skill | Why |
|----------|---------|-------|-----|
| **P1** | Bug investigation | `systematic-debugging` | No guessing - 4-phase protocol |
| **P2** | New feature implementation | `superpowers:test-driven-development` | Tests first |

---

## TDD Workflow

**Required for all new features.**

```
RED    → Write failing test
VERIFY → Run test, confirm it fails for the right reason
GREEN  → Write minimal code to pass
VERIFY → Run test, confirm it passes
REFACTOR → Clean up while keeping tests green
COMMIT → Commit the passing test and implementation
```

---

## Debugging Protocol

**Use `systematic-debugging` skill for ANY bug. No guessing.**

| Phase | Action |
|-------|--------|
| 1. Root Cause | Read errors, reproduce, check recent changes, trace data flow |
| 2. Pattern Analysis | Find working examples, compare differences |
| 3. Hypothesis | Form ONE hypothesis, test with SMALLEST change |
| 4. Implementation | Write failing test, fix root cause, verify |

---

## Emergency Procedures

### CI Fails Repeatedly (3+ attempts)

1. STOP pushing more commits
2. Run locally: `{{BUILD_COMMAND}}`
3. If still failing: create new branch from last good state, cherry-pick commits
4. If blocked > 30 minutes: ASK USER for help

### Task is Blocked

1. Document what's blocking
2. Update task status with blocker details
3. ASK USER: "I'm blocked on X because Y. Options are: A, B, C."
4. Do NOT guess without user approval

---

## Commits

Follow conventional commit format. All commits must include:

```
Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## Quick Reference

### Testing

```bash
{{TEST_COMMAND}}
{{TYPECHECK_COMMAND}}
```

---

<!-- LANGUAGE_SPECIFIC -->
