Import a Claude Code configuration archive (created by `/config:export`) onto this machine.

## Arguments

$ARGUMENTS

Required argument:

- Path to the archive: `"~/Desktop/claude-config-export-2026-06-01.tar.gz"`

Optional arguments:

- `"--dry-run"` — show what would change without applying anything (default: apply)
- `"--force"` — overwrite existing files instead of merging (default: merge)

Parse the arguments to set these variables:
- `ARCHIVE_PATH` — the path to the archive file (required)
- `DRY_RUN` — set to `1` if `--dry-run` is present
- `FORCE` — set to `1` if `--force` is present

## Instructions

### Step 1: Validate the archive

```bash
# Check file exists
[ -f "$ARCHIVE_PATH" ] || { echo "File not found: $ARCHIVE_PATH"; exit 1; }

# Extract to temp directory
IMPORT_DIR=$(mktemp -d)/claude-config-import
mkdir -p "$IMPORT_DIR"
tar xzf "$ARCHIVE_PATH" -C "$IMPORT_DIR"

# Find the extracted directory (should be claude-config-export/)
EXPORT_DIR=$(find "$IMPORT_DIR" -maxdepth 1 -type d -name 'claude-config-export' | head -1)
```

**Guard:** If `$EXPORT_DIR` is empty (directory not found), tell the user: "Archive does not contain expected `claude-config-export/` directory. This may not be a valid config export." Clean up `$IMPORT_DIR` and stop.

Verify `manifest.json` exists inside the extracted directory. If not, stop and tell the user this does not appear to be a valid config export. Always clean up `$IMPORT_DIR` on any early exit.

### Step 2: Read the manifest

```bash
cat "$EXPORT_DIR/manifest.json"
```

Extract key fields:
- `sourceHome` — the home directory on the source machine
- `sourceUser` — the username on the source machine
- `sourcePlatform` — the platform (darwin, linux)
- `categories` — what was exported
- `symlinks` — symlink targets and repo info
- `external_binaries` — binaries referenced by MCP configs

Detect the current machine:
```bash
CURRENT_HOME="$HOME"
CURRENT_USER=$(whoami)
CURRENT_PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
```

### Step 3: Present import summary

Show the user what will be imported:

```
## Import Summary

Source: geraldonyango@darwin (/Users/geraldonyango)
Target: newuser@darwin (/Users/newuser)

### Config files
 - settings.json — permissions, model, hooks, 1 MCP server
 - CLAUDE.md — global instructions
 - mcp.json — 0 MCP servers (all excluded during export)

### Directories
 - memory/ — 12 files
 - scripts/ — 1 file
 - bin/ — 1 file
 - channels/ — 1 channel
 - plugins/ — registry + marketplaces + blocklist
 - projects/ — 8 projects
 - context-mode/

### Symlinks to restore
 - skills -> futuregerald-claude-plugin/skills
 - commands -> futuregerald-claude-plugin/commands
 - agents -> futuregerald-claude-plugin/agents

### External binaries (manual setup needed)
 - codebase-memory-mcp — was at ~/.local/bin/codebase-memory-mcp

### Path fixups
 - /Users/geraldonyango -> /Users/newuser (in settings.json, mcp.json, plugin configs)
 - CLAUDE.md and memory/ files are NOT path-fixed (prose content, not config)

### Existing config detected
 - ~/.claude/ exists — will MERGE (new items added, existing items preserved)
   Use --force to overwrite instead.
```

If `--dry-run`, show this summary and stop. Say: "Run `/config:import \"<path>\"` (without --dry-run) to apply." Clean up `$IMPORT_DIR` and stop.

### Step 4: Path fixup

Replace all occurrences of `sourceHome` with `CURRENT_HOME` in config files within the staging directory (before copying to `~/.claude/`).

**Do NOT apply path fixup to:** `CLAUDE.md`, `memory/` files, or any other prose/markdown content. Only fix JSON config files.

Detect the platform-appropriate `sed` in-place syntax:
```bash
if [ "$(uname -s)" = "Darwin" ]; then
  # macOS sed requires '' after -i
  sed_inplace() { sed -i '' "$@"; }
else
  # GNU/Linux sed
  sed_inplace() { sed -i "$@"; }
fi
```

Apply path fixup:
```bash
SOURCE_HOME="<from manifest>"

# Config files
sed_inplace "s|${SOURCE_HOME}|${CURRENT_HOME}|g" "$EXPORT_DIR/settings.json" 2>/dev/null
sed_inplace "s|${SOURCE_HOME}|${CURRENT_HOME}|g" "$EXPORT_DIR/mcp.json" 2>/dev/null
sed_inplace "s|${SOURCE_HOME}|${CURRENT_HOME}|g" "$EXPORT_DIR/.mcp.json" 2>/dev/null

# Plugin configs
sed_inplace "s|${SOURCE_HOME}|${CURRENT_HOME}|g" "$EXPORT_DIR/plugins/installed_plugins.json" 2>/dev/null
sed_inplace "s|${SOURCE_HOME}|${CURRENT_HOME}|g" "$EXPORT_DIR/plugins/known_marketplaces.json" 2>/dev/null
```

### Step 5: Rename project directories

Project directories in `~/.claude/projects/` are path-encoded (e.g., `-Users-geraldonyango-Documents-dev-foo`). Replace the source home path prefix with the target:

```bash
SOURCE_PREFIX=$(echo "$SOURCE_HOME" | tr '/' '-')  # e.g., -Users-geraldonyango
TARGET_PREFIX=$(echo "$CURRENT_HOME" | tr '/' '-') # e.g., -Users-newuser

if [ -d "$EXPORT_DIR/projects" ]; then
  for dir in "$EXPORT_DIR/projects"/*/; do
    dirname=$(basename "$dir")
    newname="${dirname/$SOURCE_PREFIX/$TARGET_PREFIX}"
    if [ "$dirname" != "$newname" ]; then
      mv "$dir" "$EXPORT_DIR/projects/$newname"
    fi
  done
fi
```

Also fix paths inside project-level settings files:
```bash
find "$EXPORT_DIR/projects" -name 'settings*.json' -exec sh -c 'sed_inplace "s|'"${SOURCE_HOME}"'|'"${CURRENT_HOME}"'|g" "$1"' _ {} \;
```

### Step 6: Check for hook path warnings

Read `settings.json` from the staging directory and inspect the `hooks` section. For any hook `command` that references an absolute path NOT under `$CURRENT_HOME`, warn:

```
Warning: Hook references a path outside your home directory:
  SessionStart hook: /usr/local/bin/custom-tool
  This path may not exist on this machine. You may need to update it manually.
```

### Step 7: Apply — config files

```bash
mkdir -p ~/.claude
```

**settings.json — MERGE by default (see Step 12 for detailed merge logic):**

If `~/.claude/settings.json` exists and `FORCE` is not set, perform an additive merge (Step 12). Otherwise, copy the imported file directly.

**keybindings.json:**
```bash
if [ -f "$EXPORT_DIR/keybindings.json" ]; then
  if [ ! -f ~/.claude/keybindings.json ] || [ -n "$FORCE" ]; then
    cp "$EXPORT_DIR/keybindings.json" ~/.claude/keybindings.json
  else
    echo "Skipped: keybindings.json (already exists, use --force to overwrite)"
  fi
fi
```

**CLAUDE.md — interactive choice when both exist:**
```bash
if [ -f "$EXPORT_DIR/CLAUDE.md" ]; then
  if [ -f ~/.claude/CLAUDE.md ] && [ -z "$FORCE" ]; then
    echo "CLAUDE.md already exists on this machine."
    echo "Options:"
    echo "  1. Keep existing"
    echo "  2. Replace with imported"
    echo "  3. Show diff"
    # Ask user which option. If "3", show diff and ask again.
  else
    cp "$EXPORT_DIR/CLAUDE.md" ~/.claude/CLAUDE.md
  fi
fi
```

**MCP configs — merge when both exist:**

For each of `mcp.json` and `.mcp.json`: if the file exists on the target and `FORCE` is not set, merge by adding new MCP servers that don't already exist (same logic as Step 12's `mcpServers` merge). Otherwise, copy directly.

```bash
for mcp_file in mcp.json .mcp.json; do
  if [ -f "$EXPORT_DIR/$mcp_file" ]; then
    if [ ! -f ~/.claude/$mcp_file ] || [ -n "$FORCE" ]; then
      cp "$EXPORT_DIR/$mcp_file" ~/.claude/$mcp_file
    else
      # Merge: read both files, add servers from import that don't exist in target
      # Use jq if available, otherwise parse and merge programmatically
      echo "Merging $mcp_file..."
      # See Step 12 for merge approach
    fi
  fi
done
```

### Step 8: Apply — directories

For each directory, copy contents. In merge mode, skip files that already exist:

```bash
# memory/ — merge (add new, skip existing)
if [ -d "$EXPORT_DIR/memory" ]; then
  mkdir -p ~/.claude/memory
  for f in "$EXPORT_DIR/memory/"*; do
    [ -e "$f" ] || continue
    fname=$(basename "$f")
    if [ ! -e ~/.claude/memory/"$fname" ] || [ -n "$FORCE" ]; then
      cp -R "$f" ~/.claude/memory/
    else
      echo "Skipped: memory/$fname (already exists)"
    fi
  done
fi

# scripts/, bin/, context-mode/ — same merge logic, handles subdirectories
for dir_name in scripts bin context-mode; do
  if [ -d "$EXPORT_DIR/$dir_name" ]; then
    mkdir -p ~/.claude/$dir_name
    # Use cp -Rn for no-clobber recursive copy, fall back to item-by-item
    for f in "$EXPORT_DIR/$dir_name/"*; do
      [ -e "$f" ] || continue
      fname=$(basename "$f")
      if [ ! -e ~/.claude/$dir_name/"$fname" ] || [ -n "$FORCE" ]; then
        cp -R "$f" ~/.claude/$dir_name/
      else
        echo "Skipped: $dir_name/$fname (already exists)"
      fi
    done
  fi
done

# channels/ — copy tree, skip existing
if [ -d "$EXPORT_DIR/channels" ]; then
  mkdir -p ~/.claude/channels
  for f in "$EXPORT_DIR/channels/"*; do
    [ -e "$f" ] || continue
    fname=$(basename "$f")
    if [ ! -e ~/.claude/channels/"$fname" ] || [ -n "$FORCE" ]; then
      cp -R "$f" ~/.claude/channels/
    else
      echo "Skipped: channels/$fname (already exists)"
    fi
  done
fi

# plugins/ — registry files
if [ -d "$EXPORT_DIR/plugins" ]; then
  mkdir -p ~/.claude/plugins
  for f in installed_plugins.json known_marketplaces.json blocklist.json; do
    if [ -f "$EXPORT_DIR/plugins/$f" ]; then
      if [ ! -f ~/.claude/plugins/"$f" ] || [ -n "$FORCE" ]; then
        cp "$EXPORT_DIR/plugins/$f" ~/.claude/plugins/
      else
        echo "Skipped: plugins/$f (already exists)"
      fi
    fi
  done
fi

# projects/ — copy tree, skip existing files
if [ -d "$EXPORT_DIR/projects" ]; then
  for proj_dir in "$EXPORT_DIR/projects"/*/; do
    [ -d "$proj_dir" ] || continue
    proj_name=$(basename "$proj_dir")
    mkdir -p ~/.claude/projects/"$proj_name"
    for f in "$proj_dir"*; do
      [ -f "$f" ] || continue
      fname=$(basename "$f")
      if [ ! -f ~/.claude/projects/"$proj_name"/"$fname" ] || [ -n "$FORCE" ]; then
        cp "$f" ~/.claude/projects/"$proj_name"/
      fi
    done
    # memory subdirectory
    if [ -d "$proj_dir/memory" ]; then
      mkdir -p ~/.claude/projects/"$proj_name"/memory
      for f in "$proj_dir/memory/"*; do
        [ -f "$f" ] || continue
        fname=$(basename "$f")
        if [ ! -f ~/.claude/projects/"$proj_name"/memory/"$fname" ] || [ -n "$FORCE" ]; then
          cp "$f" ~/.claude/projects/"$proj_name"/memory/
        fi
      done
    fi
  done
fi
```

### Step 9: Restore symlinks

For each symlink in the manifest:

```bash
# Example: skills -> futuregerald-claude-plugin/skills
REPO_NAME="futuregerald-claude-plugin"
SUBPATH="skills"
LINK_NAME="skills"
ORIGINAL_TARGET="/Users/geraldonyango/Documents/dev/futuregerald-claude-plugin/skills"

# Try the original path with home dir fixed
FIXED_TARGET="${ORIGINAL_TARGET/$SOURCE_HOME/$CURRENT_HOME}"

if [ -d "$FIXED_TARGET" ]; then
  ln -sf "$FIXED_TARGET" ~/.claude/$LINK_NAME
  echo "Symlink restored: $LINK_NAME -> $FIXED_TARGET"
else
  # Search common dev directories
  FOUND=""
  for search_dir in ~/Documents/dev ~/dev ~/src ~/code ~/projects ~/repos ~/workspace; do
    candidate="$search_dir/$REPO_NAME/$SUBPATH"
    if [ -d "$candidate" ]; then
      FOUND="$candidate"
      break
    fi
  done

  if [ -n "$FOUND" ]; then
    ln -sf "$FOUND" ~/.claude/$LINK_NAME
    echo "Symlink restored: $LINK_NAME -> $FOUND"
  else
    echo "Warning: Could not find $REPO_NAME/$SUBPATH"
    echo "  Clone the repo and run: ln -sf <path>/$SUBPATH ~/.claude/$LINK_NAME"
  fi
fi
```

### Step 10: External binaries warning

For each external binary in the manifest, check if it exists at the path-fixed location:

```bash
FIXED_PATH="${ORIGINAL_PATH/$SOURCE_HOME/$CURRENT_HOME}"
if [ ! -f "$FIXED_PATH" ]; then
  echo "Missing: $BINARY_NAME"
  echo "  Was at: $ORIGINAL_PATH"
  echo "  Referenced in: $REFERENCED_IN"
  echo "  Install it or update the path in ~/.claude/settings.json"
fi
```

### Step 11: Plugin sync

If `installed_plugins.json` was imported, prompt:

```
Plugin registry imported. To install the plugins, run:
  claude plugins sync
```

Do NOT run it automatically — plugin installation may require network access and user confirmation.

### Step 12: Settings merge (detailed)

When merging `settings.json` (existing target + imported source), use these rules. This step is referenced by Step 7 when a merge is needed.

Read both JSON files. The merge is additive — the target is never reduced, only extended.

**If `jq` is available**, use it for precise JSON manipulation:

```bash
# Merge permissions.allow (union, deduplicated)
jq -s '
  .[0] as $target | .[1] as $source |
  $target * {
    permissions: {
      allow: (($target.permissions.allow // []) + ($source.permissions.allow // []) | unique)
    },
    mcpServers: (($target.mcpServers // {}) + (($source.mcpServers // {}) | to_entries | map(select(.key | in($target.mcpServers // {}) | not)) | from_entries)),
    enabledPlugins: (($target.enabledPlugins // {}) + (($source.enabledPlugins // {}) | to_entries | map(select(.key | in($target.enabledPlugins // {}) | not)) | from_entries)),
    hooks: (($target.hooks // {}) + (($source.hooks // {}) | to_entries | map(select(.key | in($target.hooks // {}) | not)) | from_entries))
  }
' ~/.claude/settings.json "$EXPORT_DIR/settings.json" > ~/.claude/settings.json.tmp \
  && mv ~/.claude/settings.json.tmp ~/.claude/settings.json
```

**If `jq` is not available**, Claude should:
1. Read both JSON files
2. Parse them as JSON objects
3. Apply these merge rules programmatically:
   - **permissions.allow**: Union of both arrays, deduplicated
   - **hooks**: For each hook event — add from source if not in target, skip if already present
   - **mcpServers**: For each server — add from source if not in target, skip if already present
   - **enabledPlugins**: Union of both objects, target values win on collision
   - **model, effortLevel, other scalars**: Keep target values
4. Write the merged result

Show the user what was added:
```
Merged settings.json:
  +2 permissions: mcp__new_server__*, Bash(docker *)
  +1 MCP server: new-server
  +1 hook: PreToolUse
  model: kept existing "opus" (imported had "sonnet")
```

**For MCP config files** (`mcp.json`, `.mcp.json`), the same add-only logic applies but is simpler — only the `mcpServers` object needs merging.

### Step 13: Report

```
Import complete.

Applied:
 - settings.json (merged: +2 permissions, +1 MCP server)
 - CLAUDE.md (replaced)
 - memory/ (added 12 files)
 - scripts/ (added 1 file)
 - bin/ (added 1 file)
 - channels/ (added 1 channel)
 - plugins/ (registry imported)
 - projects/ (added 8 projects)

Symlinks:
 - skills -> ~/Documents/dev/futuregerald-claude-plugin/skills
 - commands -> ~/Documents/dev/futuregerald-claude-plugin/commands
 - agents -> ~/Documents/dev/futuregerald-claude-plugin/agents

Action needed:
 - Run `claude plugins sync` to install plugins
 - Install codebase-memory-mcp binary (was at ~/.local/bin/codebase-memory-mcp)
```

### Step 14: Cleanup

```bash
rm -rf "$IMPORT_DIR"
```

## Rules

- **`--dry-run` shows changes without applying.** Default is to apply.
- **Default is merge, not overwrite.** Existing config is preserved. Use `--force` to overwrite.
- **Never delete existing config.** Import only adds or replaces — it does not remove items from the target.
- **Path fixup is mandatory** for JSON config files. Never path-fix prose files (`CLAUDE.md`, `memory/`).
- **Never auto-run `claude plugins sync`.** Just tell the user to run it.
- **Symlink restoration is best-effort.** If the target repo is not found, warn and continue.
- **Handle platform differences.** Define `sed_inplace` function at the start and use it for all in-place edits.
- If `~/.claude/` does not exist, create it and apply everything directly (no merge needed).
- The import must be idempotent — running it twice with the same archive should not break anything.
- **Always clean up `$IMPORT_DIR`** on success, failure, or early exit. Never leave temp directories behind.
