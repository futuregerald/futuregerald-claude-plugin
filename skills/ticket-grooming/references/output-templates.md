# Output Templates

## Writing Style (applies to all templates)

Grooming notes serve two audiences: PMs who need to understand what's wrong and what to prioritize, and engineers who need to know where to start. The visible sections must work for both.

### Rules

1. **Lead with the problem, not the solution.** TLDR explains what's broken or missing before explaining how to fix it. The reader should understand WHY before HOW.
2. **Short sentences, active voice.** "The system skips validation when..." not "A validation skip is performed by the system when..."
3. **Technical terms in context.** Use `code formatting` for model names, methods, file paths — but explain what they do. "`FindingPolicy` (the authorization check that controls who can edit findings)" not just "`FindingPolicy`."
4. **No jargon without context.** "Filters for tasks that still need work" not "the actionable scope filters." A PM should never have to guess what a phrase means.
5. **Arrows for state flows.** `Valid finding --> Pending Fix` instead of prose paragraphs describing transitions.
6. **Plain language headers.** "What we found" not "Codebase Investigation Findings."
7. **200-300 words per section max.** If longer, break into sub-sections.

### What NOT to write

- Don't explain how Rails/Ruby/React works — assume engineers know the framework
- Don't include full code blocks unless the surrounding code is non-obvious and a permalink isn't enough
- Don't leave the "why" implicit — if the reader has to guess why something matters, the note is incomplete

---

## Template: Code Tickets (Short) — DEFAULT

The sub-agent returns TWO clearly separated blocks. The short sections are visible by default. The full investigation goes inside an ADF `expand` node so readers can choose to expand it.

### Block 1 — Visible sections

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
One paragraph: what the issue is (in plain language), what's affected, and the recommended path forward. Fold in the root cause with a confidence level (e.g., "Root cause (HIGH):") and the key mechanism so readers understand the "why" without needing a separate section. A PM should be able to read this paragraph and understand the issue.

## Key Findings
- 2-3 bullets max. Include ONLY when there is a specific query, code snippet, or mechanism that makes the root cause concrete. Each bullet: the load-bearing fact with a GitHub permalink. No prose expansion.
- Omit this section entirely if the TLDR already captures the mechanism.

## Risks
- High and critical risks only. One line each. Omit low and medium risks.
- Explain the risk in terms of user/business impact, not just technical terms.

## Estimation
- One line: **Size: {T-shirt}** | {days} | Confidence: {level} | Recommend **{N} SP**.
- Second line for complexity drivers if needed.

## Recommended approach
- Bulleted per repo. Name the new classes/files and the reuse targets.
- No Option B / Option C unless a real trade-off exists worth debating.
- Brief enough that a PM can follow the direction; detailed enough that an engineer can start.

## Priority
- **P{N}** -- {One-sentence justification}

@{PM or reporter} -- {open questions, if any}
```

### Block 2 — Full investigation (collapsed by default)

This block goes inside an ADF `expand` node. Engineers opt in by clicking to expand. More technical detail is fine here.

```
## Investigation

### What we found in the code
- Relevant files, models, and functions (with GitHub permalinks)
- Database schemas/migrations involved
- Call path traces (entry point --> affected code)
- Code snippets only when necessary for clarity
- Cross-repo findings noted with repo name prefix

### History
- Related tickets (with links)
- Related PRs/commits (with links)
- Past decisions or conversations that inform this issue

### Root cause analysis
- Hypothesis 1 (confidence: high/medium/low): description with evidence
  - Counterargument considered: ...
- Hypothesis 2 (confidence: ...): description with evidence
  - Counterargument considered: ...
- Reproduction steps (if applicable)

### Risk details
- Full risk analysis with edge cases
- Dependencies and blast radius (including cross-repo impact)
- Security/performance implications

### Priority
- **Severity:** {row from matrix}
- **Urgency:** {column from matrix}
- **Priority: P{N}** -- {One-sentence justification}

### Suggested solutions (full)
- **Option A (recommended):** Full description and rationale
- **Option B:** Alternative approach with trade-offs
- **Breadcrumbs:** Key files, functions, and call paths to start from (with GitHub permalinks per repo)
```

---

## Template: Code Tickets (Full)

Use when `--full` flag is passed or `grooming-mode: full` is configured. Same content as short + expanded, but everything is visible — no collapsed section.

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
{Same as short mode — plain language, PM-readable}

## What we found in the code
{Same as Block 2 "What we found in the code"}

## History
{Same as Block 2 "History"}

## Root cause analysis
{Same as Block 2 "Root cause analysis"}

## Risks
- What could go wrong during implementation (user/business impact first, then technical detail)
- Edge cases discovered
- Dependencies and blast radius (including cross-repo impact)
- Security/performance implications

## Estimation
- **Size:** T-shirt size (S/M/L/XL) with rationale
- **Time:** Estimated duration for 1 engineer
- **Confidence:** Low / Medium / High
- **Story points:** {N} SP
- **Complexity factors:** What drives the estimate up or down
- **Similar past work:** Links to comparable completed tickets (if found)

## Priority
- **Severity:** {row from matrix}
- **Urgency:** {column from matrix}
- **Priority: P{N}** -- {One-sentence justification}

## Recommended approach
- **Option A (recommended):** Brief description and why
- **Option B:** Alternative approach with trade-offs
- **Breadcrumbs:** Key files, functions, and call paths to start from (with GitHub permalinks per repo)
```

---

## Template: Non-code tickets (process-docs)

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
{Plain language summary}

## Context
- Related tickets/decisions
- Stakeholder impact

## Estimation
- **Size:** T-shirt size with rationale
- **Time:** Estimated duration
- **Confidence:** Low / Medium / High

## Priority
- **Severity:** {row from matrix}
- **Urgency:** {column from matrix}
- **Priority: P{N}** -- {One-sentence justification}

## Recommended approach
{What to do and why}
```

---

## Formatting Rules

- **Bold** for emphasis, not ALL CAPS
- `Code formatting` for: model names, method names, file paths, status values, scope names
- Use `-->` for state transitions, not prose
- Use tables for comparisons, not paragraphs
- Bullet lists for unordered items; numbered lists for sequences
- Keep sections flat — avoid deep nesting

## Posting Rules

- **NEVER use HTML `<details>` or `<summary>` tags.** Jira does not render them.
- **Jira (full mode):** Post via `addCommentToJiraIssue` with `contentFormat: "markdown"`. Omitting `contentFormat` defaults to ADF and breaks rendering.
- **Jira (short mode):** The expand node requires posting as ADF JSON — NOT markdown. Construct the full ADF document (visible sections + expand node), write to a temp file, and post via `acli --body-file`. See [adf-posting.md](adf-posting.md) for the exact procedure and ADF skeleton template. There are no shortcuts — markdown cannot produce an expand node.
- **GitHub:** Use markdown as-is.

## Iteration Tracking

- Check if a previous "Triaging Notes" comment exists on the ticket.
- If yes: `_Groomed: {ISO_TIMESTAMP} (iteration N -- supersedes iteration N-1)_`
- If no: `_Groomed: {ISO_TIMESTAMP} (iteration 1)_`
- Do NOT edit or delete previous comments.

## Multi-Ticket Progress

Multi-ticket batch progress reporting is handled by the orchestrator, not the sub-agent. See SKILL.md for the format.
