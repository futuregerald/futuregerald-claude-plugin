# futuregerald-claude-plugin

A Claude Code plugin that adds a curated library of AI coding skills and agents, a cross-platform CLI installer, and an optional GitHub workflow integration with project management.

## Table of Contents

- [What This Plugin Does](#what-this-plugin-does)
  - [CLAUDE.md — Your Project's AI Configuration](#claudemd--your-projects-ai-configuration)
  - [Skills — Teaching Claude How to Work](#skills--teaching-claude-how-to-work)
  - [Agents — Specialized Sub-Agents](#agents--specialized-sub-agents)
  - [Development Workflow — A Structured Lifecycle](#development-workflow--a-structured-lifecycle)
- [Installation](#installation)
- [Skills and Agents Reference](#skills-and-agents-reference)
- [CLI Installer](#cli-installer)
- [Configuration](#configuration)
- [Building from Source](#building-from-source)
- [GitHub Workflow (Optional — Beta)](#github-workflow-optional--beta)
  - [Slash Commands](#slash-commands)
  - [Workflow Lifecycle](#workflow-lifecycle)
  - [Review Modes](#review-modes)
  - [Issue Tracking](#issue-tracking)
  - [Git Worktrees](#git-worktrees)
- [Attribution](#attribution)

---

## What This Plugin Does

This repository works in two ways:

- **As a Claude Code plugin** -- Installed via symlink or `--plugin-dir`. Provides slash commands, skills, and agents directly inside Claude Code sessions.
- **As a standalone CLI tool** (`skill-installer`) -- Installs skills, agents, and commands for Claude Code, GitHub Copilot, Cursor, OpenCode, and VS Code.

### CLAUDE.md — Your Project's AI Configuration

A `CLAUDE.md` file sits at the root of your project and tells Claude Code how to work in that codebase. It defines:

- **Project name and description** — what the project is
- **Key directories** — where important code lives (`src/`, `lib/`, `tests/`, etc.)
- **Commands** — how to test (`go test ./...`), typecheck (`npx tsc --noEmit`), and build
- **Development lifecycle** — the phases Claude follows: plan, implement with TDD, test, simplify, review, commit
- **Language-specific rules** — coding conventions, style guides, and best practices for the detected framework

The plugin auto-detects your project type (Go, Node.js/React/Next.js/AdonisJS/Svelte/Express, Rust, Python, Ruby, PHP) and generates a CLAUDE.md with real values filled in — no manual editing of `{{placeholders}}` required.

Generate one with:

```
/init-claude-md my-project-name
```

Or via the CLI:

```bash
skill-installer --mode config-only --target claude --yes
```

### Skills — Teaching Claude How to Work

Skills are markdown-based instruction sets that give Claude specialized knowledge and workflows. When invoked, Claude follows the skill's process exactly. The plugin includes 34 skills covering:

- **Test-driven development** — RED/GREEN/REFACTOR cycle
- **Systematic debugging** — 4-phase protocol: root cause analysis, pattern matching, hypothesis testing, implementation
- **Code review** — requesting and receiving reviews with technical rigor
- **Planning** — writing and executing implementation plans with review checkpoints
- **Brainstorming** — creative exploration before jumping to code
- **Framework expertise** — AdonisJS, React, SQLite/Turso, and more

Skills are invoked by name in Claude Code:

```
/superpowers:systematic-debugging
/superpowers:test-driven-development
/superpowers:brainstorming
```

### Agents — Specialized Sub-Agents

Agents are dispatched via the `Task` tool to handle focused work with fresh context. The plugin includes 7 agents: code quality reviewer, code simplifier (with Staff Engineer review), codebase searcher, debugger, implementer, spec reviewer, and SQL performance reviewer.

### Development Workflow — A Structured Lifecycle

The plugin defines a 10-phase development lifecycle in the generated CLAUDE.md:

```
 1. RECEIVE     Understand task, create todo list
 2. PLAN        Write implementation plan
 3. REVIEW      Staff Engineer sub-agent reviews the plan
 4. IMPLEMENT   Write code following TDD
 5. TEST        Run tests and type checking
 6. SIMPLIFY    Code-simplifier agent analyzes for improvements
 7. CODE REVIEW Code-reviewer sub-agent reviews changes
 8. SQL REVIEW  Staff Engineer audits queries for performance, security, defensive coding
 9. COMMIT      Commit to feature branch
10. PUSH + PR   Push and create PR (if gh available)
11. VERIFY CI   Check CI passes; auto-merge when green
```

Each phase has a gate — the workflow doesn't advance until the gate passes. This works entirely locally. For teams using GitHub, an optional beta workflow adds issue tracking, worktrees, and autonomous PR review (see [GitHub Workflow](#github-workflow-optional--beta) below).

---

## Installation

### As a Claude Code Plugin

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

### Via CLI Binary

Download the latest binary for your platform from [GitHub Releases](https://github.com/futuregerald/futuregerald-claude-plugin/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (M1/M2/M3) | `skill-installer_*_darwin_arm64.tar.gz` |
| macOS | Intel | `skill-installer_*_darwin_amd64.tar.gz` |
| Linux | x64 | `skill-installer_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `skill-installer_*_linux_arm64.tar.gz` |
| Windows | x64 | `skill-installer_*_windows_amd64.zip` |
| Windows | ARM64 | `skill-installer_*_windows_arm64.zip` |

**macOS / Linux:**

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
Invoke-WebRequest -Uri "https://github.com/futuregerald/futuregerald-claude-plugin/releases/latest/download/skill-installer_3.0.0_windows_amd64.zip" -OutFile "skill-installer.zip"
Expand-Archive -Path "skill-installer.zip" -DestinationPath "."
.\skill-installer.exe
```

### Via Go Install

```bash
go install github.com/futuregerald/futuregerald-claude-plugin@latest
```

The binary is named after the module (`futuregerald-claude-plugin`). Alias it for convenience:

```bash
alias skill-installer=futuregerald-claude-plugin
```

### Usage (Plugin Namespace)

After installation as a plugin, skills are available with the namespace prefix:

```
/futuregerald-claude-plugin:systematic-debugging
/futuregerald-claude-plugin:brainstorming
/futuregerald-claude-plugin:test-driven-development
```

If symlinked to `~/.claude/skills`, use the `superpowers:` prefix:

```
/superpowers:systematic-debugging
/superpowers:brainstorming
```

---

## Skills and Agents Reference

### Skills (34)

**Core Workflow:**

| Skill | Description |
|-------|-------------|
| `using-superpowers` | Skill discovery and usage patterns |
| `systematic-debugging` | 4-phase debugging protocol: root cause, pattern analysis, hypothesis, implementation |
| `writing-plans` | Implementation planning before coding |
| `executing-plans` | Plan execution with review checkpoints |
| `brainstorming` | Creative exploration before implementation |
| `verification-before-completion` | Evidence-based verification before claiming done |

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
| `subagent-driven-development` | Parallel implementation with sub-agents |
| `using-git-worktrees` | Git worktree isolation |
| `finishing-a-development-branch` | Branch completion workflow |

**Framework-Specific:**

| Skill | Description |
|-------|-------------|
| `adonisjs-best-practices` | AdonisJS v6 patterns and conventions |
| `better-auth-best-practices` | Better Auth integration |
| `javascript-testing-patterns` | Jest, Vitest, and Japa testing patterns |
| `sqlite-database-expert` | SQLite, libSQL, and Turso expertise |
| `turso-best-practices` | Turso database patterns |
| `sql-optimization-patterns` | SQL query optimization, indexing, EXPLAIN analysis, N+1 elimination |

**Design and Frontend:**

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

### Agents (7)

Agents are specialized sub-agents dispatched via the Task tool. They run with fresh context and no knowledge of the parent conversation.

| Agent | Description |
|-------|-------------|
| `code-quality-reviewer` | Reviews code for quality issues |
| `code-simplifier` | Analyzes code for simplification, with Staff Engineer review |
| `codebase-searcher` | Searches and explores codebases |
| `debugger` | Systematic bug investigation |
| `implementer` | Implements features from plans |
| `spec-reviewer` | Reviews specifications and plans |
| `sql-reviewer` | Ruthless SQL performance, security, and defensive coding audit |

### Language Templates

Used by `/init-claude-md` and `skill-installer --mode config-only` to generate framework-specific CLAUDE.md files:

| Template | Frameworks |
|----------|------------|
| `adonisjs.md` | AdonisJS v6 |
| `go.md` | Go projects |
| `nodejs.md` | Node.js |
| `php.md` | PHP / Laravel |
| `python.md` | Python |
| `react.md` | React with hooks |
| `ruby.md` | Ruby / Rails |
| `rust.md` | Rust projects |
| `svelte.md` | Svelte 5 with runes |

---

## CLI Installer

The `skill-installer` binary installs skills, agents, and commands for any supported AI coding framework -- not just Claude Code.

### Supported Frameworks

| Target | Skills Path | Agents Path | Config File | Global Support |
|--------|-------------|-------------|-------------|----------------|
| Claude Code | `.claude/skills/` | `.claude/agents/` | `CLAUDE.md` | Yes |
| GitHub Copilot | `.github/skills/` | `.github/*.agent.md` | `.github/copilot-instructions.md` | Yes |
| Cursor | `.cursor/skills/` | `.cursor/agents/` | `.cursorrules` | No |
| OpenCode | `.opencode/skills/` | `.opencode/agents/` | -- | No |
| VS Code | `.vscode/claude/skills/` | `.vscode/claude/agents/` | -- | No |

### CLI Usage

```bash
# Interactive mode (walks through framework selection and options)
skill-installer

# Install for a specific target non-interactively
skill-installer --target claude --yes

# Dry run (preview what would be installed)
skill-installer --dry-run

# List all available skills
skill-installer list

# List skills filtered by tag
skill-installer list --tag workflow

# Install globally (user-level, available to all projects)
skill-installer --global

# Skip agents or commands
skill-installer --skip-agents --skip-commands

# Create a new skill from template
skill-installer init my-skill --desc "My skill" --tag custom

# Install from a custom source
skill-installer --from /path/to/skills
skill-installer --from https://github.com/user/repo

# Choose installation mode
skill-installer --mode config-only   # Generate CLAUDE.md only (for existing global installs)
skill-installer --mode agents-only   # Install agents only
skill-installer --mode full          # Full installation (default)

# Config-only for a specific target
skill-installer --mode config-only --target cursor --yes

# Agents-only, globally
skill-installer --mode agents-only --global --target claude --yes
```

---

## Configuration

The CLI reads an optional `.skill-installer.yaml` file from the current directory:

```yaml
target: claude
mode: full  # full, config-only, or agents-only
tags: [workflow, testing]
languages: [javascript, python]
skip_claude_md: false
from: ""
```

---

## Building from Source

```bash
git clone https://github.com/futuregerald/futuregerald-claude-plugin.git
cd futuregerald-claude-plugin
make build    # builds ./skill-installer
make test     # runs tests
make install  # installs to /usr/local/bin
```

## Updating

```bash
cd ~/futuregerald-claude-plugin  # or wherever you cloned it
git pull
```

If using symlinks, changes are immediately available. No restart needed.

If using the CLI binary, download the latest release or re-run `go install` to update.

---

## Directory Structure

```
futuregerald-claude-plugin/
├── .claude-plugin/
│   └── plugin.json              # Plugin metadata (name, version, description)
├── commands/
│   ├── init-claude-md/
│   │   └── COMMAND.md           # /init-claude-md command
│   └── project/
│       ├── init.md              # /project:init
│       ├── create-issue.md      # /project:create-issue
│       ├── plan-feature.md      # /project:plan-feature
│       ├── sync-tasks.md        # /project:sync-tasks
│       ├── current.md           # /project:current
│       ├── inbox.md             # /project:inbox
│       └── cleanup.md           # /project:cleanup
├── skills/                      # 33 skill directories, each with SKILL.md
├── agents/                      # 6 agent markdown files
├── templates/
│   ├── CLAUDE-BASE.md           # Base template for generated CLAUDE.md files
│   └── languages/               # Framework-specific template snippets
├── internal/                    # Go packages for the CLI installer
├── main.go                      # CLI entry point
├── detect.go                    # Project detection heuristics
├── go.mod
├── go.sum
├── Makefile
└── .goreleaser.yaml             # Release automation config
```

---

## GitHub Workflow (Optional — Beta)

> **Beta:** The GitHub workflow integration is currently in beta and is highly opinionated. It is **not required** to take advantage of the plugin -- the skills, agents, and CLI installer all work independently without it. While the workflow has many benefits (structured development lifecycle, automated issue tracking, worktree isolation, and CI verification), read thoroughly on how it works before initializing it. By default, it will not work without the [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated locally, and you must run `/project:init` to enable it.

**Prerequisites:**

1. The plugin must be [installed](#installation) first
2. The [GitHub CLI (`gh`)](https://cli.github.com/) must be installed and authenticated (`gh auth login`)
3. You must run `/project:init` in a Claude Code session to initialize the project

### Setup

Once the plugin is installed and `gh` is authenticated, initialize the GitHub workflow for your repository:

1. **Initialize the project** (creates labels and project board):

   ```
   /project:init
   ```

   This is required once per repository before using any command that writes to GitHub.

2. **Start working.** Create issues, plan features, and let the workflow manage the rest:

   ```
   /project:create-issue "feat: add user authentication"
   /project:plan-feature "User authentication with email/password and OAuth"
   /project:current
   ```

### Slash Commands

The plugin provides eight slash commands for managing GitHub-integrated project workflows.

| Command | Description | Writes to GitHub? |
|---------|-------------|-------------------|
| `/project:init` | Create project board and standard labels. **Must run first.** | Yes |
| `/project:create-issue` | Create a GitHub issue with conventional-commit labels | Yes |
| `/project:plan-feature` | Create an epic and break it into task sub-issues | Yes |
| `/project:sync-tasks` | Sync local todo list items to GitHub issues (one-way) | Yes |
| `/project:current` | Show project status: open issues, PRs, worktrees, board | No |
| `/project:inbox` | Check for issues created by people (not by the workflow) | No |
| `/project:cleanup` | Find and remove stale worktrees (dry-run by default) | No |
| `/init-claude-md` | Generate a framework-specific CLAUDE.md for the current project | No |

Commands that write to GitHub require initialization. Read-only commands (`/project:current`, `/project:inbox`, `/project:cleanup`) work at any time.

### Workflow Lifecycle

The full development lifecycle managed by the plugin follows this sequence:

```
 1. RECEIVE TASK     Create a GitHub issue, create a git worktree + feature branch
 2. PLAN             Write an implementation plan (writing-plans skill)
 3. REVIEW PLAN      Staff Engineer sub-agent reviews the plan (must approve)
 4. IMPLEMENT        Write code following TDD in the worktree (sub-agents do the work)
 5. TEST             Run all tests and type checking (must pass)
 6. SIMPLIFY         Code-simplifier agent analyzes for improvements (Staff review)
 7. CODE REVIEW      Code-reviewer sub-agent reviews changes (must approve)
 8. SQL REVIEW       Staff Engineer audits queries for performance, security, defensive coding
 9. COMMIT           Commit to the feature branch
10. PUSH + PR        Push branch, create PR with "Closes #N" to auto-close the issue
11. VERIFY CI        Check that CI passes; fix and re-push if it fails
```

Each phase has a verification gate. The workflow does not advance until the gate passes. For example, code review must explicitly approve before SQL review runs, SQL review must approve before a commit is created, and CI must be green before work is considered done.

### Initialization

Before making any writes to GitHub (issues, PRs, labels, project items), the project must be initialized:

```
/project:init
```

This command:

1. Creates a standard set of labels for issue categorization (feature, bug, refactor, docs, test, chore, epic, task, P0-P3 priorities)
2. Removes conflicting GitHub default labels (e.g., `enhancement` and `documentation`)
3. Creates a GitHub Projects board named after the repository
4. Sets a `claude:initialized` marker label so the plugin knows the project is ready

Initialization is idempotent. Running it again with `--force` re-syncs labels without duplicating them.

#### Standard Labels

| Label | Purpose |
|-------|---------|
| `feature` | New features (maps to `feat:` commit prefix) |
| `bug` | Bug fixes (maps to `fix:` prefix) |
| `refactor` | Code improvements with no behavior change |
| `docs` | Documentation changes |
| `test` | Test additions or changes |
| `chore` | Maintenance and housekeeping |
| `epic` | Parent issue that groups related tasks |
| `task` | Sub-issue of an epic |
| `P0-critical` | Drop everything |
| `P1-high` | Do soon |
| `P2-medium` | Normal priority |
| `P3-low` | Nice to have |

### Planning Features

`/project:plan-feature` takes a feature description or a path to a PRD document and creates:

1. An **epic** issue on GitHub with requirements, task breakdown, and acceptance criteria
2. Individual **task** sub-issues, each linked back to the epic
3. All issues added to the project board

Each task is scoped to be independently implementable in a single PR. The command analyzes the codebase to determine what already exists and what needs to be built.

```
/project:plan-feature "User profile page with avatar upload and bio editing"
/project:plan-feature "docs/plans/user-profiles-prd.md"
```

### Review Modes

At the start of every piece of work, the workflow asks which review mode to use:

**Autonomous review:** After the PR is created, a code-reviewer sub-agent reviews the diff. If issues are found, fresh sub-agents fix them in the worktree. Once the review is clean and CI is green, the PR is automatically merged. Safety limits prevent infinite loops: max 3 review-fix cycles and max 3 CI-fix attempts before falling back to manual review.

**Manual review:** The PR is created and the user is notified with the PR URL, branch name, and worktree path. The user reviews and decides when to merge.

### Issue Tracking

Issues can originate from two places:

- **Locally in Claude Code** -- Most work starts here. The agent creates a GitHub issue for tracking, then works on it. These issues are marked with `<!-- source: claude-code -->` in the body.
- **From a person on GitHub** -- Someone creates an issue in the GitHub UI. Use `/project:inbox` to find these issues and pull them into a Claude Code session.

`/project:sync-tasks` bridges local todo lists and GitHub by creating issues for any pending tasks that are not already tracked. This is a one-way sync (local to GitHub) and never modifies the local todo list or existing GitHub issues.

`/project:current` provides a dashboard view of the project: open issues, active PRs, worktree status, project board columns, recently completed work, and suggested next actions.

### Git Worktrees

Every feature branch gets its own git worktree, isolating work from main and from other in-progress features. The convention is:

```
# If the repo is at ~/projects/my-app
# Worktrees are created at ~/projects/worktrees/my-app/<branch-name>

git worktree add "../worktrees/my-app/feat/user-profiles" -b feat/user-profiles
```

Branch naming follows the pattern `<type>/<short-description>`, matching the issue title prefix (e.g., `feat/user-profiles`, `fix/login-redirect`, `refactor/auth-middleware`).

`/project:cleanup` finds stale worktrees. A worktree is stale if its branch has been merged to main, its PR has been merged or closed, or it has been idle for more than 7 days with no open PR. By default, cleanup runs in dry-run mode and only reports what it would remove. Pass `--force` to actually clean up.

### Graceful Degradation

All GitHub integration is optional. If the `gh` CLI is not available or not authenticated, the plugin skips all GitHub operations and continues working normally. The development lifecycle (planning, TDD, code review, etc.) functions independently of GitHub. Even without `gh`, the plugin still uses feature branches rather than committing directly to main.

---

## Attribution

This is a collection of Claude Code skills and agents from various sources:

- **Most skills were created by others** in the Claude Code community
- **Some were modified** to fit specific workflows or fix issues
- **Some were created** from scratch

All credit goes to the original skill creators. If you are a skill author and would like attribution added or your skill removed, please open an issue.

---

## License

MIT
