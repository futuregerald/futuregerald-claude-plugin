# Antigravity configuration

Versioned copy of the Antigravity global rules, harness-native skill overrides, lifecycle subagents, and circuit-breaker hook. Embedded by `skill-installer` for `--target antigravity` (`skill-installer -t antigravity --global`); Claude Code, Zed, and OpenCode never read these files.

Personal, team, tracker and repository specifics never live in these rules, skills or hooks; they live in separate git-ignored files that these files reference (`~/.gemini/config/rules/*.local.md` and `skills/antigravity-orchestration/references/config.md`). Because nothing is stored inline, the tracked files here match the live files under `~/.gemini/` byte-for-byte.

| Here | Live path | Purpose |
|---|---|---|
| `GEMINI.md` | `~/.gemini/GEMINI.md` | Always-on global rules: parallel read/edit batching, Thin Root Supervisor (`<60k–120k`, `<=2`-file inline ceiling, `>=3`-file Q&A subagent delegation, `planner` Stage-1 subagent, and disposable `TypeName: "self"` Child Orchestrators for Phases 3½–12), check-in vs authorized runs, `Carbon Medium` default / `Argon Medium` (`/start-small`) role routing, and optional sleeping `Hydrogen` (`0.2x`) `ops` worker (`cum_turns=<N>` rotation) |
| `rules/org.example.md` | `~/.gemini/config/rules/<org>.local.md` (git-ignored) | Template for organization, tracker and repo rules |
| `skills/antigravity-orchestration/SKILL.md` | `~/.gemini/config/skills/antigravity-orchestration/SKILL.md` | On-demand reference: memory signatures, search hierarchy, Thin Root Supervisor (`planner` + `TypeName: "self"` Child Orchestrator + `~120k` wrap-up), Carbon/Argon/Hydrogen `ops` & Peer-Pipeline Swarm routing, observed Turn-1 floors (`22.7k`/`24.8k`/`29.3k`/`45.0k`), breaker rules, same-chat baton auto-chaining |
| `skills/{peer-pipeline-swarm,start,start-small,begin}/` | `~/.gemini/config/skills/{peer-pipeline-swarm,start,start-small,begin}/` | Antigravity-only `/start`, `/begin` (`Carbon Medium`), and `/start-small` (`Argon Medium`) 12-phase lifecycles and Phases 4–9 Peer-Pipeline Swarm (`impl` + `code-rev` + `sec-rev` + `monitor`) |
| `skills/{lifecycle,subagent-driven-development,future-writing-plans,plan-review,comprehensive-code-review,fix-check,editor-pass,baton-pass,future-model-router}/` | `~/.gemini/config/skills/<skill>/` | Antigravity-native overrides for core workflow skills (consuming pre-packed `build_subagent_context.py` slice/diff chunks + `## Turn-1 Parallel Read Manifest`, and tearing down sleeping `ops` workers during `baton-pass`) |
| `skills/antigravity-orchestration/references/config.example.md` | `~/.gemini/config/skills/antigravity-orchestration/references/config.md` (git-ignored) | Template for local tool paths and org names (e.g. `review-lens`) |
| `plugins/antigravity-subagents/` | `~/.gemini/config/plugins/antigravity-subagents/` | Dedicated 7-agent lifecycle plugin (`planner`, `plan-reviewer`, `code-quality-reviewer`, `security-reviewer`, `sql-reviewer`, `editor-pass-reviewer`, `implementer`) with explicit `tools:` grants, `<320k` (`350k` hard cap) subagent guard, and parallel Turn-1 chunk/manifest reading |
| `hooks.json` | `~/.gemini/config/hooks.json` | PreToolUse registration of the circuit breaker on 21 tools (including `swarm`, `read_board`, `post_message`, and `swarm_wait`) |
| `hooks/` | `~/.gemini/config/hooks/` | Circuit breaker (`autonomous-breaker.sh` → `autonomous_breaker.py`), shared rules (`breaker_rules.py`), offline tools (`session_metrics.py`, `replay_breaker.py`) and tests |

## Circuit breaker rules

| Rule | Fires when | Default mode |
|---|---|---|
| DUP | identical tool call repeated within a short window | enforce |
| EXPLORE | a helper agent exceeds `BREAKER_SUBAGENT_EXPLORE_MAX` exploratory calls (opt-in; default `0` / disabled) | enforce (when cap > 0) |
| STRIKES | 4 consecutive test failures (exempting wrap-up `.md` handoff edits) | enforce |
| BATON | 3-way context partition (`input_tokens + cache_read_tokens + output_tokens` + trailing bytes ÷ 4 since compaction): `350k` swarm-worker cap (+25k `/tmp/*.md` WIP unlock, including `SYSTEM_MESSAGE`-only workers), `350k` (`SUBAGENT_TOKEN_CAP`) `invoke_subagent` cap (wrap-up `.md` + `send_message` only), and `310k` warn / `350k` block (+50k `ACTIVE_CONTEXT.md` unlock) for Coordinator (`512k` compaction window) | enforce |
| PROBE | 3rd inline `python3 -c` probe of the same file | enforce |
| TRUELOOP | 3rd re-read of mostly already-read lines (tracking distinct `ByteOffset` pagination windows on `read_file`), or the same command, within 20 planner steps | enforce |
| H1 | an authorized main session makes a 7th code edit without dispatching a helper | enforce |
| H2 | a helper agent reaches `BREAKER_SUBAGENT_STEP_MAX` planner turns (opt-in; default `0` / disabled) | enforce (when cap > 0) |

Rules overridden with `BREAKER_<RULE>_MODE=log` write would-deny lines to `~/.gemini/metrics/breaker.jsonl` without blocking. Override any rule with `BREAKER_<RULE>_MODE=log|enforce|off`. The hook fails open: any error allows the call.

## Syncing from `~/.gemini`

```bash
./scripts/sync-antigravity-from-local.sh
```

Copies the live files verbatim (without the git-ignored local config files) and fails if any personal, org-specific, or secret string was accidentally written into a tracked file.

## Tests

```bash
PYTHONDONTWRITEBYTECODE=1 pytest -p no:cacheprovider -q internal/catalog/antigravity/hooks
```
