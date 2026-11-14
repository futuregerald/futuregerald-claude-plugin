# skill-installer (v4.0.0)

[![CI](https://github.com/futuregerald/futuregerald-claude-plugin/actions/workflows/ci.yml/badge.svg)](https://github.com/futuregerald/futuregerald-claude-plugin/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/futuregerald/futuregerald-claude-plugin)](https://github.com/futuregerald/futuregerald-claude-plugin/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/futuregerald/futuregerald-claude-plugin)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Install curated AI coding workflows, `/slash` commands, and review subagents for **Claude Code**, **Google Antigravity**, **Zed**, and **OpenCode** — without dumping 50+ noisy skill files into your config.

`skill-installer` gives you an interactive terminal wizard on **macOS, Linux, and Windows**: pick a track that matches your role, preview every skill in plain English, merge or back up existing configs before writing a single byte (with one-command rollback via `skill-installer restore`), and explore your new commands anytime with `skill-installer guide`.

```bash
# Launch the interactive 3-step terminal wizard
./skill-installer
# Or install the full Opinionated Dev Workflow for Claude Code in one command
./skill-installer -t claude -g --track opinionated --backup -y
```

---

## Table of Contents

- [Curated Installation Tracks](#curated-installation-tracks)
- [Quick Start](#quick-start)
- [Supported Harnesses & Cross-Platform Paths](#supported-harnesses--cross-platform-paths)
- [Interactive Graphical TUI & Guide](#interactive-graphical-tui--guide)
- [CLI Reference](#cli-reference)
- [Safety Backups, Restore & Uninstall](#safety-backups-restore--uninstall)
- [Live Updates (`skill-installer update` & `/plugin-update`)](#live-updates-skill-installer-update--plugin-update)
- [The 12-Phase Development Lifecycle](#the-12-phase-development-lifecycle)
- [Skill Catalog by Category](#skill-catalog-by-category)
- [Architecture](#architecture)
- [Local-Only Docker & Desktop Harness E2E Suite](#local-only-docker--desktop-harness-e2e-suite)

---

## Curated Installation Tracks

You don't need to install everything on day one. Choose one of five tracks interactively or via `--track`:

> **Note:** Not sure between **Opinionated** and **Just the Basics**? Choose **Opinionated** if you want `/start` to run caller tracing, plan review, TDD, and adversarial code/security reviews automatically. Choose **Just the Basics** if you want the 8 core skills on demand with zero workflow gates or hooks.

1. **`opinionated` — Opinionated Dev Workflow (Recommended)**
   - Installs all **32–35 workflow skills** (`32` on Claude/Zed/OpenCode, `35` on Antigravity) plus harness-native rules (`CLAUDE.md` / `GEMINI.md` / `AGENTS.md`), specialized review subagents, slash commands, and (on Antigravity) the context-window safety hook (`autonomous-breaker`, which saves a handoff before long sessions hit compaction limits).
   - Enables the complete 12-phase Development Lifecycle (`/start`, `/begin`, `/lifecycle`).
2. **`basics` — Just the Basics (Core Coding & Debugging)**
   - Installs **8 essentials** for everyday engineering without workflow ceremony: `brainstorming`, `writing-plans`, `systematic-debugging`, `test-driven-development`, `comprehensive-code-review`, `pull-request-description`, `handoff`, and `plugin-updater`.
3. **`product` — Product & Strategy Flow**
   - Installs **22 product, design & strategy skills**: `craftsmans-wisdom`, `shape`, `brainstorming`, `future-grilling`, `future-wayfinder`, `grilling-page`, `write-a-prd`, `writing-tickets`, `ticket-grooming`, `ui-design`, `huashu-design`, `no-slop-ui`, `prototype`, `visualize-plan`, `copy-edit`, `copywriting`, `marketing-psychology`, `editor-pass`, `research`, `meeting-debrief`, `team-pulse`, and `plugin-updater`.
4. **`copy-polish` — Copy, Brand & UI Polish**
   - Installs **9 design & copywriting skills**: `copy-edit`, `copywriting`, `marketing-psychology`, `editor-pass`, `impeccable`, `no-slop-ui`, `huashu-design`, `animation-principles`, and `plugin-updater`.
5. **`custom` — Pick Individual Skills**
   - Browse the full categorized catalog and pick individual skills (`--skills test-driven-development,copywriting,...` or interactively with `Space` to toggle and `d` for plain-English details; starts with only `plugin-updater` pre-checked).

---

## Quick Start

### 1. Download a Prebuilt Binary (or Build from Source)

Download the prebuilt binary for **macOS** (`darwin-arm64`, `darwin-amd64`), **Linux** (`linux-amd64`, `linux-arm64`), or **Windows** (`windows-amd64.exe`) from the [Releases page](https://github.com/futuregerald/futuregerald-claude-plugin/releases) and run it directly:

```bash
chmod +x ./skill-installer
./skill-installer
```

Or build from source with Go:

```bash
git clone https://github.com/futuregerald/futuregerald-claude-plugin.git
cd futuregerald-claude-plugin
make build
```

### 2. Run the Interactive Guided Installer

Run with no arguments in any terminal:

```bash
./skill-installer
```

From the Home menu (`Choose a workflow track (60-sec guided setup)`, `See how this works & command guide`, `Check for skill & workflow updates`, `Restore a safety backup`, `Uninstall`), the guided bot (`[ o_o ]`) walks you one calm decision per screen through three steps:
1. **Step 1 of 3 — Target App & Scope:** Pick `Google Antigravity`, `Claude Code`, `Zed Editor`, or `OpenCode`. Press `Tab` to toggle `Global (all projects)` vs `Specific project folder (<path>)`, or press `e` to set any target project directory path directly (supports `~` expansion and creates the directory if needed).
2. **Step 2 of 3 — Workflow Track:** Choose `Opinionated Dev Workflow`, `Just the Basics`, `Product & Strategy Flow`, `Copy, Brand & UI Polish`, or `Pick Individual Skills` (press `Space` or `p` on any track to preview its skills, `/slash` commands, and starter prompts).
3. **Step 3 of 3 — Config File & Safety Backup:** If your target already has a `CLAUDE.md`, `GEMINI.md`, or `AGENTS.md`, choose whether to **add/update a managed workflow section** inside your existing file (preserving your custom rules), **replace** the file with a clean template (with a timestamped backup), or **keep your existing file untouched** and install skills only.

### 3. Non-Interactive / Headless Examples

```bash
# Install the full opinionated workflow globally for Claude Code (with automatic backup)
skill-installer -t claude -g --track opinionated --backup -y

# Merge workflow rules into an existing global GEMINI.md without overwriting custom rules
skill-installer -t antigravity -g --track opinionated --merge-config -y

# Install the Basics track into a specific project directory and merge into its existing CLAUDE.md
skill-installer -t claude --project-dir ~/code/my-app --track basics --merge-config -y

# Install specific skills for OpenCode globally
skill-installer -t opencode -g --track custom --skills test-driven-development,systematic-debugging -y
```

---

## Supported Harnesses & Cross-Platform Paths

`skill-installer` embeds a unified catalog of **54 skills** (`51` shared across all harnesses + `3` Google Antigravity exclusives) alongside harness-native rules, subagents, slash commands, and context-window safety hooks for four first-class AI coding harnesses:

| Harness (`--target`) | Global Path (macOS / Linux) | Global Path (Windows) | Project Path (`--project` / `--project-dir`) | Installed Artifacts |
|---|---|---|---|---|
| **Claude Code** (`claude`) | `~/.claude/` | `%USERPROFILE%\.claude\` | `<repo>/.claude/` & `<repo>/CLAUDE.md` | `CLAUDE.md`, `skills/<name>/`, `commands/`, `agents/` |
| **Google Antigravity** (`antigravity`) | `~/.gemini/` | `%USERPROFILE%\.gemini\` | `<repo>/.agents/skills/`, `<repo>/.agents/agents/`, `<repo>/GEMINI.md` | `GEMINI.md`, `config/skills/<name>/`, `config/plugins/antigravity-subagents/`, `config/hooks/`, `config/hooks.json` |
| **Zed** (`zed`) | `~/.agents/skills/` & `~/.config/zed/AGENTS.md` | `%USERPROFILE%\.agents\skills\` & `%APPDATA%\Zed\AGENTS.md` | `<repo>/.agents/skills/` & `<repo>/AGENTS.md` | `.agents/skills/<name>/`, `AGENTS.md` |
| **OpenCode** (`opencode`) | `~/.config/opencode/` (`$XDG_CONFIG_HOME/opencode/`) | `%USERPROFILE%\.config\opencode\` (`$XDG_CONFIG_HOME\opencode\`) | `<repo>/.opencode/` & `<repo>/AGENTS.md` | `AGENTS.md`, `skills/<name>/`, `agents/`, `commands/` |

> **Windows Compatibility:** On Windows (`GOOS=windows`), `skill-installer` automatically rewrites Antigravity's `hooks.json` command from `autonomous-breaker.sh` to `"C:/Python312/python.exe" "C:/Users/<you>/.gemini/config/hooks/autonomous_breaker.py"` (or `python`/`py -3` if not resolved on `PATH`) so the context-window safety hook works natively without requiring `bash` or WSL.

---

## Interactive Graphical TUI & Guide

Open the interactive 3-tab guide at any time—even without reinstalling:

```bash
skill-installer guide
```

- **`1. What You Can Type Now`** — Concise guide to how your chosen app loads `/slash` commands and skills, plus copy-pasteable starter prompts from your installed track.
- **`2. How This Toolkit Works`** — Plain-English explainers on curated tracks, how slash commands vs. automatic skills work across harnesses, safety backups & one-command restore, and live updates.
- **`3. Skill & Command Explorer`** — Categorized catalog of all 54 skills (`51` shared + `3` Antigravity-exclusive). Press `Up`/`Down` to browse with a live detail card below, and press `a` to toggle between your track and all skills for your harness.

Running `skill-installer guide` in a non-TTY pipeline (`skill-installer guide | less`) automatically outputs a pure-ASCII reference safe across macOS, Linux, and Windows consoles.

---

## CLI Reference

```text
skill-installer [flags]
skill-installer <subcommand> [flags]
```

### Subcommands

| Subcommand | Description |
|---|---|
| `skill-installer` | Launch the interactive TUI wizard (or run headlessly with `-y` / `--config`) |
| `skill-installer guide` | Open the interactive 3-tab guide (What You Can Type Now, How This Toolkit Works, Skill & Command Explorer) |
| `skill-installer list [--category <cat>]` | List all 54 embedded skills with categories, commands, descriptions, and starter prompts |
| `skill-installer update [--check] [--self]` | Pull the latest skills and workflow updates from `main` via `gh` CLI (and optionally self-upgrade the binary) |
| `skill-installer restore [snapshot-id] [--file <path>]` | Restore a complete pre-install backup snapshot or a single backed-up file |
| `skill-installer uninstall` | Cleanly remove all files installed by `skill-installer` (leaving any other custom hooks in `hooks.json` intact) |
| `skill-installer init [-o .skill-installer.yaml]` | Generate a starter `.skill-installer.yaml` configuration file |
| `skill-installer version` | Print the `skill-installer` version (`4.0.0`) |

### Flags

| Flag | Short | Description |
|---|---|---|
| `--target <harness>` | `-t` | Target harness: `claude`, `antigravity`, `zed`, `opencode` |
| `--global` | `-g` | Install to user-level global harness directory |
| `--project` | `-p` | Install to current repository's project directory |
| `--project-dir <dir>` | `-d` | Install, update, restore, or uninstall in a specific project directory (implies `--project`) |
| `--track <track>` | | Track preset: `opinionated`, `basics`, `product`, `copy-polish`, `custom` |
| `--skills <s1,s2,...>` | | Comma-separated skill names (implies `--track custom`) |
| `--config-mode <mode>` | | How to handle an existing `CLAUDE.md` / `GEMINI.md` / `AGENTS.md`: `merge`, `replace`, or `skip` |
| `--merge-config` | | Add or update the managed workflow section inside an existing config file (`--config-mode merge`) |
| `--skip-claude-md` | | Skip updating project/global config file (`--config-mode skip`) |
| `--backup` | | Back up any existing files before overwriting |
| `--no-backup` | | Skip pre-install backup |
| `--force` | `-f` | Overwrite existing files without prompting |
| `--non-interactive` | `-y` | Run headlessly without TUI prompts |
| `--config <file>` | `-c` | Read settings from `.skill-installer.yaml` |

---

## Safety Backups, Restore & Uninstall

Whenever `skill-installer` updates or overwrites existing files, it can snapshot the pre-write state to an OS-appropriate state directory (`~/.config/skill-installer/` or `$XDG_CONFIG_HOME/skill-installer/` on macOS and Linux, `%APPDATA%\skill-installer\` on Windows):

- **Restore a full snapshot:**
  ```bash
  skill-installer restore
  # Or headlessly restore the most recent snapshot for a target:
  skill-installer restore -t antigravity -g -y
  ```
  Restoring a snapshot puts back every overwritten file byte-for-byte **and** removes any newly created files that did not exist before that install.
- **Restore a single file:**
  ```bash
  skill-installer restore <snapshot-id> --file ~/.gemini/GEMINI.md
  ```
- **Clean Uninstall:**
  ```bash
  skill-installer uninstall -t claude -g
  ```
  Reads `state.json` (or discovers legacy v3.x installations) and removes only the files managed by `skill-installer`. If your config file (`CLAUDE.md`, `GEMINI.md`, or `AGENTS.md`) uses a merged workflow section, `skill-installer update` refreshes that block in place and `skill-installer uninstall` strips only the `<!-- BEGIN SKILL-INSTALLER WORKFLOW RULES -->` … `<!-- END SKILL-INSTALLER WORKFLOW RULES -->` block while keeping your custom rules intact. On Antigravity, `uninstall` removes only the `autonomous-circuit-breaker` entry from `~/.gemini/config/hooks.json`, leaving any other custom hooks intact.

---

## Live Updates (`skill-installer update` & `/plugin-update`)

You can pull newly added or improved skills directly from `futuregerald/futuregerald-claude-plugin` (`main`) without waiting for a binary release:

### From your terminal (`gh` CLI)

```bash
# Preview available skill & binary updates
skill-installer update --check -t antigravity -g

# Apply live skill updates from main (automatically respects your installed track & skills)
skill-installer update -t antigravity -g --backup -y

# Also self-upgrade the skill-installer binary from the latest GitHub Release (with SHA256 verification)
skill-installer update --self -t antigravity -g --backup -y
```

### From inside your AI coding session (`/plugin-update`)

Every track includes the **`plugin-updater`** skill (`/plugin-update`). Ask your agent:

> `"/plugin-update Check if there are any updates to my installed skills"`

- **Path A (`skill-installer` + `gh` CLI available):** Runs `skill-installer update --check --json`, summarizes what changed in plain English, asks if you want a backup, and runs `skill-installer update`.
- **Path B (No local CLI required — GitHub MCP fallback):** Uses your connected GitHub MCP server to inspect commits on `futuregerald/futuregerald-claude-plugin` (`main`), fetches only the changed files for your harness and track, backs up modified files, and updates your local installation receipt.

---

## The 12-Phase Development Lifecycle

When you install the **Opinionated** track, you can run `/start`, `/begin`, or `/lifecycle` on any coding task to enforce all 12 quality gates:

| Phase | Name | What Happens | Skill / Command |
|---|---|---|---|
| **0** | **Shape** | Interview on what "good" means and write a confirmed brief (`craftsmans-wisdom`), then grill open decisions (`future-grilling`) or write a PRD (`write-a-prd`) | `/shape`, `/craftsmans-wisdom`, `/future-grilling`, `/write-a-prd` |
| **0½** | **UI Design** | For new pages or major UI work: generate 3 distinct interactive HTML prototypes (`design-demos/index.html`) and pick a direction before planning | `/ui-design`, `/huashu-design` |
| **1** | **Receive** | Confirm task scope, constraints, and task list | — |
| **2** | **Trace** | Map the call chain upward (every caller to the entry point) and downward (side effects, DB/IO, invisible edges) | — |
| **3** | **Plan** | Write the implementation plan with an explicit **Affected Callers** section, test cases, and verification gates | `/writing-plans`, `/visualize-plan` |
| **3½** | **Plan Review** | One-shot read-only adversarial plan review in a fresh subagent before any code is written | `/plan-review` |
| **4** | **Implement** | Strict TDD (`RED → GREEN → REFACTOR`) in an isolated implementer subagent or Peer-Pipeline Swarm | `/test-driven-development`, `/subagent-driven-development`, `/peer-pipeline-swarm` |
| **5** | **Test** | Scoped fast unit test suite passes with zero failures | — |
| **6** | **Lint** | Formatter, linter, and static analysis pass with zero offenses | — |
| **7** | **Simplify** | Refine newly written code for clarity, reuse, and deep modules | `/code-simplifier` |
| **8** | **Code Review** | Parallel read-only code-quality and security reviews (`comprehensive-code-review`), verified on fix rounds with `fix-check` | `/comprehensive-code-review`, `/fix-check` |
| **9** | **SQL Review** | Read-only query, index, and migration review when database queries change | `/sql-optimization-patterns` |
| **10** | **Blast-Radius Verify** | Walk every row in Affected Callers after all code changes complete | — |
| **11** | **Commit** | Conventional commit on a feature branch | — |
| **12** | **Handoff** | Run `editor-pass` on user-facing work, open PR (`pull-request-description`), and offer `/copy-edit` while CI runs | `/editor-pass`, `/pull-request-description`, `/copy-edit` |

---

## Skill Catalog by Category

Run `skill-installer list --long` or `skill-installer guide` to inspect every skill's full description and starter prompt (`54` skills total: `51` shared + `3` Antigravity-exclusive).

### Core Workflow (3 skills)
- `start` (`/start`), `begin` (`/begin`), `lifecycle` (`/lifecycle`)

### Product & Shaping (9 skills)
- `shape` (`/shape`), `craftsmans-wisdom` (`/craftsmans-wisdom`), `brainstorming` (`/brainstorming`), `future-grilling` (`/future-grilling`), `future-wayfinder` (`/future-wayfinder`), `grilling-page` (`/grilling-page`), `write-a-prd` (`/write-a-prd`), `writing-tickets` (`/writing-tickets`), `ticket-grooming` (`/ticket-grooming`)

### Design, Copy & UI (12 skills)
- `ui-design` (`/ui-design`), `huashu-design` (`/huashu-design`), `no-slop-ui` (`/no-slop-ui`), `impeccable` (`/impeccable`), `animation-principles` (`/animation-principles`), `copy-edit` (`/copy-edit`), `copywriting` (`/copywriting`), `marketing-psychology` (`/marketing-psychology`), `editor-pass` (`/editor-pass`), `prototype` (`/prototype`), `visualize-plan` (`/visualize-plan`), `vectorize` (`/vectorize`)

### Planning & Architecture (3 skills)
- `writing-plans` (`/writing-plans`), `plan-review` (`/plan-review`), `go-api-structure` (`/go-api-structure`)

### Engineering & TDD (7 skills)
- `test-driven-development` (`/test-driven-development`), `subagent-driven-development` (`/subagent-driven-development`), `systematic-debugging` (`/systematic-debugging`), `code-simplifier` (`/code-simplifier`), `using-git-worktrees` (`/using-git-worktrees`), `dispatching-parallel-agents` (`/dispatching-parallel-agents`), `resolving-merge-conflicts` (`/resolving-merge-conflicts`)

### Review, Security & QA (6 skills)
- `comprehensive-code-review` (`/comprehensive-code-review`), `fix-check` (`/fix-check`), `sql-optimization-patterns` (`/sql-optimization-patterns`), `review-lens` (`/review-lens`), `qa-verification` (`/qa-verification`), `skill-reviewer` (`/skill-reviewer`)

### Shipping, Ops & Team (5 skills)
- `pull-request-description` (`/pull-request-description`), `handoff` (`/handoff`), `team-pulse` (`/team-pulse`), `meeting-debrief` (`/meeting-debrief`), `datadog-dashboards` (`/datadog-dashboards`)

### Research, Browser & Meta (6 skills)
- `agent-browser` (`/agent-browser`), `research` (`/research`), `teach` (`/teach`), `future-model-router` (`/future-model-router`), `future-skill-creator` (`/future-skill-creator`), `plugin-updater` (`/plugin-update`)

### Antigravity Exclusives (3 skills)
- `antigravity-orchestration` (`/antigravity-orchestration`), `peer-pipeline-swarm` (`/peer-pipeline-swarm`), `baton-pass` (`/baton-pass`)

---

## Architecture

All distributed assets live inside `internal/catalog/` and are compiled into the Go binary via `//go:embed`:

```text
internal/
  catalog/
    embed.go              # Single //go:embed directive + 54-skill metadata registry & 5 track presets
    tracks.yaml           # Curated track definitions, plain-English skill copy, and starter prompts
    shared/skills/        # 51 shared cross-harness skills
    claude/               # Claude Code agents, slash commands, and CLAUDE.md templates
    antigravity/          # GEMINI.md, 3 Antigravity-exclusive skills + overrides, subagent plugins, and circuit-breaker hooks
    zed/                  # Zed AGENTS-GLOBAL.md and templates/AGENTS-ZED.md
    opencode/             # OpenCode AGENTS-GLOBAL.md, templates/AGENTS-OPENCODE.md, agents, and commands
  platform/               # Cross-platform (darwin/linux/windows) × harness × scope path resolution
  backup/                 # NTFS-safe indexed backup snapshots, restoration, and pruning
  state/                  # Installation receipts (state.json), legacy v3 discovery, and clean uninstallation
  installer/              # Plan computation, markdown harness adaptation, and atomic file installation
  updater/                # Live main-branch tarball updates via gh CLI and self-upgrading binary logic
  tui/                    # Bubble Tea + Lip Gloss interactive wizard, progressive disclosure table, and 3-tab guide
```

---

## Local-Only Docker & Desktop Harness E2E Suite

To run the full containerized and host-sandbox end-to-end suite locally across all four harnesses (`claude`, `antigravity`, `zed`, `opencode`) — including real installed `@anthropic-ai/claude-code` and `opencode-ai` CLIs inside Docker, Claude Desktop and Antigravity Desktop coexistence checks, and live `autonomous-breaker` hook execution:

```bash
make test-e2e
```

`e2e/e2e_test.go` is guarded by `//go:build e2e` so `make test` (`go test ./...`) and GitHub Actions CI never run the Docker E2E suite. You can also run the host-isolated desktop compatibility sandbox directly without Docker:

```bash
HOME=$(mktemp -d) go test -v -tags=e2e ./e2e -run TestHostDesktopAppsCompatibilitySandbox
```

## License

MIT
