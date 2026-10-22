# Antigravity configuration

Versioned copy of the Antigravity (Gemini) global rules, orchestration skill and circuit-breaker hook. These files are not embedded by `skill-installer` and Claude Code never reads them.

Personal, team, tracker and repository specifics never live in these rules, skills or hooks; they live in separate git-ignored files that these files reference (`~/.gemini/config/rules/*.local.md` and `skills/antigravity-orchestration/references/config.md`). Because nothing is stored inline, the tracked files here match the live files under `~/.gemini/` byte-for-byte.

| Here | Live path | Purpose |
|---|---|---|
| `GEMINI.md` | `~/.gemini/GEMINI.md` | Always-on global rules: execution discipline, check-in vs authorized runs, lifecycle tiers |
| `rules/skills-and-delegation.md` | `~/.gemini/config/rules/skills-and-delegation.md` | Always-on delegation and orchestration rules |
| `rules/org.example.md` | `~/.gemini/config/rules/<org>.local.md` (git-ignored) | Template for organization, tracker and repo rules |
| `skills/antigravity-orchestration/SKILL.md` | `~/.gemini/config/skills/antigravity-orchestration/SKILL.md` | On-demand reference: memory signatures, search hierarchy, model routing, breaker rules, baton-pass mechanics |
| `skills/antigravity-orchestration/references/config.example.md` | `~/.gemini/config/skills/antigravity-orchestration/references/config.md` (git-ignored) | Template for local tool paths and org names (e.g. `review-lens`) |
| `hooks.json` | `~/.gemini/config/hooks.json` | PreToolUse registration of the circuit breaker on 17 tools |
| `hooks/` | `~/.gemini/config/hooks/` | Circuit breaker (`autonomous-breaker.sh` → `autonomous_breaker.py`), shared rules (`breaker_rules.py`), offline tools (`session_metrics.py`, `replay_breaker.py`) and tests |

## Circuit breaker rules

| Rule | Fires when | Default mode |
|---|---|---|
| DUP | identical tool call repeated within a short window | enforce |
| EXPLORE | a helper agent exceeds 100 exploratory calls | enforce |
| STRIKES | 4 consecutive test failures | enforce |
| BATON | context estimate (transcript bytes since the last compaction ÷ 4) reaches 105k (one warn) or 130k (block until a handoff is written) | enforce |
| PROBE | 3rd inline `python3 -c` probe of the same file | log |
| TRUELOOP | 3rd re-read of mostly already-read lines, or the same command, within 20 planner steps | log |
| CHECKPOINT | an unauthorized main-session turn reaches a multiple of 45 calls | log |
| H1 | an authorized main session makes a 7th code edit without dispatching a helper | log |
| H2 | a helper agent reaches 130 planner turns | log |

`log` rules write would-deny lines to `~/.gemini/metrics/breaker.jsonl` without blocking. Override any rule with `BREAKER_<RULE>_MODE=log|enforce|off`. The hook fails open: any error allows the call.

## Syncing from `~/.gemini`

```bash
./antigravity/sync-from-local.sh
```

Copies the live files verbatim (without the git-ignored local config files) and fails if any personal or org-specific string was accidentally written into a tracked file.

## Tests

```bash
cd antigravity/hooks && python3 -m pytest -q test_*.py
```
