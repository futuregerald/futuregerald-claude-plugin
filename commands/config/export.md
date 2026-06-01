Export the Claude Code configuration from `~/.claude/` into a portable archive that can be imported on another machine.

## Arguments

$ARGUMENTS

Optional arguments:

- A path for the output archive: `"~/Desktop"` (default: current directory)
- `"--include-history"` — also include `history.jsonl` (excluded by default)

## Instructions

### Step 1: Scan `~/.claude/`

Read the top-level contents of `~/.claude/`:

```bash
ls -la ~/.claude/
```

Identify what exists. Not all items will be present on every machine.

### Step 2: Detect symlinks

Check for symlinks in `~/.claude/`:

```bash
find ~/.claude -maxdepth 1 -type l -exec sh -c 'echo "$(basename "$1") -> $(readlink "$1")"' _ {} \;
```

For each symlink, record the target path. If the target is inside a git repo, detect the repo name:

```bash
# For each symlink target:
git -C "<target_path>" rev-parse --show-toplevel 2>/dev/null
```

Record in the manifest:
- Symlink name (e.g., `skills`)
- Full target path
- Repository name (basename of git root)
- Subpath within repo (target path minus git root)

### Step 3: Detect MCP servers

Read MCP server definitions from up to three files:

```bash
cat ~/.claude/settings.json 2>/dev/null   # check .mcpServers key
cat ~/.claude/mcp.json 2>/dev/null        # check .mcpServers key
cat ~/.claude/.mcp.json 2>/dev/null       # check .mcpServers key
```

Build a combined, deduplicated list of all server names. Note which file(s) each server appears in.

### Step 4: Detect external binaries

Scan MCP server definitions for `command` fields that reference absolute paths (not `npx`, `node`, or other system commands). Record each as an external binary with its path and which config file references it.

### Step 5: Present categories to user

Show the user what will be exported with sane defaults:

```
## Export Summary

### Included (default)
 1. [x] settings.json — permissions, model, hooks, enabled plugins
 2. [x] keybindings.json — keyboard customizations
 3. [x] CLAUDE.md — global instructions
 4. [x] MCP servers — mcp.json, .mcp.json (N servers found)
 5. [x] memory/ — N files
 6. [x] scripts/ — N files
 7. [x] bin/ — N files
 8. [x] channels/ — channel configs (.env files excluded for security)
 9. [x] plugins — registry, marketplaces, blocklist
10. [x] projects/ — per-project settings & memory (N projects)
11. [x] context-mode/

### Excluded (default)
 - history.jsonl, file-history/, sessions/, session-env/
 - paste-cache/, image-cache/, shell-snapshots/
 - telemetry/, cache/, debug/, statsig/, stats-cache.json
 - tasks/, todos/, plans/, downloads/, ide/, backups/
 - plugins/cache/ (re-downloaded on sync)
 - *.backup dirs, patch.log, .DS_Store

### Symlinks detected
 - skills -> futuregerald-claude-plugin/skills
 - commands -> futuregerald-claude-plugin/commands
 - agents -> futuregerald-claude-plugin/agents

### External binaries detected
 - codebase-memory-mcp at ~/.local/bin/codebase-memory-mcp (referenced in settings.json)

Toggle any category by number, or type "ok" to proceed.
```

Let the user toggle items on/off. When they say "ok" or equivalent, proceed.

Only show items that actually exist on the machine. Skip missing items silently (e.g., if `keybindings.json` does not exist, do not list it).

### Step 6: MCP server selection

If MCP servers are included, present the server list:

```
## MCP Servers

Which servers should be EXCLUDED from the export?
(Default: include all. Enter numbers to exclude, or "none" to include all.)

 1. codebase-memory-mcp (settings.json)
 2. atlassian (mcp.json, .mcp.json)
 3. pendo (.mcp.json)
```

Apply exclusions. When a server appears in multiple files, remove it from all files where it appears.

### Step 7: Secret scanning

Before archiving, scan included files for secrets:

**Channels:** Exclude all `.env` files and `access.json` from channel directories automatically. Warn the user:
```
Security: Excluded .env and access.json from channels/ (may contain tokens).
```

**Scripts:** Read each script file. Flag any that contain patterns matching:
- `API_KEY=`, `TOKEN=`, `SECRET=`, `PASSWORD=`, `PRIVATE_KEY=`
- `Bearer ` followed by a long string
- Long base64-encoded strings (40+ chars of `[A-Za-z0-9+/=]`)

For each flagged script, show the concerning line(s) and ask:
```
Script scripts/my-script.sh contains what may be a secret:
  Line 5: TOKEN="sk-abc123..."

Include this script? (y/n)
```

Skip scripts the user declines.

**General:** After staging, strip any remaining sensitive files that may exist in any directory:

```bash
find "$EXPORT_DIR" \( -name '.env' -o -name '.secret' -o -name 'credentials.json' \) -delete
```

### Step 8: Prepare the staging directory

```bash
EXPORT_DIR=$(mktemp -d)/claude-config-export
mkdir -p "$EXPORT_DIR"
```

Copy included items into the staging directory:

```bash
# Config files (only if they exist)
# NOTE: settings.json, mcp.json, .mcp.json are written as filtered copies in Step 9.
# Do NOT copy the originals here — Step 9 handles them.
cp ~/.claude/keybindings.json "$EXPORT_DIR/" 2>/dev/null
cp ~/.claude/CLAUDE.md "$EXPORT_DIR/" 2>/dev/null

# Directories (only included ones)
cp -R ~/.claude/memory "$EXPORT_DIR/" 2>/dev/null
cp -R ~/.claude/scripts "$EXPORT_DIR/" 2>/dev/null
cp -R ~/.claude/bin "$EXPORT_DIR/" 2>/dev/null
cp -R ~/.claude/context-mode "$EXPORT_DIR/" 2>/dev/null

# Channels (exclude .env and access.json)
if [ -d ~/.claude/channels ]; then
  cp -R ~/.claude/channels/ "$EXPORT_DIR/channels/"
  find "$EXPORT_DIR/channels" -name '.env' -delete 2>/dev/null
  find "$EXPORT_DIR/channels" -name 'access.json' -delete 2>/dev/null
fi

# Plugins (registry files only, not cache)
mkdir -p "$EXPORT_DIR/plugins"
cp ~/.claude/plugins/installed_plugins.json "$EXPORT_DIR/plugins/" 2>/dev/null
cp ~/.claude/plugins/known_marketplaces.json "$EXPORT_DIR/plugins/" 2>/dev/null
cp ~/.claude/plugins/blocklist.json "$EXPORT_DIR/plugins/" 2>/dev/null

# Projects (settings.json, CLAUDE.md, memory/ only — not sessions, tasks, etc.)
if [ -d ~/.claude/projects ]; then
  mkdir -p "$EXPORT_DIR/projects"
  for proj_dir in ~/.claude/projects/*/; do
    proj_name=$(basename "$proj_dir")
    # Skip the parent catch-all directory marker
    [ "$proj_name" = "." ] && continue
    mkdir -p "$EXPORT_DIR/projects/$proj_name"
    cp "$proj_dir/settings.json" "$EXPORT_DIR/projects/$proj_name/" 2>/dev/null
    cp "$proj_dir/settings.local.json" "$EXPORT_DIR/projects/$proj_name/" 2>/dev/null
    cp "$proj_dir/CLAUDE.md" "$EXPORT_DIR/projects/$proj_name/" 2>/dev/null
    [ -d "$proj_dir/memory" ] && cp -R "$proj_dir/memory" "$EXPORT_DIR/projects/$proj_name/" 2>/dev/null
  done
fi

# Optional: history
# Only if --include-history was passed
cp ~/.claude/history.jsonl "$EXPORT_DIR/" 2>/dev/null
```

### Step 9: Write JSON config files (filtered)

Write filtered copies of all JSON config files to the staging directory. These are NOT raw copies — they have excluded MCP servers removed.

For each file (`settings.json`, `mcp.json`, `.mcp.json`):
1. Read the original from `~/.claude/`
2. Parse the JSON
3. Remove any MCP servers from `.mcpServers` that the user excluded in Step 6
4. Write the filtered result to `$EXPORT_DIR/`

If `jq` is available:
```bash
# Example: remove servers "atlassian" and "pendo" from mcp.json
jq 'del(.mcpServers.atlassian, .mcpServers.pendo)' ~/.claude/mcp.json > "$EXPORT_DIR/mcp.json"
```

If `jq` is not available, read the JSON content, parse and manipulate it programmatically (Claude can do this inline), and write the filtered result.

**All three files must be handled:** `settings.json`, `mcp.json`, and `.mcp.json`. If a file does not exist, skip it. If a file has no MCP servers to filter, copy it as-is.

### Step 10: Generate manifest

Write `manifest.json` to the staging directory with:

```json
{
  "version": 1,
  "pluginVersion": "<read from .claude-plugin/plugin.json if in the plugin repo, otherwise 'unknown'>",
  "exportedAt": "<ISO 8601 timestamp>",
  "sourceUser": "<current username>",
  "sourceHome": "<$HOME>",
  "sourcePlatform": "<uname -s | tr '[:upper:]' '[:lower:]'>",
  "categories": { ... },
  "symlinks": { ... },
  "external_binaries": [ ... ]
}
```

The `categories` object records what was included/excluded and any notes (e.g., excluded MCP servers).

### Step 11: Create the archive

```bash
OUTPUT_DIR="${1:-.}"  # user-specified or current directory
DATE=$(date +%Y-%m-%d)
ARCHIVE_NAME="claude-config-export-${DATE}.tar.gz"

tar czf "${OUTPUT_DIR}/${ARCHIVE_NAME}" -C "$(dirname "$EXPORT_DIR")" "$(basename "$EXPORT_DIR")"

# Cleanup staging
rm -rf "$(dirname "$EXPORT_DIR")"
```

### Step 12: Report

```
Export complete: ./claude-config-export-2026-06-01.tar.gz

Contents:
 - settings.json (3 MCP servers, 2 excluded)
 - CLAUDE.md
 - memory/ (12 files)
 - scripts/ (1 file)
 - bin/ (1 file)
 - channels/ (1 channel, .env excluded)
 - plugins (registry + 2 marketplaces)
 - projects/ (8 projects)
 - 3 symlinks recorded (skills, commands, agents -> futuregerald-claude-plugin)
 - 1 external binary noted (codebase-memory-mcp)

To import on another machine:
  /config:import "path/to/claude-config-export-2026-06-01.tar.gz"
```

## Rules

- **Never include `.env`, `.secret`, or `credentials.json` files** from any directory. These may contain secrets.
- **Never include `access.json`** from channel directories.
- **Always scan scripts for secrets** before including them. When in doubt, ask the user.
- **Never include the `plugins/cache/` directory.** Plugins are re-downloaded on sync.
- **Default is inclusive** — include everything useful, exclude ephemeral/machine-specific data.
- **If `~/.claude/` is empty or has nothing exportable**, inform the user and stop.
- If a category directory does not exist, skip it silently (do not error).
- The archive must be self-contained — the import command should work with just the archive file and no other context.
- Always run the general secret file cleanup (`find ... -delete`) after staging, before archiving.
