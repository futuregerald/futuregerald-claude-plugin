---
name: review-insights
description: "Query review style data to enhance PR reviews. Search past review comments, learn reviewer patterns, fetch new data from GitHub. Always invoke during cobalt repo code reviews."
trigger: /review-insights
tags: [code-review, review, training, rag, pr, cobalt]
author: Gerald Onyango <gerald.onyango@gmail.com>
---

# Review Insights

Use the **review-style-analyzer** CLI to query 7,000+ real code review comments from Cobalt's engineering team. This data captures what experienced reviewers check for, how they communicate feedback, and what topics matter most in this codebase.

## Analyzer Location

```
~/Documents/dev/cobalt-review-action/analyzer/
```

Binary: `analyzer` (build with `go build -tags fts5 -o analyzer .`)
Database: `reviews.db` in the analyzer directory (or wherever `--db` points)

**Always use the full path to the binary and DB:**

```bash
ANALYZER=~/Documents/dev/cobalt-review-action/analyzer/analyzer
DB=~/Documents/dev/cobalt-review-action/analyzer/reviews.db
```

## Slash Commands

### /review-insights

Query the review database for patterns relevant to the current PR or code change. This is the primary command and should be used during every cobalt repo code review.

**When invoked during a PR review:**

1. Identify the files and topics in the diff (models, controllers, services, tests, etc.)
2. Search the database for how the team's reviewers have commented on similar code
3. Use those patterns to calibrate your own review -- prioritize what they prioritize, ask the questions they ask, match their communication style

**Execution:**

```bash
# Step 1: Identify topics from the diff
# Map changed files to review topics:
#   app/models/         -> database, naming, validation
#   app/controllers/    -> api-design, authorization, error-handling
#   app/interactors/    -> error-handling, architecture
#   app/policies/       -> authorization, security
#   app/serializers/    -> api-design, naming
#   spec/               -> testing
#   db/migrate/         -> database, performance
#   config/             -> configuration
#   app/services/       -> architecture, error-handling
#   app/jobs/           -> concurrency, error-handling

# Step 2: Query for relevant patterns
$ANALYZER query --db $DB --topic <topic> --limit 10 --verbose
$ANALYZER search --db $DB "<relevant keywords from diff>" --limit 10

# Step 3: Check how reviewers handle this category
$ANALYZER query --db $DB --category <category> --limit 10

# Step 4: Get reviewer-specific patterns if reviewing someone's code
$ANALYZER stats --db $DB --reviewer <author_login>
```

**How to use results in your review:**

- **Questions the team asks** (curiosity=question): Adopt this inquiry style. If David asks "why not eager load here?" on database code, your review should ask similar questions on N+1 patterns.
- **Common blocking patterns** (is_blocking=true): These are the team's hard stops. If security comments are blocking, treat security findings as blocking in your review too.
- **Suggestion style** (has_code_suggestion=true): Roger provides code suggestions 16.6% of the time. When reviewing his areas (API design, testing), include concrete code alternatives.
- **Category priorities**: Testing is the #1 topic across all reviewers. Always check test coverage.
- **Tone calibration**: 75%+ of comments are conversational. Don't be formal. Ask questions, make suggestions -- don't lecture.

### /fetch-reviews

Fetch new review comment data from GitHub and import into the database.

```bash
# Fetch latest comments for a reviewer
$ANALYZER fetch --db $DB --reviewer <github_login> --repo cobalthq/cobalt-pentest-api

# Fetch since a specific date
$ANALYZER fetch --db $DB --reviewer <github_login> --since 2026-01-01

# Fetch with reply threads
$ANALYZER fetch --db $DB --reviewer <github_login> --all-threads

# Fetch for all tracked reviewers
for user in Lucianolo roger-cobalt davidgm0 mauricio-reis; do
  $ANALYZER fetch --db $DB --reviewer $user --repo cobalthq/cobalt-pentest-api --since 2026-01-01
done
```

**When to fetch:**
- Monthly, to keep the dataset current
- Before a deep review of a specific engineer's PR patterns
- When a new team member joins and you want to capture their style

### /review-stats

Display reviewer style profiles and statistics.

```bash
# All reviewers
$ANALYZER stats --db $DB

# Specific reviewer
$ANALYZER stats --db $DB --reviewer Lucianolo
```

### /search-reviews

Full-text search across all review comments.

```bash
# Search by concept
$ANALYZER search --db $DB "N+1 query eager load"

# Search by file type
$ANALYZER search --db $DB "serializer" --verbose

# Search with reviewer filter
$ANALYZER search --db $DB "authorization policy" --reviewer davidgm0

# FTS5 advanced syntax
$ANALYZER search --db $DB 'body:"race condition" AND topic:concurrency'
```

## Enhancing PR Reviews with Review Data

### Pre-Review Context Gathering

Before starting any code review on a cobalt repo, run this sequence:

```bash
# 1. What files changed?
FILE_LIST=$(git diff --name-only origin/main...HEAD)

# 2. Map to topics and search for each
# Example: if app/models/org.rb changed
$ANALYZER search --db $DB "org model" --limit 5
$ANALYZER query --db $DB --topic database --category bug --limit 5

# 3. Check what the team typically flags on this file
$ANALYZER search --db $DB "path:org.rb" --limit 10

# 4. Get category breakdown for context
$ANALYZER query --db $DB --category security --limit 5 --verbose
```

### During Review: Pattern Matching

For each finding you're about to raise, check if the team has flagged similar things:

```bash
# "Is this the kind of thing the team cares about?"
$ANALYZER search --db $DB "<your finding keywords>"

# "How do they phrase this type of feedback?"
$ANALYZER query --db $DB --topic <relevant_topic> --curiosity question --limit 5
```

If the team has 50+ comments on a topic (like testing or error-handling), weight it heavily. If they have <5 comments on a topic, it may not be a priority for this codebase.

### Post-Review: Storing New Patterns

After completing a significant review, the data can be updated:

```bash
# Fetch the latest comments (including your own and responses)
$ANALYZER fetch --db $DB --reviewer <your_github_login> --repo cobalthq/cobalt-pentest-api --since <today>

# Re-classify after any classifier updates
$ANALYZER analyze --db $DB
```

## Reviewer Profiles (Quick Reference)

Read [references/reviewer-profiles.md](references/reviewer-profiles.md) for detailed profiles of each tracked reviewer.

**Summary:**

| Reviewer | Style | Strengths | Tip |
|----------|-------|-----------|-----|
| `davidgm0` | High volume, technically deep, directive | Authorization, security, testing | Most prolific -- his patterns are the baseline |
| `roger-cobalt` | Concise, suggestion-heavy, constructive | API design, code suggestions, testing | Show don't tell -- include code snippets |
| `Lucianolo` | Question-driven, collaborative, explorative | Error handling, database, naming | Ask "why" questions, don't just directive |
| `mauricio-reis` | Balanced, database-focused, medium length | Database, naming, testing | Consistent and thorough on data layer |

## Query Patterns Reference

Read [references/query-patterns.md](references/query-patterns.md) for comprehensive query examples.

## Data Collection Reference

Read [references/data-collection.md](references/data-collection.md) for how to fetch, store, and maintain review data.

## Integration with Code Review Workflow

This skill is a **mandatory pre-step** for the `comprehensive-code-review` skill when reviewing cobalt repos. The flow:

```
1. /review-insights (this skill) -- gather team patterns
2. comprehensive-code-review -- run the full review with patterns as context
3. Calibrate findings against team norms
```

When the `comprehensive-code-review` skill gathers codebase context in Phase 1, it should also gather review insights:

```bash
# Add to {CODEBASE_CONTEXT} in the comprehensive-code-review skill:
ANALYZER=~/Documents/dev/cobalt-review-action/analyzer/analyzer
DB=~/Documents/dev/cobalt-review-action/analyzer/reviews.db

# For each topic found in the diff, query for team patterns
$ANALYZER query --db $DB --topic <topic> --limit 5
$ANALYZER search --db $DB "<relevant terms>" --limit 5
```

Pass the results as `{REVIEW_INSIGHTS_CONTEXT}` to both sub-agents so they can calibrate severity and phrasing.

## Rules

- **Always query before reviewing** cobalt repos. Even a quick search helps calibrate.
- **Don't parrot past reviews** -- use them as calibration, not templates.
- **Weight by volume** -- topics with 500+ comments (testing, database, naming) are core team concerns. Topics with <50 comments are lower priority.
- **Match the reviewer's style** when responding to their PR -- if Paul asks questions, answer with context. If Roger gives suggestions, respond with code.
- **Keep the data fresh** -- fetch new data at least monthly.
