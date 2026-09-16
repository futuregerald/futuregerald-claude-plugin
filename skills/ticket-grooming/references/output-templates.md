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

Write it exactly like this. **Blank lines are load-bearing** — the converter joins consecutive
non-blank lines into one paragraph, so a title and timestamp on adjacent lines post as a single
run-on sentence.

```
# Triaging Notes

_Groomed: {ISO_TIMESTAMP} (iteration {N})_

**What's happening:** 1-2 sentences in plain English. What's broken or missing, and who it affects. No jargon.

**Root cause:** 1-2 sentences explaining WHY. Name the specific mechanism but keep it accessible. Include confidence (high/medium/low).

**Fix:** 1-3 bullets. What to do, in which repo, touching which area. Name files/classes only if essential. Don't offer a menu of alternatives where one answer is right — but a *sequence* is fine, and "measure first, then X if it's actually slow" is often the honest answer. Say so rather than inventing certainty.

**Estimate:** {S/M/L/XL} · {days} · {N} SP · Confidence: {level}

**Risks:** Only high/critical. One line each. Omit if none.

**Priority:** P{N} — {one sentence}

@{MENTION} — {open questions, if any. Omit the whole line if there are none.}
```

`{N}` in the iteration line is supplied in the dispatch prompt — do not guess it, and do not
assume 1.

**Rules for the visible summary:**
- **One `#` heading only — the `# Triaging Notes` title.** Everything below it uses bold labels,
  not headings. No `##` or deeper anywhere in the visible block; they make it read like a report
  instead of a message.
- No "Key Findings" section — fold anything important into root cause or fix.
- No GitHub permalinks in the visible summary — those go in the details.
- `{MENTION}` is supplied in the dispatch prompt — the PM or the reporter. **Do not guess a name**
  from project files or memory; if the dispatch did not give you one, address the question to
  "the reporter" and say in your receipt that no name was supplied.
- `@Name` is **plain text and notifies nobody.** It marks who owes an answer, for human readers. A
  real Jira notification needs a mention node built from an account id; if the question actually
  needs to reach someone, tell the user so they can ping them.
- No code blocks in the visible summary. Inline `code` marks for model/method names are fine.
- "Fix" field: if 1 item, use a single sentence. If 2-3 items, use a Markdown bullet list — you write Markdown; the posting step converts it.
- Keep it to **nine blocks or fewer**: the title, the timestamp, and one block each for What's
  happening, Root cause, Fix, Estimate, Risks, Priority, and the @mention. Blank lines between
  them are mandatory (the converter needs them) and do not count. Judge length by blocks, not
  by physical lines.

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
{Same as Block 2 "Codebase findings"}

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

Posting is owned by [adf-posting.md](adf-posting.md) — build command, validation, the ADF node
reference, and the per-tracker commands all live there. Two rules matter while you are writing:

- **Never use HTML `<details>` / `<summary>`.** Jira renders them as raw text.
- **Write Markdown only.** The collapsed section needs an ADF `expand` node, which Markdown cannot
  express, so the posting step converts your file. Markdown also silently strips the
  ``[`code`](url)`` link form, which these notes use heavily — another reason not to hand Jira
  Markdown in short mode.

## Iteration Tracking

**The orchestrator determines the iteration number, not the sub-agent.** The sub-agent never sees
the ticket's existing comments, so it cannot count them — it uses the `{ITERATION}` value it is
given.

- Orchestrator: before dispatch, count existing "Triaging Notes" comments on the ticket and pass
  `{ITERATION}` = count + 1.
- Iteration 1: `_Groomed: {ISO_TIMESTAMP} (iteration 1)_`
- Later: `_Groomed: {ISO_TIMESTAMP} (iteration N -- supersedes iteration N-1)_`
- Never edit or delete previous comments.

## Paths in the posted note

**Repo-relative paths and permalinks only.** Never an absolute local path
(`/Users/<name>/...`), a scratchpad path, a hostname, or anything else describing the machine the
investigation ran on. The note is posted to a tracker, which may be public, and the details block
is the part nobody re-reads before it goes out.

## Code blocks in the details section

Fenced code blocks are supported and preserved verbatim. Use them only where a permalink is not
enough — see the Writing Style rules above. Fences must open and close on their own lines;
an unclosed fence swallows the rest of the file.

## Multi-Ticket Progress

Multi-ticket batch progress reporting is handled by the orchestrator, not the sub-agent. See SKILL.md for the format.
