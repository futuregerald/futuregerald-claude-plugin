Show the current project status: open issues, active PRs, and project board state.

## Arguments

$ARGUMENTS

Optional arguments:

- `"issues"` - Show only issues
- `"prs"` - Show only pull requests
- `"board"` - Show only project board
- No arguments: show everything

## Instructions

1. **Check gh availability:**

   ```bash
   gh auth status 2>/dev/null
   ```

   If `gh` is not available or not authenticated, tell the user: "GitHub CLI is not available. Cannot show project status." and stop.

2. **Gather data** (run these in parallel where possible):

   ```bash
   # Open issues
   gh issue list --state open --json number,title,labels,assignees,createdAt --limit 50

   # Open PRs
   gh pr list --state open --json number,title,labels,headRefName,isDraft,reviewDecision,statusCheckRollup --limit 20

   # Recent closed issues (last 10)
   gh issue list --state closed --json number,title,closedAt --limit 10

   # Recent merged PRs (last 5)
   gh pr list --state merged --json number,title,mergedAt --limit 5

   # Active worktrees (always available, no gh needed)
   git worktree list --porcelain
   ```

3. **Get project board data:**

   ```bash
   REPO_NAME=$(gh repo view --json name --jq '.name')
   PROJECT_NUM=$(gh project list --owner @me --format json --jq ".projects[] | select(.title == \"$REPO_NAME\") | .number")
   if [ -n "$PROJECT_NUM" ]; then
     gh project item-list "$PROJECT_NUM" --owner @me --format json --limit 50
   fi
   ```

4. **Check for stale worktrees.** For each worktree (excluding the main checkout):
   - Get the branch name from `git worktree list`
   - Check if the branch has been merged to main: `git branch --merged main | grep <branch>`
   - Check the last commit date on the branch: `git log -1 --format=%ci <branch>`
   - A worktree is **stale** if:
     - Its branch has already been merged to main, OR
     - Its last commit is older than 7 days with no open PR

5. **Present the status** in a clear, organized format:

   ```
   ## Project Status: <repo-name>

   ### Open Issues (<count>)
   | # | Title | Labels | Created |
   |---|-------|--------|---------|
   | 42 | feat: user profiles | feature | 2026-02-12 |

   ### Open Pull Requests (<count>)
   | # | Title | Branch | Status | CI |
   |---|-------|--------|--------|-----|
   | 15 | feat: add user profiles | feat/user-profiles | Review pending | Passing |

   ### Active Worktrees (<count>)
   | Branch | Path | Last Commit | PR | Status |
   |--------|------|-------------|-----|--------|
   | feat/user-profiles | ../worktrees/my-app/feat/user-profiles | 2026-02-12 | #15 | Active |

   ### Project Board
   | Status | Count | Issues |
   |--------|-------|--------|
   | Todo | 3 | #42, #43, #44 |
   | In Progress | 1 | #45 |
   | Done | 5 | #38, #39, #40, #41, #36 |

   ### Recently Completed
   - #41 fix: login redirect (closed 2026-02-11)
   - #40 feat: 2FA support (closed 2026-02-10)

   ### Suggested Next Actions
   - Issue #42 is ready to start → `/project:plan-feature "#42"`
   - PR #15 needs review
   ```

6. **Suggest next actions** based on the current state:
   - Unassigned issues that could be started
   - PRs that need attention (review, CI fixes)
   - Stale issues (open > 2 weeks with no activity)
   - Stale worktrees that should be cleaned up

## Rules

- Do NOT modify any data. This command is read-only.
- If the project board doesn't exist, skip that section and note it.
- If there are no issues or PRs, say so clearly instead of showing empty tables.
- Keep the output concise. If there are many items, show the most recent/relevant and summarize the rest.
