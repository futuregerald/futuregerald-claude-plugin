Sync current todo list tasks to GitHub issues (one-way: local to GitHub).

## Arguments

$ARGUMENTS

Optional arguments:

- A parent issue number to associate tasks with: `"#42"`
- A label to apply to all synced tasks: `"task"`
- No arguments: syncs all current tasks

## Instructions

1. **Check gh availability:**

   ```bash
   gh auth status 2>/dev/null
   ```

   If `gh` is not available or not authenticated, tell the user and stop.

2. **Check initialization:**

   ```bash
   gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'
   ```

   If the `claude:initialized` label does NOT exist, tell the user: "GitHub project is not initialized. Run `/project:init` first." and stop.

3. **Read the current todo list.** Use the `TaskList` tool to get all current tasks (use `TaskGet` for full details on individual tasks).

4. **List existing GitHub issues** to avoid duplicates:

   ```bash
   gh issue list --state open --limit 100 --json number,title,body
   ```

5. **For each todo item**, check if a matching GitHub issue already exists:
   - Match by exact title or by close similarity (ignore prefix differences like "feat:" vs "task:")
   - If it exists, skip it and note "already tracked as #N"
   - If it does not exist, create it

6. **Create missing issues:**

   ```bash
   gh issue create \
     --title "<conventional-commit-prefix>: <task-description>" \
     --label "task" \
     --body "$(cat <<'EOF'
   Synced from local todo list.

   ## Description
   <task description from todo>

   ## Status
   <current status from todo: pending/in-progress>

   <!-- source: claude-code -->
   EOF
   )"
   ```

   If a parent issue number was provided, include `Parent: #<number>` in the body.

7. **Add new issues to project board:**

   ```bash
   REPO_NAME=$(gh repo view --json name --jq '.name')
   PROJECT_NUM=$(gh project list --owner @me --format json --jq ".projects[] | select(.title == \"$REPO_NAME\") | .number")
   if [ -n "$PROJECT_NUM" ]; then
     gh project item-add "$PROJECT_NUM" --owner @me --url <issue-url>
   fi
   ```

8. **Report the result:**
   - Issues created (with URLs)
   - Issues skipped (already tracked or completed)
   - Issues added to project board
   - Any errors encountered

## Rules

- This is ONE-WAY sync: local todo → GitHub. Never modify the local todo list.
- Never close or update existing GitHub issues during sync.
- **Skip completed tasks** — only sync pending and in-progress tasks. Creating issues for already-completed work adds noise.
- If sync fails partway through, report what was created and what was not.
- Before creating issues, verify that task descriptions do not contain credentials, API keys, or sensitive details — especially for public repositories.
