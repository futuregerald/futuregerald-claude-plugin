# Model Map

The rules in `SKILL.md` name **roles**, never models. This file is the only place a model name appears, so a new lineup is a one-file edit that never touches the reasoning.

**Verify before relying on a row.** Lineups change every few months. Last checked: 2026-09-18.

**Sourcing.** Rows are marked by how well they are established:
- **[verified]** — confirmed directly against a tool grant, config file or API doc.
- **[vendor]** — stated by the vendor or its guidance. Plausible, not independently tested.
- **[reported]** — observed or relayed behaviour with no citation. Treat as a working assumption; if you act on it and it does not hold, fix the row.

The Google rows reflect vendor guidance for the Google AI Ultra tier and are **[vendor]** unless marked otherwise.

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
| retriever | Haiku 4.5 | Gemini 3.7 or 3.6 at thinking `low` — more literal, less prone to inventing scope **[reported]** |
| explorer | Sonnet 5 | Gemini 3.8 Flash at thinking `low` or `medium` (default) |
| orchestrator | Opus 5 (Fable 5.1 for planning and synthesis) | Gemini 3.8 Flash, thinking `high` |

**The two platforms lean on different levers, and that is deliberate.** On Anthropic the model changes per role. On Google the primary lever is the thinking level on one model — 3.8 Flash covers explorer and orchestrator by itself.

**Do not reach for Flash-Lite. [reported]** It is not in Antigravity's chat model selector. Antigravity uses it under the hood for its own lightweight background subagents, but as a chat model it is too weak at tool-calling and code quality to orchestrate anything. If the goal is speed or quota, 3.8 Flash at `low` is the answer, not a weaker model.

**The retriever row is the exception to "hold the model". [reported]** 3.7 and 3.6 also expose thinking levels, so **always name a level when you name one of them** — they are a different model, not a non-reasoning one. At `low` they earn their place for mechanical work: more literal, better at holding a strict output format instead of breaking out to explain themselves, and quicker to first token. For a format extraction, a regex, or a rename, that is better behaviour than 3.8 at `low` — not merely cheaper.

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

The same prompt can produce a few hundred output tokens at `low` and five figures at `high`. The bracket midpoints imply closer to 15–20x than the 5–15x quoted alongside them, so treat both as order-of-magnitude, not arithmetic.

**Scope of the claim:** where one model is held across several budgets — the Google column — effort is the dominant multiplier. It is *not* larger than the model lever on a platform whose tiers are separate models with different unit prices; there the two are spent differently and are not comparable on one scale. When a rolling limit drains faster than expected in a long session, effort is still the first thing to check, because it moves without you choosing it.

### Gemini lineup, by what it is good at

| Model | Best for | Why |
|---|---|---|
| 3.8 Flash | Feature work, bug finding, multi-file changes | Best reasoning, configurable thinking budget |
| 3.7 / 3.6 (name a thinking level) | Mechanical transforms, script generation, strict format extraction | Direct and literal, holds output contracts, quicker to first token, less likely to over-engineer |
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

| Agent type | Provided by | `Bash` | `Edit`/`Write` |
|---|---|---|---|
| `context-finder` | **this plugin** (`agents/context-finder.md`) | no | no |
| `Explore` | the harness | **yes** | no |
| `general-purpose` | the harness | yes | yes |

**[verified]** against `agents/context-finder.md` and the harness agent roster.

**`context-finder` does not exist for everyone.** It ships with this plugin, so a config-only install — or an install into another tool's skills directory — has no such agent. Safety advice that names it silently fails there. Where it is absent, fall back to restricting the grant in your own agent definition rather than assuming a read-only type is available.

### Google Antigravity

Subagents are spawned programmatically by the main agent and run with **workspace isolation**, so their threads do not enter the main agent's context. Three flavours: built-in roles, generic clones that inherit the main agent's prompt and environment, and subagents registered on the fly with a goal the main agent defines.

The model is selectable **per agent**, which is what makes the role mapping above usable — Gemini 3 Pro, Claude Sonnet 4.5 and GPT-OSS are all selectable targets.

`/agents` opens the Agent Manager panel to browse custom agents and watch active or finished background subagents.

The vendor describes subagents as having "workspace isolation". **[vendor, unverified]** — whether that isolates the *filesystem* or only the *context* is not established here, and only the former weakens the tree-mutation hazard. Do not downgrade the guardrails in `SKILL.md` on the strength of that phrase until you have confirmed which it means.

### Harnesses without subagents

If a harness has no way to spawn a separate context that reports back, **only the downgrade axis applies**. The isolate axis needs a subagent primitive; without one, use the carve-outs and the reasoning-budget rules and ignore the rest.

Most of what you lose is context savings. One thing you lose is a **control**, not a saving: fresh-sub-agent code review and plan review exist for objectivity, and on such a harness they cannot run at all. Say so rather than treating a self-review as equivalent.

## Sources

- [Gemini 3.8 Flash — Google AI for Developers](https://ai.google.dev/gemini-api/docs/latest-model)
- [Introducing Gemini 3.8 Flash and 3.8 Flash Cyber](https://blog.google/innovation-and-ai/models-and-research/gemini-models/3-8-flash-and-3-8-flash-cyber/)
- [Antigravity: Subagents, Hooks, Scheduled Tasks, Agent Management](https://antigravity.google/blog/google-io-2026-feature-deep-dive)
- [Agents Command (/agents) — Antigravity Docs](https://antigravity.google/docs/cli/commands/agents/)
