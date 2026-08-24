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
| 2. IMPACT ANALYSIS | Trace the call chain — up and down — of every symbol the change touches | See **System Thinking** below | Callers, callees, contract, coverage recorded |
| 3. PLAN | Write implementation plan (always required, including one-line fixes) | `writing-plans` | Plan file exists and contains the Impact Analysis |
| 4. PLAN REVIEW | Adversarial review by three fresh sub-agents, concurrently | `plan-review` → `adversarial-plan-reviewer` + `plan-blindspot-hunter` + `plan-consistency-checker` (3 concurrent) | Verdict APPROVE — zero CRITICAL, zero IMPORTANT |
| 5. IMPLEMENT | Write code following TDD | `test-driven-development` | Tests exist and pass |
| 6. TEST | `go test ./...` | — | Zero failures |
| 7. SIMPLIFY | `Agent(subagent_type="code-simplifier")` | `code-simplifier` agent | Staff review complete |
| 8. CODE REVIEW | `comprehensive-code-review` — parallel correctness + safety sub-agents | Fresh sub-agents | Reviewer approves |
| 9. SQL REVIEW | If DB touched: `Agent(subagent_type="sql-reviewer")` | `sql-optimization-patterns` skill | Reviewer approves |
| 10. BLAST-RADIUS VERIFY | Walk the IMPACT ANALYSIS caller list; confirm each still holds. **Runs last, after every code-mutating phase** | — | Every caller verified with evidence |
| 11. COMMIT | `git commit` | — | Commit created |
| 12. PUSH | Push feature branch; `gh pr create` with `Closes #N` if `gh` available | — | Branch pushed (PR created if `gh`) |
| 13. VERIFY CI | If `gh`: `gh run list`, autonomous PR review, auto-merge when green | — | CI green (if applicable) |

**Exceptions that skip planning:** pure doc updates, `git revert`.

### Mandatory Phase Rules

**All phases are MANDATORY. No exceptions. No skipping "simple" changes.**

- **PLAN REVIEW, SIMPLIFY, CODE REVIEW, and SQL REVIEW** MUST use fresh sub-agents via the Agent tool — no shared context. PLAN REVIEW dispatches three concurrently
- NEVER review your own plan or code — you wrote it, you cannot objectively review it
- **PLAN REVIEW is not CODE REVIEW.** Plan review and code review are separate gates with separate agents. Passing one never satisfies the other, and a code review cannot recover a wrong plan
- **No source file gets edited before PLAN REVIEW passes.** This includes "while I'm in here" fixes
- **"Review the plan" — in any phrasing — means invoke `plan-review`.** Never satisfy it by re-reading the plan yourself
- Give the reviewer neutral inputs only: plan path, the goal it serves, repo root, base SHA. **Never pass your own suspicions or "check X"** — a reviewer aimed at your worries inherits your blind spots
- If reviewer finds CRITICAL/IMPORTANT issues: fix, re-run tests, re-review with a fresh agent
- Only proceed after explicit reviewer approval
- `ExitPlanMode` requires a passed PLAN REVIEW
- **Any phase that mutates code re-opens BLAST-RADIUS VERIFY.** SIMPLIFY edits, and review fix-cycles edit. Re-running tests is not enough — a fix that alters a return contract regresses exactly the callers IMPACT ANALYSIS recorded as having no test. Re-walk the caller list before COMMIT

### System Thinking: Trace Before You Touch (Mandatory)

**The dominant failure mode is a locally-correct change with unconsidered downstream effects.** The code compiles, the new test passes, and something three call sites away breaks.

Before modifying any function, method, type, endpoint, or schema, reconstruct its call chain in both directions. This is the IMPACT ANALYSIS phase; its output is a required section of the plan.

- **Upward** — every direct caller, then transitively out to real entry points (handler, command, job, scheduler, public API). For each: what does it do with the return value, and which part of the contract does it rely on? Include callers outside this repo.
- **Downward** — every function called and its side effects: writes, external calls, queue sends, cache mutations, file IO. What errors propagate, and who handles them.
- **The invisible edges** — reflection, interface dispatch, struct tags, code generation, registry maps, config-driven wiring, handler names as strings. A call graph cannot see these. Grep for them deliberately, every time.
- **Contract** — current return shapes, zero values, nil cases, errors, ordering; which the change alters, and the specific callers affected by each.
- **Coverage** — which callers have tests; what test would fail if the change were wrong.

Use graph tools first, grep second and only for what graphs cannot see. Conclusions rest on files actually read.

**The bar:** "I read the function and it looks fine" is not an impact analysis. If you cannot name every caller and say what each expects, IMPACT ANALYSIS is not done — and the PLAN REVIEW reviewers will rebuild this chain independently and treat any caller you missed as at least IMPORTANT.

### Prove It, Don't Assume It

Reasoning from memory about runtime behavior is how wrong premises reach a plan. Settle it with evidence — cheapest first:

1. **Trace it** — graph tools, call paths, the code index. Fastest, usually decisive
2. **Read the actual source** — the installed dependency version, the schema, the generated file. If you can read the answer, you do not need to run it
3. **Spike it** — last resort, only for runtime behavior you cannot read off the code

If steps 1–2 leave you confident, stop and cite the evidence. Spiking what you already established wastes time and tokens. When you do spike:

- Scratch or temp directory only. Never the working tree, never repo files
- **Keep it small: one file, a few dozen lines, isolating the single behavior.** Never rebuild the app, boot the framework, or stand up a database — if proving it requires that, it is not a spike
- Two attempts, a few minutes. Then abandon it and state only what you can support
- Where independent checks would run serially, dispatch narrowly-scoped sub-agents in parallel — one question each. Do not spawn an agent for what a single search would answer

**Plan review dispatch:** see the `plan-review` skill — three agents in one message. Give each neutral inputs only — plan path, the goal it serves, repo root, base SHA. Each carries its own methodology; do not restate it, do not tailor the input per agent, and do not steer them. A **Claim Ledger** inside the plan file is not steering: it is a list of the author's own liabilities to falsify.

**Code simplifier rules:**
- Run after TEST passes, before CODE REVIEW
- Only implement APPROVED simplifications
- Re-run tests after applying changes

**SQL review rules:**
- Run after CODE REVIEW passes, before COMMIT
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
- Use `subagent-driven-development` (preferred) or `executing-plans`
- Independent tasks can run in parallel via `dispatching-parallel-agents`

**How it works (without worktrees):**

1. Create a feature branch from main: `git checkout -b <type>/<short-description>`
2. Dispatch sub-agents to implement tasks on the current branch
3. Sub-agents write code, run tests, and commit to the feature branch
4. After all tasks complete, push the branch and create a PR (if `gh` is available)

Sub-agents work in the current working directory on the active feature branch. No worktrees are needed — the orchestrator simply checks out the feature branch and dispatches work.

### PRs (when `gh` is available)

- **Always use the `pull-request-description` skill when creating or updating a PR.** This is mandatory, no exceptions.
- Use `Closes #N` (not Fixes/Resolves) in PR body to auto-close issues
- Include `Refs #N` in commit message bodies

---

## GitHub Workflow (Optional — Beta)

> **Beta:** This workflow is highly opinionated and requires the [GitHub CLI (`gh`)](https://cli.github.com/) installed, authenticated, and `/project:init` run before use. It adds structured issue tracking, git worktrees, project board management, and autonomous PR review on top of the base sub-agent workflow. **It is not required to use the plugin.** Read the README thoroughly before enabling.

### Prerequisites

- **Initialization required** before ANY GitHub write (issues, PRs, labels): run `/project:init`
- Check: `cat .claude/project.json 2>/dev/null | grep -q '"initialized": true'` (fast, local) or fall back to `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
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

- Create in RECEIVE if `gh` available
- Use conventional commit prefixes for titles: `feat:`, `fix:`, `refactor:`, etc.
- Labels created by `/project:init` map from commit prefixes (feat→feature, fix→bug, etc.)
- Workflow-created issues include `<!-- source: claude-code -->` marker; those without it are external requests
- **Epics**: parent issues labeled `epic` grouping task sub-issues. Create with `/project:plan-feature`.
- **Issues must be actionable.** When referencing code, always include specific file paths and line numbers. If a pattern repeats in N locations, list every location. An engineer should be able to start working from the issue alone without searching the codebase.

### Autonomous PR Review (Default)

After every PR is created, automatically:

1. Dispatch `code-quality-reviewer` via `Task` to review `gh pr diff`
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
- Move to "In Progress" when IMPLEMENT starts — i.e. only after PLAN REVIEW has returned APPROVE
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
| **Any code change, before writing code (PLAN)** | `writing-plans` — always required, including one-line fixes. Must contain the Impact Analysis |
| **Plan written, before implementing (PLAN REVIEW)** | `plan-review` → three concurrent reviewers (`adversarial-plan-reviewer`, `plan-blindspot-hunter`, `plan-consistency-checker`). **Also the required response to "review the plan" in any phrasing.** Never review a plan yourself |
| Code review before commit (CODE REVIEW) | `comprehensive-code-review` — parallel correctness + safety sub-agents |
| Bug investigation | `systematic-debugging` |
| New feature | `test-driven-development` (RED→GREEN→REFACTOR) |
| Database queries/mutations changed | `sql-optimization-patterns` + `sql-reviewer` agent |
| Creating or updating a pull request | `pull-request-description` — structured summary, background, test plan, rollback plan. **Mandatory for both new PRs and PR description updates.** |
| Codebase search or exploration | `future-code-search` — delegates search to Haiku/Sonnet sub-agents, keeps Opus as orchestrator. **Invoke before any Agent(Explore), Grep, or multi-file Read.** |

---

## Verification Before Completion (Always On)

Evidence before claims. Never state that something is done, fixed, passing, or working without having run the check in THIS turn and read its output.
- Before any success/completion claim: (1) identify the command that proves it, (2) run it fresh and complete, (3) read full output + exit code, (4) then claim, with the evidence.
- Don't trust a sub-agent's "success" — verify via the diff/output yourself.
- STOP-and-verify red flags: "should work", "probably", "looks correct", or "Great!/Done!" before running anything; committing or opening a PR without a green check.

## Use Your Skills

Before acting on a task, check whether an installed skill applies and use it — don't reinvent a workflow a skill already encodes.

---

## Prism Session Memory (Mandatory)

**Prism is the persistent memory layer across sessions. These rules are mandatory — no exceptions.**

### Session Start
- **Always call `session_load_context`** at the start of every session to recover prior work state. Use `standard` level by default, `deep` if resuming complex work.

### Session End
- **Always call `session_save_ledger`** before the conversation ends if any meaningful work was done (code changes, decisions, debugging, planning, reviews). Include: what was done, key decisions, files changed, and any open questions.
- **Always call `session_save_handoff`** when a task is paused, blocked, or the conversation is wrapping up with unfinished work. This lets the next session pick up seamlessly.

### After Significant Learnings
- **Call `session_save_experience`** after resolving non-trivial bugs, discovering important patterns, or making architectural decisions worth preserving.

### What Counts as "Meaningful Work"
Any session involving: code changes, debugging, architecture discussions, planning, ticket grooming, PR reviews, or decisions that affect future work. Casual Q&A or simple lookups do not require saving.

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

