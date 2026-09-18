---
name: future-model-router
description: Delegation and model routing — decide whether a piece of work runs in a sub-agent, and which model runs it. Covers codebase search and exploration, debugging loops, call-chain tracing, spikes, CI log triage, and bulk queries against tickets, logs or metrics. Invoke before any search, multi-file read, or investigation that will produce far more output than answer.
tags: [delegation, model-routing, cost-optimization, search]
---

# Delegation & Model Routing

**Two decisions, not one.** Conflating them is the common error — "don't delegate debugging" usually means "don't *downgrade* debugging", which is a different claim.

1. **Isolate?** Will doing this in the main context pull in bulk the answer doesn't need? → run it in a sub-agent, **at any model, including Opus**.
2. **Downgrade?** Can you state the shape of a correct answer *before* dispatching? → Haiku or Sonnet. If you can't, that's judgment, and judgment stays at the orchestrator's model.

The two are independent. Work can be isolated without being downgraded.

|  | Stays at Opus | Downgrade to Haiku/Sonnet |
|---|---|---|
| **Isolate (sub-agent)** | Debugging loops, call-chain tracing, spikes, CI log triage, bulk dataset queries | Symbol lookups, "does X exist", flow tracing, log/ticket/metrics sweeps |
| **Inline (main context)** | Decisions, synthesis over the conversation, writing in the user's voice, gate runs | — |

**The table is illustrative; the test governs.** Where a case is not in the table, or the table and the test disagree, apply the two questions above. The top-left cell is the one most setups miss. A debugging loop is tens of thousands of tokens of test output for a one-line root cause; isolating it is worth far more than downgrading it.

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
- *"Tell me whether this design holds up"* → no statable shape → keep at Opus.

| Model | Use for |
|---|---|
| **Haiku** | You know the name. Path lookups, symbol definitions, "does X exist", listing, mechanical extraction into a known format |
| **Sonnet** | Multi-step exploration, tracing a flow, summarizing a long document, first-pass log triage, gathering ticket or PR data |
| **Opus** | Anything containing a judgment call — root-cause analysis, design questions, synthesis, review |

**Never trade output quality for a cheaper model.** Cost and speed are the tiebreak between options that both produce the answer you need, never a reason to accept a worse one.

**Escalate once, don't retry.** A vague Haiku result goes to Sonnet, a vague Sonnet result comes back to the orchestrator. Never re-dispatch at the same tier.

## Never delegate

1. **Work whose input is the conversation.** Synthesis, decisions, "what should we do". A sub-agent starts at zero; re-supplying the context costs more than doing the work.
2. **Anything written in the user's voice.** Tickets, PR bodies, docs, messages. Sub-agents drift toward generic phrasing.
3. **Gate runs.** Tests, lint, CI — the evidence backing a completion claim has to be in the orchestrator's own transcript. A sub-agent reporting "tests pass" is a claim, not evidence.
4. **Work that fails the verification carve-out above.**

This list is **duplicated on purpose** in `CLAUDE.md` (the "Delegate by Default" section), because the rule has to fire before any skill loads and config-only installs ship no skills. The two copies must be edited together — changing one alone is a silent drift.

Code review and plan review already run in fresh sub-agents for a different reason — objectivity, not cost. That requirement is unaffected by anything here.

## Safety: isolated agents hold write tools

An investigation agent with `Edit`, `Write` or `Bash` can mutate the working tree. This has happened — a review agent reverted a worktree mid-session.

- **Know which agent types are actually read-only.** `context-finder` holds no `Bash`, `Edit` or `Write` — it physically cannot mutate anything, and that is a tool-level guarantee. **`Explore` is not read-only**: it drops `Edit`/`Write` but **keeps `Bash`**, so it can still run `git checkout`, `reset --hard` or `rm`. Check the grant before you rely on it.
- Where the agent has `Bash` at all — `Explore`, or work that genuinely needs to run a test suite — instruction is the only lever left, so layer it: **commit first**, point the agent at a SHA, and forbid `checkout`, `stash`, `reset` and file edits in the prompt, every time.

## Dispatching

**Write self-contained prompts.** The sub-agent has zero conversation context. State what you want, why, and the exact output format — paths, line numbers, signatures, call chains.

```
// Isolated but NOT downgraded — bulk work, hard reasoning
Agent({ model: "opus", subagent_type: "Explore",
  prompt: "The spec at spec/interactors/x_spec.rb:40 fails with NoMethodError.
           Run it, read the failure, trace the cause. Report ONLY: the root cause
           as file:line, the failing assertion, and a one-paragraph explanation.
           Do not edit any file. Do not run git checkout, stash, or reset." })

// Isolated AND downgraded — known answer shape
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
