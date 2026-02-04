---
name: init-claude-md
description: Initialize a CLAUDE.md file for the current project, customized for the detected framework/language
allowed-tools: Read, Write, Glob, Bash, Grep
argument-hint: [project-name]
---

# Initialize CLAUDE.md Command

Create a customized CLAUDE.md file for the current project based on detected framework and language.

## Instructions

1. **Detect the project type** by checking for:
   - `package.json` → Check for frameworks (AdonisJS, React, Svelte, Next.js, Express, etc.)
   - `go.mod` → Go project
   - `requirements.txt` / `pyproject.toml` → Python project
   - `Cargo.toml` → Rust project
   - `Gemfile` → Ruby/Rails project
   - `composer.json` → PHP/Laravel project

2. **Read the base template** from the plugin:
   ```
   {{PLUGIN_ROOT}}/templates/CLAUDE-BASE.md
   ```
   (Use the plugin's installation directory, typically `~/.claude/` or the `--plugin-dir` path)

3. **Read the appropriate language snippet** from:
   ```
   {{PLUGIN_ROOT}}/templates/languages/{language}.md
   ```

4. **Combine them** by:
   - Using the base template structure
   - Inserting language-specific content at `<!-- LANGUAGE_SPECIFIC -->` marker
   - Replacing `{{PROJECT_NAME}}` with the project name (from arg or directory name)
   - Replacing `{{FRAMEWORK}}` with detected framework
   - Replacing `{{TEST_COMMAND}}` with appropriate test command
   - Replacing `{{TYPECHECK_COMMAND}}` with appropriate typecheck command

5. **Write the result** to `./CLAUDE.md` in the current directory

6. **Report what was created** including detected framework and customizations applied

## Detection Priority

| File | Framework Detection |
|------|---------------------|
| `package.json` with `@adonisjs/*` | AdonisJS |
| `package.json` with `svelte` | Svelte |
| `package.json` with `react` | React |
| `package.json` with `next` | Next.js |
| `package.json` with `express` | Express |
| `package.json` (generic) | Node.js |
| `go.mod` | Go |
| `requirements.txt` or `pyproject.toml` | Python |
| `Cargo.toml` | Rust |
| `Gemfile` with `rails` | Rails |
| `Gemfile` (generic) | Ruby |
| `composer.json` with `laravel` | Laravel |
| `composer.json` (generic) | PHP |

**For multi-framework projects** (e.g., AdonisJS + Svelte), detect the primary backend framework and combine relevant frontend snippets.

## If CLAUDE.md Already Exists

Ask the user if they want to:
1. Overwrite it
2. Create CLAUDE.md.new for comparison
3. Cancel

## Example Output

```
Detected: AdonisJS v6 with Svelte frontend
Created: ./CLAUDE.md

Customizations applied:
- AdonisJS testing commands (node ace test)
- Lucid ORM patterns (UUID models, serialization)
- Svelte 5 runes patterns
- Inertia.js integration rules
```
