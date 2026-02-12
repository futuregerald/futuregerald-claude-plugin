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

## GitHub Workflow (Issues, Projects, PRs)

### Initialization Required

**Before making ANY write to GitHub (issues, PRs, labels, project items), the project MUST be initialized.**

Run `/project:init` to initialize. This creates the standard labels and project board.

**How to check if initialized:**

```bash
gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$' && echo "initialized" || echo "NOT initialized"
```

**If not initialized:** Do NOT create issues, PRs, labels, or modify any GitHub state. Tell the user: "GitHub project is not initialized. Run `/project:init` first." Local work (branches, commits, worktrees) is fine — only GitHub API writes are blocked.

### Standard Labels

All projects use this label set, created by `/project:init`:

| Label         | Color        | Use For                             |
| ------------- | ------------ | ----------------------------------- |
| `feature`     | `#0E8A16` 🟢 | New features (`feat:` commits)      |
| `bug`         | `#D73A4A` 🔴 | Bug fixes (`fix:` commits)          |
| `refactor`    | `#1D76DB` 🔵 | Code refactoring                    |
| `docs`        | `#0075CA` 🔵 | Documentation changes               |
| `test`        | `#BFD4F2` ⚪ | Test additions or changes           |
| `chore`       | `#D4C5F9` 🟣 | Maintenance and housekeeping        |
| `epic`        | `#7057FF` 🟣 | Parent issue grouping related tasks |
| `task`        | `#C2E0C6` 🟢 | Sub-issue of an epic                |
| `P0-critical` | `#B60205` 🔴 | Drop everything                     |
| `P1-high`     | `#D93F0B` 🟠 | Do soon                             |
| `P2-medium`   | `#FBCA04` 🟡 | Normal priority                     |
| `P3-low`      | `#0E8A16` 🟢 | Nice to have                        |

**Mapping from conventional commit prefix to label:**

| Prefix      | Label      |
| ----------- | ---------- |
| `feat:`     | `feature`  |
| `fix:`      | `bug`      |
| `refactor:` | `refactor` |
| `docs:`     | `docs`     |
| `test:`     | `test`     |
| `chore:`    | `chore`    |
| `epic:`     | `epic`     |
| `task:`     | `task`     |

### Graceful Degradation

**All GitHub integration is OPTIONAL.** Before running any `gh` command, check availability:

```bash
# Check if gh CLI is available and authenticated
gh auth status 2>/dev/null
```

If `gh` is not available, not authenticated, or any `gh` command fails: **skip all GitHub integration and continue working normally.** The Development Lifecycle works independently of GitHub. Never block work because `gh` is unavailable. Even without `gh`, still use feature branches (not main).

### Branch Protection: Never Commit to Main

**All code changes MUST go through a Pull Request.** Never commit directly to main.

The workflow for every piece of work is:

1. Create a feature branch in a git worktree
2. Implement in the worktree (sub-agents do the work)
3. Push the branch and create a PR with `Closes #N`
4. Merge after CI passes

The workflow creates the worktree directory automatically — the user never needs to create it manually.

```bash
# Create worktree for a feature (the workflow handles mkdir -p)
REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
mkdir -p "../worktrees/$REPO_NAME"
git worktree add "../worktrees/$REPO_NAME/<branch-name>" -b <branch-name>

# When done, create PR
gh pr create --title "feat: description" --body "Closes #N"

# After merge, clean up
git worktree remove "../worktrees/$REPO_NAME/<branch-name>"
```

> **Monorepo note:** If your repo lives inside a monorepo, `../worktrees/` will be inside the parent repo's tracked area. Either add `worktrees/` to the parent's `.gitignore`, or use an absolute path outside the monorepo (e.g., `~/worktrees/<repo-name>/`).

### Work Can Start Anywhere

**Most work starts locally** in Claude Code — the agent creates issues on GitHub for tracking after the fact. But work can also start from a GitHub issue created by a person.

**How to tell the difference:** Issues created by the workflow include `<!-- source: claude-code -->` at the end of the body. Issues without this marker were created by a person (or via the GitHub UI) and should be treated as external requests.

### Issue Workflow

**When to create issues:** Phase 1 (RECEIVE) of the Development Lifecycle. When you receive a task, create a GitHub issue for it if `gh` is available.

```bash
# Create a simple issue
gh issue create --title "feat: description" --label "feature" --body "..."

# Create and add to project (project name should match repo name)
REPO_NAME=$(gh repo view --json name --jq '.name')
gh issue create --title "feat: description" --label "feature" --project "$REPO_NAME" --body "..."
```

**Issue titles** should follow conventional commit prefixes when possible: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`.

### Epics

An **epic** is a parent issue labeled `epic` that groups related task sub-issues. Use epics for:

- Features that require multiple tasks
- PRDs or design documents that need to be tracked on GitHub
- Any work that will span multiple sessions or PRs

Create epics with `/project:plan-feature`. PRD files can be passed as arguments and their content becomes the epic issue body.

### Git Worktrees

**Every feature branch gets its own worktree.** This isolates work from main and from other in-progress features.

The workflow creates the worktree directory automatically — the user never needs to create it manually.

```bash
# Convention: worktrees live in a sibling directory
# If repo is at ~/projects/my-app, worktrees go to ~/projects/worktrees/my-app/

# Create worktree (workflow ensures parent directory exists)
REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
mkdir -p "../worktrees/$REPO_NAME"
git worktree add "../worktrees/$REPO_NAME/<branch-name>" -b <branch-name>

# List active worktrees
git worktree list

# Remove after merge
git worktree remove "../worktrees/$REPO_NAME/<branch-name>"
git branch -d <branch-name>
```

**Edge cases:**

- If the worktree parent directory doesn't exist, `mkdir -p` creates it (already in the workflow)
- If the worktree path already exists, check `git worktree list` and reuse it
- If the branch already exists (e.g., resuming work), omit the `-b` flag: `git worktree add <path> <existing-branch>`
- If both exist, you're resuming — just `cd` into the worktree

**Branch naming convention:** `<type>/<short-description>` matching the issue title prefix:

- `feat/user-profiles`
- `fix/login-redirect`
- `refactor/auth-middleware`

### Sub-Agents: All Tasks in Fresh Agents

**The orchestrating agent NEVER writes code.** It coordinates:

1. Creates the worktree and feature branch
2. Creates the implementation plan
3. Dispatches each task to a **fresh sub-agent** (zero prior context)
4. Reviews between tasks (spec compliance + code quality)
5. Creates the PR when all tasks pass

**Every implementation task gets its own sub-agent.** Use `superpowers:subagent-driven-development` (preferred) or `superpowers:executing-plans`. Never accumulate implementation work in the orchestrator's context.

**Independent tasks can run in parallel** via `superpowers:dispatching-parallel-agents`. The worktree ensures sub-agents don't conflict with work on main.

**Sub-agent working directory:** Always pass the worktree path to sub-agents so they operate in the isolated branch, not in the main checkout.

**What the orchestrator does:**

- Issue management (create, track, close)
- Worktree management (create, cleanup)
- Plan creation and review
- Task dispatch and coordination
- PR creation and merge

**What the orchestrator does NOT do:**

- Write application code
- Run tests (sub-agents do this)
- Edit source files
- Debug implementation issues (dispatch a fresh sub-agent instead)

### Review Mode

**At the start of every piece of work, ask the user which review mode to use:**

- **Autonomous review:** Agents review the PR, fix findings, and merge automatically once CI is green. No human intervention needed.
- **Manual review:** PR is created and the user is notified. The user reviews and decides when to merge.

This choice is made ONCE at the beginning and applies to the entire feature lifecycle. Default to asking — never assume.

### PR Workflow

**Every piece of work results in a PR.** After pushing the feature branch, create a PR that references the issue.

**Auto-close issues** by including `Closes #<issue-number>` in the PR body:

```bash
gh pr create --title "feat: add user profiles" --body "$(cat <<'EOF'
## Summary
- Added user profile page
- Added profile settings

Closes #42

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

**Key rules:**

- Use `Closes #N` (not `Fixes` or `Resolves`) for consistency
- One PR can close multiple issues: `Closes #42, Closes #43`
- The PR title should match the conventional commit format
- Always include the `Co-Authored-By` line

### Autonomous Review Process

When the user chose **autonomous review**, after the PR is created:

1. **Dispatch a review agent** to review the PR diff:

   ```bash
   # Get the diff for context
   gh pr diff <pr-number>
   ```

   Use `superpowers:code-reviewer` via the Task tool. The reviewer checks:
   - Code correctness and logic errors
   - Security issues (injection, auth bypass, data leaks)
   - Test coverage for new functionality
   - Adherence to project conventions

2. **If the reviewer finds issues:**
   - For each finding, dispatch a **fresh sub-agent** to fix it in the worktree
   - Each sub-agent commits the fix to the feature branch
   - Push the updated branch: `git push`
   - Dispatch another review agent to verify the fixes
   - Repeat until the reviewer approves (max 3 review cycles — if still failing, fall back to manual review)

3. **Wait for CI to pass:**

   ```bash
   gh pr checks <pr-number> --watch
   ```

   If CI fails, dispatch a sub-agent to investigate and fix the failure. Push the fix and wait again.

4. **Merge when review is clean and CI is green:**

   ```bash
   gh pr merge <pr-number> --squash --delete-branch
   ```

5. **Clean up:**

   ```bash
   cd <main-checkout>
   git worktree remove <worktree-path>
   git pull
   ```

6. **Report:** Notify the user that the PR was merged with a summary of what was done.

**Safety limits:**

- Max 3 review-fix cycles. If issues persist after 3 rounds, notify the user and fall back to manual review.
- Max 3 CI-fix attempts. If CI keeps failing, stop and notify the user.
- Never merge with failing CI, even in autonomous mode.
- Never merge if the reviewer has unresolved Critical findings.

### Manual Review Process

When the user chose **manual review**, after the PR is created:

1. **Notify the user:**

   ```
   PR #<number> created: <pr-url>
   Branch: <branch-name>
   Closes: #<issue-number>

   Ready for your review. The worktree is at <worktree-path>.
   ```

2. **Wait for the user's decision.** Do not merge, do not dispatch review agents.

3. **If the user asks for changes:** Dispatch sub-agents to fix them in the worktree, push, and notify the user.

4. **After the user merges (or asks the agent to merge):** Clean up the worktree.

### Project Board Updates

Project board status is managed automatically:

- **Adding items:** Slash commands add issues to the project board via `gh project item-add` when creating issues
- **Status transitions:** Configure GitHub Project automation rules to auto-move items:
  - When PR is opened → "In Progress"
  - When PR is merged → "Done"
  - When issue is closed → "Done"
- **Manual status:** Use the GitHub web UI or `/project:current` to view status

> **Why not automate status via CLI?** `gh project item-edit` requires field IDs and option IDs that vary per project and are complex to discover programmatically. GitHub's built-in project automation rules handle the common transitions without this complexity.

> **Note:** All `gh` commands use the built-in `--jq` flag for JSON filtering instead of piping to an external `jq` binary. This eliminates the `jq` dependency.

### Integration with Development Lifecycle

| Lifecycle Phase                      | GitHub Action                                                              |
| ------------------------------------ | -------------------------------------------------------------------------- |
| 1. RECEIVE                           | Create issue (if not already tracked), create worktree + feature branch    |
| 2. PLAN                              | Plan in main checkout, implementation happens in worktree                  |
| 3-7. REVIEW PLAN through CODE REVIEW | Sub-agents work in the worktree                                            |
| 8. COMMIT                            | Commits go to the feature branch. Include `Refs #N` in commit message body |
| 9. PUSH                              | Push feature branch, create PR with `Closes #N`                            |
| 10. VERIFY CI                        | PR checks show CI status. Merge after green. Clean up worktree.            |

### Slash Commands

| Command                  | Description                                  |
| ------------------------ | -------------------------------------------- |
| `/project:init`          | Create project board and standard labels     |
| `/project:create-issue`  | Create a GitHub issue with labels            |
| `/project:plan-feature`  | Create epic from feature description or PRD  |
| `/project:sync-tasks`    | Sync todo list to GitHub issues              |
| `/project:current`       | Show project status overview (read-only)     |
| `/project:inbox`         | Check for human-created issues (read-only)   |
| `/project:cleanup`       | Find and remove stale worktrees (dry-run default) |

---

## Mandatory Skills

| Priority | Trigger | Skill | Why |
|----------|---------|-------|-----|
| **P1** | Bug investigation | `systematic-debugging` | No guessing - 4-phase protocol |
| **P2** | New feature implementation | `superpowers:test-driven-development` | Tests first |
| **P3** | About to claim completion | `verification-before-completion` | Evidence before assertions |

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
