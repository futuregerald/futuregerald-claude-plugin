# `futuregerald-claude-plugin` (`skill-installer` v4.0) — Contributor & AI Agent Guide

## 1. Repository Philosophy & Purpose

`futuregerald-claude-plugin` builds **`skill-installer`**, a cross-platform (macOS, Linux, Windows) graphical CLI and packaged catalog of AI agent skills, slash commands, subagents, and harness rules for:
- **Claude Code** (`claude`)
- **Google Antigravity** (`antigravity`)
- **Zed Editor** (`zed`)
- **OpenCode** (`opencode`)

**Core Product Rule:** Never nudge users toward "install all 50+ skills at once." Every loaded skill competes for an AI agent's context window and attention. Users choose from **curated tracks** (`internal/catalog/tracks.yaml`) tailored to their role:
1. **`opinionated` ("Opinionated Dev Workflow")** — Full 12-phase engineering lifecycle (`/start`, `/begin`, `/lifecycle`) for software engineers who want strict shaping, 3-direction HTML UI prototyping, caller tracing, TDD, adversarial reviews, and blast-radius verification.
2. **`basics` ("Just the Basics")** — 8 everyday engineering skills (`brainstorming`, `writing-plans`, `systematic-debugging`, `test-driven-development`, `comprehensive-code-review`, `pull-request-description`, `handoff`, `plugin-updater`) with a lightweight config and zero hooks.
3. **`product` ("Product & Strategy Flow")** — Non-developer pack for PMs, designers, founders, and engineering leads (`craftsmans-wisdom`, `/shape`, `brainstorming`, `future-grilling`, `future-wayfinder`, `grilling-page`, `write-a-prd`, `writing-tickets`, `ticket-grooming`, `ui-design`, `huashu-design`, `no-slop-ui`, `prototype`, `visualize-plan`, `copy-edit`, `copywriting`, `marketing-psychology`, `editor-pass`, `research`, `meeting-debrief`, `team-pulse`, `plugin-updater`).
4. **`copy-polish` ("Copy, Brand & UI Polish")** — Focused pack for auditing and elevating user-facing pages, landing copy, visual hierarchy, and motion (`copy-edit`, `copywriting`, `marketing-psychology`, `editor-pass`, `impeccable`, `no-slop-ui`, `huashu-design`, `animation-principles`, `plugin-updater`).
5. **`custom` ("Pick Individual Skills")** — Interactive à la carte skill browser.

---

## 2. Core Opinionated Workflow vs. Optional Standalone Skills

**CRITICAL BOUNDARY FOR AGENTS:** Do **not** wire new or optional standalone skills into the mandatory 12-Phase Development Lifecycle unless the user explicitly asks to change the core workflow.

### The Core Opinionated 12-Phase Workflow
Only these skills form the mandatory lifecycle gates in `CLAUDE-BASE.md`, `CLAUDE-GLOBAL.md`, `GEMINI.md`, `GEMINI-BASE.md`, `AGENTS-ZED.md`, `AGENTS-OPENCODE.md`, `AGENTS-GLOBAL.md`, and `lifecycle/SKILL.md`:
- **Entry & Orchestration:** `start` (`/start`), `begin` (`/begin`), `lifecycle` (`/lifecycle`), `subagent-driven-development`, `future-model-router` (plus Antigravity-only `antigravity-orchestration`, `peer-pipeline-swarm`, and `baton-pass`)
- **Phase 0 (Shape):** `shape` (`/shape`), `craftsmans-wisdom`, `future-grilling`, `future-wayfinder`, `grilling-page`, `write-a-prd`, `brainstorming`
- **Phase 0½ (UI Design):** `ui-design` (`/ui-design`), `huashu-design`, `no-slop-ui`
- **Phases 3 & 3½ (Plan & Plan Review):** `writing-plans`, `plan-review` (`/plan-review`), `review-lens`
- **Phases 4–7 (Implement, Test, Lint, Simplify):** `test-driven-development`, `using-git-worktrees`, `systematic-debugging`, `code-simplifier`
- **Phases 8–9 (Code & SQL Review):** `comprehensive-code-review`, `fix-check` (`/fix-check`), `sql-optimization-patterns`
- **Phase 12 (Handoff):** `editor-pass`, `pull-request-description`, `copy-edit` (`/copy-edit`, combining `copywriting` + `marketing-psychology`), `handoff`

### Optional / Domain-Specific Standalone Skills
Skills such as `agent-browser`, `animation-principles`, `datadog-dashboards`, `dispatching-parallel-agents`, `future-skill-creator`, `go-api-structure`, `impeccable`, `meeting-debrief`, `prototype`, `qa-verification`, `research`, `resolving-merge-conflicts`, `skill-reviewer`, `teach`, `team-pulse`, `ticket-grooming`, `vectorize`, `visualize-plan`, and `writing-tickets` are **on-demand tools**. Never inject them as mandatory blocking steps in the 12-phase table.

---

## 3. Packaged Catalog Layout & Placement Rules (`internal/catalog/`)

There are **no** top-level `skills/`, `agents/`, `commands/`, `templates/`, `antigravity/`, or `.claude-plugin/` directories at the repository root. All embedded assets live inside `internal/catalog/`:

```text
internal/catalog/
  embed.go                       # //go:embed shared claude antigravity zed opencode tracks.yaml
  catalog.go                     # Track, skill, command, and template resolver over any fs.FS
  catalog_test.go                # Catalog & C4 cross-reference closure tests
  tracks.yaml                    # 5 curated tracks + per-skill plain-English metadata & explainers
  shared/
    skills/<skill-name>/SKILL.md # Shared skills installed across harnesses (adapted per target)
    templates/languages/*.md     # Language-specific snippets injected at <!-- LANGUAGE_SPECIFIC -->
  claude/
    agents/*.md                  # Claude Code subagents
    commands/*.md                # Native Claude slash commands (/start, /shape, /plugin-update, etc.)
    templates/CLAUDE-BASE.md     # Project CLAUDE.md template
    CLAUDE-GLOBAL.md             # Global ~/.claude/CLAUDE.md (Opinionated track)
    CLAUDE-LIGHT.md              # Global ~/.claude/CLAUDE.md (Lightweight tracks)
  antigravity/
    GEMINI.md                    # Global ~/.gemini/GEMINI.md (Opinionated track)
    GEMINI-LIGHT.md              # Global ~/.gemini/GEMINI.md (Lightweight tracks)
    hooks/                       # Circuit-breaker hooks + tests (ANTIGRAVITY ONLY)
    hooks.json                   # PreToolUse hook registration (ANTIGRAVITY ONLY)
    plugins/antigravity-subagents/
    rules/org.example.md
    skills/                      # Antigravity-exclusive skills (baton-pass, antigravity-orchestration, peer-pipeline-swarm) + Antigravity skill overrides
    templates/GEMINI-BASE.md     # Project GEMINI.md template
  zed/
    AGENTS-GLOBAL.md             # Global Zed AGENTS.md (~/.config/zed/AGENTS.md or %APPDATA%\Zed\AGENTS.md)
    templates/AGENTS-ZED.md      # Project AGENTS.md tailored for Zed (.agents/skills native /slash commands)
  opencode/
    agents/*.md                  # OpenCode subagents (.opencode/agents/*.md)
    commands/*.md                # OpenCode slash commands (.opencode/commands/*.md)
    AGENTS-GLOBAL.md             # Global OpenCode AGENTS.md (~/.config/opencode/AGENTS.md)
    templates/AGENTS-OPENCODE.md # Project AGENTS.md tailored for OpenCode
```

### Mandatory Asset Invariants
1. **`baton-pass` and circuit-breaker hooks are ANTIGRAVITY-ONLY:** Never place `baton-pass`, `antigravity-orchestration`, `peer-pipeline-swarm`, or circuit-breaker hooks in `shared/`, `claude/`, `zed/`, or `opencode/`, and never reference `baton-pass` in Claude, Zed, or OpenCode templates.
2. **Distinct harness configs:** Never copy-paste identical `CLAUDE.md` boilerplate across `zed` and `opencode`. Each harness template must reflect that tool's actual directory layout, subagent support, and slash-command mechanics.
3. **Register every skill in `tracks.yaml`:** Whenever you add or rename a skill directory under `internal/catalog/shared/skills/` or `internal/catalog/antigravity/skills/`, add its `category`, `slash_command`, `short` (1-line plain English), `long` (2–3 sentences plain English), and `starter_prompt` to `internal/catalog/tracks.yaml`.
4. **Track dependency closure (C4):** If a skill in a curated track (`opinionated`, `basics`, `product`, `copy-polish`) links to `../<other-skill>/SKILL.md`, `<other-skill>` must either be included in that same track or explicitly marked `(when installed)` / `(optional)` on that line.
5. **Host isolation in tests:** Never read or write the developer's real `~/.gemini`, `~/.claude`, `~/.agents`, or `~/.config` during tests. Always isolate `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, and `APPDATA` to `t.TempDir()`.
6. **Zero code comments:** Write no `//` or `#` explanatory comments in Go or Python files (except required `//go:embed` directives and `#!/usr/bin/env ...` shebangs).

---

## 4. Verification Gates

Run all gates before committing or opening a PR:

```bash
test -z "$(gofmt -l .)"
go vet ./...
HOME=$(mktemp -d) go test -race ./...
PYTHONDONTWRITEBYTECODE=1 pytest -p no:cacheprovider -q \
  internal/catalog/shared/skills/editor-pass/scripts \
  internal/catalog/shared/skills/future-skill-creator/scripts \
  internal/catalog/shared/skills/grilling-page/scripts \
  internal/catalog/shared/skills/team-pulse/scripts \
  internal/catalog/shared/skills/visualize-plan/scripts \
  internal/catalog/antigravity/hooks
```
