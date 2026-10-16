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
- [Codebase Memory MCP (Optional)](#codebase-memory-mcp-optional)
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

Skills are markdown-based instruction sets that give Claude specialized knowledge and workflows. When invoked, Claude follows the skill's process exactly. The plugin includes 53 skills covering:

- **Test-driven development** — RED/GREEN/REFACTOR cycle
- **Systematic debugging** — root-cause-first protocol with feedback-loop construction
- **Planning & specs** — plans, PRDs, specs, and tracer-bullet tickets, with review checkpoints
- **Code review & quality** — parallel-agent review, simplification, and skill review
- **Architecture & modeling** — deep-module design, domain models, and ADRs
- **Delivery & EM workflows** — PR descriptions, ticket grooming, team pulse, Datadog dashboards

Skills are invoked by name in Claude Code:

```
/futuregerald-claude-plugin:systematic-debugging
/futuregerald-claude-plugin:test-driven-development
/futuregerald-claude-plugin:brainstorming
```

### Agents — Specialized Sub-Agents

Agents are dispatched via the `Task` tool to handle focused work with fresh context. The plugin includes 10 agents: adversarial plan reviewer, code quality reviewer, code simplifier (with Staff Engineer review), codebase searcher, context finder (index/memory-aware read-only search), debugger, implementer, security reviewer, spec reviewer, and SQL performance reviewer.

### Development Workflow — A Structured Lifecycle

The plugin defines a 10-phase development lifecycle in the generated CLAUDE.md:

```
 1. RECEIVE     Understand task, create todo list
 2. PLAN        Write implementation plan
 3. IMPLEMENT   Write code following TDD
 4. TEST        Run tests and type checking
 5. SIMPLIFY    Code-simplifier agent analyzes for improvements
 6. CODE REVIEW Code-reviewer sub-agent reviews changes
 7. SQL REVIEW  Staff Engineer audits queries for performance, security, defensive coding
 8. COMMIT      Commit to feature branch
 9. PUSH + PR   Push and create PR (if gh available)
10. VERIFY CI   Check CI passes; auto-merge when green
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

If symlinked to `~/.claude/skills` (installed as personal skills), invoke them by plain name — no prefix:

```
/systematic-debugging
/brainstorming
```

---

## Skills and Agents Reference

### Skills (58)

**Planning & Specs:**

| Skill | Description |
|-------|-------------|
| `brainstorming` | Explore intent, requirements, and design before implementation |
| `writing-plans` | Turn a spec into a bite-sized, TDD-driven implementation plan; for multi-plan work, keep the committed roadmap of phases and the agreed strategy |
| `plan-review` | Staff-engineer review of a plan before implementing; one pass, never re-reviewed |
| `visualize-plan` | Turn a plan into a visual HTML page (flows, ER diagram, worked example, decisions) alongside the markdown |
| `subagent-driven-development` | Execute independent plan tasks via fresh sub-agents |
| `dispatching-parallel-agents` | Run 2+ independent tasks concurrently |
| `implement` | Implement a piece of work from a spec or set of tickets |
| `write-a-prd` | Create a PRD via interview + codebase exploration, with the product experience and a link to the roadmap; file as an issue |
| `writing-tickets` | Write epics, stories, initiatives, and spike tickets |
| `to-spec` | Synthesize the conversation into a spec, published to your tracker |
| `to-tickets` | Break a plan/spec into tracer-bullet tickets with blocking edges |
| `wayfinder` | Plan work too big for one session as a map of decision tickets |
| `research` | Investigate a question against primary sources; capture as Markdown |
| `future-grilling` | Fork of Matt Pocock's grilling that reads a craftsmans-wisdom brief and checks each answer against it |
| `grilling-page` | Puts any series of 3+ questions (grilling, handoff or plan open questions, reviews) on one HTML page with context, what each blocks, its source, and each option's cost; the answers come back in one paste and are recorded |
| `grill-with-docs` | Grilling that also produces ADRs and a glossary as it goes |
| `craftsmans-wisdom` | Runs before any plan: one-question-at-a-time challenge on what good means, what's written down for agents, and whether done is good; writes a point-of-view brief, then offers future-grilling on the details |

**Debugging & Code Quality:**

| Skill | Description |
|-------|-------------|
| `systematic-debugging` | Root-cause-first debugging with tight feedback-loop construction |
| `test-driven-development` | RED → GREEN → REFACTOR discipline |
| `comprehensive-code-review` | Parallel sub-agent review (correctness + safety) into one report |
| `fix-check` | After a fix round: per finding, fixed or not, where, and which test covers it; replaces a re-review |
| `code-simplifier` | Simplify recently changed code, with Staff Engineer review |
| `skill-reviewer` | Review skills for quality, size, progressive disclosure, and encoded judgment |
| `editor-pass` | Post-build quality review that returns a lettered crit sheet of pinned, specific fixes |
| `review-lens` | Surface similar past reviewer objections to calibrate findings |
| `resolving-merge-conflicts` | Resolve an in-progress git merge/rebase conflict |

**Architecture & Modeling:**

| Skill | Description |
|-------|-------------|
| `codebase-design` | Shared vocabulary for designing deep, testable modules |
| `domain-modeling` | Build and sharpen a project's domain model and ubiquitous language |
| `architecture-decision-records` | Write and maintain ADRs |
| `improve-codebase-architecture` | Find module-deepening opportunities; visual report, then grill |
| `go-api-structure` | Go service layout, interface-driven dependencies, the HTTP edge, structured logging, and bounded concurrency |

**Delivery & Dev Workflow:**

| Skill | Description |
|-------|-------------|
| `using-git-worktrees` | Isolated git worktrees for feature work |
| `baton-pass` | Pass the baton between sessions with a deterministic ACTIVE_CONTEXT.md snapshot |
| `pull-request-description` | Structured PR summary, background, test plan, and rollback plan |
| `ticket-grooming` | Investigate and groom tickets; post structured triage notes |
| `future-model-router` | Decide whether work runs in a sub-agent, and which model runs it |
| `handoff` | Compact the conversation into a handoff doc for the next session |
| `teach` | Teach a new skill or concept within the workspace |
| `future-skill-creator` | Fork of Anthropic's skill-creator that encodes judgment: decision trees, do/don't pairs, check scripts, and editing the skill's output |

**Frontend & Design:**

| Skill | Description |
|-------|-------------|
| `impeccable` | Frontend design, UX review, and UI polish |
| `no-slop-ui` | Eleven visual tells that make a UI read as AI slop, and what to do instead |
| `huashu-design` | HTML-native prototypes, slide decks, animations and infographics; 20 design philosophies, 5-dimension expert critique, MP4/GIF/PDF/editable-PPTX export |
| `prototype` | Build a throwaway prototype to answer a design question |

**Content & Media:**

| Skill | Description |
|-------|-------------|
| `nano-banana` | Image generation and editing |
| `vectorize` | Convert raster PNG/JPG into clean SVG |
| `copywriting` | Marketing copy for landing, pricing, and feature pages |
| `marketing-psychology` | Mental models and behavioral science for marketing |

**Testing & Data:**

| Skill | Description |
|-------|-------------|
| `javascript-testing-patterns` | Jest, Vitest, and Japa testing patterns |
| `sql-optimization-patterns` | Query optimization, indexing, EXPLAIN, N+1 elimination |
| `qa-verification` | Verify a ticket's acceptance criteria by running the code, not just reading it |

**Automation & EM/Observability:**

| Skill | Description |
|-------|-------------|
| `agent-browser` | Browser automation with Playwright |
| `datadog-dashboards` | Build and review Datadog dashboards that actually render the data |
| `team-pulse` | Engineering-manager team status reports (pulse) and epic/roadmap forecasts (forecast) |
| `meeting-debrief` | Strategic meeting analysis from Krisp transcripts |

### Agents (10 dispatchable)

Agents are specialized sub-agents dispatched via the Agent tool. They run with fresh context and no knowledge of the parent conversation.

**Tool grants matter.** An agent definition with no `tools:` line inherits every tool the session has loaded, which on a large MCP surface costs tens of thousands of tokens of schema on every dispatch. The agents below marked *restricted* declare an explicit minimum — measured at roughly half the dispatch cost of an unrestricted one — and the ones without `Bash` cannot mutate your working tree at all.

`codebase-searcher`, `debugger`, `implementer` and `spec-reviewer` are **prompt templates, not registered agents** — they have no frontmatter and are used as prompt text under `general-purpose` by `subagent-driven-development`. They cannot be passed as a `subagent_type`.

| Agent | Description |
|-------|-------------|
| `adversarial-plan-reviewer` | Attacks an implementation plan before code is written; runs once |
| `code-quality-reviewer` | Reviews code for quality issues |
| `code-simplifier` | Analyzes code for simplification, with Staff Engineer review |
| `codebase-searcher` | Searches and explores codebases |
| `context-finder` | Read-only, index/memory-aware search: code index → session memory → grep last; returns synthesized findings with `file:line` (runs on Sonnet) |
| `debugger` | Systematic bug investigation |
| `implementer` | Implements features from plans |
| `security-reviewer` | OWASP-aligned security audit: injection, auth, IDOR, SSRF, cryptographic failures, data exposure |
| `spec-reviewer` | Reviews specifications and plans |
| `sql-reviewer` | Ruthless SQL performance, security, and defensive coding audit |
| `investigator` | *restricted* — Read/Grep/Glob only. Literal-string and unindexed-repo search. No shell, no writes |
| `reviewer` | *restricted* — Read/Grep/Glob plus codebase-index tools, on Opus. Fresh-context review; cannot run git, so pass it a diff file |
| `runner` | *restricted* — adds `Bash`. Runs suites and triages logs, reports the verdict not the output. Can mutate: dispatch against a committed SHA |
| `writer` | *restricted* — adds `Edit`/`Write`. The only one that changes files |

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

# Full install includes optional codebase-memory-mcp setup
skill-installer --mode full --target claude
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

## Codebase Memory MCP (Optional)

The CLI installer can optionally install [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp), a server that indexes your codebase into a knowledge graph. Once configured, Claude can query function relationships, trace call paths, and understand architecture without reading every file.

### What it provides

| Tool | Purpose |
|------|---------|
| `get_architecture` | High-level overview of project structure and components |
| `search_graph` | Query the knowledge graph for function/class relationships (prefer over text grep) |
| `trace_call_path` | Trace callers and callees of a function across the codebase |
| `search_code` | Search code with semantic understanding |
| `index_repository` | Index or re-index the current repository |

### Installation

During `skill-installer --mode full`, you'll be prompted:

```
Would you like to install the Codebase Memory MCP server for codebase indexing and search? [Y/n]:
```

If accepted, the installer:
1. Downloads the platform-specific binary from GitHub releases
2. Verifies the SHA-256 checksum
3. Installs to `~/.local/bin/codebase-memory-mcp`
4. Configures the MCP server globally (`~/.claude/settings.json`) or locally (`.mcp.json`)

To skip, answer "n" or use `--yes` with `--mode config-only`.

### Usage

After installation, tell Claude to index your project:

```
index this project
```

The generated CLAUDE.md template includes rules for when Claude should use the graph tools -- during planning, debugging, and code review phases.

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
│   ├── config/                  # /config:export, /config:import
│   └── init-claude-md/
│       └── COMMAND.md           # /init-claude-md command
├── skills/                      # 53 skill directories, each with SKILL.md
├── agents/                      # 14 files: 10 dispatchable agents + 4 prompt templates
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

## Attribution

This is a collection of Claude Code skills and agents from various sources:

- **Most skills were created by others** in the Claude Code community
- **Some were modified** to fit specific workflows or fix issues
- **Some were created** from scratch

All credit goes to the original skill creators. If you are a skill author and would like attribution added or your skill removed, please open an issue.

---

## License

MIT
