# Model Map

The rules in `SKILL.md` name **roles**, never models. This file is the only place a model name appears, so a new lineup is a one-file edit that never touches the reasoning.

**Verify before relying on a row.** Lineups change every few months. Last checked: 2026-09-18.

## Roles

| Role | Selected when | Typical work |
|---|---|---|
| **retriever** | The answer has a known shape and one hop gets it | Path and symbol lookups, "does X exist", listing, mechanical extraction into a fixed format |
| **explorer** | Multi-step, but the answer's shape is still statable up front | Tracing a flow, summarizing a long document, first-pass log triage, gathering ticket or PR data |
| **orchestrator** | No statable shape — the work contains a judgment call | Root-cause analysis, design questions, synthesis, review, anything deciding rather than finding |

The orchestrator role is also whatever model is driving the session. Isolating work *at* the orchestrator role is normal and expected — see the isolate axis in `SKILL.md`.

## Provider mapping

| Role | Anthropic | Google |
|---|---|---|
| retriever | Haiku 4.5 | Gemini Flash-Lite |
| explorer | Sonnet 5 | Gemini 3.8 Flash, thinking level `low` or `medium` |
| orchestrator | Opus 5 (Fable 5.1 for planning and synthesis) | Gemini 3 Pro, or 3.8 Flash at thinking level `high` |

### Two levers, not one

**Anthropic** spends reasoning budget mainly by swapping the model.

**Google** gives you two independent levers, because Gemini 3.8 Flash exposes three thinking levels (`low`, `medium`, `high`; default `medium`). You can drop to a smaller model *or* keep the model and cut its thinking level. Prefer cutting the thinking level first when the task is well-scoped but non-trivial — it keeps the larger model's knowledge while spending less on deliberation.

This is why the skill says **"reduce the reasoning budget"** rather than "use a smaller model". The rule is the same; only how you spend it differs.

## Dispatch by harness

### Claude Code

```
Agent({ model: "haiku" | "sonnet" | "opus",
        subagent_type: "Explore" | "context-finder" | "general-purpose",
        prompt: "..." })
```

Several `Agent` calls in one message run concurrently.

**Tool grants differ and this matters for safety** (see the safety section in `SKILL.md`):

| Agent type | `Bash` | `Edit`/`Write` |
|---|---|---|
| `context-finder` | no | no |
| `Explore` | **yes** | no |
| `general-purpose` | yes | yes |

### Google Antigravity

Subagents are spawned programmatically by the main agent and run with **workspace isolation**, so their threads do not enter the main agent's context. Three flavours: built-in roles, generic clones that inherit the main agent's prompt and environment, and subagents registered on the fly with a goal the main agent defines.

The model is selectable **per agent**, which is what makes the role mapping above usable — Gemini 3 Pro, Claude Sonnet 4.5 and GPT-OSS are all selectable targets.

`/agents` opens the Agent Manager panel to browse custom agents and watch active or finished background subagents.

Because subagents get workspace isolation rather than merely a separate context, the tree-mutation hazard in `SKILL.md` is weaker here than in Claude Code — but commit-first still costs nothing and still protects you when a subagent is pointed at the shared workspace.

### Harnesses without subagents

If a harness has no way to spawn a separate context that reports back, **only the downgrade axis applies**. The isolate axis needs a subagent primitive; without one, use the carve-outs and the reasoning-budget rules and ignore the rest. Nothing in the skill degrades unsafely — you simply lose the context savings.

## Sources

- [Gemini 3.8 Flash — Google AI for Developers](https://ai.google.dev/gemini-api/docs/latest-model)
- [Introducing Gemini 3.8 Flash and 3.8 Flash Cyber](https://blog.google/innovation-and-ai/models-and-research/gemini-models/3-8-flash-and-3-8-flash-cyber/)
- [Antigravity: Subagents, Hooks, Scheduled Tasks, Agent Management](https://antigravity.google/blog/google-io-2026-feature-deep-dive)
- [Agents Command (/agents) — Antigravity Docs](https://antigravity.google/docs/cli/commands/agents/)
