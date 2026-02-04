# futuregerald-claude-plugin

Portable skills, agents, and commands for Claude Code. Includes debugging protocols, TDD workflows, code review, and multi-language project scaffolding.

## Installation

### Option 1: Clone and Symlink (Recommended)

```bash
# Clone to your preferred location
git clone https://github.com/futuregerald/claude-skills-plugin.git ~/claude-skills-plugin

# Symlink to Claude's global directory
ln -s ~/claude-skills-plugin/skills ~/.claude/skills
ln -s ~/claude-skills-plugin/agents ~/.claude/agents
ln -s ~/claude-skills-plugin/commands ~/.claude/commands
```

### Option 2: Plugin Directory Flag

```bash
# Clone anywhere
git clone https://github.com/futuregerald/claude-skills-plugin.git

# Run Claude with plugin directory
claude --plugin-dir ./claude-skills-plugin
```

### Option 3: Direct Clone to Claude Directory

```bash
git clone https://github.com/futuregerald/claude-skills-plugin.git ~/.claude/plugins/futuregerald
claude --plugin-dir ~/.claude/plugins/futuregerald
```

## Contents

### Commands

| Command | Description |
|---------|-------------|
| `/init-claude-md` | Generate a customized CLAUDE.md for your project based on detected framework/language |

### Skills (34 total)

**Core Workflow:**
| Skill | Description |
|-------|-------------|
| `using-superpowers` | Skill discovery and usage patterns |
| `systematic-debugging` | 4-phase debugging protocol (no guessing) |
| `test-driven-development` | TDD workflow: RED → GREEN → REFACTOR |
| `writing-plans` | Implementation planning before coding |
| `executing-plans` | Plan execution with review checkpoints |
| `brainstorming` | Creative exploration before implementation |

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
| `find-skills` | Skill discovery |

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
| `svelte.md` | Svelte 5 with runes |
| `react.md` | React with hooks |
| `go.md` | Go projects |
| `ruby.md` | Ruby/Rails |
| `rust.md` | Rust projects |
| `php.md` | PHP/Laravel |

## Usage

After installation, skills are available with the namespace prefix:

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

## Directory Structure

```
claude-skills-plugin/
├── .claude-plugin/
│   └── plugin.json           # Plugin manifest
├── agents/                   # Subagent definitions
│   ├── code-quality-reviewer.md
│   ├── code-simplifier.md
│   ├── codebase-searcher.md
│   ├── debugger.md
│   ├── implementer.md
│   └── spec-reviewer.md
├── commands/                 # User-invokable commands
│   └── init-claude-md/
│       └── COMMAND.md
├── skills/                   # All skills (34 total)
│   ├── systematic-debugging/
│   ├── test-driven-development/
│   └── ...
├── templates/                # CLAUDE.md templates
│   ├── CLAUDE-BASE.md
│   └── languages/
│       ├── svelte.md
│       ├── react.md
│       ├── go.md
│       ├── ruby.md
│       ├── rust.md
│       └── php.md
└── README.md
```

## Updating

```bash
cd ~/claude-skills-plugin  # or wherever you cloned it
git pull
```

If using symlinks, changes are immediately available. No restart needed.
