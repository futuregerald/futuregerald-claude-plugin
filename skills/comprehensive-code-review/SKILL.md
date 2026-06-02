---
name: comprehensive-code-review
description: Use when performing code review on a PR, reviewing code changes before merge, or when a GitHub code review is requested or received - orchestrates parallel sub-agents for correctness and safety review
tags: [quality, review, security, sql, code-review, pr, architecture, owasp, defensive-coding]
model: sonnet
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

### Framework Detection

Detect from project files: `Gemfile` (Rails), `package.json` (Next/React/Express), `go.mod` (Go), `pyproject.toml` (Django). Set `{FRAMEWORK_CONTEXT}` per [references/framework-rules.md](references/framework-rules.md). Include in BOTH sub-agent prompts.

### Requirements Resolution

1. **Jira ticket** — key in branch name or PR body, fetch via `getJiraIssue`
2. **GitHub issue** — linked in PR body (`Closes #N`, `Fixes #N`)
3. **Plan doc** — if PR body references a plan file, read it

Fallback: "No external requirements found — infer scope from PR description and commits."

### Review Comments

If a PR exists, fetch existing inline comments and review bodies:

```bash
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
PR_NUMBER=$(gh pr view --json number -q .number)
gh api repos/$REPO/pulls/$PR_NUMBER/comments \
  --jq '[.[] | {path: .path, line: .line, body: (.body | .[0:300])}] | .[-20:]'
gh api repos/$REPO/pulls/$PR_NUMBER/reviews \
  --jq '[.[] | select(.body | contains(":warning: **Superseded**") | not) | select(.body != "") | {state: .state, body: (.body | .[0:500])}] | .[-10:]'
```

If no PR or no comments: `"(none — this is a first review)"`.

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

## Phase 2: Dispatch Sub-Agents (PARALLEL)

**Both sub-agents MUST be dispatched in parallel — one message, two Agent tool invocations.** If a sub-agent fails: re-dispatch once, then record "Review Incomplete."

### Sub-Agent 1: Correctness Review

Read [references/correctness-prompt.md](references/correctness-prompt.md). Replace all `{PLACEHOLDERS}` with Phase 1 values. Covers: code quality, defensive coding, architecture, testing, scope, patterns, simplification.

### Sub-Agent 2: Safety Review

Read [references/safety-prompt.md](references/safety-prompt.md). Replace all `{PLACEHOLDERS}` with Phase 1 values. Covers: OWASP Top 10, input validation, mass assignment, conditional SQL/DB performance.

### Shared Finding Format

```
**[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
- File: `path/to/file:line_number`
- Problem: [What's wrong and why it matters]
- Recommendation: [Specific fix with code snippet if helpful]
- Impact: [What happens if not fixed]
Security findings: include CWE ID. Pattern deviations: include current vs existing pattern.
```

### Voice (applies to ALL sub-agent output)

See [references/voice.md](references/voice.md). Sub-agents MUST follow these guidelines.

## Phase 3: Consolidate Results

### Framework-Awareness Validation (MANDATORY)

Before including ANY finding, validate against the framework's type system:

1. **SQL Injection**: Does the type system make injection impossible? (Ruby `.to_i`, Go int types) **Drop if type-safe.**
2. **Missing security**: Does the framework handle it? (Rails strong params, Django CSRF) **Don't flag built-ins.**
3. **Performance**: Calibrated to actual dataset sizes? No unsubstantiated multipliers.
4. **Conventions**: Codebase convention or textbook convention? Only flag codebase deviations.

**If a sub-agent flags something the framework makes impossible, DROP it entirely.**

### Deduplication

Same `file:line` + same root cause = one finding. Use more severe categorization. For auth/injection: use Safety agent's recommendation. Note which dimensions caught it.

### Report

Use the template in [references/report-format.md](references/report-format.md). If `{FILE_LIST}` contains skill files, see the skill file review section in that reference.

## Integration

- **Replaces Phase 7 (CODE REVIEW) and Phase 8 (SQL REVIEW)** in the development lifecycle
- For PR reviews: fetch metadata, gather diff, fetch comments, dispatch sub-agents, post report
- For receiving review feedback: use `receiving-code-review` skill instead

## Rules

**NEVER:** Review code yourself | Skip the safety audit | Run sub-agents sequentially | Mark everything CRITICAL | Pass verdict with CRITICAL issues

**ALWAYS:** Dispatch BOTH sub-agents in a SINGLE parallel call | Include file:line for every finding | Provide actionable recommendations | Deduplicate across dimensions | Present unified report
