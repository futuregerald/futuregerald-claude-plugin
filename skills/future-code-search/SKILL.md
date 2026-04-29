---
name: future-code-search
description: Model routing rules for codebase search. Delegates search/exploration to cheaper models (Haiku/Sonnet) while keeping Opus as the orchestrator. Invoke this skill before any codebase search or exploration task.
tags: [search, cost-optimization, model-routing]
---

# Future Code Search — Tiered Model Routing

**Never use Opus for raw codebase exploration.** When you need to search, explore, or read code to gather context, delegate to a cheaper model via the `Agent` tool. Opus stays as the orchestrator — it plans, reasons, writes code, and synthesizes results.

## Routing Table

| Task | Model | When to use |
|------|-------|-------------|
| Simple file/symbol lookup | `haiku` | You know the name — just need the path or definition |
| Multi-step codebase exploration | `sonnet` | Tracing flows, understanding features, exploring patterns |
| Architecture overview / broad search | `sonnet` | Cross-package dependencies, high-level structure questions |
| Planning, reasoning, code writing | `opus` (default) | Implementation, PR review, debugging decisions, synthesis |

## How to Dispatch

### Simple lookup (Haiku)

```
Agent({
  model: "haiku",
  subagent_type: "Explore",
  prompt: "Find all files matching **/finding_serializer*.rb and report their paths and line counts"
})
```

### Deep exploration (Sonnet)

```
Agent({
  model: "sonnet",
  subagent_type: "Explore",
  prompt: "Trace how a finding is created: from the API controller through any interactors to the database. Report the full call chain with file paths and key method names."
})
```

### Parallel searches (multiple agents)

When you need multiple independent pieces of context, dispatch them in parallel in a single message:

```
// Both in one message — they run concurrently
Agent({ model: "haiku", subagent_type: "Explore", prompt: "Find the PentestSerializer definition..." })
Agent({ model: "sonnet", subagent_type: "Explore", prompt: "Trace the pentest creation flow..." })
```

## Rules

1. **Write specific, self-contained prompts** — the sub-agent has zero conversation context. Include what you're looking for, why, and what format you want the answer in.

2. **Use Grep/Glob directly for trivial lookups** — if you already know the file name or exact symbol, don't spawn an agent. Just grep for it. Agents are for multi-step exploration.

3. **Escalate on insufficient results** — if a Haiku agent returns vague or incomplete context, re-dispatch with Sonnet before trying the same tier again.

4. **Opus reads files directly when it knows the target** — if you already have the exact path and line range (from a previous search or the user), use `Read` directly. No agent needed for targeted reads.

5. **Parallelize independent searches** — if you need 3 different pieces of context, dispatch 3 agents in one message. Don't serialize them.

6. **Include output format in the prompt** — tell the agent exactly what to return: file paths, line numbers, method signatures, call chains. Structured output is easier to synthesize.

## Do NOT Delegate to Cheaper Models

These tasks require Opus-level reasoning and must stay with the orchestrator:

- Code writing or editing
- Code review, security review, SQL review
- Planning or architectural reasoning
- Synthesizing results from multiple searches into a decision
- Debugging decisions (root cause analysis)
- Writing commit messages, PR descriptions, or documentation

## Escalation Path

```
Can't find it? ──> Was it Haiku? ──yes──> Retry with Sonnet
                        │
                       no (already Sonnet)
                        │
                        v
               Opus reads files directly
               (fall back to manual search)
```

## Quality Check

After receiving search results from a sub-agent, the orchestrator (Opus) should:

1. Verify the results make sense given what you already know
2. Check that file paths mentioned actually exist (quick Glob if uncertain)
3. Read key files directly if the summary seems incomplete
4. Only then proceed with planning/implementation
