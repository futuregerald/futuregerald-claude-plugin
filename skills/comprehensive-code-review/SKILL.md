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

**The orchestrator precomputes ALL context and forwards it to sub-agents. Sub-agents should never re-search for information the orchestrator already gathered.**

```bash
# Git context
BASE_SHA=$(git merge-base origin/main HEAD)
HEAD_SHA=$(git rev-parse HEAD)
DIFF=$(git diff $BASE_SHA..$HEAD_SHA)
FILE_LIST=$(git diff --name-only $BASE_SHA..$HEAD_SHA)
COMMITS=$(git log --oneline $BASE_SHA..$HEAD_SHA)

# If diff exceeds ~2000 lines, prioritize security-relevant files
# (controllers, auth, models, migrations) and summarize the rest
```

For PR reviews, also fetch PR metadata using available GitHub tooling (`gh` CLI, GitHub MCP server, or API).

**Collect and populate these values before sub-agent dispatch:**

| Placeholder | Source |
|-------------|--------|
| `{DIFF}` | `git diff {BASE_SHA}..{HEAD_SHA}` — gathered ONCE by orchestrator |
| `{PR_DESCRIPTION}` | PR body from `gh pr view --json body` or user-provided context |
| `{FILE_LIST}`, `{BASE_SHA}`, `{HEAD_SHA}`, `{PR_URL}` | Git commands or PR metadata |
| `{PLAN_OR_REQUIREMENTS}` | See Requirements Resolution below |
| `{CODEBASE_CONTEXT}` | See Codebase Intelligence below — **must contain actual content** |
| `{EXISTING_REVIEW_COMMENTS}` | See Review Comments below |
| `{SCHEMA_CONTEXT}` | See Schema Context below (safety sub-agent only) |
| `{DATABASE_ENGINE}` | `config/database.yml`, `prisma/schema.prisma`, etc. Default: PostgreSQL |
| `{ORM}` | `Gemfile` (activerecord/sequel), `package.json` (prisma/knex/typeorm). Default: ActiveRecord if Rails |
| `{FRAMEWORK_CONTEXT}` | See Framework Detection below |

### Framework Detection

**Detect the framework and language before dispatch.** The framework determines what the runtime guarantees — findings that ignore framework behavior produce false positives that erode reviewer trust.

Detect from project files:
- `Gemfile` with `rails` → **Ruby on Rails** (ActiveRecord, strong params, type coercion)
- `package.json` with `next`/`react` → **Next.js / React**
- `package.json` with `express` → **Express.js**
- `requirements.txt` / `pyproject.toml` with `django` → **Django**
- `go.mod` → **Go** (statically typed — many injection classes are compile-time impossible)

Set `{FRAMEWORK_CONTEXT}` to a brief summary of framework-specific rules the sub-agents must respect. Include this in BOTH sub-agent prompts.

#### Ruby on Rails framework rules

```
## Framework: Ruby on Rails

Respect how Ruby and Rails actually work. False positives from ignoring
the framework erode trust and waste the author's time.

### Type safety in Ruby
- Ruby's `.to_i` ALWAYS returns an Integer. `"anything".to_i` → `0`.
  An Integer interpolated into a string (`"(#{int_val})"`) can ONLY
  produce a decimal number. This is NOT SQL injection — Ruby's type
  system guarantees safety. Do NOT flag integer interpolation as CWE-89.
- Ruby's `.to_f` ALWAYS returns a Float. Same type safety applies.
- `.to_s` on Integer/Float cannot produce SQL metacharacters.
- Only flag SQL injection when STRINGS from user/external input are
  interpolated into queries without `conn.quote()` or parameterization.

### ActiveRecord conventions
- `conn.quote(value)` is for string values in raw SQL. Using it on
  integers is unnecessary (it returns the decimal string anyway).
- `conn.quote_column_name` / `conn.quote_table_name` are for dynamic
  identifiers. Hardcoded string constants do NOT need quoting.
- `where(column: value)` is parameterized — not injection.
- `find_by(column: value)` is parameterized — not injection.
- `pluck(:column)` returns typed Ruby values from the DB adapter.

### Rails mass assignment
- Strong parameters (`params.require().permit()`) handle mass assignment
  in controllers. Internal service objects that don't accept user params
  do NOT need strong parameters.
- `create!` / `update!` with hardcoded hashes is safe.

### Sidekiq / ActiveJob
- Without `retry_on`, Sidekiq uses its OWN retry mechanism (25 retries,
  exponential backoff). `retry_on` is ActiveJob's DSL and is optional.
  Note the default behavior but don't flag absence as CRITICAL.
- `perform_now` runs synchronously (not queued). Used in rake tasks.

### Rails conventions to NOT flag
- `ENV.fetch('KEY')` raising KeyError is intentional fail-fast, not a bug.
- `Time.current` vs `Time.now` — `Time.current` is the Rails convention.
- `present?` / `blank?` are Rails core extensions, not custom code.
- `squish` on strings is a Rails method that collapses whitespace.
```

#### Go framework rules

```
## Framework: Go

- Go is statically typed. Integer values CANNOT contain SQL injection
  when used with `fmt.Sprintf("%d", val)` or direct interpolation.
- String values from user input CAN be dangerous in `fmt.Sprintf`
  SQL construction — flag these.
- `database/sql` with `?` placeholders is parameterized.
```

#### JavaScript/TypeScript framework rules

```
## Framework: JavaScript/TypeScript

- JS has NO integer type — all numbers are IEEE 754 doubles. However,
  `parseInt(val, 10)` returns NaN for non-numeric strings, not a
  dangerous value. `NaN` in SQL causes a query error, not injection.
- Template literals with user strings ARE dangerous in raw SQL.
- ORMs (Prisma, TypeORM, Knex) parameterize by default when using
  their query builder APIs. Raw SQL methods (`.raw()`, `.$queryRaw()`)
  need manual parameterization.
```

Add rules for other frameworks as encountered. The key principle: **understand what the language runtime guarantees before claiming a vulnerability exists.**

### Requirements Resolution

Gather from all available sources (GitHub MCP, `gh` CLI, Atlassian MCP, file reads):

1. **Jira ticket** — key in branch name or PR body (e.g., `DL-1234`), fetch via `getJiraIssue`
2. **GitHub issue** — linked in PR body (`Closes #N`, `Fixes #N`)
3. **Plan doc** — if PR body references a plan file, read it

If no PR exists (local branch), skip GitHub issue. Extract Jira key from branch name.
Combine with labels: `**From Jira DL-1234:** ...` / `**From PR description:** ...`
Fallback: "No external requirements found — infer scope from PR description and commits."

### Review Comments

If a PR exists, fetch existing inline comments and non-superseded review bodies. Pass both to sub-agents so they can check whether prior findings have been addressed.

```bash
# Get repo and PR number
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
PR_NUMBER=$(gh pr view --json number -q .number)

# Inline comments (file + line + body, last 20)
gh api repos/$REPO/pulls/$PR_NUMBER/comments \
  --jq '[.[] | {path: .path, line: .line, body: (.body | .[0:300])}] | .[-20:]'

# Non-superseded review bodies (top-level summaries only, last 10)
gh api repos/$REPO/pulls/$PR_NUMBER/reviews \
  --jq '[.[] | select(.body | contains(":warning: **Superseded**") | not) | select(.body != "") | {state: .state, body: (.body | .[0:500])}] | .[-10:]'
```

Format `{EXISTING_REVIEW_COMMENTS}` as:

```
### path/to/file.rb:42
> IMPORTANT — N+1 query: eager load :org association...

### path/to/file.rb:87
> CRITICAL — missing index on org_id...
```

If no PR exists or no comments: set `{EXISTING_REVIEW_COMMENTS}` to `"(none — this is a first review)"`.

### Schema Context

If the diff touches DB files (models, migrations, services with queries), extract the relevant `create_table` blocks from `db/schema.rb`:

```bash
# Replace <table> with each table name found in the diff
grep -A 40 'create_table "<table>"' db/schema.rb
```

Include the raw `create_table` blocks verbatim in `{SCHEMA_CONTEXT}`. If no DB files in diff: set `{SCHEMA_CONTEXT}` to `"(skipped — no database-touching files changed)"`.

### Codebase Intelligence

**Always populate `{CODEBASE_CONTEXT}` with actual search output — never with instructions to search.**

**Use the `future-code-search` model routing rules to delegate search work to cheaper models.** The orchestrator should NOT run raw grep/glob searches itself. Instead:

1. **Dispatch a Sonnet exploration agent** for multi-step codebase context gathering:

```
Agent({
  model: "sonnet",
  subagent_type: "Explore",
  prompt: "I'm reviewing a PR that changes these files: {FILE_LIST}.
  I need codebase context for a code review. For each changed file, find:
  1. Similar patterns in the codebase (interactors, policies, serializers, controllers)
  2. Existing callbacks on changed models
  3. Related factories in spec/factories/
  4. If codebase-memory-mcp is available, use get_architecture and trace_call_path on affected areas.
  5. Otherwise, run 3-5 targeted grep searches.

  Report findings as labeled sections with file paths and line numbers:
  ### Interactor pattern examples
  ### Existing policies
  ### Callbacks on ChangedModel
  ### Related factories"
})
```

2. **Dispatch a Haiku agent** for simple lookups (finding specific files, checking if a method exists):

```
Agent({
  model: "haiku",
  subagent_type: "Explore",
  prompt: "Find all files matching **/policies/*_policy.rb and list their paths"
})
```

3. **Parallelize independent searches** — dispatch multiple agents in a single message when gathering different categories of context.

Set `{CODEBASE_CONTEXT}` to the **combined output** from these agents, labelled by section:

```
### Interactor pattern examples
<agent output>

### Existing policies
<agent output>

### Callbacks on ChangedModel
<agent output>
```

## Phase 2: Dispatch Sub-Agents (PARALLEL)

**CRITICAL:** Both sub-agents MUST be dispatched in parallel. Each gets a fresh context — no shared state. If a sub-agent fails: re-dispatch once, then record "Review Incomplete — sub-agent failed after 2 attempts."

### Shared Output Format (used by BOTH agents)

```
For EACH finding:
**[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short title]**
- File: `path/to/file:line_number`
- Problem: [What's wrong and why it matters]
- Recommendation: [Specific fix with code snippet if helpful]
- Suggestion: [If simple one-line fix, include exact corrected line prefixed with SUGGESTION:]
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

Read [references/correctness-prompt.md](references/correctness-prompt.md) for the full prompt template. Replace all `{PLACEHOLDERS}` with values from Phase 1.

Covers: code quality, defensive coding, architecture, testing, scope verification, pattern consistency, and simplification opportunities.

### Sub-Agent 2: Safety Review

Read [references/safety-prompt.md](references/safety-prompt.md) for the full prompt template. Replace all `{PLACEHOLDERS}` with values from Phase 1.

Covers: OWASP Top 10 (injection, auth, data exposure, SSRF, crypto), input validation, mass assignment, and conditional SQL/DB performance review.

## Phase 3: Consolidate Results

After BOTH sub-agents return, merge findings into a single report.

### Framework-Awareness Validation (MANDATORY)

Before including ANY finding in the final report, the orchestrator MUST validate it against the detected framework's type system and conventions:

1. **SQL Injection claims**: Does the language's type system make the injection impossible? In Ruby, `.to_i` returns `Integer` — interpolating an Integer into SQL cannot produce injection. In Go, static typing prevents string injection via int params. **Drop findings where the type system provides the guarantee.**
2. **Missing security patterns**: Does the framework already handle this? Rails strong params in controllers, Django CSRF middleware, etc. **Don't flag framework-provided protections as missing.**
3. **Performance claims**: Are they calibrated to actual dataset sizes? "2-5x slower" needs justification. If the data volume is known (e.g., 200K rows), estimate the real-world impact in seconds, not multipliers.
4. **Convention claims**: Is the finding about the framework's convention or the codebase's convention? Only flag deviations from the codebase's established patterns, not from what a textbook says.

**If a sub-agent flags something the framework makes impossible, DROP the finding entirely** — do not downgrade it to MINOR. False positives waste the author's time and erode trust in the review process.

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
3. Fetch existing review comments and schema context
4. Dispatch sub-agents with all populated placeholders
5. Post the consolidated report as a PR comment if the user requests it

### For receiving code review

When you receive review feedback on your own PR, use `receiving-code-review` skill instead.

### For skill file reviews

If `{FILE_LIST}` contains any `SKILL.md` files or files under a `skills/` directory, dispatch an **additional** sub-agent for skill quality review:

```
Agent tool:
  subagent_type: "code-quality-reviewer"
  description: "Skill quality review"
  prompt: |
    You are reviewing skill files for quality. First, read the skill-reviewer
    skill by invoking: Skill tool with skill: "skill-reviewer"

    Then apply its checklist to every SKILL.md and reference file in this diff.

    ## Diff
    ```diff
    {DIFF}
    ```

    ## File List
    {FILE_LIST}

    Read each SKILL.md file in the diff (use the Read tool — the diff may
    truncate large files). For each skill, produce the review table and
    verdict from the skill-reviewer checklist.
```

Include the skill review findings in the consolidated report under a `## Skill Quality` section after Simplification Opportunities. Skill review findings use the same severity ratings (CRITICAL/IMPORTANT/MINOR).

## Rules

**NEVER:** Review code yourself | Skip the safety audit | Run sub-agents sequentially | Mark everything CRITICAL | Pass verdict with CRITICAL issues

**ALWAYS:** Dispatch BOTH sub-agents in a SINGLE parallel call (one message, two Agent tool invocations) | Include file:line for every finding | Provide actionable recommendations | Deduplicate across dimensions | Present unified report

**Parallel dispatch is mandatory.** Both Agent tool calls must appear in the same response. Sequential dispatch wastes tokens and time:

```
# CORRECT — both dispatched in one response:
Agent(subagent_type="code-quality-reviewer", prompt="...{DIFF}...{CODEBASE_CONTEXT}...")
Agent(subagent_type="security-reviewer",     prompt="...{DIFF}...{SCHEMA_CONTEXT}...")

# WRONG — sequential (never do this):
Agent(subagent_type="code-quality-reviewer", ...)   # wait for result...
Agent(subagent_type="security-reviewer", ...)        # then dispatch second
```
