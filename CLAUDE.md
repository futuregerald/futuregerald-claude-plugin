# futuregerald-claude-plugin - Claude Code Configuration

## Key Directories

- `internal/`
- `docs/`

---

## Development Lifecycle (MASTER WORKFLOW)

**MANDATORY: Create a todo list using TaskCreate for every non-trivial task.**

| Phase | Action | Skill/Tool | Gate |
|-------|--------|------------|------|
| 1. RECEIVE | Understand task, create todo list | `TaskCreate` | Todo list exists |
| 2. PLAN | Write implementation plan | `superpowers:writing-plans` | Plan document created |
| 3. REVIEW PLAN | Staff Engineer reviews plan | `superpowers:code-reviewer` via `Task` | Reviewer approves |
| 4. IMPLEMENT | Write code following TDD | `superpowers:test-driven-development` | Tests exist and pass |
| 5. TEST | `go test ./...` | — | Zero failures |
| 6. SIMPLIFY | `Task(subagent_type="code-simplifier")` | `code-simplifier` agent | Staff review complete |
| 7. CODE REVIEW | `Task(subagent_type="superpowers:code-reviewer")` | Fresh sub-agent | Reviewer approves |
| 8. SQL REVIEW | `Task(subagent_type="superpowers:code-reviewer")` with SQL audit prompt | `sql-optimization-patterns` skill + `sql-reviewer` agent | Reviewer approves |
| 9. COMMIT | `git commit` | — | Commit created |
| 10. PUSH | Push feature branch; `gh pr create` with `Closes #N` if `gh` available | — | Branch pushed (PR created if `gh`) |
| 11. VERIFY CI | If `gh`: `gh run list`, autonomous PR review, auto-merge when green | — | CI green (if applicable) |

**Exceptions that skip planning:** pure doc updates, `git revert`.

### Mandatory Phase Rules

**All phases are MANDATORY. No exceptions. No skipping "simple" changes.**

- **Phases 3, 6, 7, 8** MUST use `Task` tool (fresh sub-agent, no shared context)
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

**SQL review rules:**
- Run after code review passes (Phase 7), before commit (Phase 9)
- Dispatch a fresh Staff Engineer sub-agent using the `sql-reviewer` agent template
- The reviewer audits ALL database queries, mutations, and ORM usage for: **performance**, **security**, and **defensive coding**
- CRITICAL findings MUST be fixed. Re-run tests after fixes, then re-run SQL review
- IMPORTANT findings: fix if possible, otherwise open a GitHub issue immediately
- Max 3 review cycles before escalating to user

**Pre-existing issues found during review:**
- If reviewer flags a pre-existing issue in code you're touching, **fix it** — you own that code path
- Only exception: issue is in completely unrelated code your changes don't touch

**Unaddressed work MUST be tracked:**
- Any improvement, follow-up, or deferred fix identified during work (code simplifier suggestions, reviewer findings, TODOs) that is NOT addressed in the current branch MUST be tracked
- If `gh` is available: file as a GitHub issue. Otherwise: add to the todo list or note in a `TODO.md`
- This includes: approved simplifications deferred to a follow-up, pre-existing issues in unrelated code, scope-expanding suggestions
- Never silently drop findings — if you're not fixing it now, track it

---

## Branching and Sub-Agents

### Branch Protection

- **Never commit to main.** All changes go through feature branches (and PRs when `gh` is available).
- Branch naming: `<type>/<short-description>` (e.g., `feat/user-profiles`, `fix/login-redirect`)

### Sub-Agent Workflow

**The orchestrating agent NEVER writes code.** It coordinates:
- Branch management, plan management, task dispatch, and (optionally) PR creation
- Every implementation task gets a fresh sub-agent pointed at the feature branch
- Use `superpowers:subagent-driven-development` (preferred) or `superpowers:executing-plans`
- Independent tasks can run in parallel via `superpowers:dispatching-parallel-agents`

**How it works (without worktrees):**

1. Create a feature branch from main: `git checkout -b <type>/<short-description>`
2. Dispatch sub-agents to implement tasks on the current branch
3. Sub-agents write code, run tests, and commit to the feature branch
4. After all tasks complete, push the branch and create a PR (if `gh` is available)

Sub-agents work in the current working directory on the active feature branch. No worktrees are needed — the orchestrator simply checks out the feature branch and dispatches work.

### PRs (when `gh` is available)

- Use `Closes #N` (not Fixes/Resolves) in PR body to auto-close issues
- Include `Refs #N` in commit message bodies

---

## GitHub Workflow (Optional — Beta)

> **Beta:** This workflow is highly opinionated and requires the [GitHub CLI (`gh`)](https://cli.github.com/) installed, authenticated, and `/project:init` run before use. It adds structured issue tracking, git worktrees, project board management, and autonomous PR review on top of the base sub-agent workflow. **It is not required to use the plugin.** Read the README thoroughly before enabling.

### Prerequisites

- **Initialization required** before ANY GitHub write (issues, PRs, labels): run `/project:init`
- Check: `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
- If not initialized: block GitHub writes, allow local work (branches, commits)
- **Graceful degradation**: if `gh` unavailable (`gh auth status 2>/dev/null`), skip all GitHub integration and continue normally. Never block work.

### Git Worktrees

When the GitHub workflow is active, every feature branch gets its own worktree for full isolation:
```bash
REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
mkdir -p "../worktrees/$REPO_NAME"
git worktree add "../worktrees/$REPO_NAME/<branch>" -b <branch>
# Cleanup after merge:
git worktree remove "../worktrees/$REPO_NAME/<branch>"
```

- If worktree/branch already exists, reuse it (omit `-b` for existing branch)
- Monorepo: use `~/worktrees/<repo-name>/` to avoid parent repo tracking
- Sub-agents receive the worktree path and work there instead of the main working directory

### Issues

- Create in Phase 1 (RECEIVE) if `gh` available
- Use conventional commit prefixes for titles: `feat:`, `fix:`, `refactor:`, etc.
- Labels created by `/project:init` map from commit prefixes (feat→feature, fix→bug, etc.)
- Workflow-created issues include `<!-- source: claude-code -->` marker; those without it are external requests
- **Epics**: parent issues labeled `epic` grouping task sub-issues. Create with `/project:plan-feature`.
- **Issues must be actionable.** When referencing code, always include specific file paths and line numbers. If a pattern repeats in N locations, list every location. An engineer should be able to start working from the issue alone without searching the codebase.

### Autonomous PR Review (Default)

After every PR is created, automatically:

1. Dispatch `superpowers:code-reviewer` via `Task` to review `gh pr diff`
2. Post feedback on the GitHub PR via `gh pr review` (approve or request-changes)
3. If issues found: dispatch fresh sub-agents to fix → push → re-review (max 3 cycles)
4. Wait for CI: `gh pr checks <pr-number> --watch` (fix failures via sub-agent, max 3 attempts)
5. When CI passes: merge, cleanup, and pull:
   ```bash
   gh pr merge <pr> --squash --delete-branch   # merges + deletes remote branch
   git worktree remove <worktree-path>          # removes local worktree (if used)
   git branch -d <branch-name>                  # deletes local branch
   git pull                                     # updates main
   ```

**Auto-merge is mandatory when CI is green.** Do not ask for user confirmation. Post-merge cleanup (branch deletion + pull) is also mandatory — never leave stale branches.

**Safety limits:** Max 3 review cycles, max 3 CI fixes. Never merge with failing CI or unresolved Critical findings.

### Project Board (Kanban)

- Columns: Todo → In Progress → Done
- Move to "In Progress" when implementation starts (Phase 4)
- Move to "Done" after PR merged and cleaned up
- Use `gh project item-edit` with `--jq` for filtering (no external `jq`)

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
| Database queries/mutations changed | `sql-optimization-patterns` + `sql-reviewer` agent |
| About to claim completion | `verification-before-completion` |

---

## Emergency Procedures

**CI fails 3+ times:** Stop pushing. Run `go build ./...` locally. If still failing, branch from last good state + cherry-pick. If blocked >30min, ask user.

**Task blocked:** Document blocker, update task status, ask user with options A/B/C. Never guess.

---

## Commits

- Conventional commit format

## Quick Reference
```bash
go test ./...
```

---

## Go Rules

### Style

- Follow `gofmt` and `go vet` conventions
- Use short variable names for short scopes
- Return early to reduce nesting
- Handle errors explicitly, don't ignore them
```go
// Before
func processItems(items []Item) ([]Result, error) {
    results := []Result{}
    for i := 0; i < len(items); i++ {
        item := items[i]
        if item.Valid {
            result, err := process(item)
            if err != nil {
                return nil, err
            }
            results = append(results, result)
        }
    }
    return results, nil
}

// After
func processItems(items []Item) ([]Result, error) {
    var results []Result
    for _, item := range items {
        if !item.Valid {
            continue
        }
        result, err := process(item)
        if err != nil {
            return nil, err
        }
        results = append(results, result)
    }
    return results, nil
}
```

### Best Practices

- Use `defer` for cleanup
- Keep interfaces small (1-3 methods)
- Accept interfaces, return concrete types
- Use table-driven tests
- Prefer composition over inheritance (embedding)

### Testing
```bash
go test ./...             # Run all tests
go test -v ./...          # Verbose
go test -cover ./...      # With coverage
go vet ./...              # Static analysis
```

