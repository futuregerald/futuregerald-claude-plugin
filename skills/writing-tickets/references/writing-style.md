# Writing Style

Rules and templates for clear, actionable tickets. Dual audience: readable by PMs, actionable by engineers.

## Core Rules

1. **Lead with the problem, not the solution.** First section is always "What problem are we solving?" or equivalent. The reader should understand WHY before HOW.
2. **Plain language headers.** "When is a work order done?" not "Work Order Completion Definition". Headers should be questions or phrases a human would say.
3. **Short sentences, active voice.** "The system checks all tasks" not "A check is performed against all associated tasks by the system."
4. **Arrows for state flows.** `Valid finding --> Pending Fix` instead of prose paragraphs describing transitions.
5. **Technical terms in context.** Use `code formatting` for model names, scopes, file paths, method names. But explain what they do -- don't assume the reader knows.
6. **Consolidated acceptance criteria.** No redundant ACs. Merge overlapping ones. Each AC must be independently testable.
7. **Situate in hierarchy.** Always link parent (initiative/epic), sibling tickets, and dependencies. Show where this work fits.
8. **Unresolved questions explicit.** Separate numbered section. Each question must be specific enough that someone could answer it without re-reading the whole ticket.
9. **End state defined.** Every ticket must state what success looks like. For stories/epics, use a "Key results" section. For initiatives, embed this in the scope -- each deliverable line should make the end state clear.
10. **200-300 words per section max.** If a section is longer, break it into sub-sections.
11. **No "In scope" on stories/tasks.** Stories and tasks already have "Desired behavior" (observable outcomes) and "Technical approach" (implementation details). Adding an "In scope" section duplicates both. Use "Out of scope" for guardrails, but never "In scope" -- that content belongs in the existing sections. Only initiatives use "Scope" (they lack a technical approach section).

## Section Templates

### Initiative

```markdown
## Overview
{2-3 sentences: what this initiative is, at a glance. A new reader
should understand the initiative after reading this paragraph alone.}

## Why does this matter?
{1-2 paragraphs: business driver, customer impact, strategic value.
What happens if we don't do this? What opportunity do we miss?}

## Key results
{Numbered list: measurable outcomes that define success}

## Scope
{What's in scope, organized by theme/workstream}

## Out of scope
{Explicit list of adjacent work we are NOT doing}

## Epics
{List of child epics with links and one-line descriptions}

## Dependencies
{Cross-team, cross-domain, or external dependencies with links.
What must be true or done before this initiative can succeed?}

## Blockers
{Current blockers preventing progress. If none, omit this section.
For each: what's blocked, who/what is blocking, and suggested resolution.}

## Timeline
{Key milestones or deadlines, if known}

## Unresolved questions
{Numbered list}
```

### Epic

```markdown
## What problem are we solving?
{1-2 paragraphs: the problem in plain language, why it matters}

## Key results
{Numbered list: what does the system look like when this epic is done?}

## Part 1: {Plain language name}
{Describe the first logical chunk of work. 200-300 words max.
Use arrows for state flows, code formatting for technical terms.}

## Part 2: {Plain language name}
{Next chunk. Same rules.}

## Part N: ...

## Dependencies
{Links to parent initiative, blocking/blocked tickets, related PRs.
What must exist or be completed before this epic can proceed?}

## Blockers
{Current blockers preventing progress. If none, omit this section.
For each: what's blocked, who/what is blocking, and suggested resolution.}

## Related context
{Links to Confluence docs, ADRs, prior art PRs, Datadog dashboards}

## Unresolved questions
{Numbered list, specific enough to answer without re-reading}

## Acceptance criteria
{Checklist. Non-redundant. Each independently testable.}
```

### Story

```markdown
## What are we doing?
{1-2 sentences: the specific change, in plain language}

## Why?
{1-2 sentences: link to parent epic, explain how this fits}

## Current behavior
{What happens today. Be specific -- reference actual code paths.}

## Desired behavior
{What should happen after this change. Use arrows for flows.
Include guardrails: what must NOT change (e.g., "auto-staffing unaffected").}

## Out of scope
{Explicit list of adjacent work we are NOT doing. Omit if obvious.}

## Technical approach
{How to implement. Reference specific files, methods, models.
This section is for engineers -- technical detail is expected.}

## Edge cases
{Bullet list of edge cases and how each should be handled}

## Acceptance criteria
{Checklist. Non-redundant. Each independently testable.}

## Related
{Parent epic, sibling stories, relevant PRs, docs}
```

### Spike

```markdown
## What do we need to learn?
{The specific question this spike should answer}

## Why don't we know this yet?
{Brief context on why this is unknown -- what we've looked at so far}

## What we know so far
{Summary of investigation findings, with links to code/tickets}

## Experiment plan
{Concrete steps to find the answer. Not implementation -- exploration.}
1. {Step 1: what to try}
2. {Step 2: what to try}
3. ...

## Exit criteria
{How we know the spike is done. What artifact or decision comes out.}
- [ ] {Specific question answered}
- [ ] {Decision documented in parent epic / ADR / Confluence}
- [ ] {Findings shared with team}

## Time box
{Suggested time limit. Spikes should not be open-ended.}

## Related
{Parent epic, the ticket that spawned this spike, relevant code}
```

## Formatting Rules

- **Bold** for emphasis, not ALL CAPS
- `Code formatting` for: model names, method names, file paths, status values, scope names
- Use `-->` for state transitions, not prose
- Use tables for comparisons, not paragraphs
- Use blockquotes `>` for callouts that need attention (tech lead review, open questions)
- Bullet lists for items with no ordering; numbered lists for sequences
- Keep acceptance criteria as a flat checklist, not nested

## What NOT to Write

- Don't explain how Rails/Ruby/React works -- assume the reader knows the framework
- Don't include implementation code in epics -- that's for stories and PRs
- Don't list every file that might change -- focus on the key ones
- Don't use jargon without context ("the actionable scope filters...") -- say what it does ("filters for tasks that still need work")
- Don't write acceptance criteria that duplicate each other in different words
- Don't leave the "why" implicit -- if the reader has to guess why, the ticket is incomplete
