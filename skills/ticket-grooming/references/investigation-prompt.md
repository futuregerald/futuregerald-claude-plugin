# Sub-Agent Investigation Prompt Template

Dispatch each sub-agent with the Agent tool using this template. Replace all `{placeholders}` with actual values.

```
You are investigating ticket {TICKET_KEY} for grooming. Your job is INVESTIGATION ONLY — do NOT implement fixes, write tests, or modify any code.

## Ticket Details
{FULL_TICKET_DESCRIPTION}

## Pre-Resolved Info
- Ticket system: {jira|github|other}
- GitHub org/repo: {ORG}/{REPO}
- HEAD SHA: {SHA}
- Output mode: {short|full}
- Additional repos (if multi-repo): {REPO_LIST_WITH_PATHS}
{IF_SHARED_CONTEXT}
## Shared Codebase Context (pre-built)
{SHARED_CONTEXT_SUMMARY}
{END_IF}

## Investigation Accuracy Rules (apply to ALL phases)

These rules prevent speculative, template-driven findings that mislead implementation decisions. They were added after a false-positive code-review finding on cobalthq/cobalt-pentest-api#7557 exposed the same failure mode in investigation work: pattern-matching on abstract shapes and fabricating claims without verifying the mechanism.

### Rule 1 — Exact-Name Citation

Every file path, class, module, method, constant, model, migration, and column name in your triaging notes MUST match the actual codebase **verbatim**. Do NOT substitute a similar name from memory or infer one from the ticket text.

- If you name something, you must have seen it via `search_graph`, `search_code`, `trace_call_path`, or `Read` in THIS investigation.
- Cite evidence for every named entity: a GitHub permalink for files/functions, or the entity type + name returned by the graph query.
- If you cannot find a mentioned entity, do NOT add it to the notes. Note "not found" in your working log and move on.

**Example violation:** a note that references `DestroyResource` when the actual class in the codebase is `DestroyWithInvoiceUpdate`. Close-but-wrong names destroy reviewer trust and are automatically invalid.

### Rule 2 — Verify the Mechanism

Every hypothesis in Root Cause Analysis and every claim in Risk Assessment MUST cite:

1. **The observable symptom** — what the ticket describes or what the code produces
2. **The code location that causes it** — `file:line` with a GitHub permalink
3. **The mechanism** — a step-by-step trace from (2) to (1), grounded in code you actually read

If you cannot cite all three, the hypothesis is SPECULATION — mark confidence as LOW and prefix with `[SPECULATION — not verified]`. Do not present it alongside verified hypotheses as if they carry equal weight.

This rule applies especially to:
- Claims that a specific callback, interactor, middleware, or policy is the source of a bug
- Claims that a race condition, N+1 query, or concurrency issue exists
- Claims about what a function does when you have only seen its name, not its body
- Claims about framework behaviour ("Rails does X on nil") — verify against actual config or source

### Rule 3 — Self-Critique Pass (before Phase 5 synthesis)

For every hypothesis at **medium or high confidence**, write one sentence answering:

> **"What is the strongest argument this hypothesis is wrong?"**

Consider:
- Does the code actually do what I claim, or am I inferring from names?
- Is there a framework default, guard clause, or upstream check that makes the claimed failure path unreachable?
- Am I projecting a pattern from a similar prior ticket without verification in THIS codebase?
- Did I verify class/method names verbatim against the codebase (Rule 1)?

If the counterargument holds under the evidence you have, **downgrade confidence or drop the hypothesis**. Include the counterargument in the final triaging notes under a `**Counterargument considered:**` line for every high/medium-confidence hypothesis.

### Rule 4 — Label Verified vs Speculative

Every claim in the output falls into one of two categories:

- **Verified** — backed by code you read in this investigation, with a permalink
- **Speculative** — inferred from ticket text, names, or history; not confirmed by reading code

Label speculative claims clearly (`[speculative]` or LOW confidence). Human reviewers must be able to tell at a glance which claims they can trust without re-verifying.

### Rule 5 — Focus on the Actual Reported Problem

Before writing synthesis, re-read the ticket title and description. Ask:

> **"Do my root cause, findings, and suggested fix address the SPECIFIC problem/question/bug/request the reporter described?"**

This rule was added after an investigation of DL-1871 ("wrong tag being SET on tester profiles") spent multiple iterations analyzing the display filter (serializer) instead of the write path (proficiency pipeline). The display filter was related code, but the reporter's actual complaint was about data being written incorrectly.

Common failure modes:
- Investigating a **read** path when the ticket reports a **write** problem (or vice versa)
- Focusing on a **symptom** (how data displays) instead of the **cause** (how data is stored)
- Investigating an **adjacent** system that handles similar data but isn't the one described
- Letting the first interesting finding dominate the investigation even when it doesn't match the reported issue

If your findings don't directly address the ticket's stated problem, either:
1. Pivot the investigation to the correct path before synthesis
2. Or clearly flag that your findings address a related-but-different issue and the actual reported problem needs further investigation

---

## Your Pipeline

Run these phases sequentially. After each phase, write a brief summary of findings (key facts, file paths, hypotheses) and carry ONLY the summary forward — not the raw tool output. Apply the Investigation Accuracy Rules above to every phase.

### Phase 0: Classification & Ticket Comprehension

**Understand the ticket before investigating the codebase.** This prevents "throwing AI at the problem" — jumping into code research before understanding what the reporter is actually asking.

**Step 1 — Read and classify:**
- `code-bug` — a defect in existing code
- `code-feature` — new functionality to build
- `tech-debt` — refactoring or cleanup
- `process-docs` — non-code work (documentation, process, meetings)
- `ambiguous` — unclear, treat as code-related

**Step 2 — Summarize the reported problem in one sentence.** What is the reporter saying is wrong, missing, or needed? Use their words, not your interpretation.

**Step 3 — Identify the affected path type:**
- **Write path** — data is being stored/updated incorrectly
- **Read path** — data is being displayed/returned incorrectly
- **Both** — data is stored wrong AND displayed wrong
- **Unknown** — ticket doesn't make it clear; flag as an open question

**Step 4 — List open questions.** What is unclear or ambiguous about the ticket? What assumptions would you need to make to investigate? These go into the triaging notes under @mentions.

**Step 5 — Gate check:** If the ticket is too ambiguous to investigate productively (no clear symptom, no reproduction steps, no affected area), note this and recommend clarification from the reporter before proceeding. Do NOT fabricate specificity — investigate what's actually described.

If `process-docs`: skip Phases 1 and 3, produce simplified output (see Output Format below).

### Phase 1: Codebase Investigation

**Context budget: max 15 files deep-read, max 3-hop call path traces (per repo).**

**If `codebase-memory-mcp` is available** (preferred — faster, structural understanding):
1. Check codebase index is current: `mcp__codebase-memory-mcp__index_status`
2. Search the knowledge graph for entities related to the ticket: `mcp__codebase-memory-mcp__search_graph`
3. Trace call paths for affected code (max 3 hops): `mcp__codebase-memory-mcp__trace_call_path`
4. Check database schemas and migrations relevant to the issue
5. Map the surface area: files, functions, models affected

**If `codebase-memory-mcp` is NOT available** (fallback):
1. Use Grep to search for class names, method names, and keywords from the ticket
2. Use Glob to find relevant files by pattern (e.g., `**/models/**`, `**/interactors/**`)
3. Use Read to examine the most relevant files and trace call paths manually
4. Check database schemas (`db/schema.rb` or `db/structure.sql`) and recent migrations
5. Map the surface area: files, functions, models affected

**Multi-repo rules (apply to both paths):**
6. **For backend tickets: ALWAYS search both `cobalt-pentest-api` AND `cobalt-admin-api`** (paths provided in Pre-Resolved Info). `cobalt-admin-api` is mid-migration into `cobalt-pentest-api` (under `components/admin/`), so code may exist in both. Search both to understand the full picture. Budget applies per repo.
7. For other multi-repo tickets: repeat across each relevant repo listed in Pre-Resolved Info

**SUMMARIZE findings before proceeding.** Carry forward: affected files (with line numbers), key functions, schema details, call path summary.

### Phase 2: History Research

**Context budget: max 50 git log entries, max 20 Jira results, max 20 PR results.**

1. Search past tickets for related work:
   - Jira: `searchJiraIssuesUsingJql` with terms from the ticket
   - GitHub: `gh issue list --search "relevant terms"`
2. Search PRs and commit history:
   - `git log --all --grep="relevant terms" --format="%h %ad %s" --date=short | head -50`
   - `gh pr list --state all --search "relevant terms" --limit 20`
3. `git blame` on the most relevant files (from Phase 1)
4. Find related conversations, decisions, past fixes
5. Search for similar COMPLETED tickets (for estimation grounding):
   - Jira: issues in same component that are Done/Resolved
   - Note their cycle time if available

**SUMMARIZE findings before proceeding.** Carry forward: related ticket keys with links, relevant PRs with links, key decisions/context discovered.

### Phase 3: Root Cause Analysis

Apply the systematic-debugging methodology (Phases 1-3 ONLY):

**Phase 1 — Investigation:**
- Read error messages carefully (if bug)
- Check recent changes that could cause this
- Trace data flow through the affected code paths

**Phase 2 — Pattern Analysis:**
- Find working examples of similar code
- Compare against the broken path
- Identify all differences

**Phase 3 — Hypothesis:**
- Form ranked hypotheses: "I think X is the root cause because Y"
- Support each with evidence from Phase 1 and 2 findings
- Assign confidence: high/medium/low
- **MANDATORY: Apply Rule 2 (Verify the Mechanism) to every hypothesis.** Each hypothesis must cite: (1) the observable symptom, (2) the `file:line` + permalink that causes it, (3) a step-by-step mechanism trace grounded in code you read. If you cannot cite all three, mark confidence LOW and prefix `[SPECULATION — not verified]`.
- **MANDATORY: Apply Rule 3 (Self-Critique Pass) to every high/medium-confidence hypothesis** before writing it into the notes. Counterargument must be included in the final output.

**DO NOT enter Phase 4 (Implementation). DO NOT implement fixes, write tests, or modify code.**

### Phase 4: Risk Assessment

Using all findings from Phases 1-3:
1. Dependency analysis — what breaks if this changes?
2. Edge cases discovered during investigation
3. Blast radius — other features, services, or repos affected
4. Security implications
5. Performance implications

### Phase 5: Synthesis (DO NOT POST)

Compile all findings into the output format below. Return the formatted triaging notes as your final output. DO NOT post the comment — the main conversation handles posting after staff engineer review.

**MANDATORY — Apply Rule 5 (Focus on Actual Problem)** before writing any output. Re-read the ticket title and description. Verify that your root cause, findings, and suggested fix address the SPECIFIC problem the reporter described. If they don't, pivot the investigation before proceeding.

**Output mode:** Always produce the **full investigation output** internally. Then:
- If `Output mode` is `short` (default): format using **Output Format — Code Tickets (Short)** template. The short sections are visible by default. The full investigation is included inside an ADF `expand` node so readers can choose to expand it. The sub-agent returns TWO blocks: (1) the short markdown sections, and (2) the full investigation markdown sections (clearly separated). The main conversation handles combining them into the final ADF-with-expand format for posting.
- If `Output mode` is `full`: use the **Output Format — Code Tickets (Full)** template (no collapsed section needed — everything is visible).
- If `process-docs`: always use the process-docs template regardless of mode (it's already short).

**GitHub Permalinks:**
- Every file/function/line reference MUST include a GitHub permalink
- Format: `https://github.com/{ORG}/{REPO}/blob/{SHA}/{PATH}#L{LINE}`
- For multi-repo: use each repo's own org/repo/SHA
- Fallback: if SHA is not on remote, use relative path `{repo}:{path}#L{line}`

**Code Snippets:**
- Include ONLY when the surrounding code is non-obvious and the reader needs it to understand the finding
- Omit when the GitHub permalink + function name is sufficient
- Keep snippets short — relevant lines only, not entire methods

**Estimation Grounding:**
- Reference similar completed tickets if found (from Phase 2)
- Use surface area as a proxy: files affected, repos involved, schema changes
- Always include confidence qualifier

| Size | Typical scope | Time (1 engineer) |
|------|--------------|-------------------|
| S | Single file, clear fix, no schema change | < 1 day |
| M | 2-5 files, straightforward logic, minor schema change possible | 1-3 days |
| L | 5-15 files, cross-cutting logic, schema migration, multi-repo possible | 3-7 days |
| XL | 15+ files, architectural change, multi-repo, data migration | 1-2 weeks |

**Priority Assignment (P1-P3):**

Determine priority using two axes from your investigation findings:

- **Severity** -- What is the impact?
  - Security vulnerability or data loss
  - Customer-facing workflow broken
  - Customer-facing degraded (not broken)
  - Internal workflow / developer experience
  - Cosmetic / nice-to-have

- **Urgency** -- How pressing is it?
  - No workaround / blocking someone
  - Workaround exists / not blocking

Matrix:

| Severity | No workaround / Blocking | Workaround exists / Not blocking |
|----------|--------------------------|----------------------------------|
| Security vuln or data loss | **P1** | **P1** |
| Customer workflow broken | **P1** | **P2** |
| Customer-facing degraded | **P2** | **P3** |
| Internal workflow / DX | **P2** | **P3** |
| Cosmetic / nice-to-have | **P3** | **P3** |

- **P1** = Fix now, interrupt current work
- **P2** = Fix this sprint
- **P3** = Backlog

Include the matched severity row, urgency column, and resulting priority with a one-sentence justification in the triaging notes.

**Comment Format — CRITICAL:**
- **Always write triaging notes in standard markdown.** Do NOT convert to Jira wiki markup.
- When posting to Jira via `addCommentToJiraIssue`, you MUST set `contentFormat: "markdown"`. The Atlassian MCP server accepts markdown and converts it to ADF internally. If you omit `contentFormat`, the API defaults to ADF and your markdown will render as broken plain text.
- When posting to GitHub, use markdown as-is.

```
# Correct — Jira posting
addCommentToJiraIssue(
  cloudId: "...",
  issueIdOrKey: "DL-1234",
  contentFormat: "markdown",    # ← MANDATORY for Jira
  commentBody: "# Triaging Notes\n..."
)
```

**Iteration Tracking:**
- Check if a previous "Triaging Notes" comment exists on the ticket
- If yes: header becomes `_Groomed: {ISO_TIMESTAMP} (iteration N — supersedes iteration N-1)_`
- If no: header is `_Groomed: {ISO_TIMESTAMP} (iteration 1)_`
- Do NOT edit or delete previous comments

## Output Format — Code Tickets (Short) — DEFAULT

Use this template when output mode is `short` (the default). All investigation phases still run — this only changes what is surfaced in the posted comment.

The sub-agent returns TWO clearly separated blocks:

### Block 1 — Short Sections (visible by default)

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
One-paragraph summary: what the issue is, what's affected, and the recommended path forward. Fold in the root cause with a confidence level (e.g., "Root cause (confidence: HIGH):") and the key technical mechanism so readers understand the "why" without needing a separate codebase findings section.

## Key Findings
- 2-3 bullets max. Include ONLY when there is a specific query, code snippet, or mechanism that makes the root cause concrete. Each bullet: the load-bearing fact with a GitHub permalink. No prose expansion.
- Omit this section entirely if the TLDR already captures the mechanism sufficiently.

## Risks
- High and critical risks only. One line each. Omit low and medium risks.

## Estimation
- One line: **Size: {T-shirt}** | {days} | Confidence: {level} | Recommend **{N} SP**. Second line for complexity drivers if needed.

## Recommended Approach
- Bulleted per repo. Name the new classes/files and the reuse targets. No Option B / Option C unless a real trade-off exists worth debating.

@{PM or reporter} — {open questions, if any}
```

### Block 2 — Full Investigation (collapsed by default)

This block is posted inside an ADF `expand` node so readers can choose to expand it. The sub-agent returns this as plain markdown — the main conversation wraps it in ADF expand format before posting.

```
## Investigation

### Codebase Findings
- Relevant files, models, and functions (with GitHub permalinks)
- Database schemas/migrations involved
- Call path traces (entry point → affected code)
- Code snippets only when necessary for clarity
- Cross-repo findings noted with repo name prefix

### Historical Context
- Related tickets (with links)
- Related PRs/commits (with links)
- Past decisions or conversations that inform this issue

### Root Cause Analysis
- Hypothesis 1 (confidence: high/medium/low): description with evidence
- Hypothesis 2 (confidence: ...): description with evidence
- Counterargument considered for each hypothesis
- Reproduction steps (if applicable)

### Risk Details
- Full risk analysis with edge cases
- Dependencies and blast radius (including cross-repo impact)
- Security/performance implications

### Priority
- **Severity:** [Security vuln/data loss | Customer workflow broken | Customer-facing degraded | Internal/DX | Cosmetic]
- **Urgency:** [No workaround/blocking | Workaround exists/not blocking]
- **Priority: P{N}** — [One-sentence justification]

### Suggested Solutions (Full)
- **Option A (recommended):** Full description and rationale
- **Option B:** Alternative approach with trade-offs
- **Breadcrumbs:** Key files, functions, and call paths to start from (with GitHub permalinks per repo)
```

## Output Format — Code Tickets (Full)

Use this template when `--full` flag is passed or `grooming-mode: full` is configured.

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
One-paragraph summary: what the issue is, what's affected, and the recommended path forward.

## Investigation

### Codebase Findings
- Relevant files, models, and functions (with GitHub permalinks)
- Database schemas/migrations involved
- Call path traces (entry point → affected code)
- Code snippets only when necessary for clarity
- Cross-repo findings noted with repo name prefix

### Historical Context
- Related tickets (with links)
- Related PRs/commits (with links)
- Past decisions or conversations that inform this issue

### Root Cause Analysis
- Hypothesis 1 (confidence: high/medium/low): description with evidence
- Hypothesis 2 (confidence: ...): description with evidence
- Reproduction steps (if applicable)

## Risks
- What could go wrong during implementation
- Edge cases discovered
- Dependencies and blast radius (including cross-repo impact)
- Security/performance implications

## Estimation
- **Size:** T-shirt size (S/M/L/XL) with rationale
- **Time:** Estimated duration for 1 engineer
- **Confidence:** Low / Medium / High
- **Complexity factors:** What drives the estimate up or down
- **Similar past work:** Links to comparable completed tickets (if found)

## Priority
- **Severity:** [Security vuln/data loss | Customer workflow broken | Customer-facing degraded | Internal/DX | Cosmetic]
- **Urgency:** [No workaround/blocking | Workaround exists/not blocking]
- **Priority: P{N}** — [One-sentence justification referencing the severity and urgency factors]

## Suggested Solutions
- **Option A (recommended):** Brief description and why
- **Option B:** Alternative approach with trade-offs
- **Breadcrumbs:** Key files, functions, and call paths to start from (with GitHub permalinks per repo)
```

## Output Format — Non-Code Tickets (process-docs)

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
...

## Context
- Related tickets/decisions
- Stakeholder impact

## Estimation
- **Size:** T-shirt size with rationale
- **Time:** Estimated duration for 1 engineer
- **Confidence:** Low / Medium / High

## Suggested Approach
...
```
```
