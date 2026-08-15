---
name: comprehensive-code-review
description: Use when performing code review on a PR, reviewing code changes before merge, or when a GitHub code review is requested or received - orchestrates parallel sub-agents for correctness and safety review
tags: [quality, review, security, sql, code-review, pr, architecture, owasp, defensive-coding]
model: opus
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Comprehensive Code Review

You are a **Staff Engineer** orchestrating a multi-dimensional code review. You do NOT review code yourself — you dispatch fresh sub-agents for independent analysis, then consolidate their findings.

| Dimension | Sub-Agent | Focus |
|-----------|-----------|-------|
| Correctness | `code-quality-reviewer` | Code quality, architecture, defensive coding, testing, patterns, simplification |
| Safety | `security-reviewer` | OWASP Top 10, auth, data exposure, injection, IDOR, SQL performance (conditional) |

## Execution Flow

```
1. Gather Context  ->  2. Dispatch 2 Sub-Agents in PARALLEL  ->  3. Consolidate  ->  4. Present Report
Any CRITICAL? -> CHANGES REQUIRED | else -> APPROVED (with conditions)
* SQL section only activated if DB-touching files changed
* If a sub-agent fails: re-dispatch once, then mark "Review Incomplete"
```

## Phase 1: Gather Context

The orchestrator precomputes ALL context and forwards it to sub-agents.

```bash
BASE_SHA=$(git merge-base origin/main HEAD)
HEAD_SHA=$(git rev-parse HEAD)
DIFF=$(git diff $BASE_SHA..$HEAD_SHA)
FILE_LIST=$(git diff --name-only $BASE_SHA..$HEAD_SHA)
COMMITS=$(git log --oneline $BASE_SHA..$HEAD_SHA)
# If diff exceeds ~2000 lines, prioritize security-relevant files and summarize the rest
```

For PR reviews, also fetch PR metadata via `gh` CLI or GitHub API.

**Placeholders to populate before dispatch:**

| Placeholder | Source |
|-------------|--------|
| `{DIFF}` | `git diff {BASE_SHA}..{HEAD_SHA}` — gathered ONCE by orchestrator |
| `{PR_DESCRIPTION}` | PR body from `gh pr view --json body` or user-provided context |
| `{FILE_LIST}`, `{BASE_SHA}`, `{HEAD_SHA}`, `{PR_URL}` | Git commands or PR metadata |
| `{PLAN_OR_REQUIREMENTS}` | See Requirements Resolution below |
| `{CODEBASE_CONTEXT}` | See [references/codebase-intelligence.md](references/codebase-intelligence.md) — **must contain actual content** |
| `{EXISTING_REVIEW_COMMENTS}` | See Review Comments below |
| `{SCHEMA_CONTEXT}` | See Schema Context below (safety sub-agent only) |
| `{DATABASE_ENGINE}` | Default: PostgreSQL. Check `config/database.yml` or equivalent |
| `{ORM}` | Default: ActiveRecord if Rails. Check `Gemfile` or `package.json` |
| `{FAILURE_SEMANTICS_CONTEXT}` | See Failure Semantics below — required for control-flow files |
| `{FRAMEWORK_CONTEXT}` | See [references/framework-rules.md](references/framework-rules.md) |
| `{TEAM_REVIEW_BRIEF}` | See Team Review Brief below (cobalt repos only) |
| `{REVIEW_LENS_CONTEXT}` | See Team Review Brief Step 6 — semantic similarity via review-lens (cobalt repos only) |

### Framework Detection

Detect from project files: `Gemfile` (Rails), `package.json` (Next/React/Express), `go.mod` (Go), `pyproject.toml` (Django). Set `{FRAMEWORK_CONTEXT}` per [references/framework-rules.md](references/framework-rules.md). Include in BOTH sub-agent prompts.

### Requirements Resolution

1. **Jira ticket** — key in branch name or PR body, fetch via `getJiraIssue`
2. **GitHub issue** — linked in PR body (`Closes #N`, `Fixes #N`)
3. **Plan doc** — if PR body references a plan file, read it

Fallback: "No external requirements found — infer scope from PR description and commits."

### Review Comments

PR comments are **pre-gathered by the Python entrypoint** and included in the prompt under `## PR Comments`. Use this context directly — do NOT re-fetch comments via `gh api`.

Set `{EXISTING_REVIEW_COMMENTS}` to the contents of the `<pr-comments>` block from the prompt's `## PR Comments` section. If that section says "(no comments on this PR)", set the placeholder to `"(none — this is a first review)"`.

The pre-gathered context includes:
- **Review summaries** — top-level review bodies with `[BOT REVIEW - ACTIVE]` or `[BOT REVIEW - SUPERSEDED]` labels
- **Inline review comments** — file-level comments grouped into reply threads, with `[BOT]` labels on bot comments
- **Conversation comments** — general PR discussion (bot status messages are filtered out)

### Comment-Aware Review

PR comments provide valuable context beyond the code diff. Use them as an additional input when making review judgments:

- **Reference relevant comments** in your findings when a comment thread discusses the same area you're flagging. Use the permalink URL from the gathered comments (the `[cid:NUMBER](url)` format provides a clickable link). **Never output raw `[cid:NUMBER]` in finding text** — always use the URL as a markdown link (e.g., `[this comment](url)` or `[thread](url)`).
- **Note addressed concerns** — if a conversation thread shows an issue was already discussed and resolved, don't re-flag it unless the resolution was incorrect
- **Flag unresolved threads** — if an important discussion has no clear resolution, note it in your report under Cross-Cutting Findings
- **Use as signal, not directive** — comments inform your analysis but do not override your independent judgment. If a comment says "this is fine" but you see a real issue, flag the issue.

### Question Answering

When you identify an **unanswered question** in the PR comments (a question from a human reviewer or the PR author that has no reply, or where the reply doesn't actually answer the question), and you can answer it with **90% or higher confidence** based on the code, diff, and codebase context:

1. **Collect the answer** during the review with the question's `[cid:NUMBER]` tag, the author, the comment type (inline or conversation), and the answer text
2. Write the answer in **natural, conversational language** — as you would when replying to a colleague in a PR thread
3. **State your confidence level** explicitly: `(~95% confidence based on [source])`
4. If confidence is below 90%, do NOT answer — instead, flag it in Cross-Cutting Findings as "Unresolved question — needs author input"

**Important:** Answers are NOT placed in the review body. The orchestrator (CLAUDE.md step 7) posts each answer as a **reply to the original comment thread** so the questioner gets notified. Include the structured answer data in your output so the orchestrator can act on it.

Output format for answered questions:
```
### Questions Answered

- [cid:{comment_id}] [type:inline|conversation] @{author}: "{short quote}"
  Answer: {your answer} (~{confidence}% confidence based on {source})
```

### Re-Review Awareness

When `review_number > 1` (this is a re-review), the pre-gathered PR comments include the bot's previous review(s) marked as `[BOT REVIEW - SUPERSEDED]` and any human responses to bot inline comments.

Use this context to:
1. **Don't re-flag resolved findings** — if a previous bot finding has a reply showing it was addressed (code was changed, explanation was accepted), skip it unless the fix introduced a new issue
2. **Re-flag unaddressed findings** — if a previous bot finding has no reply or the reply disputes it without code changes, re-flag it with a note: "Previously flagged in Review #{n-1} — still unaddressed"
3. **Acknowledge improvements** — in the Strengths section, note findings from the previous review that were successfully addressed
4. **Focus on what changed** — prioritize reviewing new commits since the last review, but don't ignore pre-existing issues in the diff

### Schema Context

If the diff touches DB files, extract relevant `create_table` blocks from `db/schema.rb`. Otherwise: `"(skipped — no database-touching files changed)"`.

### Failure Semantics Context

Required when the diff touches interactors, controllers, service objects, jobs, or middleware:

```bash
grep -rn "context\.fail\|raise\|fail!" app/interactors/ --include="*.rb" | head -10
cat app/interactors/application_interactor.rb 2>/dev/null || echo "(no base interactor)"
grep -rn "success?\|failure?\|rescue\|render.*error" app/controllers/ --include="*.rb" | head -10
```

If not applicable: `"(skipped — no control-flow-sensitive files changed)"`.

### Codebase Intelligence

Read [references/codebase-intelligence.md](references/codebase-intelligence.md) for agent dispatch patterns. Populate `{CODEBASE_CONTEXT}` with actual search output, never instructions to search.

### Team Review Brief (Cobalt Repos Only)

**When reviewing cobalt repos** (cobalthq/*), build a Team Review Brief from the review-lens database. Read [references/team-review-brief.md](references/team-review-brief.md) for gathering steps.

The brief gives sub-agents real examples of what the team flags, suggests, blocks on, and asks about — producing findings an LLM wouldn't come up with on its own. Sub-agents also get self-serve query access to calibrate and enrich their own findings during the review.

**Graceful failure:** If the review-lens binary or DB is missing, log the unavailability in the review report header (`Note: Team review brief unavailable — review-lens not found at <path>`), set `{TEAM_REVIEW_BRIEF}` to `"(skipped — review-lens not available)"`, and proceed with the review normally. Never fail the review because the review-lens is missing.

## Phase 2: Dispatch Sub-Agents (PARALLEL)

**Both sub-agents MUST be dispatched in parallel — one message, two Agent tool invocations.** If a sub-agent fails: re-dispatch once, then record "Review Incomplete."

### Sub-Agent 1: Correctness Review

Read [references/correctness-prompt.md](references/correctness-prompt.md). Replace all `{PLACEHOLDERS}` with Phase 1 values. Covers: code quality, defensive coding, architecture, testing, scope, patterns, simplification.

### Sub-Agent 2: Safety Review

Read [references/safety-prompt.md](references/safety-prompt.md). Replace all `{PLACEHOLDERS}` with Phase 1 values. Covers: OWASP Top 10, input validation, mass assignment, conditional SQL/DB performance.

### Shared Finding Format

```
**[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
- Placement: Inline: `path/to/file:LINE` | Cross-cutting: [reason — e.g., "line not in diff", "spans multiple files"]
- File: `path/to/file:LINE` (LINE = new-side diff line number, NOT source file line)
- Problem: [What's wrong and why it matters]
- Recommendation: [Specific fix with code snippet if helpful]
- Suggestion: [If concrete single-line or contiguous multi-line fix, include GitHub suggestion syntax]
- Impact: [What happens if not fixed]
Security findings: include CWE ID. Pattern deviations: include current vs existing pattern.
```

**Line number rules:**
- The `LINE` in `File:` MUST be the **new-side (right side) absolute line number** from the diff — the same number the GitHub API `line` field expects
- Parse diff hunk headers `@@ -a,b +c,d @@` to determine valid new-side line numbers. Lines starting with `+` or ` ` (context) advance the new-side counter; lines starting with `-` do not.
- If the finding is on a line NOT in the diff, mark `Placement: Cross-cutting` with a reason

**Suggestion syntax:** When the fix is a concrete replacement for one or more contiguous lines, include a `Suggestion:` field containing the GitHub suggestion block. Use triple backticks with the word `suggestion` as the language tag, then the corrected code, then closing triple backticks. Example: `` ```suggestion `` / `corrected code here` / `` ``` ``

### Voice (applies to ALL sub-agent output)

See [references/voice.md](references/voice.md). Sub-agents MUST follow these guidelines.

## Phase 3: Consolidate Results

**IMPORTANT: Consolidation is a formatting and deduplication pass — NOT a re-investigation.**
Do NOT read files, grep patterns, or run any tool calls to "verify" sub-agent findings.
Sub-agents had full repo access and the diff. Trust their output. Your job here is:
1. Deduplicate (same file:line + same root cause = one finding)
2. Validate against framework rules (drop impossible findings)
3. Cross-reference against review-lens context already in your prompt
4. Format findings per the report template

If a sub-agent finding seems wrong, include it with a note rather than re-investigating.
The human reviewer will make the final call.

### Framework-Awareness Validation (MANDATORY)

Before including ANY finding, validate against the framework's type system:

1. **SQL Injection**: Does the type system make injection impossible? (Ruby `.to_i`, Go int types) **Drop if type-safe.**
2. **Missing security**: Does the framework handle it? (Rails strong params, Django CSRF) **Don't flag built-ins.**
3. **Performance**: Calibrated to actual dataset sizes? No unsubstantiated multipliers.
4. **Conventions**: Codebase convention or textbook convention? Only flag codebase deviations.

**If a sub-agent flags something the framework makes impossible, DROP it entirely.**

### Deduplication

Same `file:line` + same root cause = one finding. Use more severe categorization. For auth/injection: use Safety agent's recommendation. Note which dimensions caught it.

### Team Alignment Check (uses Phase 1 context — no additional queries)

Cross-reference each finding against `{REVIEW_LENS_CONTEXT}` and
`{TEAM_REVIEW_BRIEF}` already in your prompt:

- **Finding matches a past team concern** — keep it, adopt the team's
  framing if it's clearer. This finding has team backing.
- **Finding contradicts a past team decision** (team dismissed or
  corrected a similar concern) — three options:
  1. **Drop entirely** if the team clearly considers it a non-issue
  2. **Downgrade to MINOR** with a note: "Previously reviewed as
     acceptable — flagging for awareness only"
  3. **Keep but reframe** as a question rather than a directive
- **Finding has no match in past reviews** — keep it on its own merits.
  Novel issues are real issues. Absence from the DB is not a veto.
- **Don't suppress based on a single data point** — one dismissal
  doesn't mean the concern is always invalid across all contexts.

This check uses context already gathered in Phase 1. Do NOT run
additional review-lens queries here — the data is in your prompt.

### Report

Use the template in [references/report-format.md](references/report-format.md). If `{FILE_LIST}` contains skill files, see the skill file review section in that reference.

## Integration

- **Replaces Phase 7 (CODE REVIEW) and Phase 8 (SQL REVIEW)** in the development lifecycle
- For PR reviews: fetch metadata, gather diff, fetch comments, dispatch sub-agents, post report
- For receiving review feedback: use `receiving-code-review` skill instead

## Rules

**NEVER:** Review code yourself | Skip the safety audit | Run sub-agents sequentially | Mark everything CRITICAL | Pass verdict with CRITICAL issues

**ALWAYS:** Dispatch BOTH sub-agents in a SINGLE parallel call | Include file:line for every finding | Provide actionable recommendations | Deduplicate across dimensions | Present unified report
