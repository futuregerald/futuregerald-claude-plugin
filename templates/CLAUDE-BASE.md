# {{PROJECT_NAME}} - Claude Code Configuration

## Project Overview

{{PROJECT_DESCRIPTION}}

## Key Directories

{{KEY_DIRECTORIES}}

---

## Development Lifecycle (MASTER WORKFLOW)

**MANDATORY: Create a todo list using TaskCreate for every non-trivial task.**

| Phase | Action | Skill/Tool | Gate |
|-------|--------|------------|------|
| 1. RECEIVE | Understand task, create todo list | `TaskCreate` | Todo list exists |
| 2. PLAN | Write implementation plan | `superpowers:writing-plans` | Plan document created |
| 3. REVIEW PLAN | Staff Engineer reviews plan | `superpowers:code-reviewer` via `Task` | Reviewer approves |
| 4. IMPLEMENT | Write code following TDD | `superpowers:test-driven-development` | Tests exist and pass |
| 5. TEST | `{{TEST_COMMAND}}` + `{{TYPECHECK_COMMAND}}` | — | Zero failures |
| 6. SIMPLIFY | `Task(subagent_type="code-simplifier")` | `code-simplifier` agent | Staff review complete |
| 7. CODE REVIEW | `Task(subagent_type="superpowers:code-reviewer")` | Fresh sub-agent | Reviewer approves |
| 8. COMMIT | `git commit` | — | Commit created |
| 9. PUSH | Push feature branch, `gh pr create` with `Closes #N` | — | PR created |
| 10. VERIFY CI | `gh run list`, autonomous PR review, auto-merge when green | — | CI green, merged |

**Exceptions that skip planning:** pure doc updates, `git revert`.

### Mandatory Phase Rules

**All phases are MANDATORY. No exceptions. No skipping "simple" changes.**

- **Phases 3, 6, 7** MUST use `Task` tool (fresh sub-agent, no shared context)
- NEVER review your own plan or code — you wrote it, you cannot objectively review it
- If reviewer finds CRITICAL/IMPORTANT issues: fix, re-run tests, re-review
- Only proceed after explicit reviewer approval
- `ExitPlanMode` requires prior staff engineer approval of the plan

**Plan review prompt template:**
```
Task(subagent_type="superpowers:code-reviewer", prompt="
  Review this plan: <path>. Verify: file paths accurate, codebase facts correct,
  no missing edge cases, response shapes match actual patterns, nothing already implemented.
")
```

**Code simplifier rules:**
- Run after tests pass (Phase 5), before code review (Phase 7)
- Only implement APPROVED simplifications
- Re-run tests after applying changes

**Pre-existing issues found during review:**
- If reviewer flags a pre-existing issue in code you're touching, **fix it** — you own that code path
- Only exception: issue is in completely unrelated code your changes don't touch

**Unaddressed work MUST become GitHub issues:**
- Any improvement, follow-up, or deferred fix identified during work (code simplifier suggestions, reviewer findings, TODOs) that is NOT addressed in the current PR MUST be filed as a GitHub issue
- This includes: approved simplifications deferred to a follow-up, pre-existing issues in unrelated code, scope-expanding suggestions
- Never silently drop findings — if you're not fixing it now, track it

---

## GitHub Workflow

### Prerequisites

- **Initialization required** before ANY GitHub write (issues, PRs, labels): run `/project:init`
- Check: `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
- If not initialized: block GitHub writes, allow local work (branches, commits, worktrees)
- **Graceful degradation**: if `gh` unavailable (`gh auth status 2>/dev/null`), skip all GitHub integration and continue normally. Never block work.

### Branch Protection

- **Never commit to main.** All changes go through PRs.
- Branch naming: `<type>/<short-description>` (e.g., `feat/user-profiles`, `fix/login-redirect`)

### Git Worktrees

Every feature branch gets its own worktree:

```bash
REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
mkdir -p "../worktrees/$REPO_NAME"
git worktree add "../worktrees/$REPO_NAME/<branch>" -b <branch>
# Cleanup after merge:
git worktree remove "../worktrees/$REPO_NAME/<branch>"
```

- If worktree/branch already exists, reuse it (omit `-b` for existing branch)
- Monorepo: use `~/worktrees/<repo-name>/` to avoid parent repo tracking

### Issues

- Create in Phase 1 (RECEIVE) if `gh` available
- Use conventional commit prefixes for titles: `feat:`, `fix:`, `refactor:`, etc.
- Labels created by `/project:init` map from commit prefixes (feat→feature, fix→bug, etc.)
- Workflow-created issues include `<!-- source: claude-code -->` marker; those without it are external requests
- **Epics**: parent issues labeled `epic` grouping task sub-issues. Create with `/project:plan-feature`.
- **Issues must be actionable.** When referencing code, always include specific file paths and line numbers. If a pattern repeats in N locations, list every location. An engineer should be able to start working from the issue alone without searching the codebase.

### PRs

- Use `Closes #N` (not Fixes/Resolves) in PR body to auto-close issues
- Include `Refs #N` in commit message bodies

### Autonomous PR Review (Default)

After every PR is created, automatically:

1. Dispatch `superpowers:code-reviewer` via `Task` to review `gh pr diff`
2. Post feedback on the GitHub PR via `gh pr review` (approve or request-changes)
3. If issues found: dispatch fresh sub-agents to fix → push → re-review (max 3 cycles)
4. Wait for CI: `gh pr checks <pr-number> --watch` (fix failures via sub-agent, max 3 attempts)
5. When CI passes: merge, cleanup, and pull:
   ```bash
   gh pr merge <pr> --squash --delete-branch   # merges + deletes remote branch
   git worktree remove <worktree-path>          # removes local worktree
   git branch -d <branch-name>                  # deletes local branch
   git pull                                     # updates main
   ```

**Auto-merge is mandatory when CI is green.** Do not ask for user confirmation. Post-merge cleanup (worktree + local branch deletion + pull) is also mandatory — never leave stale worktrees or branches.

**Safety limits:** Max 3 review cycles, max 3 CI fixes. Never merge with failing CI or unresolved Critical findings.

### Project Board (Kanban)

- Columns: Todo → In Progress → Done
- Move to "In Progress" when implementation starts (Phase 4)
- Move to "Done" after PR merged and worktree cleaned
- Use `gh project item-edit` with `--jq` for filtering (no external `jq`)

### Sub-Agents

**The orchestrating agent NEVER writes code.** It coordinates:
- Worktree/issue/plan management, task dispatch, PR creation
- Every implementation task gets a fresh sub-agent with the worktree path
- Use `superpowers:subagent-driven-development` (preferred) or `superpowers:executing-plans`
- Independent tasks can run in parallel via `superpowers:dispatching-parallel-agents`

### Slash Commands

- `/project:init` — **Run first.** Creates board + labels
- `/project:create-issue`, `/project:plan-feature`, `/project:sync-tasks`
- `/project:current`, `/project:inbox` — read-only, work before init
- `/project:cleanup` — stale worktrees (dry-run default)

---

## Mandatory Skills

| Trigger | Skill |
|---------|-------|
| Bug investigation | `systematic-debugging` |
| New feature | `superpowers:test-driven-development` (RED→GREEN→REFACTOR) |
| About to claim completion | `verification-before-completion` |

---

## Emergency Procedures

**CI fails 3+ times:** Stop pushing. Run `{{BUILD_COMMAND}}` locally. If still failing, branch from last good state + cherry-pick. If blocked >30min, ask user.

**Task blocked:** Document blocker, update task status, ask user with options A/B/C. Never guess.

---

## Commits

- Conventional commit format

## Quick Reference

```bash
{{TEST_COMMAND}}
{{TYPECHECK_COMMAND}}
```

---

<!-- LANGUAGE_SPECIFIC -->
