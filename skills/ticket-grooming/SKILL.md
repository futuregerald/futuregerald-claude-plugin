---
name: ticket-grooming
description: "Deeply investigate and groom tickets by dispatching sub-agents for codebase investigation, history research, root cause analysis, and risk assessment. Use when the user says 'groom', 'triage', or asks to investigate a ticket. Posts structured 'Triaging Notes' as a comment on the ticket. Default output is short-form (TLDR, key findings, risks, estimation, recommended approach) with full investigation collapsed in an ADF expand node. Use --full for the complete report. Always searches both cobalt-pentest-api and cobalt-admin-api for backend tickets. Works with Jira, GitHub Issues, or any ticketing system."
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
- **Default is short.** Short posts: TLDR (with root cause + confidence), Key Findings (2-3 bullets max), Risks (high/critical only), Estimation, Recommended Approach, @mentions -- plus the full investigation collapsed inside an ADF `expand` node.

## Pre-Flight (Main Conversation)

Before dispatching any sub-agents, complete these steps:

### 1. Ensure Codebase Index is Current

If `codebase-memory-mcp` is available: call `index_status`, re-index if stale. For multi-repo, check each repo. Sub-agents verify via `index_status` but do NOT re-index.

If `codebase-memory-mcp` is NOT available: skip. Sub-agents use Grep/Glob/Read instead.

### 2. Detect Ticket System

Resolution order:
1. **Config override** -- check CLAUDE.md for `### Ticket Grooming` section
2. **URL pattern** -- `*.atlassian.net` = Jira, `github.com/*/issues/*` = GitHub
3. **Key pattern** -- `XX-1234` = Jira, `#1234` = GitHub
4. **Verbal description** -- no ticket exists; investigate and ask user where to post

| System | Read | Search | Post | Delete |
|--------|------|--------|------|--------|
| Jira | `getJiraIssue` | `searchJiraIssuesUsingJql` | `addCommentToJiraIssue` | `acli jira workitem comment delete` |
| GitHub | `gh issue view` | `gh issue list`, `gh pr list` | `gh issue comment` | `gh api -X DELETE` |
| Other | Ask user | Best effort | Ask user | Ask user |

**acli fallback:** When MCP lacks a capability (deleting comments, bulk edits), use `acli` CLI. Always prefer MCP for reads/writes. `acli --body-file` accepts ADF JSON only -- it does NOT convert markdown.

### 3. Resolve GitHub Remote Info

For each repo: extract org/repo from `git remote get-url origin`, get HEAD SHA via `git rev-parse HEAD`, verify it is pushed. Fallback: use relative paths instead of permalinks.

### 4. Backend Issues: Resolve Both API Repos

For backend tickets, **always include both repos:**

| Repo | Path | GitHub |
|------|------|--------|
| `cobalt-pentest-api` | `~/Documents/dev/cobalt-pentest-api` | `cobalthq/cobalt-pentest-api` |
| `cobalt-admin-api` | `~/Documents/dev/cobalt-admin-api` | `cobalthq/cobalt-admin-api` |

Resolve GitHub info for both during pre-flight. Pass both to the sub-agent.

**Migration context:** `cobalt-admin-api` is mid-migration into `cobalt-pentest-api` (under `components/admin/`). Search both to understand the full picture, but **fix only in `cobalt-pentest-api`**. Note which repo the fix targets and why.

Skip for clearly frontend-only tickets.

### 5. Read Tickets and Detect Shared Context (Multi-Ticket Only)

When grooming 2+ tickets: read all, check for overlapping components/files. If overlap found, run shared codebase investigation once and pass as pre-built context. Otherwise dispatch independently.

### 6. Dispatch Sub-Agents

- **1 sub-agent per ticket** (two-level only -- sub-agents do NOT spawn their own)
- **Max 3 concurrent.** Queue additional as slots free.
- Each sub-agent gets fresh context -- no shared state between tickets.
- Each is fully independent -- failure in one does not affect others.

## Sub-Agent Prompt Template

Dispatch each sub-agent with the Agent tool using the full template in **[references/investigation-prompt.md](references/investigation-prompt.md)**. Replace all `{placeholders}` with actual values resolved during pre-flight.

The investigation prompt includes: Investigation Accuracy Rules (5 rules), Pipeline phases 0-5 (Classification, Codebase Investigation, History Research, Root Cause Analysis, Risk Assessment, Synthesis), Output Format templates (short and full), GitHub Permalink rules, Estimation table, and the P1-P3 Priority Matrix (severity x urgency decision grid).

## After Investigation Sub-Agent Returns

Back in the main conversation, dispatch a **staff engineer review sub-agent** before posting. Use the full template in **[references/staff-review-prompt.md](references/staff-review-prompt.md)** with `model: "sonnet"`.

The review covers: Rule 5 alignment check (is the note about the right problem?), Rule 1-3 enforcement, correctness verification, defensive coding/security, and pattern matching.

## Auto-Fix and Post

After the staff engineer review returns:

1. **PASS:** Post as-is.
2. **PASS WITH NOTES:** Post as-is. Mention notes to user in conversation summary.
3. **NEEDS FIXES:** Apply all fixes automatically, then post corrected version. Report what was fixed.

**Posting rules:**
- Always write triaging notes in **standard markdown** (not Jira wiki markup).
- **Jira via MCP:** Set `contentFormat: "markdown"` on `addCommentToJiraIssue`. Omitting it defaults to ADF and breaks rendering.
- **Short mode with expand:** Requires ADF JSON for the `expand` node. See **[references/adf-posting.md](references/adf-posting.md)** for the full ADF reference and two-part posting shortcut.
- **Full mode:** Use MCP with `contentFormat: "markdown"` directly.
- **GitHub:** Use markdown as-is.

**Iteration tracking:**
- Check if a previous "Triaging Notes" comment exists on the ticket.
- If yes: `_Groomed: {ISO_TIMESTAMP} (iteration N -- supersedes iteration N-1)_`
- If no: `_Groomed: {ISO_TIMESTAMP} (iteration 1)_`
- Do NOT edit or delete previous comments.

**After posting (all verdicts):**
1. Add the label `has_notes` to the ticket to indicate it has been groomed. Use `editJiraIssue` (Jira) or `gh issue edit --add-label` (GitHub) to add the label without removing existing labels.
2. Set the priority field on the ticket based on the Priority section in the triaging notes. Use `editJiraIssue` with the `priority` field (Jira) or add a `priority:P{N}` label (GitHub).

**`--dry-run`:** Display final notes in conversation. Ask "Post to ticket?" If confirmed, post and add label.

**Multi-ticket batch reporting:**
```
Grooming 3 tickets...
  - DL-1234: Triaging notes posted (review: PASS)
  - DL-1235: Triaging notes posted (review: NEEDS FIXES -- 2 corrections applied)
  - DL-1236: In progress -- staff engineer review
```

## Verbal Description (No Ticket)

When the user describes an issue without a ticket key:
1. Run the full investigation pipeline.
2. Present findings in conversation (not posted).
3. Ask: "Should I create a ticket with these triaging notes?" (respect project rules about ticket creation).

## Error Handling

| Failure | Behavior |
|---------|----------|
| Ticket not found / inaccessible | Fail fast, tell the user |
| Sub-agent exceeds context budget | Summarize what it has, note "investigation truncated due to complexity" |
| Comment post fails | Output triaging notes in conversation so nothing is lost |
| Comment posted with wrong format | Delete via `acli jira workitem comment delete --key {KEY} --id {ID}`, re-post with `contentFormat: "markdown"` |
| MCP tools unavailable | Inform user which tools are needed; fall back to `acli` for unsupported ops |
| Need to delete a Jira comment | Use `acli jira workitem comment delete --key {KEY} --id {ID}` |
| Repo not cloned locally | Skip that repo, note in findings, ask user for path |
| One sub-agent fails in a batch | Others continue; failure reported with partial findings |
| GitHub remote detection fails | Use relative paths instead of permalinks |
| Staff engineer review fails | Post unreviewed notes with disclaimer |

## Configuration

Add to CLAUDE.md to customize behavior:

```markdown
### Ticket Grooming
- Default ticket system: jira
- Jira site: zombie.atlassian.net
- GitHub org: cobalt-io
- dry-run: false
- grooming-mode: short  # short (default) | full (--full flag overrides this)
- Repos:
  - cobalt-pentest-api: ~/Documents/dev/cobalt-pentest-api
  - cobalt-admin-api: ~/Documents/dev/cobalt-admin-api
  - cobalt-web: ~/Documents/dev/cobalt-web
```

## Skills Referenced

| Skill | When | Scope |
|-------|------|-------|
| `systematic-debugging` | Phase 3 (always for code tickets) | Phases 1-3 only -- investigation, not implementation |
| `dispatching-parallel-agents` | Multi-ticket invocations | Parallel sub-agent dispatch with max 3 concurrency |
| `code-reviewer` (built-in) | Step 6 -- staff engineer review | Correctness, defensive coding, security, pattern matching |

## Reference Files

| File | Contents |
|------|----------|
| [references/investigation-prompt.md](references/investigation-prompt.md) | Full sub-agent investigation prompt template (Accuracy Rules 1-5, Pipeline phases 0-5, Output Format templates, Estimation table, P1-P3 Priority Matrix) |
| [references/staff-review-prompt.md](references/staff-review-prompt.md) | Staff engineer review sub-agent prompt template (verification strategy, review checklist, verdict format) |
| [references/adf-posting.md](references/adf-posting.md) | ADF expand node reference and posting details (JSON examples, acli commands, two-part posting shortcut) |

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
