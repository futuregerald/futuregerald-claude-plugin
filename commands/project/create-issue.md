Create a GitHub issue for this project.

## Arguments

$ARGUMENTS

The arguments can be:

- A title string: `"fix: login redirect broken after 2FA"`
- A title + body: `"fix: login redirect broken" "After enabling 2FA, the login redirect goes to /dashboard instead of the original URL"`
- Just a description of what to create (you'll generate the title and body)

## Instructions

1. **Check gh availability:**

   ```bash
   gh auth status 2>/dev/null
   ```

   If `gh` is not available or not authenticated, tell the user and stop. Do not block any other work.

2. **Check initialization:**

   ```bash
   gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'
   ```

   If the `claude:initialized` label does NOT exist, tell the user: "GitHub project is not initialized. Run `/project:init` first." and stop.

3. **Parse the arguments.** If the user gave a plain description, generate:
   - A title using conventional commit format (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`)
   - A body with a clear description

4. **Determine labels.** Based on the title prefix, use the standard project labels:

   | Prefix      | Label      |
   | ----------- | ---------- |
   | `feat:`     | `feature`  |
   | `fix:`      | `bug`      |
   | `docs:`     | `docs`     |
   | `refactor:` | `refactor` |
   | `test:`     | `test`     |
   | `chore:`    | `chore`    |

   If the user specified labels explicitly, use those instead.

5. **Create the issue.** Always append `<!-- source: claude-code -->` to the body so the workflow can distinguish agent-created issues from human-created ones:

   ```bash
   gh issue create --title "<title>" --label "<labels>" --body "<body>

   <!-- source: claude-code -->"
   ```

6. **Attempt to add to project board** (non-blocking if it fails):

   ```bash
   REPO_NAME=$(gh repo view --json name --jq '.name')
   PROJECT_NUM=$(gh project list --owner @me --format json --jq ".projects[] | select(.title == \"$REPO_NAME\") | .number")
   if [ -n "$PROJECT_NUM" ]; then
     gh project item-add "$PROJECT_NUM" --owner @me --url <issue-url>
   fi
   ```

7. **Report the result:** Show the issue URL and number.
