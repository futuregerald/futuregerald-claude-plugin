Plan a feature by creating an epic issue and breaking it into task sub-issues on GitHub.

## Arguments

$ARGUMENTS

The arguments can be:

- A feature description: `"User profile page with avatar upload and bio editing"`
- A path to a PRD or design document: `"docs/plans/2026-02-12-user-profiles.md"`
- A feature description + PRD path: `"Add dark mode" "docs/plans/dark-mode-design.md"`

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

3. **Gather feature context:**
   - If a PRD/design doc path was provided, read it and extract requirements, acceptance criteria, and task breakdown
   - If only a description was provided, analyze the codebase to understand:
     - What already exists that this feature touches
     - What new files/components are needed
     - Database changes required
     - Frontend and backend work required

4. **Create the epic issue:**

   ```bash
   gh issue create \
     --title "epic: <feature-title>" \
     --label "epic" \
     --body "$(cat <<'EOF'
   ## Overview
   <1-2 sentence summary>

   ## Requirements
   <If from a PRD, include the key requirements. If from description, generate them.>
   - <requirement 1>
   - <requirement 2>
   - ...

   ## Tasks
   Sub-issues will be created for each task below.

   - [ ] Task 1
   - [ ] Task 2
   - ...

   ## Acceptance Criteria
   - [ ] <criterion 1>
   - [ ] <criterion 2>

   ## Source
   <If from a PRD: "PRD: `<file-path>`">
   <If from description: "Described in Claude Code session">

   <!-- source: claude-code -->
   EOF
   )"
   ```

5. **Break into task sub-issues.** For each task identified:

   ```bash
   gh issue create \
     --title "task: <specific-task>" \
     --label "task" \
     --body "$(cat <<'EOF'
   Parent: #<epic-issue-number>

   ## Description
   <what this task does>

   ## Implementation Notes
   - <relevant file paths>
   - <approach>

   ## Acceptance Criteria
   - [ ] <criterion>

   <!-- source: claude-code -->
   EOF
   )"
   ```

6. **Update the epic issue body** to include links to all created sub-issues:

   ```bash
   gh issue edit <epic-number> --body "<updated body with issue links>"
   ```

7. **Add all issues to the project board:**

   ```bash
   REPO_NAME=$(gh repo view --json name --jq '.name')
   PROJECT_NUM=$(gh project list --owner @me --format json --jq ".projects[] | select(.title == \"$REPO_NAME\") | .number")
   if [ -n "$PROJECT_NUM" ]; then
     gh project item-add "$PROJECT_NUM" --owner @me --url <epic-issue-url>
     gh project item-add "$PROJECT_NUM" --owner @me --url <task-1-url>
     gh project item-add "$PROJECT_NUM" --owner @me --url <task-2-url>
     # ... for each task
   fi
   ```

8. **Report the result:** Show:
   - Epic issue URL and number
   - List of all task sub-issue URLs and numbers
   - Whether they were added to the project board
   - If sourced from a PRD, note which document was used

## Guidelines for Task Breakdown

- **Simple features** (1-2 areas of code): 2-4 tasks
- **Medium features** (frontend + backend + DB): 4-6 tasks
- **Complex features** (multiple systems, new patterns): 6-10 tasks

Each task should be:

- Independently implementable (can be a single PR)
- Small enough to complete in one session
- Clear about what "done" means

Do NOT create tasks for: planning, code review, testing (those are part of the Development Lifecycle for each task).

## Security

Before creating issues, verify that the body content does not contain credentials, API keys, internal URLs, or sensitive details — especially for public repositories. PRD files may contain internal context that should not be published.
