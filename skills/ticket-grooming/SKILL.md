---
name: ticket-grooming
description: "Deeply investigate and groom tickets by dispatching sub-agents for codebase investigation, history research, root cause analysis, and risk assessment. Use when the user says 'groom', 'triage', or asks to investigate a ticket. Posts structured 'Triaging Notes' as a comment on the ticket. Works with Jira, GitHub Issues, or any ticketing system."
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
- **Flags:** `--dry-run` (preview without posting)

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

| System | Read ticket | Search history | Post comment |
|--------|------------|----------------|--------------|
| Jira | `getJiraIssue` | `searchJiraIssuesUsingJql` | `addCommentToJiraIssue` |
| GitHub | `gh issue view` | `gh issue list`, `gh pr list` | `gh issue comment` |
| Other | Ask user | Best effort | Ask user |

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

### 4. Read Tickets and Detect Shared Context (Multi-Ticket Only)

When grooming 2+ tickets:
1. Read all tickets
2. If 2+ tickets reference the same component/package/file paths → build shared context:
   - Run codebase investigation for the shared area once
   - Produce a shared context summary
   - Pass to each sub-agent as pre-built context
3. If no overlap → dispatch independently

### 5. Dispatch Sub-Agents

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
- Additional repos (if multi-repo): {REPO_LIST_WITH_PATHS}
{IF_SHARED_CONTEXT}
## Shared Codebase Context (pre-built)
{SHARED_CONTEXT_SUMMARY}
{END_IF}

## Your Pipeline

Run these phases sequentially. After each phase, write a brief summary of findings (key facts, file paths, hypotheses) and carry ONLY the summary forward — not the raw tool output.

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

**Format Conversion:**
- If posting to Jira, convert markdown to Jira wiki markup before posting:
  - `# H` → `h1. H`, `## H` → `h2. H`, `### H` → `h3. H`
  - `**bold**` → `*bold*`
  - `` `code` `` → `{{code}}`
  - ```` ```lang ```` → `{code:lang}...{code}`
  - `[text](url)` → `[text|url]`
  - `- item` → `* item`
  - `> quote` → `{quote}...{quote}`
- If posting to GitHub, use markdown as-is

**Iteration Tracking:**
- Check if a previous "Triaging Notes" comment exists on the ticket
- If yes: header becomes `_Groomed: {ISO_TIMESTAMP} (iteration N — supersedes iteration N-1)_`
- If no: header is `_Groomed: {ISO_TIMESTAMP} (iteration 1)_`
- Do NOT edit or delete previous comments

## Output Format — Code Tickets

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

Dispatch a new sub-agent with **fresh context** (no shared state with the investigation agent). This agent reviews the triaging notes for errors and missed issues.

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

## Review Checklist

For each claim in the triaging notes, verify it against the actual codebase:

### Correctness
- [ ] Every file path mentioned exists and is correct
- [ ] Every function/method/class referenced exists at the stated location
- [ ] Every line number in GitHub permalinks points to the correct code
- [ ] Schema claims match actual database schema (check db/schema.rb or migrations)
- [ ] Call path traces are accurate (verify with grep/read, not just the knowledge graph)
- [ ] Related ticket references are accurate (correct keys, correct statuses)
- [ ] The root cause / gap analysis is logically sound

### Defensive Coding & Security
- [ ] If file uploads are involved: validates file types, enforces size limits, sanitizes filenames
- [ ] If user input is involved: checks for injection (SQL, XSS, command injection)
- [ ] If authorization is involved: verifies Pundit policies cover the new path
- [ ] If new API params are added: checks strong parameter whitelisting
- [ ] Suggested solutions do not introduce N+1 queries
- [ ] Suggested solutions handle edge cases (nil values, empty arrays, concurrent access)
- [ ] Cleanup/rollback paths are addressed (e.g., orphaned records on state reversal)

### Pattern Matching
- [ ] Suggested approach follows existing repo conventions (interactors, concerns, serializers)
- [ ] Similar completed work is referenced and the pattern is correctly identified
- [ ] The suggested solution uses the same abstractions as the reference implementation (not a novel approach when a proven one exists)
- [ ] Test expectations align with existing test patterns (request specs, model specs, interactor specs)

### Estimation
- [ ] T-shirt size is reasonable given the surface area
- [ ] Complexity factors are complete (nothing major missing)

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
| MCP tools unavailable | Inform user which tools are needed, proceed with what's available |
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
- Jira site: zombie.atlassian.net
- GitHub org: cobalt-io
- dry-run: false
- Repos:
  - cobalt-pentest-api: ~/Documents/dev/cobalt-pentest-api
  - cobalt-admin-api: ~/Documents/dev/cobalt-admin-api
  - cobalt-web: ~/Documents/dev/cobalt-web
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
