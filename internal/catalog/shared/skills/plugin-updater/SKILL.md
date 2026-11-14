---
name: plugin-updater
description: Check for and apply live skill and workflow updates from futuregerald/futuregerald-claude-plugin main branch using skill-installer, gh CLI, or GitHub MCP. Use when the user asks "are there updates to the plugin?", "check skill updates", "update my skills", or invokes /plugin-update.
author: Gerald Onyango
tags: [maintenance, updates, cli]
---

# Plugin & Skill Updater (`/plugin-update`)

Keep installed skills, slash commands, subagents, and harness configuration up to date with `futuregerald/futuregerald-claude-plugin` (`main` branch) without losing local customizations (`*.local.md`, `references/config.md`) or skipping user confirmation.

## What good output looks like

- **For the user:** a concise, plain-English summary of what changed since their installed commit (new skills, updated skills, or a newer `skill-installer` binary release), followed by a single confirmation prompt before touching any files.
- **The default to prevent:** dumping raw commit SHAs or diffs into chat, overwriting files before asking, or failing when `gh` CLI is not installed even though the GitHub MCP server is connected.

## Decision Tree: Which Update Path to Run

1. **Check for `skill-installer` and authenticated `gh` CLI first (Path A — Preferred):**
   ```bash
   command -v skill-installer >/dev/null 2>&1 && command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1
   ```
   - If exit code is `0`, follow **Path A (`skill-installer` + `gh` CLI)** below.
2. **Fallback to GitHub MCP when `gh` CLI or `skill-installer` is unavailable (Path B):**
   - If `github` MCP tools (`list_commits`, `get_file_contents`) are available in your tool list, follow **Path B (GitHub MCP Direct Sync)** below.
3. **If neither `gh` CLI nor GitHub MCP is authenticated:**
   - Tell the user which option is quickest:
     - Run `gh auth login` and then `skill-installer update` in their terminal, or
     - Enable and sign in to the GitHub MCP server in their editor/harness settings and re-run `/plugin-update`.

---

## Path A: `skill-installer` + `gh` CLI

1. **Check for updates without modifying any files:**
   ```bash
   skill-installer update --check --json
   ```
   *(If `skill-installer` is older than v4.0 and lacks `update --check --json`, inspect `~/.config/skill-installer/state.json` or `%APPDATA%\skill-installer\state.json` and run `gh api repos/futuregerald/futuregerald-claude-plugin/commits/main` directly.)*

2. **Summarize what changed in plain English:**
   - If already at the latest `main` commit and latest binary version, report that everything is up to date and stop.
   - Otherwise, list:
     - **Updated or added skills** in the user's active track/installation
     - **Recent commit highlights** (1 line per change, plain English)
     - **Newer `skill-installer` binary release**, if available

3. **Ask before applying:**
   - Ask the user once:
     1. Whether to apply the skill/config updates now
     2. Whether to keep the automatic safety backup enabled (default: **Yes**, stored in `~/.config/skill-installer/backups/` or `%APPDATA%\skill-installer\backups\`, restorable anytime via `skill-installer restore`)

4. **Apply confirmed updates:**
   - With safety backup (default):
     ```bash
     skill-installer update --apply --backup -y
     ```
   - If the user explicitly asked to skip backup:
     ```bash
     skill-installer update --apply --no-backup -y
     ```
   - If a newer CLI binary release was also confirmed:
     ```bash
     skill-installer update --self-upgrade -y
     ```
   - Report the updated skills, new commit SHA, and backup snapshot ID.

---

## Path B: GitHub MCP Direct Sync (No `gh` CLI Required)

1. **Read current installation state:**
   - Check `~/.config/skill-installer/state.json` (macOS/Linux, or `$XDG_CONFIG_HOME/skill-installer/state.json`) or `%APPDATA%\skill-installer\state.json` (Windows).
   - Identify the active harness (`claude`, `antigravity`, `zed`, or `opencode`), scope (`global` or `project`), installed `track`, `skills` list, and `commit_sha`.
   - If `state.json` does not exist yet (e.g. a legacy v3.x install), inspect the active harness skill directory (`.claude/skills`, `~/.claude/skills`, `.agents/skills`, `~/.gemini/config/skills`, or `.opencode/skills`) to list currently installed skill folder names.

2. **Query `futuregerald/futuregerald-claude-plugin` (`main`) via GitHub MCP:**
   - Call `list_commits` on `owner: "futuregerald"`, `repo: "futuregerald-claude-plugin"`, `sha: "main"`, `perPage: 15`.
   - Compare the latest commit SHA on `main` against `commit_sha` from `state.json`.
   - Fetch `internal/catalog/tracks.yaml` via `get_file_contents` if needed to see new skills added to the user's track.

3. **Present changelog and ask confirmation:**
   - Summarize the commits and affected skills clearly.
   - Ask the user to confirm before writing any files, and ask if they want a backup copy of modified files saved first.

4. **Apply updates safely:**
   - Never overwrite `*.local.md` or `references/config.md`.
   - For each updated skill in the user's installed set, fetch its `SKILL.md` (and any changed `references/` or `scripts/` files, excluding `test_*.py`) from `internal/catalog/shared/skills/<skill>/` (or `internal/catalog/<harness>/skills/<skill>/` when a harness override exists on `antigravity`).
   - Adapt harness paths if installing onto `antigravity`, `zed`, or `opencode` (replace `.claude/skills` and `CLAUDE.md` references with the active harness's native paths and strip `model:` / `effort:` frontmatter keys on non-Claude harnesses).
   - Update `commit_sha` and `updated_at` in `state.json` and report the completed update.
