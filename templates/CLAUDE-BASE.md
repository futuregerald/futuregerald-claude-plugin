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
| 3. IMPLEMENT | Write code following TDD | `superpowers:test-driven-development` | Tests exist and pass |
| 4. TEST | `{{TEST_COMMAND}}` + `{{TYPECHECK_COMMAND}}` | — | Zero failures |
| 5. SIMPLIFY | `Task(subagent_type="code-simplifier")` | `code-simplifier` agent | Staff review complete |
| 6. REVIEW | `comprehensive-code-review` skill — dispatches 5 parallel sub-agents (code quality, pattern consistency, SQL, security, simplification) | Fresh sub-agents | Findings documented by severity |
| 7. FIX FINDINGS | Address all CRITICAL findings; address or explicitly defer IMPORTANT findings (tracking issue + user approval); track MINOR findings | — | All CRITICAL + IMPORTANT resolved; MINOR tracked |
| 8. COMMIT | `git commit` | — | Commit created |
| 9. PUSH | Push feature branch; `gh pr create` with `Closes #N` if `gh` available | — | Branch pushed (PR created if `gh`) |
| 10. VERIFY CI | If `gh`: `gh run list`, autonomous PR review, auto-merge when green | — | CI green (if applicable) |

**Exceptions that skip planning:** pure doc updates, `git revert`.

### Mandatory Phase Rules

**All phases are MANDATORY. No exceptions. No skipping "simple" changes.**

- **Phases 5, 6** MUST use `Task` tool or `Agent` tool (fresh sub-agents, no shared context)
- NEVER review your own code — you wrote it, you cannot objectively review it
- If reviewer finds CRITICAL/IMPORTANT issues: fix, re-run tests, re-review
- Only proceed after explicit reviewer approval

**Code simplifier rules:**
- Run after tests pass (Phase 4), before review (Phase 6)
- Only implement APPROVED simplifications
- Re-run tests after applying changes

**Review rules (Phase 6 — `comprehensive-code-review`):**
- Invoke the `comprehensive-code-review` skill — it orchestrates all review dimensions in parallel
- Covers: code quality, pattern consistency, SQL performance, security (OWASP), and simplification
- SQL sub-agent is automatically dispatched if DB-touching files changed; skipped otherwise
- **CRITICAL findings:** MUST be fixed before commit — hard block on Phase 7. Re-run tests and re-review after fixes.
- **IMPORTANT findings:** MUST be addressed before commit — fix the code, or (if scope-expanding) open a tracking issue AND get explicit user approval to defer. Cannot silently skip.
- **MINOR findings:** address if straightforward; otherwise open a tracking issue. Do not block commit.
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
- Move to "In Progress" when implementation starts (Phase 3)
- Move to "Done" after PR merged and cleaned up
- Use `gh project item-edit` with `--jq` for filtering (no external `jq`)

### Slash Commands

- `/project:init` — **Run first.** Creates board + labels
- `/project:create-issue`, `/project:plan-feature`, `/project:sync-tasks`
- `/project:current`, `/project:inbox` — read-only, work before init
- `/project:cleanup` — stale worktrees (dry-run default)

---

## Codebase Graph (codebase-memory-mcp)

If the codebase-memory-mcp server is configured, use these tools proactively — don't wait to be asked.

| Context | Tool | Purpose |
|---------|------|---------|
| Phase 2 (PLAN) | `get_architecture` | Understand affected areas before planning |
| Phase 6 (REVIEW) | `search_graph` | Check impact radius. Callers come from grep — see the caveat below; SQL callers traced by SQL sub-agent |
| Debugging (`systematic-debugging`) | `search_graph`, `trace_call_path` | Understand call chains before guessing — corroborate any caller claim with grep |
| Searching for relationships | `search_graph` | Prefer over text grep when searching for function/class relationships |

**Rules:**
- During **debugging**, use `trace_call_path` and `search_graph` to understand the call chain before proposing fixes. Don't guess — trace. Corroborate callers with grep before acting on them.
- During **review** (Phase 6), ALWAYS use `search_graph` to check the impact radius of changes.
- During **planning** (Phase 2), use `get_architecture` to understand the affected areas.
- Use `search_graph` over text grep when searching for function relationships, not just text matches.

**What the graph does and does not prove.** A row proves a symbol EXISTS at that file:line. A symbol's ABSENCE proves nothing, and a zero-caller result is a prompt to grep — not a conclusion that nothing calls it. Call graphs cannot see reflection, interface dispatch, registry maps, config-driven wiring, or handler names held as strings, and in codebases built on those the graph will confidently report no callers for symbols that have them. **Never close "verify no callers are broken" on graph output alone.**

### Reuse Before You Build

**Before writing a new function, type, helper, or job, check whether it already exists.** Duplication caught in review is duplication someone already paid to write.

- Search the **name** — `search_graph(name_pattern="<stem>")`.
- Search the **body**, because a duplicate is usually named differently. Pull distinctive tokens out of what you are about to write — constants, called functions, type names — and search those too. A new `AddTwoBusinessDays` is found by its name; an existing `skipWeekend` that references a weekday table is only found by the body token.
- Count it — `query_graph("MATCH (m:Method) WHERE m.name CONTAINS '<stem>' RETURN m.name AS name, count(*) AS n ORDER BY n DESC")`. Read the top rows — the count alone means nothing, because a common verb matches every class of its kind. Skip framework verbs and single short words; a stem worth counting is specific and multi-word.
- **Read the candidate's body before reusing or rejecting it.** A matching name is a coincidence until you have read the code.

Absence of a name is not absence of the capability. "I found nothing" is only credible with the queries attached.

### Match the Pattern When You Do Build

Reuse says don't write it. This says: when you must write it, make it look like the code next to it. New code that solves an old problem a new way is the most common defect in AI-assisted changes, and it passes tests every time.

**Find the canonical exemplar first — one file, not a survey.** The most recently changed sibling in the same directory is the pattern, because it is the one that most recently passed review:

```bash
# Exclude the files you are changing: yours is by construction the newest in
# the directory, so without this you rank the change as its own gold standard.
for f in "$(dirname "$FILE")"/*.<ext>; do
  [ "$f" = "$FILE" ] && continue
  echo "$(git log -1 --format=%ad --date=short -- "$f") $f"
done | sort -r | head -3
```

Match it on **naming, structure, error handling, logging, dependency wiring, and the shape and location of its tests.** Then confirm the family agrees with `search_graph(file_pattern=...)` and `search_graph(qn_pattern=...)`.

- **Cite the exemplar in the plan.** "Modelled on `<path>:<line>`" is checkable; "follows codebase conventions" is not.
- **Where the neighbours disagree with each other, the newest one wins** — and say that you found disagreement, because an inconsistent directory is itself worth reporting.
- **A convention you can quote from the repo's own docs outranks one you inferred from code.**
- **Diverge only deliberately, and say so in the PR with the reason.** An undocumented divergence reads to every future reader as an accident.

### Pre-Work: Read the Repo's Written Conventions

**Before starting any work on a repo**, read what the team already wrote down:

```bash
ls docs/adr/*.md 2>/dev/null | head -60
ls docs/good-practices.md docs/*GUIDELINES*.md docs/*PATTERNS*.md CONTRIBUTING.md docs/CONTRIBUTING.md 2>/dev/null
```

Read `CONTRIBUTING.md` for branching, testing and deployment; the `docs/adr/` titles, then the two or three that bear on the change; and any guideline document in full. These are far more specific than anything you would infer from reading code, and a convention you can quote outranks one you inferred.

---

## Mandatory Skills

| Trigger | Skill |
|---------|-------|
| Bug investigation | `systematic-debugging` |
| New feature | `superpowers:test-driven-development` (RED→GREEN→REFACTOR) |
| Database queries/mutations changed | `sql-optimization-patterns` + `sql-reviewer` agent |
| Code review (Phase 6+7), reviewing a PR, or GitHub code review requested | `comprehensive-code-review` — orchestrates parallel sub-agents for code quality, patterns, SQL, security, and simplification |
| Creating or updating a pull request | `pull-request-description` — structured summary, background, test plan, rollback plan. **Mandatory for both new PRs and PR description updates.** |
| About to claim completion | `verification-before-completion` |

---

## Communication Style (Always On)

**Plain language over jargon. Concise. Clear.** This applies to everything you write for the user — chat, PR reviews, commit messages, plan documents, issue comments. Not just when asked.

### Plain language

- Use the ordinary word when it says the same thing. "Crashes when the list is empty" beats "exhibits undefined behavior at zero cardinality."
- When a precise term is genuinely needed — `NoMethodError`, `context.fail!`, `SIGPIPE` — use it and add a short clause so the reader doesn't have to look it up.
- Name the thing that goes wrong, not the category it belongs to. "The comma lands inside the comment, so the real last property never gets one" beats "improper delimiter placement in comment-adjacent context."
- No filler. Cut "it should be noted that", "it is worth mentioning", "importantly", "essentially".

### Concise

- Lead with the answer. Reasoning after, and only as much as changes what the user does.
- Say it once. Don't restate the same point in different words, and don't summarise a section you just wrote.
- Length matches the stakes. A one-line answer to a one-line question. Don't pad a small finding into a report.
- No emoji unless asked.

### Still be specific

Concise does not mean vague. Keep the details that let the user act or check your work:

- File paths with line numbers, ticket keys, dates, names, exact commands and their output
- For risks and blockers: which ticket, which person, by when
- For findings: what breaks, under what input, and what happens as a result

### By task type

- **Project management** (tickets, team status): structured summary, clear takeaway up top
- **Engineering planning**: trade-offs and risks stated plainly, with a recommendation
- **Writing**: match the tone the user specifies
- **Code review**: write for the person fixing it — what's wrong, why it matters, what to do, in that order. Titles say what's broken, not a CWE number.

---

## Jira Integration (when Atlassian MCP is configured)

### Posting Comments

When posting comments to Jira via `addCommentToJiraIssue`, you **MUST** set `contentFormat: "markdown"`. Write all comment bodies in standard markdown. Omitting `contentFormat` causes the API to default to ADF, which renders markdown as broken plain text.

### acli Fallback

When an operation is not available via the Atlassian MCP server (e.g., deleting comments, bulk edits), use the `acli` CLI instead:

```bash
# Delete a comment
acli jira workitem comment delete --key PROJ-1234 --id 12345

# List comments (to find IDs)
acli jira workitem comment list --key PROJ-1234
```

Always prefer MCP tools for reads and writes. Use `acli` only when MCP lacks the capability.

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
