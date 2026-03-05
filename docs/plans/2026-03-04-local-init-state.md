# Store Project Initialization State Locally Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Store project initialization state in `.claude/project.json` (committed to repo) so initialization checks are fast, offline-capable, and don't depend solely on a GitHub label.

**Architecture:** During `project:init`, write a `.claude/project.json` file with initialization metadata and ensure `.gitignore` allows it. All commands that check initialization read this file first (instant, no API call), falling back to the label check for backwards compatibility.

**Tech Stack:** Markdown skill files (no compiled code changes needed)

---

### Task 1: Update `project:init` to write `.claude/project.json`

**Files:**
- Modify: `commands/project/init.md:63-71` (check if already initialized)
- Modify: `commands/project/init.md:106-116` (after project board creation, before report)

**Step 1: Update the "Check if already initialized" step (step 3) to check the local file first**

Replace lines 63-71 of `commands/project/init.md` with:

```markdown
3. **Check if already initialized:**

   First, check for a local project file (fast, no API call):

   ```bash
   cat .claude/project.json 2>/dev/null | grep -q '"initialized": true'
   ```

   If the local file exists with `initialized: true` and `--force` was NOT passed, tell the user:
   "Project is already initialized. Use `/project:init --force` to re-sync labels."
   and stop.

   If no local file, fall back to the label check for backwards compatibility:

   ```bash
   gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'
   ```

   If the `claude:initialized` label exists and `--force` was NOT passed, tell the user:
   "Project is already initialized. Use `/project:init --force` to re-sync labels."
   and stop.
```

**Step 2: Add a new step 7 (before report) to write the local project file and update .gitignore**

Insert a new step between the current step 6 (project board) and step 7 (report). This becomes the new step 7, and the report becomes step 8.

Add this after the project board step (after line 116):

```markdown
7. **Write local initialization state:**

   Ensure `.gitignore` allows `.claude/project.json`:

   ```bash
   if [ -f .gitignore ] && grep -q '\.claude/\*' .gitignore; then
     if ! grep -q '!\.claude/project\.json' .gitignore; then
       # Add the exception right after the .claude/* line
       sed -i '' '/^\.claude\/\*/a\
!.claude/project.json' .gitignore
     fi
   fi
   ```

   Write the project state file:

   ```bash
   mkdir -p .claude
   cat > .claude/project.json << 'PROJ_EOF'
   {
     "initialized": true,
     "initializedAt": "TIMESTAMP",
     "version": 1
   }
   PROJ_EOF
   ```

   Replace `TIMESTAMP` with the current ISO 8601 timestamp (e.g., `date -u +"%Y-%m-%dT%H:%M:%SZ"`).

   Stage and commit the changes:

   ```bash
   git add .gitignore .claude/project.json
   git commit -m "chore: add project initialization state

   Store initialization state in .claude/project.json for fast,
   offline-capable init checks.

   Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
   ```

   > **Note:** This is one of the rare cases where the init command creates a commit directly on the current branch. The file must be committed to be useful across sessions and for other developers.
```

**Step 3: Update the report step (now step 8) to mention the local file**

In the report section, add a line under "### Project Board":

```markdown
   ### Local State
   - Written: `.claude/project.json`
   - Gitignore updated: `.claude/project.json` is now tracked
```

**Step 4: Verify the full init.md reads correctly**

Read the file end-to-end and confirm no step numbering conflicts or duplicate instructions.

**Step 5: Commit**

```bash
git add commands/project/init.md
git commit -m "feat: write .claude/project.json during project init

Stores initialization state locally for fast, offline-capable checks.
Updates .gitignore to allow .claude/project.json.

Refs #<issue>"
```

---

### Task 2: Update initialization checks in `create-issue`, `plan-feature`, `sync-tasks`

**Files:**
- Modify: `commands/project/create-issue.md:23-29`
- Modify: `commands/project/plan-feature.md:23-29`
- Modify: `commands/project/sync-tasks.md:23-29`

All three files have an identical "Check initialization" step. Replace each with the same updated version.

**Step 1: Update `create-issue.md` initialization check**

Replace lines 23-29 with:

```markdown
2. **Check initialization:**

   First, check for a local project file (fast, no API call):

   ```bash
   cat .claude/project.json 2>/dev/null | grep -q '"initialized": true'
   ```

   If not found locally, fall back to the label check:

   ```bash
   gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'
   ```

   If NEITHER check passes, tell the user: "GitHub project is not initialized. Run `/project:init` first." and stop.
```

**Step 2: Apply the same change to `plan-feature.md`**

Replace the identical block (lines 23-29) with the same text from Step 1.

**Step 3: Apply the same change to `sync-tasks.md`**

Replace the identical block (lines 23-29) with the same text from Step 1.

**Step 4: Commit**

```bash
git add commands/project/create-issue.md commands/project/plan-feature.md commands/project/sync-tasks.md
git commit -m "feat: check .claude/project.json before label fallback

All project commands now check the local file first for fast,
offline-capable initialization verification.

Refs #<issue>"
```

---

### Task 3: Update CLAUDE-BASE.md template and CLAUDE.md

**Files:**
- Modify: `templates/CLAUDE-BASE.md:113-114`
- Modify: `CLAUDE.md:110-111`

**Step 1: Update the prerequisites check in `CLAUDE-BASE.md`**

Replace line 114:
```
- Check: `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
```

With:
```
- Check: `cat .claude/project.json 2>/dev/null | grep -q '"initialized": true'` (fast, local) or fall back to `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
```

**Step 2: Apply the same change to `CLAUDE.md`**

Replace the identical line (line 111) with the same text.

**Step 3: Commit**

```bash
git add templates/CLAUDE-BASE.md CLAUDE.md
git commit -m "docs: update init check instructions to prefer local file

Refs #<issue>"
```

---

### Task 4: Update README.md

**Files:**
- Modify: `README.md:508-514`

**Step 1: Update the initialization description**

Replace lines 508-514:

```markdown
This command:

1. Creates a standard set of labels for issue categorization (feature, bug, refactor, docs, test, chore, epic, task, P0-P3 priorities)
2. Removes conflicting GitHub default labels (e.g., `enhancement` and `documentation`)
3. Creates a GitHub Projects board named after the repository
4. Sets a `claude:initialized` marker label so the plugin knows the project is ready
```

With:

```markdown
This command:

1. Creates a standard set of labels for issue categorization (feature, bug, refactor, docs, test, chore, epic, task, P0-P3 priorities)
2. Removes conflicting GitHub default labels (e.g., `enhancement` and `documentation`)
3. Creates a GitHub Projects board named after the repository
4. Sets a `claude:initialized` marker label on GitHub
5. Writes `.claude/project.json` to the repo for fast, offline-capable initialization checks
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: document .claude/project.json in README

Refs #<issue>"
```

---

### Task 5: Update the learning-journey-adonis CLAUDE.md (consumer repo)

**Files:**
- Modify: `/Users/geraldonyango/Documents/dev/learning-journey-monorepo/learning-journey-adonis/CLAUDE.md` (the line referencing the init check)

**Step 1: Find and update the init check line**

The CLAUDE.md in the consumer repo has:
```
- Check: `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
```

Replace with:
```
- Check: `cat .claude/project.json 2>/dev/null | grep -q '"initialized": true'` or fall back to `gh label list --json name --jq '.[].name' | grep -q '^claude:initialized$'`
```

**Step 2: Commit (in the consumer repo)**

```bash
git add CLAUDE.md
git commit -m "docs: update init check to prefer local .claude/project.json

Refs futuregerald/futuregerald-claude-plugin#<issue>"
```

---

## Summary of Changes

| File | Change |
|------|--------|
| `commands/project/init.md` | Check local file first; write `.claude/project.json` + update `.gitignore` |
| `commands/project/create-issue.md` | Check local file first, fall back to label |
| `commands/project/plan-feature.md` | Check local file first, fall back to label |
| `commands/project/sync-tasks.md` | Check local file first, fall back to label |
| `templates/CLAUDE-BASE.md` | Update prerequisites check instruction |
| `CLAUDE.md` | Update prerequisites check instruction |
| `README.md` | Document `.claude/project.json` in init description |
| Consumer repo `CLAUDE.md` | Update init check instruction |
