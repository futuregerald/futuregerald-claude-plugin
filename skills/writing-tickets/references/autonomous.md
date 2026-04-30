# Autonomous Mode

Run the full skill end-to-end without user input. Self-grill, self-answer, log assumptions.

## When to Use

Triggered by `--autonomous` flag or when the user says "write this autonomously" / "don't ask me questions" / "use your best judgment".

## Flow Differences

| Phase | Interactive | Autonomous |
|-------|------------|------------|
| Investigation | Same | Same |
| Grill | Ask user, wait for answers | Self-grill, self-answer via investigation |
| Unknowns | Ask user first | Log as assumption with confidence |
| Writing | Same | Same |
| Review | Same | Same |
| Output | Present, ask to post | Present with assumptions section, ask to post |
| Sub-tasks | Offer after approval | Draft alongside parent, present all at once |

## Self-Grilling

Run through every question from [grilling.md](grilling.md) for the ticket type. For each question, answer it yourself using investigation findings.

**Format each self-grill answer:**
```
**[Self-grill Q{N}]: {Question}**
Answer: {Your answer}
Confidence: {High|Medium|Low}
Basis: {What evidence supports this -- file path, ticket key, metric, etc.}
```

**Confidence definitions:**
- **High:** Direct evidence from code, ticket, or metric. You read the file, you saw the data.
- **Medium:** Inferred from patterns, similar code, or related tickets. Consistent but not directly confirmed.
- **Low:** Best guess. Limited or no direct evidence. Could be wrong.

## Dispatching Sub-Agents for Answers

When a self-grill question can't be answered from what's already known, dispatch a sub-agent to investigate before answering. Don't guess when you can look.

```
Agent(sonnet): "I need to answer: '{grill question}'.
Investigate {specific area} to find evidence.
Check: {files/tickets/metrics to look at}.
Return: findings with confidence assessment."
```

**Decision tree:**
1. Can I answer from investigation already done? --> Answer directly.
2. Can a sub-agent find the answer? --> Dispatch, then answer.
3. Neither? --> Log as Low confidence assumption.

## Assumption Logging

Every Medium and Low confidence answer becomes an assumption in the final ticket.

**In the ticket output, add an Assumptions section:**
```markdown
## Assumptions

| # | Assumption | Confidence | Basis |
|---|-----------|------------|-------|
| 1 | {What was assumed} | Medium | {Why -- what evidence exists} |
| 2 | {What was assumed} | Low | {Why -- what we couldn't verify} |

> Low-confidence assumptions should be validated before implementation begins.
> Consider creating spike tickets for assumptions that carry implementation risk.
```

## Spike Ticket Drafting

In autonomous mode, automatically draft spike tickets for:
- Any Low confidence assumption that affects implementation approach
- Any question where investigation found contradictory evidence
- Any area where the codebase has no tests and the change is non-trivial

**Spike tickets follow the spike template from [writing-style.md](writing-style.md).**

Present spike tickets alongside the main ticket:
```
**Main ticket:** [Epic/Story title]
**Spike tickets drafted (2):**
1. Spike: {Question to answer} -- spawned from assumption #2
2. Spike: {Question to answer} -- spawned from contradictory findings in investigation
```

## Context Management

Autonomous mode generates more internal reasoning. Keep the orchestrator lean:

- **Delegate all investigation to sub-agents** -- don't read files directly in the orchestrator
- **Sub-agent results should be summarized** -- return findings, not raw file contents
- **Self-grill log stays internal** -- only assumptions appear in the final ticket
- **Don't re-investigate** -- if a sub-agent already found the answer, use it

## Final Output

Present everything at once:

1. **The ticket** (formatted per writing-style.md, with Assumptions section)
2. **Spike tickets** (if any)
3. **Investigation summary** (brief: what was checked, what tools were available/unavailable)
4. **Self-grill log** (collapsed or summarized -- user can expand if they want to see the reasoning)

Then ask: "Want me to create these in Jira?" (unless `--direct` was specified).
