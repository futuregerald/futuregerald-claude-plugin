---
name: future-model-router
description: Delegation and model routing — decide whether a piece of work runs in a sub-agent, and how much reasoning budget it gets. Provider-neutral; the model map is a reference file. Covers codebase search and exploration, debugging loops, call-chain tracing, spikes, CI log triage, and bulk queries against tickets, logs or metrics. Invoke before any search, multi-file read, or investigation that will produce far more output than answer.
tags: [delegation, model-routing, cost-optimization, search]
---

# Delegation & Model Routing

**Two decisions, not one.** Conflating them is the common error — "don't delegate debugging" usually means "don't *downgrade* debugging", which is a different claim.

1. **Isolate?** Will doing this in the main context pull in bulk the answer doesn't need? → run it in a sub-agent, **at any role, including the orchestrator's own model**.
2. **Downgrade?** Can you state the shape of a correct answer *before* dispatching? → **reduce the reasoning budget**. Two levers, usable together: a **smaller model**, and a **lower effort level** on whatever model you pick. If you can't state the shape, that's judgment, and judgment keeps its budget.

The two are independent. Work can be isolated without being downgraded.

|  | Stays at orchestrator | Downgrade |
|---|---|---|
| **Isolate (sub-agent)** | Debugging loops, call-chain tracing, spikes, CI log triage, bulk dataset queries | Symbol lookups, "does X exist", flow tracing, log/ticket/metrics sweeps |
| **Inline (main context)** | Decisions, synthesis over the conversation, writing in the user's voice, gate runs | Lower the effort level — the one downgrade available without leaving the context |

**The table is illustrative; the test governs.** Where a case is not in the table, or the table and the test disagree, apply the two questions above. The top-left cell is the one most setups miss. A debugging loop is tens of thousands of tokens of test output for a one-line root cause; isolating it is worth far more than downgrading it.

## The routing decision has a budget

**Two questions, answered from what you already know, with zero tool calls.** If you have to investigate to decide whether to delegate, the investigation *is* the work — do it inline and stop routing.

- **One pass, no deliberation.** Routing at high effort is the over-thinking failure this document warns about, applied to itself. If the answer is not obvious in one pass, it is a tie.
- **Ties go inline.** The costs are asymmetric: guessing wrong toward inline wastes some context, while guessing wrong toward delegation costs a full dispatch round trip *plus* the verification of whatever comes back.
- **Decide once per task, not per step.** Re-routing at every sub-step is where the tax compounds. Route when the task arrives; revisit only if its shape changes materially.

A routing decision that takes longer than the work it was routing has cost more than it saved, and nothing in the system will tell you that happened.

## Axis 1 — Isolate?

Isolate when the work **produces far more output than answer**:

- Debugging and test-failure iteration — run, read, hypothesize, re-run. The answer is "root cause is X at `file:line`".
- Call-chain tracing for an impact analysis — dozens of files read, a caller list returned.
- Spikes — throwaway code in a scratchpad that proves one fact.
- CI and build-log triage.
- Queries over large external datasets — tickets (JQL), logs, metrics, warehouse tables.
- Reading a large file or directory to answer a narrow question about it.

**Floor cost.** A dispatch is not free: it costs a prompt plus the agent's own reasoning. If **two tool calls with small output** would answer it, do it inline. Don't spawn an agent for what a single grep answers.

**Verification carve-out.** If trusting the answer would require reading the same bulk the agent read, isolating saved nothing. Do it inline, or change the question to one whose answer is checkable on its own (a `file:line`, a count, a diff).

## Axis 2 — Downgrade?

The test is **"can I state the shape of a correct answer before dispatching?"**

- *"Come back with the file path, line number, and every caller"* → shape is known → downgrade.
- *"Tell me whether this design holds up"* → no statable shape → keep at the orchestrator role.

| Role | Use for |
|---|---|
| **retriever** | You know the name. Path lookups, symbol definitions, "does X exist", listing, mechanical extraction into a known format |
| **explorer** | Multi-step exploration, tracing a flow, summarizing a long document, first-pass log triage, gathering ticket or PR data |
| **orchestrator** | Anything containing a judgment call — root-cause analysis, design questions, synthesis, review |

**Roles, not model names.** Which model fills each role — and whether your provider spends budget by swapping models or by lowering a thinking level — is in `references/model-map.md`. That file is the only place a model name appears, so a lineup change never edits the rules.

**Never trade output quality for a cheaper model.** Cost and speed are the tiebreak between options that both produce the answer you need, never a reason to accept a worse one.

**But more budget is not automatically better output.** On mechanical work, a high reasoning budget degrades the result: the model refactors code you did not ask it to touch, adds unsolicited error handling and commentary, and second-guesses a request that was already unambiguous. "Rename this variable" does not improve with deliberation — it gets embellished. Matching the budget to the task protects the output, not just the bill.

### Effort is the second lever

Effort — reasoning budget, thinking level, whatever the harness calls it — is set independently of the model, and it behaves differently from swapping models in two ways worth knowing:

- **It works inline.** You cannot change your own model mid-session, but you can spend less deliberation on a routine turn. It is the only downgrade available without dispatching.
- **It is itself a context and quota cost.** Reasoning tokens are output tokens: they accumulate in the transcript that produced them and they count against rate limits. The spread between the lowest and highest setting is an order of magnitude or more on the same prompt, so this is the largest single multiplier in the whole system — larger than the choice of model.
- **It changes behaviour, not just depth.** A low setting is more literal and more likely to do exactly what was asked. A high setting deliberates, and deliberation on an unambiguous request turns into scope it invented.

Match effort to the same test: a statable answer shape means low effort will reach it, and will reach it more faithfully. Reserve high effort for the judgment calls that keep their full budget anyway — genuine root-cause work, concurrency, architecture.

Where a harness sets effort per agent definition rather than per dispatch, set it there — an agent whose whole job is mechanical retrieval should not be defined at high effort.

**A non-reasoning model is a third option.** Where the lineup still offers an older model that does not deliberate at all, it can beat the newest model at its lowest setting for strict-format work — mechanical transforms, format extraction, regex, anything where breaking out of the output contract to explain itself is the failure mode. See `references/model-map.md`.

**Escalate once, don't retry.** A vague retriever result goes to the explorer role; a vague explorer result comes back to the orchestrator. Never re-dispatch at the same tier.

## Never delegate

1. **Work whose input is the conversation.** Synthesis, decisions, "what should we do". A sub-agent starts at zero; re-supplying the context costs more than doing the work.
2. **Anything written in the user's voice.** Tickets, PR bodies, docs, messages. Sub-agents drift toward generic phrasing.
3. **Gate runs.** Tests, lint, CI — the evidence backing a completion claim has to be in the orchestrator's own transcript. A sub-agent reporting "tests pass" is a claim, not evidence.
4. **Work that fails the verification carve-out above.**

This list is **duplicated on purpose** in `CLAUDE.md` (the "Delegate by Default" section), because the rule has to fire before any skill loads and config-only installs ship no skills. The two copies must be edited together — changing one alone is a silent drift.

Code review and plan review already run in fresh sub-agents for a different reason — objectivity, not cost. That requirement is unaffected by anything here.

## Safety: isolated agents hold write tools

An investigation agent with `Edit`, `Write` or `Bash` can mutate the working tree. This has happened — a review agent reverted a worktree mid-session.

- **Check the agent type's real tool grant before trusting it to be read-only.** Dropping `Edit`/`Write` does not imply dropping `Bash`, and an agent with `Bash` can still run `git checkout`, `reset --hard` or `rm`. Per-harness grants are in `references/model-map.md`; in Claude Code, `context-finder` is the real guarantee and `Explore` is **not** read-only.
- Where the agent holds `Bash` at all — or the work genuinely needs to run a test suite — instruction is the only lever left, so layer it: **commit first**, point the agent at a SHA, and forbid `checkout`, `stash`, `reset` and file edits in the prompt, every time.
- Where the harness gives subagents their own isolated workspace, this hazard is weaker — but commit-first costs nothing and still protects a subagent pointed at the shared tree.

## Dispatching

**Write self-contained prompts.** The sub-agent has zero conversation context, and it does not necessarily inherit your MCP servers either — a child without a tool must not call it or claim it did. State what you want, why, the evidence you already have, and the exact output format: paths, line numbers, signatures, call chains.

The prompt shape below is portable; the call syntax is Claude Code's. Your harness's dispatch syntax is in `references/model-map.md`.

```
// Isolated, NOT downgraded — bulk work, judgment required → orchestrator role
Agent({ model: "opus", subagent_type: "Explore",
  prompt: "The spec at spec/interactors/x_spec.rb:40 fails with NoMethodError.
           Run it, read the failure, trace the cause. Report ONLY: the root cause
           as file:line, the failing assertion, and a one-paragraph explanation.
           Do not edit any file. Do not run git checkout, stash, or reset." })

// Isolated AND downgraded — answer shape is stated up front → retriever role
Agent({ model: "haiku", subagent_type: "Explore",
  prompt: "Find the definition of InvoiceSerializer. Report the file path and line number only." })
```

**Parallelize only genuinely independent questions.** Agents cannot see each other's work, so three agents over overlapping paths read the same files three times. When several questions share a subject, send **one** agent with a multi-part prompt instead of N agents.

## After a sub-agent returns

Sub-agent output is **evidence, never a completion claim**.

1. Check the result against what you already know.
2. Spot-check that cited paths and line numbers exist.
3. Read the key file directly if the summary looks thin.
4. In the answer to the user, mark which claims came from a sub-agent and which you verified yourself.
