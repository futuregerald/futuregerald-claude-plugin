# Grill Phase

Embedded grill-me: challenge assumptions and force clarity before writing. Calibrated intensity by ticket type.

## Principles

- Walk down each branch of the decision tree, resolving dependencies one by one
- For each question, provide your recommended answer with reasoning
- If a question can be answered by investigating the codebase, investigate instead of asking
- Don't let vague answers slide -- push back until the answer is specific and testable
- Surface contradictions found during investigation

## Intensity by Ticket Type

### Initiative (2-3 strategic questions)

Focus on why, scope, and success criteria.

**Required questions:**
1. **Why this, why now?** What's the business driver? What happens if we don't do this?
2. **What does success look like?** What are the key results? How will we measure them?
3. **What's explicitly out of scope?** What adjacent work are we NOT doing?

**Optional (if unclear):**
- Who are the stakeholders and what do they each need from this?
- What's the timeline pressure and why?

### Epic (4-6 questions covering scope, edges, and dependencies)

Focus on boundaries, edge cases, and what's been tried before.

**Required questions:**
1. **What's the end state?** Describe what the system looks like when this epic is done. Be specific.
2. **What are the edge cases?** Walk through the unhappy paths. What happens when X fails?
3. **What depends on this, and what does this depend on?** Map the dependency chain.
4. **Has this been attempted before?** Check Jira/GitHub for prior art. If found: "I found {ticket/PR} which attempted something similar. What's different this time?"

**Optional (based on investigation findings):**
- "Investigation found {X} works differently than the ticket assumes. How should we handle this?"
- "The data model supports {A} but not {B}. Should we change the model or change the approach?"
- "There's no feature flag coverage mentioned. What should be flagged?"
- "Who is the operator performing this action? What's their workflow today?"

### Story (1-2 focused questions)

Stories under a well-grilled epic need less interrogation. Focus on implementation clarity.

**Required questions:**
1. **Is the approach clear?** Can an engineer read this and start coding without asking questions?
2. **What could go wrong?** Any race conditions, data integrity risks, or side effects?

**Optional (if the story is complex):**
- "This touches {callback/observer/hook}. Have you accounted for the side effect?"
- "The test coverage in this area is {thin/strong}. Should we add specific test cases?"

### Spike (Minimal)

Spikes exist to find answers. Don't grill what we don't know yet.

**One question:**
1. **What specific question should this spike answer?** Frame the experiment with a clear exit criterion.

## Interactive Mode

Ask questions one at a time. Wait for the user's answer before proceeding. Provide your recommended answer with each question so the user can just confirm or correct.

Format:
```
**[Question N of M]: {Question}**

My recommendation: {Your recommended answer based on investigation}.
{Brief reasoning}.
```

When all questions are resolved, summarize the shared understanding before proceeding to write:
```
**Shared understanding:**
- {Key decision 1}
- {Key decision 2}
- ...

Moving to write the ticket. Correct?
```

## Autonomous Mode

Self-grill: ask and answer every question yourself using investigation findings. For each:

```
**[Self-grill Q{N}]: {Question}**
Answer: {Your answer based on investigation}
Confidence: {High|Medium|Low}
Basis: {What investigation finding supports this answer}
```

- **High confidence:** Investigation directly confirms the answer (read the code, found the ticket, saw the metric)
- **Medium confidence:** Investigation is consistent with the answer but doesn't directly confirm it (inferred from patterns, similar code elsewhere)
- **Low confidence:** Best guess based on limited evidence. Flag as assumption in the ticket.

After self-grilling, compile all Low/Medium confidence answers into the ticket's "Assumptions" section.
