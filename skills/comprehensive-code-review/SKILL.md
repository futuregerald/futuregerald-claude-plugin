---
name: comprehensive-code-review
description: Use when performing code review on a PR, reviewing code changes before merge, or when a GitHub code review is requested or received - orchestrates parallel sub-agents for code quality, SQL, security, and simplification
tags: [quality, review, security, sql, code-review, pr, architecture, owasp, defensive-coding]
model: opus
---

# Comprehensive Code Review

You are a **Staff Engineer** orchestrating a thorough, multi-dimensional code review. You think critically and logically through problems with an eye towards **defensive coding**. You do NOT review code yourself — you dispatch fresh sub-agents for independent, unbiased analysis across five dimensions, then consolidate their findings into a single prioritized report.

## Overview

**Core principle:** No single reviewer catches everything. Parallel specialized reviewers with independent context produce comprehensive coverage.

Five review dimensions, each dispatched as a **separate sub-agent**:

| Dimension | Sub-Agent | Focus |
|-----------|-----------|-------|
| Code Quality | `superpowers:code-reviewer` | Correctness, architecture, defensive coding, testing |
| Pattern Consistency | `superpowers:code-reviewer` (Pattern prompt) | Codebase idioms, conventions, existing patterns |
| SQL Performance | `superpowers:code-reviewer` (SQL prompt) | Query performance, N+1, injection, defensive DB patterns |
| Security Audit | `superpowers:code-reviewer` (Security prompt) | OWASP Top 10, auth, data exposure, injection, IDOR |
| Code Simplifier | `code-simplifier` agent | Clarity, reuse, dead code, naming, efficiency |

## When to Use

- Phase 7 (CODE REVIEW) of the development lifecycle
- When reviewing a GitHub PR (own or teammate's)
- When receiving a code review request
- Before merging any feature branch
- When the user says "review this PR", "review these changes", "code review"

## Execution Flow

```
1. Gather Context
   ↓
2. Dispatch 5 Sub-Agents in PARALLEL
   (Code Quality, Patterns, SQL*, Security, Simplifier)
   ↓
3. Collect All Results
   ↓
4. Deduplicate & Merge
   ↓
5. Present Unified Report
   ↓
Any CRITICAL? → yes → CHANGES REQUIRED
             → no  → APPROVED (with conditions)

* SQL only dispatched if DB-touching files changed
* If any sub-agent fails: re-dispatch once, then mark dimension as "Review Incomplete"
```

## Phase 1: Gather Context

Before dispatching sub-agents, collect and resolve all placeholders used in sub-agent prompts.

```bash
# Git context
BASE_SHA=$(git merge-base origin/main HEAD)
HEAD_SHA=$(git rev-parse HEAD)
git diff --stat $BASE_SHA..$HEAD_SHA
git diff --name-only $BASE_SHA..$HEAD_SHA
git log --oneline $BASE_SHA..$HEAD_SHA

# For PR reviews: get stable SHAs from PR metadata
gh pr view {PR_NUMBER} --json baseRefOid,headRefOid,title,body
```

**Collect and populate these values before sub-agent dispatch:**

| Placeholder | Where to find it |
|-------------|-----------------|
| `{BASE_SHA}` | `git merge-base origin/main HEAD` or `gh pr view --json baseRefOid` |
| `{HEAD_SHA}` | `git rev-parse HEAD` or `gh pr view --json headRefOid` |
| `{DESCRIPTION}` | Summary of what changed (from commits + PR description) |
| `{PLAN_OR_REQUIREMENTS}` | Jira ticket description, plan doc path, or PR body from `gh pr view`. If none available, use "No plan reference — infer from code and PR description." |
| `{FILE_LIST}` | `git diff --name-only {BASE_SHA}..{HEAD_SHA}` |
| `{DATABASE_ENGINE}` | Check `config/database.yml` (Rails), `prisma/schema.prisma`, `knexfile.js`, etc. Default: PostgreSQL if not determinable. |
| `{ORM}` | Check `Gemfile` for `activerecord`/`sequel`, `package.json` for `prisma`/`knex`/`typeorm`, etc. Default: ActiveRecord if Rails project. |
| `{PR_URL}` | From `gh pr view --json url` or provided by user |
| `{CODEBASE_CONTEXT}` | See Codebase Intelligence section below |

**Skip SQL sub-agent if:** No database-touching files changed (no models, migrations, controllers with queries, services with queries). State this explicitly in the report.

### Codebase Intelligence

**If `codebase-memory-mcp` is available**, run these before dispatching sub-agents and include results as `{CODEBASE_CONTEXT}`:

```
mcp__codebase-memory-mcp__get_architecture          # overview of affected areas
mcp__codebase-memory-mcp__trace_call_path(...)       # callers of changed functions
mcp__codebase-memory-mcp__search_code(query=...)     # similar existing implementations
mcp__codebase-memory-mcp__search_graph(query=...)    # impact radius
```

**If `codebase-memory-mcp` is NOT available**, gather equivalent context using Grep and Glob:

```bash
# Find existing examples of the same patterns
grep -r "class.*Controller" app/controllers/ | head -5
grep -r "def call" app/interactors/ | head -5
grep -r "class.*Serializer" app/serializers/ | head -5
```

Set `{CODEBASE_CONTEXT}` to whichever of these you ran, or to "Not available — sub-agent should perform its own pattern discovery using Grep and Glob." Never leave it as an unfilled placeholder.

## Phase 2: Dispatch Sub-Agents (PARALLEL)

**CRITICAL:** All sub-agents MUST be dispatched in parallel using the Agent tool. Do NOT run them sequentially. Each sub-agent gets a fresh context — no shared state.

**If a sub-agent fails or returns a malformed response:** Re-dispatch it once. If it fails again, record that dimension as "Review Incomplete — sub-agent failed after 2 attempts" in the final report and surface this to the user before presenting a final verdict.

### Sub-Agent 1: Code Quality Review

```
Agent tool:
  subagent_type: "superpowers:code-reviewer"
  description: "Code quality review"
  prompt: |
    You are a Staff Engineer performing a code quality review. Think critically
    and logically. Focus on defensive coding — what can go wrong, what edge cases
    are missed, what assumptions are fragile.

    ## What Was Changed
    {DESCRIPTION}

    ## Requirements/Plan
    {PLAN_OR_REQUIREMENTS}

    ## Git Range
    Base: {BASE_SHA}
    Head: {HEAD_SHA}

    Run: git diff {BASE_SHA}..{HEAD_SHA}

    ## Review Focus

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

    ## Output Format

    For EACH finding:
    **[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
    - File: `path/to/file:line_number`
    - Problem: [What's wrong and why it matters]
    - Recommendation: [Specific fix with code snippet if helpful]
    - Impact: [What happens if not fixed]

    End with:
    ### Assessment
    **Verdict:** APPROVED | APPROVED WITH CONDITIONS | CHANGES REQUIRED
    **Summary:** [1-2 sentences]
```

### Sub-Agent 2: Pattern & Idiom Consistency Review

This sub-agent verifies that new code follows the established patterns, idioms, and conventions of the codebase. It uses codebase-memory-mcp (if available) and direct code search to compare against existing implementations.

```
Agent tool:
  subagent_type: "superpowers:code-reviewer"
  description: "Pattern consistency review"
  prompt: |
    You are a Staff Engineer who deeply understands this codebase. Your job is to
    verify that new/changed code follows the established patterns, idioms, and
    conventions already in use. Inconsistent code creates maintenance burden and
    confuses future developers.

    ## What Was Changed
    {DESCRIPTION}

    ## Git Range
    Base: {BASE_SHA}
    Head: {HEAD_SHA}

    Run: git diff {BASE_SHA}..{HEAD_SHA}

    ## Codebase Context
    {CODEBASE_CONTEXT}
    (If this is empty or says "Not available", use Grep and Glob yourself to find 3-5 existing
    examples of each pattern before comparing against the changed code.)

    ## Review Process

    **Step 1: Identify the patterns in the changed code.**
    For each changed file, identify what patterns are being used:
    - Controller patterns (thin/fat, before_action usage, response format)
    - Service/interactor patterns (call conventions, error handling, return types)
    - Model patterns (validations, scopes, callbacks, associations)
    - Test patterns (setup, assertion style, shared examples, factories)
    - Error handling patterns (rescue, raise, error classes)
    - Serialization patterns (what serializer, what fields exposed)
    - Authorization patterns (Pundit, before_action, scoping)

    **Step 2: Search for how the codebase already does this.**
    Use Grep and Glob to find 3-5 existing examples of the same pattern in the
    codebase. If codebase-memory-mcp is available, also use:
    - search_code to find similar implementations
    - search_graph to find related functions/classes
    - get_architecture to understand the module structure

    **Step 3: Compare and flag deviations.**
    For each pattern in the changed code, compare against existing examples:
    - Is the same approach used? (e.g., interactor vs service object vs inline)
    - Is the same naming convention followed?
    - Is the same error handling pattern used?
    - Is the same test structure used?
    - Are the same gems/libraries used for the same purpose?

    ## What to Flag

    **IMPORTANT — Pattern Deviation:**
    Code uses a different pattern than the rest of the codebase for the same task.
    Example: New code uses a plain service class when the codebase uses interactors.

    **IMPORTANT — Convention Violation:**
    Code breaks a naming, structure, or organizational convention.
    Example: Test file not in the expected directory, method naming doesn't match.

    **MINOR — Idiom Inconsistency:**
    Code works but uses a less idiomatic approach for the language/framework.
    Example: Using `each` + `push` instead of `map`, `if x != nil` instead of `if x`.

    **Not a finding:**
    Intentional, documented deviation with an explicit comment explaining the reason.

    **Important:** Do NOT decide unilaterally that a new pattern is "strictly better" and
    suppress the finding. All deviations from established patterns must be reported — you may
    add a note that the new approach appears superior, but still flag it as MINOR so the
    orchestrating Staff Engineer can make the judgment call. The decision to adopt a new
    pattern belongs to the human reviewer, not the sub-agent.

    ## Output Format

    For EACH finding:
    **[IMPORTANT/MINOR] — Pattern Consistency — [Short title]**
    - File: `path/to/file:line_number`
    - Current code: [What the changed code does]
    - Existing pattern: [What the codebase already does, with example file:line]
    - Recommendation: [Use the established pattern, with code snippet]
    - Impact: [Inconsistency cost — maintenance burden, confusion, etc.]

    End with:
    ### Assessment
    **Verdict:** APPROVED | APPROVED WITH CONDITIONS | CHANGES REQUIRED
    **Summary:** [Pattern consistency assessment in 1-2 sentences]
```

### Sub-Agent 3: SQL Performance Review (conditional)

**Only dispatch if database-touching files changed.**

```
Agent tool:
  subagent_type: "superpowers:code-reviewer"
  description: "SQL performance review"
  prompt: |
    You are a Staff Engineer specializing in database performance, security,
    and defensive coding. Ruthlessly audit every database query, mutation,
    and ORM interaction in the changed code.

    First, read the sql-optimization-patterns skill: invoke Skill tool with
    skill: "sql-optimization-patterns"

    ## Database Context
    Database: {DATABASE_ENGINE}
    ORM: {ORM}

    ## What Was Changed
    {DESCRIPTION}

    ## Files to Review
    {FILE_LIST}

    ## Git Range
    Base: {BASE_SHA}
    Head: {HEAD_SHA}

    Run: git diff {BASE_SHA}..{HEAD_SHA}

    ## Checklist

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

    ## Output Format

    For EACH finding:
    **[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
    - File: `path/to/file:line_number`
    - Problem: [What's wrong]
    - Recommendation: [Specific fix]
    - Impact: [What happens if not fixed]

    End with:
    ### Assessment
    **Verdict:** APPROVED | APPROVED WITH CONDITIONS | CHANGES REQUIRED
    **Summary:** [Database performance and safety assessment in 1-2 sentences]
```

### Sub-Agent 4: Security Audit

```
Agent tool:
  subagent_type: "superpowers:code-reviewer"
  description: "Security audit"
  prompt: |
    You are a Staff Security Engineer performing a security-focused code audit.
    Your job is to find vulnerabilities before they reach production. Think like
    an attacker — what can be exploited?

    ## What Was Changed
    {DESCRIPTION}

    ## Git Range
    Base: {BASE_SHA}
    Head: {HEAD_SHA}

    Run: git diff {BASE_SHA}..{HEAD_SHA}

    ## Security Checklist (OWASP-aligned)

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
    - User-controlled URLs passed to HTTP clients (Net::HTTP, Faraday, HTTParty, fetch)?
    - Are outbound requests restricted to an allowlist of domains?
    - Internal metadata endpoints (AWS 169.254.x.x, cloud metadata) accessible via user-supplied URLs?
    - CWE-918

    ### Cryptographic Failures (IMPORTANT)
    - Weak hash algorithms for security-sensitive data? (MD5, SHA1 — use bcrypt/argon2 for passwords, SHA-256+ for tokens)
    - Hardcoded encryption keys, salts, or IVs?
    - Secrets in ENV but `.env` files committed to version control?
    - Sensitive data in JWT payload without encryption (only signing)?
    - CWE-327, CWE-798

    ### Logging & Monitoring (MINOR)
    - Security events not logged (failed auth, permission denied)
    - Missing audit trail for sensitive operations
    - Error messages leaking internal details

    ## Verdict Thresholds

    - **APPROVED**: No critical or important security findings
    - **APPROVED WITH CONDITIONS**: Important findings present — each must have a Jira issue opened before this verdict applies
    - **CHANGES REQUIRED**: Any critical security findings — must be fixed before merge, no exceptions

    ## Output Format

    For EACH finding:
    **[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
    - File: `path/to/file:line_number`
    - Problem: [What's wrong — describe the attack vector]
    - Recommendation: [Specific remediation with code snippet]
    - Impact: [Severity if exploited — data breach, privilege escalation, etc.]
    - CWE: [CWE ID, e.g., CWE-89 (SQL Injection), CWE-79 (XSS), CWE-862 (Missing Authorization)]

    End with:
    ### Assessment
    **Verdict:** APPROVED | APPROVED WITH CONDITIONS | CHANGES REQUIRED
    **Summary:** [Security posture assessment in 1-2 sentences]
```

### Sub-Agent 5: Code Simplifier

```
Agent tool:
  subagent_type: "code-simplifier"
  description: "Code simplification review"
  prompt: |
    Review the recently modified code for simplification opportunities.

    ## Git Range
    Base: {BASE_SHA}
    Head: {HEAD_SHA}

    Focus on files changed in this range:
    git diff --name-only {BASE_SHA}..{HEAD_SHA}

    Analyze for: unnecessary complexity, redundant code, naming improvements,
    dead code, and language-specific best practices.

    ## Required Output Format

    After your Phase 1 analysis and Phase 2 Staff Engineer review, present
    your final report with the following structure for each APPROVED item:

    **[APPROVED/DEFERRED] — [Short title]**
    - File: `path/to/file:line_number`
    - Current: [What the code does now]
    - Simplified: [What it should be, with code snippet]
    - Rationale: [Why this is simpler or clearer]

    End with:
    ### Assessment
    **Verdict:** CHANGES SUGGESTED | NO CHANGES NEEDED
    **Summary:** [1-2 sentences on overall simplification state of the changed code]
```

## Phase 3: Consolidate Results

After ALL sub-agents return, merge their findings into a single report.

### Deduplication Rules

Two findings are the **same** if they reference the same `file:line` AND the same root cause (not merely the same file). When merging:

- Note which dimensions caught it: "Caught by: Code Quality, Security"
- Use the more severe categorization when sub-agents disagree
- For auth/injection findings: use the Security sub-agent's recommendation
- For all other findings: use the more specific of the two recommendations
- Keep findings distinct if they share a file but have different root causes — different dimensions catch different problems

### Unified Report Format

```markdown
# Comprehensive Code Review Report

**PR:** {PR_URL or branch name}
**Reviewed:** {DATE}
**Git Range:** {BASE_SHA}..{HEAD_SHA}
**Files Changed:** {COUNT}

---

## CRITICAL ({count})

### 1. [Short title]
- **File:** `path/to/file:line_number`
- **Dimension:** {Code Quality | Patterns | SQL | Security | Simplification}
- **Problem:** [Clear description]
- **Recommendation:** [Specific fix with code if helpful]
- **Impact:** [What happens if not fixed]

---

## IMPORTANT ({count})

### 1. [Short title]
- **File:** `path/to/file:line_number`
- **Dimension:** {Code Quality | Patterns | SQL | Security | Simplification}
- **Problem:** [Clear description]
- **Recommendation:** [Specific fix]
- **Impact:** [What happens if not fixed]

---

## MINOR ({count})

### 1. [Short title]
- **File:** `path/to/file:line_number`
- **Dimension:** {Code Quality | Patterns | SQL | Security | Simplification}
- **Problem:** [Clear description]
- **Recommendation:** [Suggestion]

---

## Simplification Opportunities

[APPROVED items from Code Simplifier — each with file:line, current code, and simplified version.
Mark each as "Approved — implement" or "Deferred — track as follow-up".]

---

## Strengths

[What was done well across all dimensions — be specific with file:line references]

---

## Overall Assessment

| Dimension | Verdict | Critical | Important | Minor |
|-----------|---------|----------|-----------|-------|
| Code Quality | {verdict} | {n} | {n} | {n} |
| Pattern Consistency | {verdict} | {n} | {n} | {n} |
| SQL Performance | {verdict or N/A} | {n} | {n} | {n} |
| Security | {verdict} | {n} | {n} | {n} |
| Simplification | {verdict} | {n} | {n} | {n} |

**Final Verdict:** {APPROVED | APPROVED WITH CONDITIONS | CHANGES REQUIRED}

**Action Required:**
- CRITICAL: Must fix before merge ({count} items)
- IMPORTANT: Should fix, or create Jira ticket ({count} items)
- MINOR: At author's discretion ({count} items)
```

## Integration with Workflows

### As Phase 7 replacement

This skill replaces the standalone Phase 7 (CODE REVIEW) and Phase 8 (SQL REVIEW) in the development lifecycle. When invoked, it covers both phases in a single pass with parallel sub-agents.

### For GitHub PR reviews

When reviewing a teammate's PR or responding to "review this PR":

1. Get stable SHAs: `gh pr view {PR_NUMBER} --json baseRefOid,headRefOid,title,body,url`
2. Use returned `baseRefOid` as `BASE_SHA` and `headRefOid` as `HEAD_SHA`
3. Get diff for context: `gh pr diff {PR_NUMBER}` or `git diff {BASE_SHA}..{HEAD_SHA}`
4. Dispatch sub-agents with populated placeholders
5. Post the consolidated report as a PR comment (if user requests): `gh pr comment {PR_NUMBER} --body "..."`

### For receiving code review

When you receive review feedback on your own PR, use `receiving-code-review` skill instead — that skill handles how to respond to feedback.

## Rules

**NEVER:**
- Review code yourself — always dispatch sub-agents
- Skip the security audit — every change has a security surface
- Run sub-agents sequentially when they can run in parallel
- Mark everything as CRITICAL — use severity accurately
- Give a passing verdict when CRITICAL issues exist

**ALWAYS:**
- Dispatch ALL applicable sub-agents in a single parallel call
- Include file:line references for every finding
- Provide actionable recommendations (not just "fix this")
- State explicitly if SQL review was skipped (no DB changes) and why
- Deduplicate findings across dimensions
- Present the unified report, not raw sub-agent outputs
