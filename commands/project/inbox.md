Check for new GitHub issues that were created by a person (not by the workflow).

## Arguments

$ARGUMENTS

Optional arguments:

- `"all"` - Show all open human-created issues
- `"#42"` - Pull a specific issue into the current session
- No arguments: show unassigned human-created issues

## Instructions

1. **Check gh availability:**

   ```bash
   gh auth status 2>/dev/null
   ```

   If `gh` is not available or not authenticated, tell the user and stop.

2. **Fetch open issues:**

   ```bash
   gh issue list --state open --limit 50 --json number,title,body,labels,assignees,createdAt,author
   ```

3. **Filter out agent-created issues.** Check each issue's body for `<!-- source: claude-code -->`. Issues WITHOUT this marker were created by a person.

4. **If a specific issue number was provided** (`#42`):
   - Fetch the full issue details:
     ```bash
     gh issue view 42 --json number,title,body,labels,comments
     ```
   - Present the issue to the user with a summary
   - Ask: "Would you like me to work on this issue?"
   - If yes, the issue number becomes the tracking reference for the Development Lifecycle

5. **If showing inbox** (no specific issue):
   - Present human-created issues in a table:

     ```
     ## Inbox: Issues Created by People

     | # | Title | Author | Labels | Created |
     |---|-------|--------|--------|---------|
     | 42 | feat: add dark mode | @username | enhancement | 2026-02-12 |

     <count> issue(s) from people. Use `/project:inbox "#42"` to pull one into your session.
     ```

6. **If no human-created issues exist:**
   - Report: "No new issues from people. All open issues were created by the workflow."

## Rules

- This command is READ-ONLY. It does not create, modify, or close any issues.
- The `<!-- source: claude-code -->` marker is the ONLY way to distinguish agent vs. human issues.
- When pulling an issue into a session, do NOT automatically start working — present it and wait for user confirmation.
