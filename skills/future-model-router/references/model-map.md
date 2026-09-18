# Model Map

The rules in `SKILL.md` name **roles**, never models. This file is the only place a model name appears, so a new lineup is a one-file edit that never touches the reasoning.

**Verify before relying on a row.** Lineups change every few months. Last checked: 2026-09-18. The Google rows reflect vendor guidance for the Google AI Ultra tier.

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
| retriever | Haiku 4.5 | Gemini 3.7 or 3.6 — non-reasoning, literal, no deliberation latency |
| explorer | Sonnet 5 | Gemini 3.8 Flash, thinking `low` or `medium` (default) |
| orchestrator | Opus 5 (Fable 5.1 for planning and synthesis) | Gemini 3.8 Flash, thinking `high` |

**The two platforms lean on different levers, and that is deliberate.** On Anthropic the model changes per role. On Google the primary lever is the thinking level on one model — 3.8 Flash covers explorer and orchestrator by itself.

**Do not reach for Flash-Lite.** It is not in Antigravity's chat model selector at all. Antigravity uses it under the hood for its own lightweight background subagents, but as a chat model it is too weak at tool-calling and code quality to orchestrate anything. If the goal is speed or quota, 3.8 Flash at `low` is the answer, not a weaker model.

**The retriever row is the exception to "hold the model".** 3.7 and 3.6 still earn their place for mechanical work: they are direct and literal, they hold strict output formats without breaking out to explain themselves, and they stream immediately because there is no thinking trace to compute first. For a format extraction, a regex, or a rename, that is better behaviour than 3.8 at `low` — not merely cheaper.

### Two levers, on both platforms

Model and effort are set independently, and both platforms expose both — they just default to different ones. Prefer cutting effort first when the task is well-scoped but non-trivial: it keeps the larger model's knowledge while spending less on deliberation.

| Platform | Model lever | Effort lever |
|---|---|---|
| Anthropic / Claude Code | `model:` on the dispatch, or in the agent definition | Reasoning effort in the agent definition's frontmatter (`.claude/agents/*.md`) — supported by the harness, and easy to leave unset by accident |
| Google / Antigravity | Model selectable per agent | Thinking level on Gemini 3.8 Flash: `low`, `medium`, `high`; default `medium` |

This is why the skill says **"reduce the reasoning budget"** rather than "use a smaller model".

### What the effort lever actually costs

Thinking tokens are **output tokens**. They count against rate limits and rolling quotas, and the input price does not change between settings — only the volume generated does. Approximate, for Gemini 3.8 Flash:

| Level | Typical thinking tokens | Relative burn |
|---|---|---|
| `low` | ~250 – 1,000 | 1x |
| `medium` (default) | ~1,000 – 4,000 | 2–4x |
| `high` | ~8,000 – 16,000+ | 5–15x+ |

The same prompt can produce 500 output tokens at `low` and 10,000+ at `high`. **That spread is the largest single multiplier in this document — bigger than the choice of model.** When a rolling limit is draining faster than expected during a long session, this is the first thing to check.

### Gemini lineup, by what it is good at

| Model | Best for | Why |
|---|---|---|
| 3.8 Flash | Feature work, bug finding, multi-file changes | Best reasoning, configurable thinking budget |
| 3.7 / 3.6 | Mechanical transforms, script generation, strict format extraction | Direct and literal, holds output contracts, no deliberation latency, will not over-engineer |
| Claude / GPT frontier | A second opinion, architecture review, quota fallback | Separate quota pool, different training biases when stuck |

That last row is a real reason to switch that is neither isolating nor downgrading: a separate quota pool routes around congestion or a drained limit on your primary provider. Treat it as an operational fallback, not a routing rule.

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
