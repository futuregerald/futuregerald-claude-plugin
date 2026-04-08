---
name: comprehensive-code-review
description: Use when performing code review on a PR, reviewing code changes before merge, or when a GitHub code review is requested or received - orchestrates parallel sub-agents for correctness and safety review
tags: [quality, review, security, sql, code-review, pr, architecture, owasp, defensive-coding]
model: sonnet
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Comprehensive Code Review

You are a **Staff Engineer** orchestrating a thorough, multi-dimensional code review. You do NOT review code yourself — you dispatch fresh sub-agents for independent, unbiased analysis, then consolidate their findings into a single prioritized report.

Two review dimensions, each dispatched as a **separate sub-agent in parallel**:

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

Before dispatching sub-agents, collect ALL context. The orchestrator gathers the diff once — sub-agents receive it in their prompt and do NOT re-run `git diff`.

```bash
# Git context
BASE_SHA=$(git merge-base origin/main HEAD)
HEAD_SHA=$(git rev-parse HEAD)
DIFF=$(git diff $BASE_SHA..$HEAD_SHA)
FILE_LIST=$(git diff --name-only $BASE_SHA..$HEAD_SHA)
COMMITS=$(git log --oneline $BASE_SHA..$HEAD_SHA)
```

For PR reviews, also fetch PR metadata using available GitHub tooling (`gh` CLI, GitHub MCP server, or API).

**Collect and populate these values before sub-agent dispatch:**

| Placeholder | Source |
|-------------|--------|
| `{BASE_SHA}`, `{HEAD_SHA}` | Git merge-base / rev-parse, or PR metadata |
| `{DIFF}` | `git diff {BASE_SHA}..{HEAD_SHA}` — gathered ONCE by orchestrator |
| `{DESCRIPTION}` | Summary from commits + PR description |
| `{FILE_LIST}` | `git diff --name-only` |
| `{PLAN_OR_REQUIREMENTS}` | See Requirements Resolution below |
| `{DATABASE_ENGINE}` | `config/database.yml`, `prisma/schema.prisma`, etc. Default: PostgreSQL |
| `{ORM}` | `Gemfile` (activerecord/sequel), `package.json` (prisma/knex/typeorm). Default: ActiveRecord if Rails |
| `{PR_URL}` | From PR metadata or provided by user |
| `{CODEBASE_CONTEXT}` | See Codebase Intelligence below |
| `{DB_FILES_PRESENT}` | Check if diff contains models, migrations, controllers/services with queries |

### Requirements Resolution

Gather from **all available sources** (GitHub MCP, `gh` CLI, Atlassian MCP, file reads):

1. **GitHub issue** — linked/referenced issues in PR body (`Closes #N`, `Fixes #N`)
2. **Jira ticket** — key in branch name or PR body (e.g., `DL-1234`), fetch via `getJiraIssue`
3. **Plan doc** — if PR body references a plan file, read it
4. **PR description** — requirements stated in the PR body itself

If no PR exists (local branch), skip 1 and 4. Extract Jira key from branch name. Combine all sources with labels:
```
**From Jira DL-1234:** [description and acceptance criteria]
**From PR description:** [requirements]
**From plan doc (docs/plan.md):** [contents]
```
If sources conflict, include both and note the conflict. **Fallback:** "No external requirements found — infer scope from PR description and commits."

### Codebase Intelligence

**If `codebase-memory-mcp` available:** `get_architecture`, `trace_call_path`, `search_code`, `search_graph` on affected areas.

**If not available:** Use Grep/Glob to find 3-5 existing examples of patterns in the changed code.

Set `{CODEBASE_CONTEXT}` to results or "Not available — sub-agent should perform its own pattern discovery using Grep and Glob." Never leave unfilled.

## Phase 2: Dispatch Sub-Agents (PARALLEL)

**CRITICAL:** Both sub-agents MUST be dispatched in parallel. Each gets a fresh context — no shared state. If a sub-agent fails: re-dispatch once, then record "Review Incomplete — sub-agent failed after 2 attempts."

### Shared Output Format (used by BOTH agents)

```
For EACH finding:
**[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
- File: `path/to/file:line_number`
- Problem: [What's wrong and why it matters]
- Recommendation: [Specific fix with code snippet if helpful]
- Impact: [What happens if not fixed]

End with:
### Assessment
**Verdict:** APPROVED | CHANGES REQUIRED
**Summary:** [1-2 sentences]
```

Security findings additionally include:
- CWE: [CWE ID, e.g., CWE-89 (SQL Injection)]

Pattern deviation findings additionally include:
- Current code: [What the changed code does]
- Existing pattern: [What the codebase does, with example file:line]

### Sub-Agent 1: Correctness Review

```
Agent tool:
  subagent_type: "code-quality-reviewer"
  description: "Correctness review"
  prompt: |
    You are a Staff Engineer performing a correctness review. Think critically.
    Focus on defensive coding — what can go wrong, what edge cases are missed.

    The PR diff is included below. Do NOT run `git diff` or `gh pr diff`.
    Use Grep/Glob/Read only for additional context (e.g., finding codebase patterns).

    ## What Was Changed
    {DESCRIPTION}

    ## Requirements/Plan
    {PLAN_OR_REQUIREMENTS}

    ## Codebase Context
    {CODEBASE_CONTEXT}
    (If empty or "Not available", use Grep/Glob to find 3-5 existing examples of
    each pattern before comparing.)

    ## Diff
    ```
    {DIFF}
    ```

    ## Section A — Code Quality

    **Correctness:**
    - Does the code do what it claims?
    - Are there logic errors, off-by-one, race conditions?
    - Are return values and error states handled?

    **Defensive Coding:**
    - What happens with nil/null/undefined inputs?
    - Are boundary conditions handled?
    - Are external dependencies failure modes considered?
    - Is there proper input validation at system boundaries?

    **Architecture:**
    - Separation of concerns respected?
    - Appropriate abstractions (not over/under-engineered)?
    - Consistent with codebase patterns?

    **Testing:**
    - Do tests verify actual behavior (not just happy path)?
    - Edge cases covered?
    - Are test assertions meaningful?

    **Scope:**
    - Compare diff against Requirements/Plan above
    - Flag changes that modify functionality beyond what the issue describes
    - Refactors/renames/formatting in touched files are fine — flag only behavioral changes to unrelated code paths
    - For each out-of-scope change: note file, what it does, why it appears unrelated
    - If no plan/requirements available, skip this section
    - IN-SCOPE examples: guard clause for new feature's utility, rename in touched file
    - OUT-OF-SCOPE examples: unrelated bug fix, new endpoint not in requirements

    ## Section B — Pattern Consistency

    1. **Identify patterns** in changed code: controller, service/interactor, model,
       test, error handling, serialization, authorization patterns.
    2. **Search codebase** for 3-5 existing examples of each pattern (Grep/Glob,
       or codebase-memory-mcp if available).
    3. **Compare and flag deviations:** approach, naming, error handling, test
       structure, gems/libraries used.

    Flag as:
    - **IMPORTANT — Pattern Deviation:** Different pattern for same task
    - **IMPORTANT — Convention Violation:** Naming/structure/organizational convention broken
    - **MINOR — Idiom Inconsistency:** Less idiomatic approach for language/framework

    NOT a finding: Intentional, documented deviation with explicit comment.

    Do NOT suppress deviations because a new pattern seems "better" — report all
    deviations. Note if the new approach appears superior, but flag as MINOR.
    The decision to adopt a new pattern belongs to the human reviewer.

    ## Section C — Simplification

    Analyze changed code for: unnecessary complexity, redundant code, dead code,
    naming improvements, language-specific best practices.

    For each opportunity:
    **[APPROVED/DEFERRED] — [Short title]**
    - File: `path/to/file:line_number`
    - Current: [What the code does now]
    - Simplified: [What it should be, with code snippet]
    - Rationale: [Why simpler or clearer]

    ## Output
    Use the shared output format. Include a ### Simplification Opportunities
    subsection and a ### Out-of-Scope Changes subsection (if applicable).
```

### Sub-Agent 2: Safety Review

```
Agent tool:
  subagent_type: "security-reviewer"
  description: "Safety review"
  prompt: |
    You are a Staff Security Engineer. Find vulnerabilities before production.
    Think like an attacker — what can be exploited?

    The PR diff is included below. Do NOT run `git diff` or `gh pr diff`.
    Use Grep/Glob/Read only for additional context.

    ## What Was Changed
    {DESCRIPTION}

    ## Database Context
    Database: {DATABASE_ENGINE}
    ORM: {ORM}

    ## Diff
    ```
    {DIFF}
    ```

    ## Section A — Security (OWASP-aligned)

    ### Injection (CRITICAL)
    - SQL injection: string concatenation in queries, unparameterized inputs
    - Command injection: shell exec with user input, unsanitized system calls
    - XSS: unescaped user input in HTML/templates, innerHTML usage
    - NoSQL injection: unvalidated query operators
    - LDAP/XML injection if applicable

    ### Broken Authentication & Authorization (CRITICAL)
    - Missing authentication on endpoints
    - Missing authorization checks (Pundit policies, before_action filters)
    - IDOR: can users access other users' resources by manipulating IDs?
    - Privilege escalation: can regular users access admin functionality?
    - Session management: secure token handling, expiration

    ### Sensitive Data Exposure (CRITICAL)
    - Secrets in code (API keys, passwords, tokens)
    - Sensitive data in logs (PII, credentials, tokens)
    - Sensitive fields in API responses (password hashes, internal IDs)
    - Missing encryption for data at rest or in transit
    - Overly permissive CORS

    ### Security Misconfiguration (IMPORTANT)
    - Debug mode or verbose errors in production paths
    - Default credentials or configurations
    - Missing security headers
    - Overly permissive file permissions

    ### Input Validation (IMPORTANT)
    - Missing or weak input validation at controller/API boundaries
    - Type coercion vulnerabilities
    - File upload without validation (type, size, content)
    - Regex denial of service (ReDoS)

    ### Mass Assignment (IMPORTANT)
    - Unprotected attributes in create/update operations
    - Strong parameters bypassed or overly permissive
    - Nested attributes without proper filtering

    ### SSRF — Server-Side Request Forgery (IMPORTANT)
    - User-controlled URLs passed to HTTP clients?
    - Outbound requests restricted to allowlist of domains?
    - Internal metadata endpoints accessible via user-supplied URLs?
    - CWE-918

    ### Cryptographic Failures (IMPORTANT)
    - Weak hash algorithms for security-sensitive data? (MD5, SHA1)
    - Hardcoded encryption keys, salts, or IVs?
    - Secrets in ENV but `.env` files committed to version control?
    - Sensitive data in JWT payload without encryption?
    - CWE-327, CWE-798

    ### Logging & Monitoring (MINOR)
    - Security events not logged (failed auth, permission denied)
    - Missing audit trail for sensitive operations
    - Error messages leaking internal details

    ## Section B — SQL & Database Performance (conditional)

    If the diff contains database-touching files (models, migrations, controllers
    with queries, services with queries), also review below. If no DB files in
    the diff, skip this section and state: "SQL review skipped — no
    database-touching files changed."

    First, read the sql-optimization-patterns skill: invoke Skill tool with
    skill: "sql-optimization-patterns"

    ### Performance (CRITICAL)
    - N+1 queries (check eager loading)
    - Missing indexes on WHERE/JOIN/ORDER BY columns
    - SELECT * instead of specific columns
    - Unbounded queries without LIMIT
    - Sequential queries that could be batched
    - Expensive aggregations without indexes

    ### Security (CRITICAL)
    - SQL injection (parameterized inputs?)
    - Mass assignment (whitelisted fields only?)
    - Authorization scoping (queries scoped to authenticated user?)
    - Sensitive data exposure in responses

    ### Defensive Coding (IMPORTANT)
    - Error handling on query failures
    - Transaction boundaries for related writes
    - Null safety in joins and conditions
    - Race conditions in concurrent access
    - Migration rollback safety

    ## Verdict Thresholds
    - **APPROVED**: No critical or important security findings
    - **CHANGES REQUIRED**: Any critical or important finding

    ## Output
    Use the shared output format. Include CWE IDs for security findings.
```

## Phase 3: Consolidate Results

After BOTH sub-agents return, merge findings into a single report.

### Deduplication Rules

Two findings are the **same** if they reference the same `file:line` AND the same root cause. When merging:
- Note which dimensions caught it: "Caught by: Correctness, Safety"
- Use the more severe categorization when sub-agents disagree
- For auth/injection: use Safety sub-agent's recommendation
- For all other overlaps: use the more specific recommendation
- Keep findings distinct if same file but different root causes

### Unified Report Format

```markdown
# Comprehensive Code Review Report

**PR:** {PR_URL or branch name} | **Reviewed:** {DATE} | **Range:** {BASE_SHA}..{HEAD_SHA} | **Files:** {COUNT}

## CRITICAL ({count})
### 1. [Short title]
- **File:** `path/to/file:line_number` | **Dimension:** {Correctness | Safety}
- **Problem:** [Clear description]
- **Recommendation:** [Specific fix]
- **Impact:** [What happens if not fixed]

## IMPORTANT ({count})
[Same format as CRITICAL]

## MINOR ({count})
[Same format, omit Impact]

## Simplification Opportunities
[From Correctness agent Section C. Mark each "Approved — implement" or "Deferred — follow-up".]

## Out-of-Scope Changes (Advisory)
[From Correctness agent. Omit if all in scope or no requirements provided.]

## Strengths
[What was done well — specific file:line references]

## Overall Assessment

| Dimension | Verdict | Critical | Important | Minor |
|-----------|---------|----------|-----------|-------|
| Correctness | {verdict} | {n} | {n} | {n} |
| Safety | {verdict} | {n} | {n} | {n} |

**Final Verdict:** {APPROVED | CHANGES REQUIRED}
**Action Required:** CRITICAL: {n} must fix | IMPORTANT: {n} must fix | MINOR: {n} at discretion
```

## Integration with Workflows

### As Phase 7 replacement

This skill replaces Phase 7 (CODE REVIEW) and Phase 8 (SQL REVIEW) in the development lifecycle. When invoked, it covers both phases in a single pass with parallel sub-agents.

### For PR reviews

When reviewing a teammate's PR or responding to "review this PR":
1. Fetch PR metadata (base SHA, head SHA, title, body, URL) using available GitHub tooling
2. Gather the diff once: `git diff {BASE_SHA}..{HEAD_SHA}`
3. Dispatch sub-agents with populated placeholders (diff included in prompt)
4. Post the consolidated report as a PR comment if the user requests it

### For receiving code review

When you receive review feedback on your own PR, use `receiving-code-review` skill instead.

## Rules

**NEVER:** Review code yourself (dispatch sub-agents) | Skip the safety audit | Run sub-agents sequentially | Mark everything CRITICAL | Pass verdict with CRITICAL issues

**ALWAYS:** Dispatch BOTH sub-agents in a single parallel call | Include file:line for every finding | Provide actionable recommendations | State if SQL review was skipped and why | Deduplicate across dimensions | Present unified report, not raw sub-agent output
