Find and clean up stale git worktrees whose branches have been merged or are idle.

## Arguments

$ARGUMENTS

Optional arguments:

- `"--dry-run"` - Show what would be removed without removing anything (default)
- `"--force"` - Actually remove stale worktrees and delete their branches
- No arguments: defaults to dry-run

## Instructions

1. **List all worktrees:**

   ```bash
   git worktree list --porcelain
   ```

   Parse the output to get each worktree's path and branch.
   Skip the main worktree (the primary checkout).

2. **For each worktree, determine if it's stale.** A worktree is stale if ANY of these are true:
   - **Branch merged:** `git branch --merged main` includes the branch name
   - **PR merged/closed:** `gh pr list --head <branch> --state merged --json number` returns results (or `--state closed`)
   - **Idle:** Last commit on the branch is older than 7 days AND no open PR exists for it

   Check merged status:

   ```bash
   # Check if branch is merged into main
   git branch --merged main | grep -q "<branch-name>"

   # Check PR status (if gh available)
   gh pr list --head "<branch-name>" --state all --json number,state,mergedAt --limit 1
   ```

   Check last activity:

   ```bash
   git log -1 --format="%ci" "<branch-name>"
   ```

3. **Present findings:**

   ```
   ## Worktree Cleanup

   ### Stale Worktrees (<count>)
   | Branch | Path | Reason | Last Commit |
   |--------|------|--------|-------------|
   | feat/done-feature | ../worktrees/my-app/feat/done-feature | Branch merged to main | 2026-02-10 |
   | fix/old-bug | ../worktrees/my-app/fix/old-bug | Idle 11 days, no PR | 2026-02-01 |

   ### Active Worktrees (<count>)
   | Branch | Path | PR | Last Commit |
   |--------|------|----|-------------|
   | feat/user-profiles | ../worktrees/my-app/feat/user-profiles | #15 (open) | 2026-02-12 |
   ```

4. **If `--dry-run` (default):**
   - Show what would be removed
   - Say: "Run `/project:cleanup "--force"` to remove stale worktrees."

5. **If `--force`:**
   - **Confirm with user before each removal** if the worktree has uncommitted changes
   - For each stale worktree:

     ```bash
     # Check for uncommitted changes first
     cd <worktree-path>
     git status --porcelain

     # If clean (or user confirmed), remove
     cd <main-checkout>
     git worktree remove <worktree-path>
     git branch -d <branch-name>  # -d (safe delete, fails if not merged)
     ```

   - If `git branch -d` fails (branch not fully merged), warn the user and skip unless they confirm `-D` (force delete)
   - Report what was removed and what was skipped

## Rules

- **Default is dry-run.** Never remove anything without `--force`.
- **Never remove the main worktree.**
- **Never force-delete branches** (`-D`) without explicit user confirmation — data loss risk.
- If a worktree has uncommitted changes, warn and skip unless the user confirms.
- This command does NOT close GitHub issues or PRs — it only cleans up local worktrees and branches.
