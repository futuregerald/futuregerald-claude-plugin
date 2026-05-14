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

The visible portion should read like a Slack message from a senior engineer — conversational, direct, no section bloat. Reserve all technical evidence for the collapsed details section.

The sub-agent returns TWO clearly separated blocks:

### Block 1 — Visible summary

Write in plain language. A PM should understand the problem, the fix, and the risk without expanding the details section.

```
Triaging Notes
Groomed: {ISO_TIMESTAMP} (iteration {N})

What's happening: 1-2 sentences in plain English. What's broken or missing, and who it affects. No jargon.

Root cause: 1-2 sentences explaining WHY. Name the specific mechanism but keep it accessible. Include confidence (high/medium/low).

Fix: 1-3 bullets. What to do, in which repo, touching which area. Name files/classes only if essential. No Option B unless there's a real trade-off.

Estimate: {S/M/L/XL} · {days} · {N} SP · Confidence: {level}

Risks: Only high/critical. One line each. Omit if none.

Priority: P{N} — {one sentence}

@{PM or reporter} — {open questions, if any. Omit if no questions.}
```

**Rules for the visible summary:**
- No markdown headers (`##`) — use bold labels instead. Keeps it compact.
- No "Key Findings" section — fold anything important into root cause or fix.
- No GitHub permalinks in the visible summary — those go in the details.
- No code blocks in the visible summary. Inline `code` marks for model/method names are fine.
- "Fix" field: if 1 item, use a single sentence. If 2-3 items, use a `bulletList` in the ADF (see adf-posting.md).
- Total visible summary should be **under 15 lines** when rendered.

### Block 2 — Full investigation details (collapsed)

Goes inside an ADF `expand` node. Engineers opt in by clicking to expand. Technical depth is expected here — use markdown headers, permalinks, and code snippets as needed.

```
## Codebase findings
- Relevant files, models, and functions (with GitHub permalinks)
- Database schemas/migrations involved
- Call path traces (entry point --> affected code)

## History
- Related tickets and PRs (with links)

## Root cause analysis (full)
- Each hypothesis with evidence, permalink, mechanism trace
- Counterarguments considered

## Risk details
- Full risk analysis, blast radius, security/performance

## Priority
- Severity · Urgency · P{N} with justification

## Breadcrumbs
- Key files, functions, and call paths to start from (with GitHub permalinks per repo)
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
