# futuregerald-claude-plugin

A curated collection of skills, agents, and commands for AI coding tools. Includes debugging protocols, TDD workflows, code review, multi-language project scaffolding, and a CLI installer that works across Claude Code, GitHub Copilot, Cursor, OpenCode, and VS Code.

This repository works in two ways:

- **As a Claude Code plugin** -- installed via symlink or `--plugin-dir`
- **As a standalone CLI tool** (`skill-installer`) -- installs skills, agents, and commands for any supported AI coding framework

## Attribution

This is a collection of Claude Code skills and agents from various sources:

- **Most skills were created by others** in the Claude Code community
- **Some were modified** by me to fit my workflow or fix issues
- **Some were created** by me

**All credit goes to the original skill creators.** I'm sharing this collection to make it easier to set up Claude Code on new machines. If you're a skill author and would like attribution added or your skill removed, please open an issue.

## Installation

### As Claude Code Plugin

#### Option 1: Clone and Symlink (Recommended)

```bash
# Clone to your preferred location
git clone https://github.com/futuregerald/futuregerald-claude-plugin.git ~/futuregerald-claude-plugin

# Symlink to Claude's global directory
ln -s ~/futuregerald-claude-plugin/skills ~/.claude/skills
ln -s ~/futuregerald-claude-plugin/agents ~/.claude/agents
ln -s ~/futuregerald-claude-plugin/commands ~/.claude/commands
```

#### Option 2: Plugin Directory Flag

```bash
# Clone anywhere
git clone https://github.com/futuregerald/futuregerald-claude-plugin.git

# Run Claude with plugin directory
claude --plugin-dir ./futuregerald-claude-plugin
```

#### Option 3: Direct Clone to Claude Directory

```bash
git clone https://github.com/futuregerald/futuregerald-claude-plugin.git ~/.claude/plugins/futuregerald
claude --plugin-dir ~/.claude/plugins/futuregerald
```

### Via CLI (`skill-installer`)

The CLI installs skills, agents, and commands for any supported framework -- not just Claude Code.

#### Download from Releases

Download the latest binary for your platform from [GitHub Releases](https://github.com/futuregerald/futuregerald-claude-plugin/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (M1/M2/M3) | `skill-installer_*_darwin_arm64.tar.gz` |
| macOS | Intel | `skill-installer_*_darwin_amd64.tar.gz` |
| Linux | x64 | `skill-installer_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `skill-installer_*_linux_arm64.tar.gz` |
| Windows | x64 | `skill-installer_*_windows_amd64.zip` |
| Windows | ARM64 | `skill-installer_*_windows_arm64.zip` |

**macOS/Linux:**
```bash
# Download and extract (example for macOS Apple Silicon)
curl -LO https://github.com/futuregerald/futuregerald-claude-plugin/releases/latest/download/skill-installer_3.0.0_darwin_arm64.tar.gz
tar -xzf skill-installer_3.0.0_darwin_arm64.tar.gz
chmod +x skill-installer

# Move to PATH (optional)
sudo mv skill-installer /usr/local/bin/

# Run
skill-installer
```

**Windows (PowerShell):**
```powershell
# Download and extract
Invoke-WebRequest -Uri "https://github.com/futuregerald/futuregerald-claude-plugin/releases/latest/download/skill-installer_3.0.0_windows_amd64.zip" -OutFile "skill-installer.zip"
Expand-Archive -Path "skill-installer.zip" -DestinationPath "."

# Run
.\skill-installer.exe
```

#### Install with Go

If you have Go installed:

```bash
go install github.com/futuregerald/futuregerald-claude-plugin@latest
```

Then run (note: the binary is named after the module):

```bash
futuregerald-claude-plugin
```

Or alias it for convenience:

```bash
alias skill-installer=futuregerald-claude-plugin
```

## CLI Usage

Running `skill-installer` with no arguments starts an interactive installer that walks you through selecting a target framework, skills, and options.

```bash
# Interactive mode
skill-installer

# Install for a specific target (skip prompts)
skill-installer --target claude --yes

# Dry run (preview what would be installed)
skill-installer --dry-run

# List all available skills
skill-installer list

# List skills filtered by tag
skill-installer list --tag workflow

# Install globally (Claude Code and GitHub Copilot)
skill-installer --global

# Skip agents or commands
skill-installer --skip-agents --skip-commands

# Create a new skill
skill-installer init my-skill --desc "My skill" --tag custom

# Install from a custom source (local path or remote repo)
skill-installer --from /path/to/skills
skill-installer --from https://github.com/user/repo
```

## Supported Frameworks

| Target | Skills Path | Agents Path | Config File | Global Support |
|--------|------------|-------------|-------------|----------------|
| Claude Code | `.claude/skills/` | `.claude/agents/` | `CLAUDE.md` | Yes |
| GitHub Copilot | `.github/skills/` | `.github/*.agent.md` | `.github/copilot-instructions.md` | Yes |
| Cursor | `.cursor/skills/` | `.cursor/agents/` | `.cursorrules` | No |
| OpenCode | `.opencode/skills/` | `.opencode/agents/` | - | No |
| VS Code | `.vscode/claude/skills/` | `.vscode/claude/agents/` | - | No |

## Contents

### Commands (8 total)

| Command | Description |
|---------|-------------|
| `/init-claude-md` | Generate a customized CLAUDE.md for your project based on detected framework/language |
| `/project:init` | Create project board and standard labels (run first) |
| `/project:create-issue` | Create a GitHub issue with labels |
| `/project:plan-feature` | Create epic from feature description or PRD |
| `/project:sync-tasks` | Sync todo list to GitHub issues |
| `/project:current` | Show project status overview (read-only) |
| `/project:inbox` | Check for human-created issues (read-only) |
| `/project:cleanup` | Find and remove stale worktrees (dry-run default) |

### Skills (33 total)

**Core Workflow:**
| Skill | Description |
|-------|-------------|
| `using-superpowers` | Skill discovery and usage patterns |
| `systematic-debugging` | 4-phase debugging protocol (no guessing) |
| `test-driven-development` | TDD workflow: RED -> GREEN -> REFACTOR |
| `writing-plans` | Implementation planning before coding |
| `executing-plans` | Plan execution with review checkpoints |
| `brainstorming` | Creative exploration before implementation |
| `verification-before-completion` | Evidence before assertions — verify before claiming done |

**Code Quality:**
| Skill | Description |
|-------|-------------|
| `code-simplifier` | Code simplification analysis |
| `requesting-code-review` | Code review requests |
| `receiving-code-review` | Processing review feedback |
| `error-handling-patterns` | Error handling across languages |

**Development Workflow:**
| Skill | Description |
|-------|-------------|
| `dispatching-parallel-agents` | Parallel task execution |
| `subagent-driven-development` | Parallel implementation |
| `using-git-worktrees` | Git worktree isolation |
| `finishing-a-development-branch` | Branch completion workflow |

**Framework-Specific:**
| Skill | Description |
|-------|-------------|
| `adonisjs-best-practices` | AdonisJS v6 patterns and conventions |
| `better-auth-best-practices` | Better Auth integration |
| `javascript-testing-patterns` | Jest/Vitest/Japa testing patterns |
| `sqlite-database-expert` | SQLite/libSQL/Turso expertise |
| `turso-best-practices` | Turso database patterns |

**Design & Frontend:**
| Skill | Description |
|-------|-------------|
| `frontend-design` | Production-grade frontend interfaces |
| `ui-design` | Refactoring UI methodology |
| `design-principles` | Linear/Notion/Stripe-inspired design |

**Other:**
| Skill | Description |
|-------|-------------|
| `api-design-principles` | REST and GraphQL API design |
| `architecture-decision-records` | ADR documentation |
| `code-search` | Fast codebase search |
| `skill-creator` | Creating new skills |
| `writing-skills` | Skill authoring |
| `copywriting` | Marketing copy writing |
| `marketing-psychology` | Mental models for marketing |
| `programmatic-seo` | Template-based SEO pages at scale |
| `agent-browser` | Browser automation with Playwright |
| `baoyu-article-illustrator` | Article illustration generation |
| `create-auth-skill` | Auth layer creation |

### Agents (6 total)

| Agent | Description |
|-------|-------------|
| `code-quality-reviewer` | Reviews code for quality issues |
| `code-simplifier` | Analyzes code for simplification with Staff Engineer review |
| `codebase-searcher` | Searches and explores codebases |
| `debugger` | Systematic bug investigation |
| `implementer` | Implements features from plans |
| `spec-reviewer` | Reviews specifications and plans |

### Language Templates

Used by `/init-claude-md` to generate framework-specific CLAUDE.md files:

| Template | Frameworks |
|----------|------------|
| `adonisjs.md` | AdonisJS v6 |
| `go.md` | Go projects |
| `nodejs.md` | Node.js |
| `php.md` | PHP/Laravel |
| `python.md` | Python |
| `react.md` | React with hooks |
| `ruby.md` | Ruby/Rails |
| `rust.md` | Rust projects |
| `svelte.md` | Svelte 5 with runes |

## Usage (Claude Code Plugin)

After installation as a plugin, skills are available with the namespace prefix:

```
/futuregerald-claude-plugin:systematic-debugging
/futuregerald-claude-plugin:brainstorming
/futuregerald-claude-plugin:test-driven-development
```

Or if symlinked to `~/.claude/skills`, use the `superpowers:` prefix:

```
/superpowers:systematic-debugging
/superpowers:brainstorming
```

### Generate Project-Specific CLAUDE.md

```
/init-claude-md my-project-name
```

This detects your framework (AdonisJS, React, Svelte, Go, etc.) and generates a customized CLAUDE.md with:
- Mandatory workflow sections (TDD, debugging, pre-push checklist)
- Language-specific conventions and patterns
- Appropriate test/typecheck commands

## Configuration

The CLI reads an optional `.skill-installer.yaml` file from the current directory:

```yaml
target: claude
tags: [workflow, testing]
languages: [javascript, python]
skip_claude_md: false
from: ""
```

## Building from Source

```bash
git clone https://github.com/futuregerald/futuregerald-claude-plugin.git
cd futuregerald-claude-plugin
make build    # builds ./skill-installer
make test     # runs tests
make install  # installs to /usr/local/bin
```

## Directory Structure

```
futuregerald-claude-plugin/
├── .claude-plugin/
│   └── plugin.json
├── internal/
│   ├── config/config.go
│   └── installer/
│       ├── installer.go
│       └── installer_test.go
├── agents/
├── commands/
│   ├── init-claude-md/
│   └── project/
│       ├── init.md
│       ├── create-issue.md
│       ├── plan-feature.md
│       ├── sync-tasks.md
│       ├── current.md
│       ├── inbox.md
│       └── cleanup.md
├── skills/
├── templates/
├── main.go
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Updating

```bash
cd ~/futuregerald-claude-plugin  # or wherever you cloned it
git pull
```

If using symlinks, changes are immediately available. No restart needed.

If using the CLI binary, download the latest release or re-run `go install` to update.
