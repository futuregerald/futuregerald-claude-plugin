# Plan: Config Export/Import Commands

**Date:** 2026-06-01
**Branch:** `feat/export-import-config`
**Goal:** Add `/config:export` and `/config:import` commands to the plugin so users can portably migrate their Claude Code setup between machines.

---

## Problem

Migrating a Claude Code installation requires manually identifying config files, MCP servers, plugins, skills, symlinks, scripts, and memory — then manually copying, fixing paths, and re-linking on the target machine. There's no built-in way to do this.

## Solution

Two new commands under `commands/config/`:

- **`/config:export`** — Scans `~/.claude/`, presents categorized components with sane defaults, lets user exclude items (e.g., specific MCP servers), and generates a portable `.tar.gz` archive with a manifest.
- **`/config:import`** — Reads the archive, shows what will be applied, fixes hardcoded paths for the new machine, restores symlinks, and triggers plugin sync.

---

## Design

### Archive Format

Output filename: `claude-config-export-<YYYY-MM-DD>.tar.gz`

A `.tar.gz` containing:
```
claude-config-export/
  manifest.json              # metadata, categories, source machine info
  settings.json              # filtered (excluded MCP servers removed)
  keybindings.json           # keyboard customizations (if exists)
  mcp.json                   # filtered (if exists)
  .mcp.json                  # filtered (if exists)
  CLAUDE.md                  # global instructions
  memory/                    # persistent memories
  scripts/                   # custom scripts (secret-scanned)
  bin/                       # custom binaries
  channels/                  # channel configs (.env files EXCLUDED)
  plugins/
    installed_plugins.json   # plugin registry (not cache)
    known_marketplaces.json  # third-party plugin sources
    blocklist.json           # blocked plugins
  projects/                  # per-project settings (settings.json, CLAUDE.md, memory/ only)
  context-mode/              # context mode config
```

### Manifest Schema (`manifest.json`)

```json
{
  "version": 1,
  "pluginVersion": "3.8.0",
  "exportedAt": "2026-06-01T12:00:00Z",
  "sourceUser": "geraldonyango",
  "sourceHome": "/Users/geraldonyango",
  "sourcePlatform": "darwin",
  "categories": {
    "settings": { "included": true, "files": ["settings.json"] },
    "keybindings": { "included": true, "files": ["keybindings.json"], "note": "if exists" },
    "mcp_servers": { "included": true, "files": ["mcp.json", ".mcp.json"], "excluded_servers": ["atlassian", "pendo"] },
    "global_instructions": { "included": true, "files": ["CLAUDE.md"] },
    "memory": { "included": true, "path": "memory/" },
    "scripts": { "included": true, "path": "scripts/", "note": "secret-scanned" },
    "bin": { "included": true, "path": "bin/" },
    "channels": { "included": true, "path": "channels/", "note": ".env files excluded" },
    "plugins": { "included": true, "files": ["plugins/installed_plugins.json", "plugins/known_marketplaces.json", "plugins/blocklist.json"] },
    "projects": { "included": true, "path": "projects/", "note": "settings and memory only" },
    "context_mode": { "included": true, "path": "context-mode/" }
  },
  "symlinks": {
    "skills": { "target": "/Users/geraldonyango/Documents/dev/futuregerald-claude-plugin/skills", "repo": "futuregerald-claude-plugin", "subpath": "skills" },
    "commands": { "target": "/Users/geraldonyango/Documents/dev/futuregerald-claude-plugin/commands", "repo": "futuregerald-claude-plugin", "subpath": "commands" },
    "agents": { "target": "/Users/geraldonyango/Documents/dev/futuregerald-claude-plugin/agents", "repo": "futuregerald-claude-plugin", "subpath": "agents" }
  },
  "external_binaries": [
    { "name": "codebase-memory-mcp", "path": "/Users/geraldonyango/.local/bin/codebase-memory-mcp", "referenced_in": "settings.json" }
  ]
}
```

### Category Defaults

| Category | Default | Why |
|----------|---------|-----|
| `settings.json` | INCLUDE | Core config — permissions, model, hooks, plugins |
| `keybindings.json` | INCLUDE | Keyboard customizations (portable, no machine-specific content) |
| `mcp.json` / `.mcp.json` | INCLUDE (with server filtering) | MCP server definitions |
| `CLAUDE.md` | INCLUDE | Global instructions |
| `memory/` | INCLUDE | Cross-session memory |
| `scripts/` | INCLUDE (secret-scanned) | Custom scripts — scan for secrets before including |
| `bin/` | INCLUDE | Custom binaries |
| `channels/` | INCLUDE (`.env` excluded) | Channel configs — `.env` files stripped to avoid leaking tokens |
| `plugins/installed_plugins.json` | INCLUDE | Plugin registry for re-install |
| `plugins/known_marketplaces.json` | INCLUDE | Third-party plugin source registrations |
| `plugins/blocklist.json` | INCLUDE | Blocked plugin list |
| `projects/` (settings + memory only) | INCLUDE | Per-project customization |
| `context-mode/` | INCLUDE | Context mode preferences |
| `history.jsonl` | EXCLUDE | Machine-specific conversation history |
| `file-history/` | EXCLUDE | Machine-specific |
| `sessions/`, `session-env/` | EXCLUDE | Machine-specific |
| `paste-cache/`, `image-cache/` | EXCLUDE | Ephemeral cache |
| `shell-snapshots/` | EXCLUDE | Machine-specific |
| `telemetry/` | EXCLUDE | Machine-specific |
| `cache/`, `debug/` | EXCLUDE | Ephemeral |
| `tasks/`, `todos/`, `plans/` | EXCLUDE | Session-scoped |
| `statsig/`, `stats-cache.json` | EXCLUDE | Machine-specific |
| `downloads/`, `ide/`, `backups/` | EXCLUDE | Machine-specific |
| `plugins/cache/` | EXCLUDE | Re-downloaded on sync |
| `*.backup` dirs | EXCLUDE | Migration artifacts |
| `patch.log` | EXCLUDE | Debug output |
| `.DS_Store` | EXCLUDE | macOS metadata |

### Security: Secret Scanning

Before including files in the archive, scan for secrets:

1. **Channels**: Exclude all `.env` files and `access.json` from channel directories. These contain bot tokens and access control data.
2. **Scripts**: Read each script file and warn if it contains patterns like `API_KEY=`, `TOKEN=`, `SECRET=`, `PASSWORD=`, `Bearer `, or long base64 strings. Present flagged scripts and let user confirm or skip each one.
3. **General**: Never include files named `.env`, `.secret`, `credentials.json`, or similar patterns.

### MCP Server Filtering

During export, Claude:
1. Reads all MCP server definitions from `settings.json.mcpServers`, `mcp.json.mcpServers`, and `.mcp.json.mcpServers`
2. Presents a deduplicated list of all servers found, noting which file(s) each appears in
3. User selects which to **exclude** (default: include all)
4. Exclusions are applied to whichever file(s) contain the server
5. Filtered copies are written to the archive

### Symlink Handling

During export:
- Detect symlinks in `~/.claude/` (`skills`, `commands`, `agents`, any others)
- Record in manifest: target path, inferred repo name (basename of git root), subpath within repo

During import:
- If the target repo exists at the same relative location, re-create symlink
- If not found, search common dev directories (`~/Documents/dev/`, `~/dev/`, `~/src/`, `~/code/`, `~/projects/`)
- If still not found, warn and skip (user can fix manually)

### Path Fixup on Import

The import command:
1. Reads `manifest.json` to get `sourceHome`
2. Detects current `$HOME`
3. Replaces all occurrences of `sourceHome` with current `$HOME` in:
   - `settings.json`
   - `mcp.json` / `.mcp.json`
   - `plugins/installed_plugins.json`
   - `plugins/known_marketplaces.json`
4. Does NOT replace paths inside `CLAUDE.md` or `memory/` (those are prose, not config)
5. Warns about any hooks in `settings.json` that reference paths not under `$HOME` (cannot be auto-fixed)
6. Renames project directories: decode path-encoded names (e.g., `-Users-geraldonyango-Documents-dev-foo` becomes `-Users-newuser-Documents-dev-foo`) by replacing the source user's path prefix with the target user's

### Settings Merge Strategy (Import)

If `~/.claude/` already exists on the target:
- **settings.json**: Merge — import adds permissions, hooks, MCP servers, enabled plugins that don't already exist. Existing entries are NOT overwritten. User is shown a diff of what will change.
- **CLAUDE.md**: If target has one, show both and ask which to keep (or merge manually)
- **memory/**: Merge — new files are added, existing files with same name are skipped (warn)
- **Everything else**: Copy if not present, skip with warning if present

### Import: `--dry-run` Mode

When invoked with `--dry-run`, the import command shows everything that would change without applying anything. This follows the same pattern as `/project:cleanup`.

---

## Files to Create

| File | Purpose |
|------|---------|
| `commands/config/export.md` | Export command |
| `commands/config/import.md` | Import command |

No Go code, no tests (these are markdown command instructions for Claude to follow).

---

## Task Breakdown

### Task 1: Create `commands/config/export.md`
- Full command instructions for scanning, categorizing, filtering, archiving
- Must handle: category presentation, MCP filtering, symlink detection, manifest generation, archive creation

### Task 2: Create `commands/config/import.md`
- Full command instructions for reading archive, presenting contents, path fixup, symlink restoration, merge strategy, plugin sync
- Must handle: existing config detection, merge vs overwrite, path replacement, missing repo warning
