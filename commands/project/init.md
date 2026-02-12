Initialize the GitHub project for this repository. Creates the project board and standard labels.

**This command MUST be run before any other `/project:` command that writes to GitHub.**

## Arguments

$ARGUMENTS

No arguments required. Optional:

- `--force` - Re-run initialization even if already initialized (recreates missing labels, skips existing ones)

## Instructions

1. **Check gh availability:**

   ```bash
   gh auth status 2>/dev/null
   ```

   If `gh` is not available or not authenticated, tell the user and stop.

2. **Offer to allow-list `gh` commands.** Check if `Bash(gh:*)` is already in `.claude/settings.local.json` under `permissions.allow`. If NOT present, ask the user:

   > "Would you like to auto-allow all `gh` CLI commands? This adds `Bash(gh:*)` to `.claude/settings.local.json` so you won't be prompted for each GitHub operation."

   If the user agrees:
   - Read `.claude/settings.local.json` (create if it doesn't exist)
   - Parse the JSON. If `permissions.allow` array doesn't exist, create it
   - Add `"Bash(gh:*)"` to the `permissions.allow` array (if not already present)
   - Write the file back with proper JSON formatting
   - If the user declines, skip this step and continue

3. **Check if already initialized:**

   ```bash
   gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'
   ```

   If the `claude:initialized` label exists and `--force` was NOT passed, tell the user:
   "Project is already initialized. Use `/project:init --force` to re-sync labels."
   and stop.

4. **Create standard labels.** For each label below, create it if it doesn't exist. If it exists with wrong color/description, update it.

   ```bash
   # Type labels
   gh label create "feature"    --color "0E8A16" --description "New feature or enhancement"       --force
   gh label create "bug"        --color "D73A4A" --description "Something isn't working"           --force
   gh label create "refactor"   --color "1D76DB" --description "Code improvement, no behavior change" --force
   gh label create "docs"       --color "0075CA" --description "Documentation changes"             --force
   gh label create "test"       --color "BFD4F2" --description "Test additions or changes"         --force
   gh label create "chore"      --color "D4C5F9" --description "Maintenance and housekeeping"      --force

   # Workflow labels
   gh label create "epic"       --color "7057FF" --description "Parent issue grouping related tasks" --force
   gh label create "task"       --color "C2E0C6" --description "Sub-issue of an epic"              --force

   # Priority labels
   gh label create "P0-critical" --color "B60205" --description "Drop everything"                  --force
   gh label create "P1-high"     --color "D93F0B" --description "Do soon"                          --force
   gh label create "P2-medium"   --color "FBCA04" --description "Normal priority"                  --force
   gh label create "P3-low"      --color "0E8A16" --description "Nice to have"                     --force

   # Initialization marker
   gh label create "claude:initialized" --color "EEEEEE" --description "Marker: GitHub workflow initialized" --force
   ```

5. **Remove conflicting default labels** that overlap with our standard set. Only remove these specific defaults if they exist — do NOT remove any other labels:

   ```bash
   # These GitHub defaults overlap with our labels
   gh label delete "enhancement" --yes 2>/dev/null || true
   gh label delete "documentation" --yes 2>/dev/null || true
   ```

6. **Create the project board** (if it doesn't already exist):

   ```bash
   REPO_NAME=$(gh repo view --json name --jq '.name')
   PROJECT_NUM=$(gh project list --owner @me --format json --jq ".projects[] | select(.title == \"$REPO_NAME\") | .number")
   if [ -z "$PROJECT_NUM" ]; then
     gh project create --owner @me --title "$REPO_NAME"
   else
     echo "Project board '$REPO_NAME' already exists (project #$PROJECT_NUM)"
   fi
   ```

7. **Report the result:**

   ```
   ## GitHub Project Initialized

   ### Labels Created
   | Label | Color | Description |
   |-------|-------|-------------|
   | feature | 🟢 | New feature or enhancement |
   | bug | 🔴 | Something isn't working |
   | ... (list all) |

   ### Project Board
   - Created/verified: <project-name>

   ### Next Steps
   - Create issues: `/project:create-issue`
   - Plan a feature: `/project:plan-feature`
   - Check status: `/project:current`
   ```

## Rules

- This command is SAFE to run multiple times — `--force` on `gh label create` updates existing labels without duplicating them.
- Do NOT delete any labels that are not in the "conflicting defaults" list above.
- Do NOT modify any existing issues, PRs, or project board items.
- If any individual label creation fails, continue with the rest and report failures at the end.
