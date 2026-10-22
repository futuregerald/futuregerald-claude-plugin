# Antigravity configuration snapshot

A versioned copy of the author's Antigravity (Gemini) setup. These files are not installed by `skill-installer` and Claude Code never reads them. The live copies sit under `~/.gemini/`; this directory exists so changes can be reviewed and rolled back.

| Here | Live path | What it is |
|---|---|---|
| `GEMINI.md` | `~/.gemini/GEMINI.md` | Always-on global rules: execution discipline, check-in vs authorized runs, lifecycle tiers |
| `rules/skills-and-delegation.md` | `~/.gemini/config/rules/` | Always-on delegation and orchestration rules |
| `skills/antigravity-orchestration/SKILL.md` | `~/.gemini/config/skills/antigravity-orchestration/` | On-demand reference: memory signatures, search hierarchy, model routing, breaker rules, baton-pass mechanics. Kept out of the top-level `skills/` so Claude Code does not load it |
| `hooks.json` | `~/.gemini/config/hooks.json` | PreToolUse registration of the circuit breaker on 17 tools |
| `hooks/` | `~/.gemini/config/hooks/` | Circuit breaker (`autonomous-breaker.sh` → `autonomous_breaker.py`), shared rules (`breaker_rules.py`), offline tools (`session_metrics.py`, `replay_breaker.py`) and their tests |

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

## Updating this snapshot

```bash
./antigravity/sync-from-local.sh
```

The script copies the live files here and redacts personal and employer details (names, home paths, tracker IDs, org names). It exits non-zero if anything it should have redacted remains. Re-run it after changing the live files, then commit.

## Tests

```bash
cd antigravity/hooks && python3 -m pytest -q test_*.py
```

Some tests read the live `~/.gemini/config/hooks.json`, so they pass only on a machine with this setup installed. CI does not run them.
