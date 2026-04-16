---
name: ticket-grooming
description: "Deeply investigate and groom tickets by dispatching sub-agents for codebase investigation, history research, root cause analysis, and risk assessment. Use when the user says 'groom', 'triage', or asks to investigate a ticket. Posts structured 'Triaging Notes' as a comment on the ticket. Default output is short-form (TLDR, key findings, risks, estimation, recommended approach). Use --full for the complete report. Works with Jira, GitHub Issues, or any ticketing system."
tags: [workflow, project-management, debugging]
model: opus
---

# Ticket Grooming

Deeply investigate one or more tickets by dispatching isolated sub-agents, then post structured "Triaging Notes" as a comment on the ticket.

**Announce at start:** "Using ticket-grooming to investigate [ticket(s)]."

## Inputs

Extract from the user's message:
- **Ticket key(s) or URL(s)** (e.g., `DL-1234`, `https://zombie.atlassian.net/browse/DL-1234`, `#42`)
- OR a **verbal description** of the issue
- **Flags:**
  - `--dry-run` (preview without posting)
  - `--full` (post the full-form report with codebase findings, historical context, root cause analysis, and breadcrumbs)
- **Mode resolution:** `--full` flag > `grooming-mode` config in CLAUDE.md > default (`short`)
- **Default is short.** The short format posts: TLDR (with root cause + confidence), Key Findings (2-3 bullets max), Risks (medium+ only), Estimation, Recommended Approach, @mentions. The full investigation still runs internally — short only changes what is surfaced in the posted comment.

## Pre-Flight (Main Conversation)

Before dispatching any sub-agents, the main conversation MUST complete these steps:

### 1. Ensure Codebase Index is Current

```
Call mcp__codebase-memory-mcp__index_status
If stale → call mcp__codebase-memory-mcp__index_repository (once)
For multi-repo → check/index each repo
```

Sub-agents use `index_status` to verify. They do NOT re-index.

### 2. Detect Ticket System

Resolution order:
1. **Config override** — check CLAUDE.md for `### Ticket Grooming` section
2. **URL pattern** — `*.atlassian.net` = Jira, `github.com/*/issues/*` = GitHub
3. **Key pattern** — `XX-1234` (uppercase letters + hyphen + digits) = Jira, `#1234` = GitHub
4. **Verbal description** — no ticket exists; investigate and ask user where to post

| System | Read ticket | Search history | Post comment | Delete comment |
|--------|------------|----------------|--------------|----------------|
| Jira | `getJiraIssue` | `searchJiraIssuesUsingJql` | `addCommentToJiraIssue` | `acli jira workitem comment delete --key {KEY} --id {ID}` |
| GitHub | `gh issue view` | `gh issue list`, `gh pr list` | `gh issue comment` | `gh api -X DELETE` |
| Other | Ask user | Best effort | Ask user | Ask user |

### acli Fallback

When an operation is not available via the Atlassian MCP server (e.g., deleting comments, bulk edits), use the `acli` CLI instead:

```bash
# Delete a comment
acli jira workitem comment delete --key DL-1234 --id 12345

# List comments (to find IDs)
acli jira workitem comment list --key DL-1234

# Other useful acli commands
acli jira workitem view --key DL-1234
acli jira workitem edit --key DL-1234 --field "labels=has_notes"
```

Always prefer MCP tools for reads and writes. Use `acli` when MCP lacks the capability (deletes, comment updates, bulk operations).

**CRITICAL — acli content formatting:** When posting or updating comment content via acli, NEVER use `--body` or `--body-file` with markdown — these post plain/unformatted text. Always use `--body-adf` with a properly constructed ADF JSON file for formatted content. For simple text-only comments (no formatting needed), `--body` is fine.

### 3. Resolve GitHub Remote Info

For each repo involved:
```bash
# Get org/repo from remote (handles SSH and HTTPS)
git remote get-url origin
# SSH:   git@github.com:org/repo.git → org/repo
# HTTPS: https://github.com/org/repo.git → org/repo

# Get HEAD SHA (verify it's pushed)
git rev-parse HEAD
git branch -r --contains HEAD  # if empty, use latest remote SHA

# Fallback: if detection fails, use relative paths instead of permalinks
```

### 4. Multi-Repo Investigation (if applicable)

If the project spans multiple repositories (e.g., separate API repos, frontend + backend, microservices), check CLAUDE.md for a `Repos:` config section listing them.

- Resolve GitHub remote info (org/repo, HEAD SHA) for **each** relevant repo during pre-flight step 3.
- Pass all repos to the sub-agent as `Additional repos`.
- The sub-agent must search and index each repo during Phase 1 (Codebase Investigation).
- If a ticket is clearly scoped to a single repo, skip this step.

**Migration context:** If repos are mid-migration (code moving from one repo to another), note in triaging notes which repo the fix should target and which contains legacy copies.

### 5. Read Tickets and Detect Shared Context (Multi-Ticket Only)

When grooming 2+ tickets:
1. Read all tickets
2. If 2+ tickets reference the same component/package/file paths → build shared context:
   - Run codebase investigation for the shared area once
   - Produce a shared context summary
   - Pass to each sub-agent as pre-built context
3. If no overlap → dispatch independently

### 6. Dispatch Sub-Agents

- **1 sub-agent per ticket** (two-level architecture only — sub-agents do NOT spawn their own sub-agents)
- **Max 3 concurrent.** Queue additional tickets as slots free up.
- Each sub-agent gets **fresh context** — no shared state between tickets.
- Each sub-agent is fully independent — a failure in one does not affect others.

## Sub-Agent Prompt Template

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

### Phase 0: Classification

Read the ticket and classify it:
- `code-bug` — a defect in existing code
- `code-feature` — new functionality to build
- `tech-debt` — refactoring or cleanup
- `process-docs` — non-code work (documentation, process, meetings)
- `ambiguous` — unclear, treat as code-related

If `process-docs`: skip Phases 1 and 3, produce simplified output (see Output Format below).

### Phase 1: Codebase Investigation

**Context budget: max 15 files deep-read, max 3-hop call path traces.**

1. Check codebase index is current: `mcp__codebase-memory-mcp__index_status`
2. Search the knowledge graph for entities related to the ticket: `mcp__codebase-memory-mcp__search_graph`
3. Trace call paths for affected code (max 3 hops): `mcp__codebase-memory-mcp__trace_call_path`
4. Check database schemas and migrations relevant to the issue
5. Map the surface area: files, functions, models affected
6. For multi-repo tickets: repeat across each relevant repo (budget applies per repo)

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
- If `Output mode` is `short` (default): format using **Output Format — Code Tickets (Short)** template, which includes a collapsed `<details>` section containing the full investigation. The short sections are visible by default; the full investigation is expandable.
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

```
# Triaging Notes
_Groomed: {ISO_TIMESTAMP} (iteration {N})_

## TLDR
One-paragraph summary: what the issue is, what's affected, and the recommended path forward. Fold in the root cause with a confidence level (e.g., "Root cause (confidence: HIGH):") and the key technical mechanism so readers understand the "why" without needing a separate codebase findings section.

## Key Findings
- 2-3 bullets max. Include ONLY when there is a specific query, code snippet, or mechanism that makes the root cause concrete. Each bullet: the load-bearing fact with a GitHub permalink. No prose expansion.
- Omit this section entirely if the TLDR already captures the mechanism sufficiently.

## Risks
- Medium, high, and critical risks only. One line each. Omit low risks entirely.

## Estimation
- One line: **Size: {T-shirt}** | {days} | Confidence: {level} | Recommend **{N} SP**. Second line for complexity drivers if needed.

## Recommended Approach
- Bulleted per repo. Name the new classes/files and the reuse targets. No Option B / Option C unless a real trade-off exists worth debating.

@{PM or reporter} — {open questions, if any}

---

<details>
<summary>Full Investigation Details (click to expand)</summary>

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

</details>
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

## After Investigation Sub-Agent Returns

Back in the main conversation, dispatch a **staff engineer review sub-agent** before posting.

### 6. Staff Engineer Review

Dispatch a new sub-agent with **fresh context** using `model: "sonnet"` for faster verification. This agent reviews the triaging notes for errors and missed issues.

**Staff Engineer Review Sub-Agent Prompt Template:**

```
You are a staff engineer reviewing triaging notes for ticket {TICKET_KEY} before they are posted. Your job is to catch errors, missed risks, and deviations from repo patterns. You have fresh context — verify everything independently.

## Triaging Notes to Review
{FULL_TRIAGING_NOTES_FROM_INVESTIGATION_AGENT}

## Ticket Details
{FULL_TICKET_DESCRIPTION}

## Pre-Resolved Info
- GitHub org/repo: {ORG}/{REPO}
- HEAD SHA: {SHA}

## Verification Strategy: Graph First, Then Targeted Reads

**Use the codebase knowledge graph as your primary verification tool.** Do NOT re-read every file mentioned in the notes. Instead:

1. `mcp__codebase-memory-mcp__search_graph` — verify entities exist (classes, modules, methods, files)
2. `mcp__codebase-memory-mcp__get_architecture` — validate component boundaries and patterns
3. `mcp__codebase-memory-mcp__trace_call_path` — verify call path claims in 1 query instead of reading each file
4. **Only fall back to Read/Grep** for specific line-number verification or when the graph lacks detail (e.g., config values, schema columns)

**Budget: max 15 tool calls for correctness checks.** The graph should handle most verification in 5-8 queries.

## Review Checklist

Focus on high-risk claims. Skip low-risk items (Jira ticket statuses, estimation opinions).

**Enforce the Investigation Accuracy Rules** from the investigation agent's prompt. The investigation agent was told to follow Rules 1–5 (exact-name citation, verify-the-mechanism, self-critique, label verified vs speculative, focus on actual problem). Your job here is to catch violations.

### Rule 1 enforcement — Exact-Name Citation
- [ ] Every class, method, module, file, and column named in the notes EXISTS in the codebase (batch-verify via `search_graph` or `search_code`)
- [ ] If any named entity cannot be found, it is a violation — flag and reject

### Rule 2 enforcement — Verify the Mechanism
- [ ] Every high/medium-confidence hypothesis cites a `file:line` + permalink + concrete mechanism trace
- [ ] Mechanism traces are grounded in actual code (spot-check via Read for the most load-bearing hypothesis)
- [ ] Hypotheses lacking all three elements (symptom, location, mechanism) are downgraded to LOW/speculative — flag if not

### Rule 3 enforcement — Self-Critique
- [ ] Each high/medium-confidence hypothesis includes a `**Counterargument considered:**` line
- [ ] The counterargument is substantive (not boilerplate like "this might not be the root cause")

### Rule 5 enforcement — Focus on Actual Reported Problem
- [ ] Re-read the ticket title and description independently
- [ ] Verify that the root cause hypothesis addresses the SPECIFIC problem the reporter described (not an adjacent or related issue)
- [ ] Verify that the suggested fix would resolve the reporter's stated complaint
- [ ] If findings address a read path but the ticket reports a write problem (or vice versa), flag as misaligned
- [ ] If the investigation focused on a symptom (display) rather than the cause (storage/logic), flag as misaligned

### Correctness (use graph first)
- [ ] Key file paths and classes exist (batch-verify via search_graph)
- [ ] Call path traces are accurate (verify via trace_call_path)
- [ ] Schema claims match actual database schema (one Read of db/schema.rb or db/structure.sql)
- [ ] The root cause / gap analysis is logically sound

### Defensive Coding & Security (highest priority)
- [ ] If authorization is involved: verifies Pundit policies cover the new path
- [ ] If new API params are added: checks strong parameter whitelisting
- [ ] Suggested solutions handle edge cases (nil values, empty arrays, concurrent access)
- [ ] Cleanup/rollback paths are addressed (e.g., orphaned records on state reversal)
- [ ] Security-sensitive claims are verified with targeted file reads (not just graph)

### Pattern Matching (use get_architecture + graph)
- [ ] Suggested approach follows existing repo conventions (verify via get_architecture)
- [ ] The suggested solution uses the same abstractions as the reference implementation
- [ ] Authorization pattern is correctly identified (controller vs interactor level)

## Output Format

Return your review as a structured report:

### Errors Found
List each error with:
- **What:** The specific claim that is wrong
- **Why:** What the actual state is (with evidence — file path, line number, grep result)
- **Fix:** The corrected text that should replace it in the triaging notes

### Missed Risks
List any risks or edge cases the investigation missed:
- **Risk:** Description
- **Evidence:** How you found it
- **Addition:** Text to add to the triaging notes

### Pattern Deviations
List any places the suggested solution deviates from repo conventions:
- **Deviation:** What was suggested vs what the repo does
- **Evidence:** Reference to the existing pattern
- **Fix:** How to correct the suggestion

### Verdict
- **PASS** — No issues found, safe to post as-is
- **PASS WITH NOTES** — Minor issues that don't affect correctness (list them for context but no fixes needed)
- **NEEDS FIXES** — Issues found; provide the corrected triaging notes sections
```

### 7. Auto-Fix and Post

After the staff engineer review sub-agent returns:

1. **PASS:** Post the triaging notes as-is.
2. **PASS WITH NOTES:** Post the triaging notes as-is. Mention the notes to the user in the conversation summary.
3. **NEEDS FIXES:** Apply all fixes from the review to the triaging notes automatically, then post the corrected version. Report what was fixed in the conversation summary.

**Posting to Jira — MANDATORY:**
- **MCP** (`addCommentToJiraIssue`): MUST pass `contentFormat: "markdown"`. Omitting it defaults to ADF and renders markdown as broken plain text.
- **acli** (`workitem comment create/update`): MUST use `--body-adf` with a properly constructed ADF JSON file. Do NOT use `--body` or `--body-file` for markdown content — these post plain text that renders unformatted.
- **Preferred for new comments:** MCP with `contentFormat: "markdown"` (simplest — handles conversion automatically).
- **Preferred for updating comments:** acli with `--body-adf`, or delete via acli + re-post via MCP.

**After posting (all verdicts):** Add the label `has_notes` to the ticket to indicate it has been groomed. Use `editJiraIssue` (Jira) or `gh issue edit --add-label` (GitHub) to add the label without removing existing labels.

If `--dry-run`: display the final (potentially corrected) triaging notes in the conversation. Ask: "Post to ticket?" If confirmed, post and add the `has_notes` label.

For multi-ticket batches, report as each completes:
```
Grooming 3 tickets...
  - DL-1234: Triaging notes posted (review: PASS)
  - DL-1235: Triaging notes posted (review: NEEDS FIXES — 2 corrections applied)
  - DL-1236: In progress — staff engineer review
```

## Error Handling

| Failure | Behavior |
|---------|----------|
| Ticket not found / inaccessible | Fail fast, tell the user |
| Sub-agent exceeds context budget | Summarize what it has, note "investigation truncated due to complexity" |
| Comment post fails | Output triaging notes in the conversation so nothing is lost |
| Comment posted with wrong format | Delete the bad comment via `acli jira workitem comment delete --key {KEY} --id {ID}`, then re-post with correct `contentFormat: "markdown"` |
| MCP tools unavailable | Inform user which tools are needed. For Jira operations not supported by MCP (deleting comments, bulk edits), fall back to `acli` CLI |
| Need to delete a Jira comment | MCP does not support comment deletion. Use `acli jira workitem comment delete --key {KEY} --id {ID}` |
| Repo not cloned locally (multi-repo) | Skip that repo, note it in findings, ask user for path |
| One sub-agent fails in a batch | Others continue; failure reported with partial findings |
| GitHub remote detection fails | Use relative paths instead of permalinks |
| Staff engineer review fails | Post the unreviewed notes with a note: "Posted without staff engineer review (review agent failed)" |

## Verbal Description (No Ticket)

When the user describes an issue without a ticket key:
1. Run the full investigation pipeline
2. Present findings in the conversation (not posted anywhere)
3. Ask: "Should I create a ticket with these triaging notes?" (respect project rules about ticket creation — e.g., ask before creating Jira tickets)

## Configuration

Add to CLAUDE.md to customize behavior:

```markdown
### Ticket Grooming
- Default ticket system: jira
- Jira site: your-site.atlassian.net
- GitHub org: your-org
- dry-run: false
- grooming-mode: short  # short (default) | full (--full flag overrides this)
- Repos:
  - my-api: ~/dev/my-api
  - my-web: ~/dev/my-web
```

## Skills Referenced

| Skill | When | Scope |
|-------|------|-------|
| `systematic-debugging` | Phase 3 (always for code tickets) | Phases 1-3 only — investigation, not implementation |
| `dispatching-parallel-agents` | Multi-ticket invocations | Parallel sub-agent dispatch with max 3 concurrency |
| `code-reviewer` (built-in) | Step 6 — staff engineer review | Correctness, defensive coding, security, pattern matching |

## Installation

### User-level (personal, works across all projects)
```
~/.claude/skills/ticket-grooming/SKILL.md
```

### Project-level (shared with team via repo)
```
.claude/skills/ticket-grooming/SKILL.md
```

Project-level overrides user-level if both exist.
