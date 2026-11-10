# Model Map

The rules in `SKILL.md` name **roles**, never models. This file is the only place a model name appears, so a new lineup is a one-file edit that never touches the reasoning.

**Verify before relying on a row.** Lineups change every few months. Last checked: 2026-11-10.

**A lineup discrepancy you will hit immediately.** Antigravity IDE (`swarm{action: "models", all: true}`) exposes `gemini-4-carbon-{low,medium,high}`, `gemini-4-argon-{low,medium,high}`, `gemini-3.8-flash-{low,medium,high}`, `claude-opus-5-5-{low,medium,high}`, `claude-sonnet-5-5-{low,medium,high}`, and `gpt-oss-120b-medium`, whereas `invoke_subagent` only exposes the 4-slot enum `Model: "inherit" | "pro" | "flash" | "flash_lite"`. Meanwhile, the Gemini CLI on the machine where the CLI numbers below were taken (v0.58.0) exposes `gemini-3.5-flash`, `gemini-3.5-flash-lite` and `gemini-3.1-flash-lite`; `gemini-3.5-pro` and `gemini-3-pro` both return `ModelNotFoundError`, and `-m gemini-3.1-flash` silently resolves to `gemini-3.5-flash`. So the chat/IDE lineup, the `invoke_subagent` enum, and the CLI lineup are **not the same set**. Check what your entry point actually accepts before copying a model name out of this file.

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

| Role | Anthropic | Google (Antigravity) |
|---|---|---|
| retriever | Haiku 4.5 | **Flash-Lite** (`Model: "flash_lite"`) **only for processing pre-fed input** (max 3 turns, never code discovery) **[verified]**; otherwise `gemini-3.8-flash-low` (`Model: "flash"` or 3.7/3.6 `low` where exposed) **[reported]** |
| explorer | Sonnet 5.5 (`low` / `medium`) | **Gemini 3.8 Flash** (`Model: "flash"`, `low` or `medium`, max 15 turns) for bounded search; **Gemini 4 Argon (`medium`)** (`Model: "inherit"` or `gemini-4-argon-medium` in `swarm`) for TDD stream implementation **[verified]** |
| orchestrator | Opus 5.5 (`medium` default; `high` for complex architecture & adversarial review) | **Gemini 4 Carbon** (`medium` default for planning/orchestration; `high` for net-new multi-layer architecture/migrations and adversarial review; **Gemini 4 Argon** as execution/quota fallback) **[verified roster; Carbon vs. Argon turn/cost curve unmeasured]** |

**The two platforms lean on different levers, and that is deliberate.** On Anthropic the model changes per role. On Google the thinking level on one model covers explorer and orchestrator by itself — **and the model lever is available too, for subagents**, where Flash-Lite is selectable. So the retriever role there can drop a model tier rather than only a thinking level. Both platforms expose both levers; they differ in which one is the default reach.

### Measured on Gemini CLI — read the scope before the numbers

Three retriever-shaped questions, same repo and same ground truth used for the Anthropic
numbers in `SKILL.md`, run through Gemini CLI 0.58.0 read-only, n=2 per arm. **[verified]**

**Scope, stated first because it bounds what these numbers mean.** The two arms are Flash and
**Flash-Lite**, and **Flash-Lite is selectable for Antigravity subagents [reported]** — so this
pair *is* a routing option there, which makes the accuracy findings below the operative ones
rather than the cost ones.

Two caveats on transferring the numbers. First, this is the **CLI's** Flash-Lite, not
Antigravity's: different harness, different scaffolding, and a model's tool-calling behaviour is
shaped by both. Treat the direction as transferable and the magnitudes as not. Second, **the
thinking-level lever is still unmeasured** — Gemini CLI 0.58.0 has no thinking/effort flag and
`~/.gemini/settings.json` has no model block, so it cannot be set from a terminal. The Google
column therefore has one lever measured (model swap) and one not (thinking level).

| Arm | Total tokens | Tool calls | API latency | Score |
|---|---|---|---|---|
| `gemini-3.5-flash` | ~1,950,000 | 38 | ~170s | 18/19 |
| `gemini-3.5-flash-lite` | ~658,000 | 26 | ~19s | 17/19 |

The lighter model was **3.0x cheaper in tokens and roughly 9x faster** — the opposite direction
from the Anthropic arms, where the lighter model cost 2.03x more and ran 1.86x slower. So **a
lighter model can go either way on cost: the direction is a property of the pair, not of the
rule.** Measure the pair you intend to use.

**Where Flash-Lite lost its points matters more than the total, and the total hides it.** Both
arms dropped the counting question, so set that aside and look at the rest:

| Run | Location answers (`file:line`) |
|---|---|
| flash, both runs | **perfect** |
| flash-lite run 1 | one line number off by 2 (`1911` for `1913`) |
| flash-lite run 2 | **handler placed in the wrong file entirely** — `main.js:1298` for `ipcHandlers.js:6957` |

**Flash-Lite erred in exactly the deliverable a retriever exists to produce.** Flash erred only on
the count. 17-versus-18 reads as a rounding difference; "names the wrong file, confidently, in one
run out of two" does not. It also fails in the worst way for this role — a wrong path is checkable
in seconds by anyone who looks, and silently wrong to anyone who does not.

Three further things that showed up, all of them on the **Flash** arm as well, and all of which
any Gemini routing advice has to account for:

1. **Neither Gemini arm could count.** Asked how many times one token appears in one file
   (true answer 331, confirmed five ways), the four runs answered 349, 338, 247 and 139. Both
   Anthropic arms answered 331, every run. Gemini leaned on a match-*listing* search tool
   (17–29 calls a run) and counted the results by eye; it ran a shell command at all in only
   one run of four. **The filter-at-the-source rule in `SKILL.md` is therefore a correctness
   control here, not just a cost one.** Say "run `grep -c`", not "count the matches".
2. **`-m` does not pin the serving model.** Both `gemini-3.5-flash` runs silently served part
   of the work from `gemini-3-flash-preview` — 9 of 43 requests in one run.
3. **It went off-task.** One run made two web searches while answering questions about a local
   file; both flash-lite runs made unrelated `update_topic` calls. No Anthropic arm made an
   off-task call.

**Flash-Lite: usable as a narrow retriever subagent, never as an orchestrator. [reported +
measured]** It is selectable for Antigravity subagents and is genuinely much cheaper and faster.
As a *chat* or orchestrating model it stays too weak at tool-calling and code quality — that half
of the original guidance holds.

**The line is processing versus finding, and it is finer than "simple work".** Flash-Lite is for
bounded, mechanical work on material *you hand it*, where the output shape is rigid and cheap to
check. It is not for going and looking.

**Use it for:**

- **Distilling output you already captured** — pulling the failing specs, their `file:line` and
  error messages out of a 2,000-line test log. It does not need to know *why* they failed.
- **Deterministic reshaping** — structured data into a table, a changelog, a fixed schema.
- **Boilerplate from an exemplar you point at** — "using this spec as the template, generate the
  cases listed in that file".
- **Binary checks across named files** — "is this flag true or false in each of these five".
- **Non-technical summarising and translation**, where nothing turns on code causality.

**Do not use it for:** code discovery or "where is X defined"; adversarial review, where it
rubber-stamps with generic praise **[reported]**; writing implementation code; or root-cause
debugging, where it fixates on the symptom. **And never hand it a counting question** — tell it to
run a counting primitive and report the number, never to count what it sees.

**Three questions before you dispatch it.** All three must be yes:

1. **Can you pre-feed it the exact input?** If it has to hunt, use the bigger model.
2. **Is the output format strictly mechanical?** Table, list, key-value, extracted lines.
3. **Can you verify the result without re-reading the raw input?** If checking it means redoing
   the work, isolating to a cheap model saved nothing.

Question 3 is the verification carve-out from `SKILL.md`, reached independently — which is some
evidence the carve-out is the real constraint and not a stylistic preference.

**Why "no code discovery" is the firm one:** measured, its errors were both *location* errors —
a line number off by two, and a handler placed in the wrong file entirely — while every stronger
model was perfect on the same questions. A wrong path is the deliverable, and it propagates into
whatever plan, edit or ticket consumes it.

**What changed, and what did not. [reported + measured]** The original guidance rejected
Flash-Lite partly on availability and partly on capability. **Availability is no longer a
reason** — it is selectable for subagents — and **cost is not a reason either**, since it is the
cheaper and faster arm. What survives is capability, sharpened: not an orchestrator, not a finder,
fine as a processor of input you hand it.

**One divergence worth recording, because it runs against the operator guidance.** That guidance
says Flash-Lite burns 8–10 tool rounds where Flash takes 2, making it *slower* in practice. The
runs here measured the opposite on Gemini: Flash-Lite used **26** tool calls to Flash's **38**,
and finished roughly 9x faster. The 2.5x-more-rounds pattern *was* observed — but on the other
platform, where the smaller model took 10 calls to the larger model's 4. So "the cheap model
flails and costs you more" is real, and is not universal. **It is a property of the pair.**
Flash-Lite's problem in these runs was not effort spent; it was confident wrong answers, cheaply
and quickly.

**The retriever row is the exception to "hold the model". [reported]** 3.7 and 3.6 also expose thinking levels, so **always name a level when you name one of them** — they are a different model, not a non-reasoning one. At `low` they earn their place for mechanical work: more literal, better at holding a strict output format instead of breaking out to explain themselves, and quicker to first token. For a format extraction, a regex, or a rename, that is better behaviour than 3.8 at `low` — not merely cheaper.

### Two levers, on both platforms

Model and effort are set independently, and both platforms expose both — they just default to different ones. Prefer cutting effort first when the task is well-scoped but non-trivial: it keeps the larger model's knowledge while spending less on deliberation.

| Platform | Model lever | Effort lever |
|---|---|---|
| Anthropic / Claude Code | `model:` on the dispatch, or in the agent definition | **`effort:`** in the agent definition — `low`, `medium`, `high`, `xhigh`, `max`, overriding the session level; and **`claude --effort <level>`** for a whole session. **[verified]** — named in the subagent frontmatter reference and exercised through both routes |
| Google / Antigravity | Model selectable per agent | Thinking level on Gemini 3.8 Flash: `low`, `medium`, `high`; default `medium`. **[vendor, unmeasured]** — and not settable from Gemini CLI 0.58.0, which has no thinking/effort flag and no model block in `settings.json`, so it cannot be benchmarked from a terminal |

This is why the skill says **"reduce the reasoning budget"** rather than "use a smaller model".

**Harness gotcha, measured:** a subagent definition dropped into the agents directory
mid-session is **not** picked up — the registry loads at session start, and a dispatch to the
new type fails with "Agent type not found" listing only the types that existed then. To
exercise a new definition without restarting, run it headless: `claude -p "<task>" --agent
<name> --output-format json`, which also reports `num_turns`, `duration_ms`,
`total_cost_usd` and `usage.output_tokens_details.thinking_tokens` — the instrumentation these
measurements were taken with.

### What the effort lever actually costs

Thinking tokens are **output tokens**. They count against rate limits and rolling quotas, and the input price does not change between settings — only the volume generated does. Approximate, for Gemini 3.8 Flash:

| Level | Typical thinking tokens | Relative burn |
|---|---|---|
| `low` | ~250 – 1,000 | 1x |
| `medium` (default) | ~1,000 – 4,000 | 2–4x |
| `high` | ~8,000 – 16,000+ | 5–15x+ |

The same prompt can produce a few hundred output tokens at `low` and five figures at `high`. The bracket midpoints imply closer to 15–20x than the 5–15x quoted alongside them, so treat both as order-of-magnitude, not arithmetic.

**Measured against those brackets.** On a mechanical single-parameter rename, thinking tokens
came back at **0–215 across eight runs**, at `low` *and* at `high`, on both the session flag
and the agent field. The brackets above describe what a task with genuine deliberation in it
costs; they are not what an unambiguous task costs at a high setting. Effort cannot spend a
budget on a question that has nothing to think about, so the burn table is an upper envelope,
not a per-turn expectation.

**Scope of the claim:** where one model is held across several budgets — the Google column — effort is the dominant multiplier. It is *not* larger than the model lever on a platform whose tiers are separate models with different unit prices; there the two are spent differently and are not comparable on one scale. When a rolling limit drains faster than expected in a long session, effort is still the first thing to check, because it moves without you choosing it.

### Choosing medium vs high on Gemini Flash **[reported]**

Operator guidance, not measured here. Recorded because it fills the gap the CLI could not reach
— the thinking level is unsettable from a terminal, so this is the only account of that lever.

**Stay on `medium` by default; it is the right baseline for roughly 85% of work.** Reach for
`high` only where the question is *"what did we fail to consider?"*

| Stay on `medium` | Switch to `high` |
|---|---|
| Executing an approved plan — TDD tasks, implementing to a spec | **Adversarial plan review** — imagining invisible failure modes: dynamic dispatch, unindexed callers, concurrency traps, contract changes existing tests still pass |
| Mechanical refactoring — extract, rename, restructure, split | **Root-cause work on subtle bugs** — races, deadlocks, transaction-isolation quirks, state corruption surfacing three services downstream |
| Targeted discovery — file reads, greps, graph queries, glue scripts | **Architecture, migrations, zero-downtime schema changes** — ordering constraints, locks, backward compatibility, severe rollback risk |
| Interactive pairing, where latency is the point | **Intricate algorithmic or parser logic** — where one branch condition or off-by-one breaks everything |

Time-to-first-token is reported as 2–4x faster on `medium`, which is most of why it is the
default for conversational work.

**This is the skill's own test, arrived at independently.** The rule of thumb offered — *use
`medium` when the shape of the solution is already known; go `high` when the question is what
you failed to consider* — is the statable-answer-shape test from `SKILL.md`, applied to the
effort lever on a different platform. That convergence is the main reason this section is worth
keeping: two routes to the same discriminator.

**One corroboration and one divergence against what was measured here.**

- **Corroborates:** the guidance states that on mechanical work a higher effort setting yields
  the identical code diff. That is exactly what eight runs of a seeded rename found on the other
  platform — identical four-line diff at `low` and `high`, refactor bait untouched. Two
  platforms, two instruments, same result. This is the strongest evidence in this document that
  **effort does not change mechanical output**, and it is why that claim was removed from
  `SKILL.md` rather than merely softened.
- **Diverges:** the guidance says high roughly *doubles latency* on mechanical work. The runs
  here found no reliable latency difference — the `high` arm's own two runs differed by 2.4x,
  more than the two levels differed from each other. Both can hold: where one model spans
  several budgets, effort is the dominant multiplier, whereas separate model tiers spend it
  differently. Do not carry a latency figure across platforms.

### Gemini lineup, by what it is good at

| Model | Best for | Why |
|---|---|---|
| **Gemini 4 Carbon** (`low` / `medium` / `high`) | Planning (`medium`; `high` for ~10% complex architecture/migrations) and adversarial plan/code/security/SQL/editorial review (`high` or `Model: "inherit"` with high-deliberation prompt) **[verified roster; cost/turn curve vs. Argon unmeasured]** | Newest frontier reasoning tier in Antigravity; shares quota reset window with Argon and 3.8 Flash |
| **Gemini 4 Argon** (`low` / `medium` / `high`) | Phase 4 TDD implementation (`medium`), parallel `swarm` stream authors (`gemini-4-argon-medium`), and primary session fallback **[verified]** | Measured `27/28 (96.4%)` vs. Flash 3.8 `23/28 (82.1%)` with 5.0× fewer turns and 5.93× fewer input tokens |
| **Gemini 3.8 Flash** (`low` / `medium` / `high`) | Bounded code search (`<=15` turns), `<50 LOC` mechanical boilerplate, and `swarm_monitor` stall cron (`gemini-3.8-flash-low`) | Cheap per-token rate for bounded mechanical tasks; flails on ambiguous CLI errors and nuanced regex/timeout invariants **[verified]** |
| **3.7 / 3.6** (name a thinking level, where exposed) | Mechanical transforms, script generation, strict format extraction **[reported]** | Direct and literal, holds output contracts, quicker to first token |
| **Claude 5.5 Opus / Sonnet & GPT-OSS 120B** | A second opinion, architecture review, separate quota pool | Separate rolling quota reset window when Google pool is congested or vice versa **[verified]** |

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

| Agent type | Provided by | `Bash` | `Edit`/`Write` | Network |
|---|---|---|---|---|
| `context-finder` | **this plugin** | no | no | no |
| `investigator` | **this plugin** | no | no | no |
| `reviewer` | **this plugin** | no | no | no |
| `runner` | **this plugin** | **yes** | no | forbidden by prompt only |
| `writer` | **this plugin** | **yes** | **yes** | forbidden by prompt only |
| `Explore` | the harness | **yes** | no | — |
| `general-purpose` | the harness | **yes** | **yes** | — |

`runner` and `writer` hold `Bash`, so their restrictions on destructive git, `rm` and network access are **instructions, not enforcement**. Treat them as the mitigation layer, not the control.

**[verified]** against `agents/context-finder.md` and the harness agent roster.

**`context-finder` does not exist for everyone.** It ships with this plugin, so a config-only install — or an install into another tool's skills directory — has no such agent. Safety advice that names it silently fails there. Where it is absent, fall back to restricting the grant in your own agent definition rather than assuming a read-only type is available.

### Google Antigravity

Subagents are spawned programmatically by the main agent and run with **workspace isolation**, so their threads do not enter the main agent's context. Three flavours: built-in roles, generic clones that inherit the main agent's prompt and environment, and subagents registered on the fly with a goal the main agent defines.

The model is selectable **per agent**, which is what makes the role mapping above usable — Gemini 3 Pro, Claude Sonnet 4.5 and GPT-OSS are all selectable targets.

#### Measured on Antigravity (3-Arm TypeScript/Go Controlled Study + 46-Subagent Production Corpus, 2026-11-07) **[verified]**

1. **`Model: "inherit"` vs. `Model: "pro"` on Argon Sessions (`invoke_subagent`):**
   - `invoke_subagent` only accepts `Model: "inherit" | "pro" | "flash" | "flash_lite"`.
   - When the parent session is set to **Argon** (`Gemini 4 Argon (Medium)` or `(High)`), **`Model: "inherit"` is the ONLY value that runs Argon in the subagent.**
   - Passing `Model: "pro"` routes to the fixed Gemini 3.1 Pro / 3 Pro slot (`pipeline_beta` planner `2a54de88`), which took **34 turns / 2.01M input tokens** vs. Argon `Model: "inherit"` (`pipeline_alpha` planner `7eee7623`) at **13 turns / 955k input tokens**, and missed 7 repo/spec invariants (`20/28` vs `27/28` on the 84-assertion adversarial test suite).

2. **Argon Medium (`Model: "inherit"`) vs. Gemini 3.8 Flash (`Model: "flash"`) Implementer on the Exact Same Plan (`pipeline_gamma` vs `pipeline_alpha`):**

| Arm (Identical 549-line `PLAN.md`) | Turns | Shell Calls | Wall Clock | Cumulative Input Tokens | Output Tokens | Adversarial Score (28 checks) |
|---|---:|---:|---:|---:|---:|---:|
| **Argon Medium (`Model: "inherit"`, `5f5fbbc4`)** | **15** | **5** | **550.0s** | **1,309,954** | **44,099** | **27 / 28 (96.4%)** |
| **Gemini 3.8 Flash (`Model: "flash"`, `c11a4297`)** | **75** | **53** | **683.0s** | **7,766,269** | **67,751** | **23 / 28 (82.1%)** |

   - **Why Flash 3.8 cost 5.93× more input tokens and took 5× more turns:** When `npx vitest run` filtered out the benchmark directory due to `extension/vitest.config.ts`, Argon Medium inspected the config and passed `--root` on its 2nd command (15 turns total); Flash 3.8 flailed for 45 consecutive turns and 53 shell commands. Across 19 historical production streams (TypeScript + Go), Argon Medium was also **19.2% faster on mean wall-clock time** (`886.8s` vs `1,098.2s`, with Flash 3.8 hitting a 320-turn / 87.4-min runaway in `5593a7ac`).
   - **Where Flash 3.8 failed on correctness (`A10`, `A11`, `C8`, `C9`):**
     - Checked `/\b(500|502|503|504)\b/` *before* `QuotaExceededError` / `SyntaxError` / `OperationError`, misclassifying quota (`"5000KB"`) and syntax (`"position 500"`) errors as retryable HTTP 500 errors (`A10`).
     - Used `setTimeout(() => controller.abort())` without `Promise.race`, hanging forever when `pushFn` or `fetchFn` ignored `AbortSignal` (`A11`, `C8`).
     - Retained `${parsed.search}` (`?token=secret`) in `sanitizedUrl` (`C9`).
   - **Price-per-prompt parity:** Because Flash 3.8 burned **5.93× more input tokens** and **1.54× more output tokens** (plus extra Phase 8 `fix-check` rework rounds), Argon Medium's total cost per completed prompt is roughly equal to or **lower** than Flash 3.8 even at a 4×–5× per-token price ratio.
   - **Planner vs. Plan Reviewer Effort (`Argon Medium` Planner + `Argon High` Reviewer):** Across 6 historical plans drafted at Medium thinking, a single read-only pass of `Argon High` `plan-review` caught **13 CRITICAL and 20 IMPORTANT** cross-boundary bugs before coding. Keeping the Planner on `Argon Medium` (for ~90% of work) + `Argon High` on `plan-review` achieves the 96.4% ceiling without paying the High thinking tax across the 8 non-planning phases of the Lead session.

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
