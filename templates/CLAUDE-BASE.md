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
