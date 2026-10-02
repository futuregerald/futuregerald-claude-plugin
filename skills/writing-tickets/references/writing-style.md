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
8. **Unresolved questions explicit.** Separate numbered section. Each question must be specific enough that someone could answer it without re-reading the whole ticket. Epics are the exception: they carry no open questions (see the Epic template).
9. **End state defined.** Every ticket must state what success looks like. For stories/epics, use a "Key results" section. For initiatives, embed this in the scope -- each deliverable line should make the end state clear.
10. **200-300 words per section max.** If a section is longer, break it into sub-sections.
11. **No "In scope" on stories/tasks.** Stories and tasks already have "Desired behavior" (observable outcomes) and "Technical approach" (implementation details). Adding an "In scope" section duplicates both. Use "Out of scope" for guardrails, but never "In scope" -- that content belongs in the existing sections. Only initiatives use "Scope" (they lack a technical approach section).

## Brevity Discipline

**Initiatives and epics must be scannable in under 2 minutes.** If a PM or engineer can't understand what this work is and where it stands by skimming, the ticket is too long. Every section must earn its place.

### What to cut

| Cut this | It belongs in |
|----------|---------------|
| **"Current State" / "How it works today"** sections | Child stories (as "Current behavior") or Confluence docs. The epic audience already knows the system. |
| **Completed research / done tickets** | Git history or a "Related context" link. Don't list done work inline — it's not actionable. |
| **Per-workstream acceptance criteria** in initiatives | Child epics. The initiative's job is to define scope and key results, not detailed ACs per area. |
| **Multi-option analyses for decided questions** | Comments or ADRs. Once a decision is made, state the decision — don't preserve the deliberation. |
| **User stories** in initiatives/epics | Child stories. The initiative/epic describes the problem and scope, not individual user journeys. |
| **Implementation-level detail** (specific files, methods, code examples) in epics | Child stories. Epics describe what and why, not how. |
| **Confluence links, spreadsheets, prior art** inline in scope sections | A single "Related context" section at the bottom. Don't clutter scope with reference material. |

### What to keep

- **Problem statement** — always. 2-3 sentences max.
- **Scope** — what's being built, as a numbered list with 1-2 sentence descriptions per workstream. Not paragraphs.
- **Out of scope** — explicit exclusions. Bullet list.
- **Key decisions** — decisions made in comments or conversations, surfaced into the body so they're visible. Don't leave decisions buried in comment threads.
- **Open questions** (initiatives only) — only questions that are genuinely still open. Collapse answered questions into "Key decisions."
- **Dependencies** — what blocks this work.
- **Next step** — what happens next (assign tech lead, break into epics, etc.).

### When rewriting an existing ticket

1. **Read all comments.** Decisions and answers live in the thread, not the description. Extract them.
2. **Fold resolved questions into "Key decisions."** If a question was asked in the description and answered in a comment, the description should reflect the answer — not preserve the question.
3. **Drop completed work.** If a research spike or prerequisite is done, remove it from the description. It's history.
4. **Cut the "current state" section.** If the ticket previously explained how the system works today, that was useful during drafting but clutters the final version. Anyone working this ticket will investigate the codebase themselves.
5. **Test the 2-minute rule.** Read the rewrite as if you're a PM or engineer seeing it for the first time. Can you understand the problem, scope, decisions, and open questions in a quick skim? If not, cut more.

## Section Templates

### Initiative

```markdown
## Problem
{2-4 sentences: what's broken or missing, and why it matters now.
A new reader should understand the initiative after this paragraph alone.
Do NOT describe how the system works today -- that's docs, not the ticket.}

## What this initiative delivers
{Numbered list: one line per workstream/deliverable. 1-2 sentences each.
This is the scope. Keep it scannable -- no paragraphs per item.}

## Out of scope
{Explicit bullet list of adjacent work we are NOT doing}

## Key dependencies
{Table: What | Owner | Blocks. Only items that gate progress.}

## Key decisions
{Decisions made in comments, conversations, or reviews -- surfaced here
so they're visible, not buried in threads. Format: decision + who decided.
Omit this section if no decisions have been made yet.}

## Open questions
{Numbered list -- only genuinely unresolved questions. If a question
was asked and answered, move the answer to Key decisions instead.
Each question must name a suggested owner.}

## Suggested epics
{Numbered list: one line per epic, aligned to the deliverables above.
No links until the epics actually exist.}

## Next step
{One sentence: what happens next to move this forward.}

## Related context
{Links to Confluence docs, ADRs, dashboards, completed research.
This is the only place for reference material -- not inline in scope.}
```

**What changed from the old template:** Dropped "Key results" (redundant with scope when scope is well-written), dropped "Timeline" (use Jira fields), dropped "Blockers" (use "Key dependencies" -- blockers are transient and belong in comments), added "Key decisions" (prevents decisions from being lost in comment threads), added "Next step" (every initiative should have a clear next action), added "Related context" as the single home for reference material. "Why does this matter?" was merged into "Problem" -- if the problem statement doesn't convey urgency, it's not a good problem statement.

### Epic

Use this format whenever you write an epic, and always when you also create the stories under it. Product and engineering are both the audience. Write in the user's voice, in plain language, and don't explain or justify decisions.

```markdown
Part of {parent initiative link}.

## What problem are we solving?
{2-3 sentences: what's missing for the user, and what we'll do about it}

## How it works
{3-5 bullets: the end-to-end flow in plain language. No tables, endpoints or file names.}

## Designs
{Link to the design file or flow. Then, for each key screen:
a bold linked title, followed by an embedded screenshot.
Note in one line anything the designs don't cover yet, and who owns it.}

## Key results
{2-4 bullets: what's true for the user when this epic is done}

## Success metrics
{3-4 bullets: **Name:** what we measure. No invented numeric targets.}

## Key decisions
{Bullets: decisions already made, stated as facts. No reasoning, no options considered.}

## Out of scope
{Bullets: adjacent work we're not doing}

## Dependencies
{Bullets: what must land first, with links and owners}

## Related context
{Links: product answers, existing PRs, ADRs, upstream tickets}
```

**An epic never contains:**
- User stories or acceptance criteria. Those go in the child stories.
- A list of stories, sizes or estimates. Jira shows the children.
- An open questions section. Resolve open questions with the user before writing. Anything that's still open goes to the person who owns it, not into the description.
- Implementation detail: tables, columns, endpoints, flags, file paths.

### Story (child of an epic)

Use this format for every story created under an epic. A story is a user story plus acceptance criteria. Engineers decide how to build it.

```markdown
## User story
As a {role}, I want {capability}, so that {outcome}.

Design: {link to the screen, if there is one}

## Acceptance criteria
- [ ] {Observable behaviour a PM could check in the product}
- [ ] ...
```

**Rules:**
- Every acceptance criterion describes behaviour the user can see, never how it's built.
- No technical notes, table or column names, endpoints, feature flag names, file paths or SQL. Keep that thinking in a local plan, not in the ticket.
- Name a related ticket only when the behaviour depends on it, for example "works the same way as {link}".
- 3-6 acceptance criteria. If there are more, the story is too big: split it.
- A story with no user-facing surface (for example, recording data other stories read) is still written as a user story, from the user's point of view: "As an {owner}, I want the product to keep a record of…".

### Task (child of an epic)

For work that isn't a user capability, such as rollout, measurement or a staff preview.

```markdown
## What are we doing?
{One sentence}

## Acceptance criteria
- [ ] {Outcome, not implementation}
```

### Story (standalone)

Use this only for a single story that isn't being created under a new epic, for example a bug fix or a follow-up from grooming.

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

### Writing to Jira

- Send descriptions as Atlassian Document Format (ADF), not markdown. Markdown checklists turn into literal `\[ \]` bullets and links on inline code are dropped. Acceptance criteria become an ADF `taskList`.
- Neither `acli` nor the Atlassian MCP can attach files. Save design screenshots locally next to the drafts, keep the design links in the description, and tell the user which files to drag in.
- Read every created or edited ticket back and check its structure before saying it's done.

## What NOT to Write

- Don't explain how Rails/Ruby/React works -- assume the reader knows the framework
- Don't include implementation code in epics -- that's for stories and PRs
- Don't list every file that might change -- focus on the key ones
- Don't use jargon without context ("the actionable scope filters...") -- say what it does ("filters for tasks that still need work")
- Don't write acceptance criteria that duplicate each other in different words
- Don't leave the "why" implicit -- if the reader has to guess why, the ticket is incomplete
- Don't include "Current State" sections in epics/initiatives -- the audience knows the system; put this in child stories if needed
- Don't list completed research or done prerequisite tickets in the description -- link them in "Related context" if relevant
- Don't preserve multi-option analyses after a decision is made -- state the decision, drop the deliberation
- Don't write user stories in initiatives or epics -- those belong in child tickets
- Don't leave answered questions in the "Unresolved questions" list -- move answers to "Key decisions"
- Don't put implementation detail in stories under an epic -- engineers don't need every detail defined
- Don't add a stories-and-sizes table or an open questions section to an epic
