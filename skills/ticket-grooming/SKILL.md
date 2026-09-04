---
name: ticket-grooming
description: "Investigate and groom tickets by dispatching sub-agents for codebase research, history, root cause analysis, and risk assessment. Posts structured Triaging Notes as a comment. Default is short-form (TLDR, key findings, risks, estimation, approach) with full investigation collapsed. Use --full for the complete report. Works with Jira, GitHub Issues, or any ticketing system."
tags: [workflow, project-management, debugging]
model: opus
---

# Ticket Grooming

Investigate one or more tickets, then post structured "Triaging Notes" as a comment. Notes are written for two audiences: PMs who need to understand the issue and prioritize it, and engineers who need to know where to start.

**Announce at start:** "Using ticket-grooming to investigate [ticket(s)]."

## Inputs

Extract from the user's message:
- **Ticket key(s) or URL(s)** (e.g., `ABC-1234`, `https://your-site.atlassian.net/browse/ABC-1234`, `#42`)
- OR a **verbal description** of the issue
- **Flags:**
  - `--dry-run` — preview without posting
  - `--full` — post the full report (no collapsed section)
- **Mode:** `--full` flag > `grooming-mode` in CLAUDE.md > default (`short`)

## Depth Calibration

Not every ticket needs the full pipeline. Match investigation depth to the ticket's complexity.

| Signal | What to run | Staff review? |
|--------|-------------|---------------|
| **Cosmetic** (label, copy, UI text) | Phases 0, 1 (shallow), 5. Skip root cause and risk assessment. | No |
| **Standard bug or feature** | Full pipeline (phases 0-5) | Yes |
| **Security / data integrity** | Full pipeline + security checklist in staff review | Yes |

**How to detect:** Classify during Phase 0 (understand the ticket). If the ticket is clearly cosmetic — no logic change, no data change, no authorization change — use the shallow path. When in doubt, use the full pipeline.

## Pre-Flight

Before dispatching sub-agents:

### 1. Check codebase index

If `codebase-memory-mcp` is available: call `index_status`, re-index if stale. Sub-agents verify via `index_status` but do NOT re-index.

### 2. Detect ticket system

Resolution order: CLAUDE.md config > URL pattern (`*.atlassian.net` = Jira, `github.com` = GitHub) > key pattern (`XX-1234` = Jira, `#1234` = GitHub) > ask user.

| System | Read | Search | Post |
|--------|------|--------|------|
| Jira | `getJiraIssue` | `searchJiraIssuesUsingJql` | `addCommentToJiraIssue` |
| GitHub | `gh issue view` | `gh issue list`, `gh pr list` | `gh issue comment` |

**acli fallback:** Use `acli` CLI when MCP lacks a capability (deleting comments, bulk edits). `acli --body-file` accepts ADF JSON only.

### 3. Resolve GitHub remote info

For each repo: extract org/repo from `git remote get-url origin`, get HEAD SHA via `git rev-parse HEAD`, verify pushed. Fallback: relative paths instead of permalinks.

### 4. Backend tickets: resolve every configured repo

Read the `Repos:` list from the configuration (see [references/configuration.md](references/configuration.md)) and resolve each one's path and GitHub slug.

Search **all** of them, not just the first — a ticket whose code lives in the second repo otherwise comes back as "no code found". If the configuration notes that one repo is migrating into another, search both and propose fixes only in the destination. Skip backend repos entirely for frontend-only tickets.

### 5. Multi-ticket: detect shared context

When grooming 2+ tickets: read all, check for overlapping components. If overlap found, run shared investigation once and pass as context. Otherwise dispatch independently.

### 6. Dispatch sub-agents

- 1 sub-agent per ticket (two-level only — sub-agents don't spawn their own)
- Max 3 concurrent. Queue additional as slots free.
- Each gets fresh context — no shared state between tickets.

## Investigation

Dispatch each sub-agent using the template in **[references/investigation-prompt.md](references/investigation-prompt.md)**. The prompt assembles content from these reference files:

| File | What it covers |
|------|---------------|
| [accuracy-rules.md](references/accuracy-rules.md) | 7 rules for grounding findings in actual code (no hallucinating names, verify mechanisms, challenge hypotheses, label verified vs speculative, stay on the reporter's problem, respect framework behavior, leave verification trails) |
| [pipeline.md](references/pipeline.md) | Investigation phases: understand the ticket, investigate the code, check history, find root cause, assess risks, write notes |
| [output-templates.md](references/output-templates.md) | Writing style rules (PM-readable, plain language, technical terms in context) and templates for short, full, and non-code tickets |
| [estimation-priority.md](references/estimation-priority.md) | T-shirt sizing table and P1-P3 priority matrix |

## Staff Engineer Review

After the investigation sub-agent returns, dispatch a review sub-agent using **[references/staff-review-prompt.md](references/staff-review-prompt.md)** with `model: "opus"`.

The review checks:
- Is the note about the reporter's actual problem? (highest priority)
- Are all named entities verified in the codebase?
- Are hypotheses backed by evidence with counterarguments?
- Are visible sections PM-readable? (no unexplained jargon, plain language)
- Correctness, security, and pattern adherence

**Skip staff review only for cosmetic tickets** (per depth calibration above).

## Post the Notes

After staff review:

| Verdict | Action |
|---------|--------|
| **PASS** | Post as-is |
| **PASS WITH NOTES** | Post as-is, mention notes to user |
| **NEEDS FIXES** | Apply fixes, post corrected version, report what changed |

**Posting rules:** See [output-templates.md](references/output-templates.md) for full posting and iteration tracking details.
- **Jira (full mode):** Set `contentFormat: "markdown"` on `addCommentToJiraIssue`. Omitting it breaks rendering.
- **Jira (short mode):** MUST use ADF JSON — markdown cannot produce the expand node. Construct the full ADF document (visible sections + expand node), write to `/tmp/triaging-notes-{TICKET_KEY}.json`, post via `acli --body-file`. See [adf-posting.md](references/adf-posting.md) for the exact procedure and skeleton template. NEVER use HTML `<details>` tags — Jira does not render them.
- **GitHub:** Post markdown directly.

**After posting:**
1. Add `has_notes` label to the ticket
2. Set priority field based on the Priority section in the notes

**`--dry-run`:** Show notes in conversation. Ask "Post to ticket?" before posting.

**Multi-ticket progress:**
```
Grooming 3 tickets...
  - ABC-1234: Posted (review: PASS)
  - ABC-1235: Posted (review: NEEDS FIXES -- 2 corrections applied)
  - ABC-1236: In progress -- staff review
```

## No Ticket? No Problem.

When the user describes an issue verbally (no ticket key):
1. Run the investigation pipeline (default to standard depth unless the description is clearly cosmetic)
2. Present findings in conversation
3. Ask: "Should I create a ticket with these notes?" (respect project rules about ticket creation)

## Reference Files

| File | Contents |
|------|----------|
| [investigation-prompt.md](references/investigation-prompt.md) | Sub-agent prompt template with placeholder assembly |
| [accuracy-rules.md](references/accuracy-rules.md) | 7 investigation accuracy rules |
| [pipeline.md](references/pipeline.md) | Investigation phases 0-5 |
| [output-templates.md](references/output-templates.md) | Writing style rules and output templates (short, full, non-code) |
| [estimation-priority.md](references/estimation-priority.md) | T-shirt estimation table and P1-P3 priority matrix |
| [staff-review-prompt.md](references/staff-review-prompt.md) | Staff engineer review prompt and checklist |
| [adf-posting.md](references/adf-posting.md) | ADF expand node reference and posting details |
| [error-handling.md](references/error-handling.md) | What to do when things fail |
| [framework-detection.md](references/framework-detection.md) | Framework detection logic and per-framework investigation rules |
| [configuration.md](references/configuration.md) | CLAUDE.md configuration options |

## Skills Referenced

| Skill | When |
|-------|------|
| `systematic-debugging` | Root cause analysis (phases 1-3 only, no implementation) |
| `dispatching-parallel-agents` | Multi-ticket invocations (max 3 concurrent) |
