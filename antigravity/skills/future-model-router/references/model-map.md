# Model Map & Dispatch Reference (Antigravity & Gemini)

Authoritative model IDs, relative token costs, and dispatch syntax for Antigravity (`invoke_subagent` and `swarm`) and Gemini CLI (`~/.gemini/GEMINI.md` and `templates/GEMINI-BASE.md`).

## 1. Model Tiers & Relative Cost

| Tier / ID | Relative Cost | Compaction Window | Primary Roles |
|---|---|---|---|
| `gemini-4-argon-medium` (`Argon Medium`) | `1.0×` (baseline) | `256k` (warn `205k`, block `235k`) | Default Coordinator UI model (Phases 0–3, 5–7, 10–12); Phase 4 TDD Implementer (`impl` in `swarm` or `Model: "inherit"` in `invoke_subagent`) |
| `gemini-4-carbon-high` / `gemini-4-carbon-medium` (`Carbon`) | `2.0×` | `256k` (warn `205k`, block `235k`) | Adversarial reviewers (`plan-rev`, `sec-rev`, `sql-rev`, `editor-rev`); manual UI switch for the ~10% hardest architectural, auth/crypto, concurrency, or zero-downtime migration plans |
| `claude-opus-5-5-high` (`Claude Opus 5.5 High`) | Cross-family high deliberation | — | Primary Staff Code-Quality Reviewer (`code-rev`) in `peer-pipeline-swarm` (fallback: `gemini-4-carbon-high` when Anthropic quota is exhausted) |
| `gemini-4-hydrogen-low` / `gemini-4-hydrogen-high` (`Hydrogen`) | `0.2×` | `256k` (warn `205k`, block `235k`) | Non-code chat, triage, fast Q&A, and the `swarm_monitor` stall watchdog (`monitor`, fallback `gemini-3.8-flash-low`) |
| `gemini-3.8-flash` (`Model: "flash"`) | ~`0.25×` per token (but `5.9×` more tokens on complex TDD) | `256k` | Bounded read-only search (`≤15` turns) and `<50` LOC single-file mechanical boilerplate/renames only |
| `Model: "flash_lite"` | ~`0.1×` | — | Pre-fed MCP/file extraction (`≤3` turns max; never for code discovery) |

## 2. `antigravity-subagents` Plugin Roster (`~/.gemini/config/plugins/antigravity-subagents/agents/`)

| `TypeName` | `tools:` Grant | Schema Floor | `Model` Argument | Purpose |
|---|---|---|---|---|
| `plan-reviewer` | `[read_file]` | `~31k` | `"inherit"` | Phase 3½ read-only adversarial plan audit (zero spikes, zero `run_command`) |
| `code-quality-reviewer` | `[read_file]` | `~31k` | `"inherit"` | Phase 8 read-only Staff Code-Quality audit and `fix-check` verification |
| `security-reviewer` | `[read_file]` | `~31k` | `"inherit"` | Phase 8 read-only Principal Security (OWASP) audit |
| `sql-reviewer` | `[read_file]` | `~31k` | `"inherit"` | Phase 9 read-only SQL performance, indexing, and injection audit |
| `editor-pass-reviewer` | `[read_file]` | `~31k` | `"inherit"` | Phase 12 read-only post-build quality audit against the point-of-view brief |
| `implementer` | `[read_file, edit_file, write_file, run_command, manage_task]` | `~42k` | `"inherit"` | Phase 4 TDD implementer (runs scoped fast unit/lint gates and returns 4-section handoff) |

## 3. Dispatch Examples

### A. `invoke_subagent` (Harness-Enforced Tool Grants)

```text
invoke_subagent({
  TypeName: "implementer",
  Role: "Phase 4 TDD Implementer",
  Model: "inherit",
  Prompt: "Read /tmp/subagent-context-<project>-<stream>.md and PLAN.md first..."
})
```

### B. `swarm` (Exact Model Pinning for `peer-pipeline-swarm` & One-Shot Carbon Reviews)

```text
swarm({
  action: "add",
  name: "impl",
  model: "gemini-4-argon-medium",
  workspace: "share",
  brief: "Read ~/.gemini/config/plugins/antigravity-subagents/agents/implementer.md, /tmp/plan-<project>-<stream>.md, and /tmp/subagent-context-<project>-<stream>.md..."
})
```
