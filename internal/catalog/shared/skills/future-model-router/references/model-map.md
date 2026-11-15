# Model Map

The rules in `SKILL.md` name **roles**, never models. This file is the only place a model name appears, so a new lineup is a one-file edit that never touches the reasoning.

**Verify before relying on a row.** Lineups change every few months. Last checked: 2026-11-15.

**Check what your entry point actually accepts before copying a model name out of this file.** In Claude Code, the `Agent` tool accepts `model: "haiku" | "sonnet" | "opus"` (or inherits from the agent definition's YAML frontmatter), and agent frontmatter supports `effort: low | medium | high | xhigh | max` plus explicit `tools:` grants. Other CLI/IDE harnesses expose different model identifiers and frontmatter fields; harness-specific overrides document their own rosters.

**Sourcing.** Rows are marked by how well they are established:
- **[verified]** — confirmed directly against a tool grant, config file, benchmark run, or API doc.
- **[vendor]** — stated by the vendor or its guidance. Plausible, not independently tested.
- **[reported]** — observed or relayed behaviour with no citation. Treat as a working assumption; if you act on it and it does not hold, fix the row.

## Roles

| Role | Selected when | Typical work |
|---|---|---|
| **retriever** | The answer has a known shape and the input is already bounded | Distilling a captured log, deterministic format extraction, binary checks across named files (max 3 turns, zero open-ended code hunting) |
| **explorer** | Multi-step, but the answer's shape is still statable up front | Wide `>=8`-query caller scouting, noisy build/test failure reproduction, tracing a flow, summarizing a long document, TDD implementation against an approved plan |
| **orchestrator** | No statable shape — the work contains a judgment call | Root-cause analysis, architecture & migration planning, synthesis, adversarial plan/code/security/SQL review |

The orchestrator role is also whatever model is driving the session. Isolating work *at* the orchestrator role is normal and expected — see the isolate axis in `SKILL.md`.

## Provider mapping (Claude Code & Cross-Harness Baseline)

| Role | Claude Code (`Agent` tool & agent frontmatter) | Key measured constraint |
|---|---|---|
| **retriever** | **Haiku 4.5** (`model: "haiku"`) — **strictly for pre-fed bounded input** (`<=3` turns, zero code discovery) **[verified]** | Sent hunting across a repo, Haiku took `10` tool calls vs. Sonnet's `4` and cost **`2.03x` more context** (`61,776` vs. `30,384` tokens) while running `1.86x` slower **[verified]** |
| **explorer** | **Sonnet 5.5** (`model: "sonnet"`, `effort: "low"` for `context-finder` & `runner`; `effort: "medium"` for `implementer`) **[verified]** | `effort: low` keeps larger-model location accuracy on `>=8`-query caller scouting and noisy build/test reproduction without paying deliberation overhead |
| **orchestrator** | **Opus 5.5** (`model: "opus"`, `effort: "medium"` default for Lead/Planner; `effort: "high"` for `adversarial-plan-reviewer`, `spec-reviewer`, `code-quality-reviewer`, `security-reviewer`, `sql-reviewer`, and complex architecture/migrations) **[verified]** | Single fresh `high`-effort review pass caught `2` CRITICAL + `12` IMPORTANT defects missed by the authoring context (`98,745` tokens vs. `~31k` restricted dispatch floor) **[verified]** |

### Two levers: Model tier and `effort`

Model and effort are set independently. Prefer cutting `effort` first when the task is well-scoped but requires accurate code navigation: it keeps Sonnet/Opus symbol and path accuracy while spending near-zero thinking tokens on mechanical steps.

| Mechanism | Where configured | Measured behaviour |
|---|---|---|
| `model:` | `Agent({ model: ... })` or `.claude/agents/<name>.md` YAML frontmatter | Controls capability tier (`opus`, `sonnet`, `haiku`). Never use `haiku` for open-ended file hunting (`2.03x` context penalty from extra search turns) or multi-file TDD **[verified]** |
| `effort:` | `.claude/agents/<name>.md` YAML frontmatter (`low`, `medium`, `high`, `xhigh`, `max`) or `claude --effort <level>` | Thinking tokens are billed as output tokens and accumulate in transcript context. On unambiguous mechanical tasks, `low` and `high` produced the **identical diff** (`0–215` thinking tokens across 8 runs); reserve `high` for *"what did we fail to consider?"* **[verified]** |
| `tools:` | `.claude/agents/<name>.md` YAML frontmatter | Explicit minimal tool list cuts null-task dispatch floor by **57%** (`56,887` -> `24,561` tokens) and enforces read-only safety at the harness boundary **[verified]** |

**Harness gotcha, measured:** a subagent definition dropped into `.claude/agents/` mid-session is **not** picked up — the registry loads at session start, and a dispatch to the new type fails with `"Agent type not found"`. To exercise a new definition without restarting, run it headless: `claude -p "<task>" --agent <name> --output-format json`, which also reports `num_turns`, `duration_ms`, `total_cost_usd` and `usage.output_tokens_details.thinking_tokens`.

### Fresh One-Shot `Agent` Dispatches vs. Warm `resume` vs. Agent Teams

1. **Fresh One-Shot `Agent` vs. Warm `Agent` `resume` (`+58%` penalty) [verified]:**
   - Resuming a finished sub-agent carrying ~98k of prior context cost **`101,679` tokens (`20.4s`)** vs. **`64,211` tokens (`15.4s`)** for a fresh sub-agent doing the identical follow-up work (**`+58%` tokens, `+32%` wall time**).
   - Always prefer **one fresh `Agent` with a batched multi-part prompt** (or pre-packed `/tmp/subagent-context-<project>-<stream>.md` via `build_subagent_context.py`) over resuming warm sub-agents across phases.
2. **Experimental Multi-Session Agent Teams (`CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`, `~7x` token multiplier) [vendor + reported]:**
   - Multi-session Agent Teams spin up persistent concurrent Claude Code sessions with inter-agent messaging, multiplying total token spend by roughly **`~7x`** compared to standard single-session orchestration.
   - Do **not** use Agent Teams for routine sequential lifecycle phases, read-only code/plan reviews, or persistent "sleeping" command runners. Keep `context-finder` and `runner` (`model: sonnet`, `effort: low`) as fresh single-shot `Agent` dispatches, and reserve experimental Agent Teams strictly for when the user explicitly asks for `3+` independent parallel implementation workstreams.

### Haiku / Small-Model Retriever Guardrails

**The line is processing versus finding, and it is finer than "simple work".** Small retriever models (`haiku`, or lightweight tiers on other providers) are for bounded, mechanical work on material *you hand them*, where the output shape is rigid and cheap to check. They are not for going and looking.

**Three questions before you dispatch `haiku` — all three must be yes:**

1. **Can you pre-feed it the exact input (in `<=3` turns)?** If it has to hunt across files, use `context-finder` (`sonnet`, `effort: low`).
2. **Is the output format strictly mechanical?** Table, list, key-value, extracted lines.
3. **Can you verify the result without re-reading the raw input?** If checking it means redoing the work, isolating to a cheap model saved nothing.

**Never use `haiku` for:** open-ended code discovery (`2.03x` flailing penalty), adversarial review (rubber-stamps with generic praise), multi-file TDD implementation, or counting matches by eye (always use `grep -c` or a counting primitive).

## Dispatch by harness

### Claude Code

```typescript
Agent({
  model: "haiku" | "sonnet" | "opus",
  subagent_type: "context-finder" | "runner" | "implementer" | "code-quality-reviewer" | "security-reviewer",
  prompt: "..."
})
```

Several `Agent` calls in one message run concurrently.

**Tool grants and effort levels across shipped Claude Code agents [verified]:**

| Agent type | Provided by | Default model & effort | `Bash` | `Edit`/`Write` | Read-only enforced by harness? |
|---|---|---|---|---|---|
| `context-finder` | **this plugin** | `sonnet` (`effort: low`) | no | no | **yes** (`Read, Grep, Glob, ToolSearch, MCP`) |
| `investigator` | **this plugin** | `inherit` | no | no | **yes** |
| `reviewer` | **this plugin** | `inherit` (`effort: high`) | no | no | **yes** |
| `adversarial-plan-reviewer` | **this plugin** | `inherit` (`effort: high`) | no | no | **yes** (`Read, Grep, Glob, ToolSearch`, explicit read-only MCP allowlist) |
| `spec-reviewer` | **this plugin** | `opus` (`effort: high`) | no | no | **yes** (`Read, Grep, Glob, ToolSearch`, explicit read-only MCP allowlist) |
| `code-quality-reviewer` | **this plugin** | `opus` (`effort: high`) | no | no | **yes** (`Read, Grep, Glob, ToolSearch`, explicit read-only MCP allowlist) |
| `security-reviewer` | **this plugin** | `inherit` (`effort: high`) | no | no | **yes** (`Read, Grep, Glob, ToolSearch`, explicit read-only MCP allowlist) |
| `sql-reviewer` | **this plugin** | `inherit` (`effort: high`) | no | no | **yes** (`Read, Grep, Glob, ToolSearch`, explicit read-only MCP allowlist) |
| `runner` | **this plugin** | `sonnet` (`effort: low`) | **yes** | no | prompt mitigation only (`Bash` present) |
| `implementer` | **this plugin** | `sonnet` (`effort: medium`) | **yes** | **yes** | no (TDD implementation agent) |
| `writer` | **this plugin** | `sonnet` | **yes** | **yes** | no |
| `Explore` | the harness | session default | **yes** | no | no (`Bash` present) |
| `general-purpose` | the harness | session default | **yes** | **yes** | no |

All six review agents (`reviewer`, `adversarial-plan-reviewer`, `spec-reviewer`, `code-quality-reviewer`, `security-reviewer`, `sql-reviewer`) omit `Bash`, `Edit`, and `Write` from their `tools:` frontmatter. Because they physically cannot run `git diff`, the orchestrator **must** pre-pack the diff and gate output into `/tmp/subagent-context-<project>-<stream>.md` (via `build_subagent_context.py post-code`) before dispatching them.

`runner` holds `Bash`, so its restrictions on destructive git, `rm`, and network access are **instructions, not enforcement** — commit or checkpoint work first when dispatching it.

**Plugin agent availability:** Agents marked **this plugin** do not exist in a `config-only` install or in harnesses without `.claude/agents/`. Where absent, fall back to restricting `tools:` in your own agent definition rather than assuming a read-only type is present.

### Harnesses without subagents (e.g., Zed single-thread mode)

If a harness has no way to spawn a separate context that reports back, **only the downgrade and turn-batching axes apply**: filter bulky command output at the shell, batch parallel file reads (`5–12` files/turn, `400–800` lines/read), and lower reasoning effort on mechanical turns. Fresh-subagent plan and code review cannot run for objectivity on a single-context harness; state that explicitly rather than treating self-review as equivalent.
