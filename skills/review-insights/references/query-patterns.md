# Query Patterns Reference

Comprehensive examples for querying the review-style-analyzer database.

```bash
ANALYZER=~/Documents/dev/cobalt-review-action/analyzer/analyzer
DB=~/Documents/dev/cobalt-review-action/analyzer/reviews.db
```

## By Topic

```bash
# What does the team say about error handling?
$ANALYZER query --db $DB --topic error-handling --limit 20

# Database review patterns (N+1, indexes, migrations)
$ANALYZER query --db $DB --topic database --limit 20 --verbose

# Authorization/policy patterns
$ANALYZER query --db $DB --topic authorization --limit 15

# API design feedback
$ANALYZER query --db $DB --topic api-design --limit 15

# Naming conventions the team enforces
$ANALYZER query --db $DB --topic naming --limit 20
```

## By Category

```bash
# Security findings (130 comments, avg 55 words -- the most detailed)
$ANALYZER query --db $DB --category security --limit 20 --verbose

# Performance concerns (227 comments)
$ANALYZER query --db $DB --category performance --limit 20

# Bug callouts (430 comments)
$ANALYZER query --db $DB --category bug --limit 20

# Architecture discussions
$ANALYZER query --db $DB --category architecture --limit 15

# Testing feedback (501 comments -- the most frequent non-general category)
$ANALYZER query --db $DB --category testing --limit 20
```

## By Sentiment + Topic (combining filters)

```bash
# Negative comments about bugs (the things that actually block PRs)
$ANALYZER query --db $DB --sentiment negative --category bug --limit 10

# Constructive suggestions about architecture
$ANALYZER query --db $DB --sentiment constructive --topic architecture --limit 10

# Questions about error handling (collaborative review style)
$ANALYZER query --db $DB --curiosity question --topic error-handling --limit 10

# Positive feedback (what does "good" look like to this team?)
$ANALYZER query --db $DB --sentiment positive --limit 20
```

## By Reviewer

```bash
# David's security focus (he has 2.3% security, highest on team)
$ANALYZER query --db $DB --reviewer davidgm0 --category security --limit 10

# Roger's code suggestions (16.6% suggestion rate)
$ANALYZER query --db $DB --reviewer roger-cobalt --limit 20 --verbose

# Paul's questions (26.6% question rate -- most collaborative style)
$ANALYZER query --db $DB --reviewer Lucianolo --curiosity question --limit 15

# Mauricio's database reviews (19.5% database topic -- highest)
$ANALYZER query --db $DB --reviewer mauricio-reis --topic database --limit 10
```

## By PR

```bash
# All comments on a specific PR
$ANALYZER query --db $DB --pr 6687 --verbose

# A specific reviewer's comments on a PR
$ANALYZER query --db $DB --pr 7078 --reviewer Lucianolo
```

## Full-Text Search

```bash
# Concept search
$ANALYZER search --db $DB "N+1 query eager load" --limit 10
$ANALYZER search --db $DB "race condition thread safety" --limit 10
$ANALYZER search --db $DB "strong parameters mass assignment" --limit 10
$ANALYZER search --db $DB "interactor context fail" --limit 10

# File-specific patterns
$ANALYZER search --db $DB "path:org.rb" --limit 10
$ANALYZER search --db $DB "path:structure.sql" --limit 10
$ANALYZER search --db $DB "path:policy" --limit 10

# Topic-scoped search
$ANALYZER search --db $DB "topic:security AND authorization" --limit 10

# Reviewer-scoped search
$ANALYZER search --db $DB "pundit policy" --reviewer davidgm0 --limit 10

# Phrase search
$ANALYZER search --db $DB '"error handling"' --limit 10
$ANALYZER search --db $DB '"missing test"' --limit 10

# Prefix search (wildcard)
$ANALYZER search --db $DB "serializ*" --limit 10
$ANALYZER search --db $DB "refact*" --limit 10
```

## Statistics

```bash
# Full team overview
$ANALYZER stats --db $DB

# Individual profiles
$ANALYZER stats --db $DB --reviewer davidgm0
$ANALYZER stats --db $DB --reviewer roger-cobalt
$ANALYZER stats --db $DB --reviewer Lucianolo
$ANALYZER stats --db $DB --reviewer mauricio-reis
```

## Export for RAG / Training

```bash
# Export everything
$ANALYZER export --db $DB --output all_reviews.json

# Export one reviewer's data
$ANALYZER export --db $DB --reviewer Lucianolo --output paul_reviews.json

# Pipe to jq for custom filtering
$ANALYZER export --db $DB | jq '[.[] | select(.category == "security")]' > security_reviews.json
$ANALYZER export --db $DB | jq '[.[] | select(.is_blocking == true)]' > blocking_reviews.json
$ANALYZER export --db $DB | jq '[.[] | select(.word_count > 50)]' > detailed_reviews.json
```

## Thread Analysis

```bash
# See comments with the most discussion
$ANALYZER threads --db $DB --limit 20

# Threads by a specific reviewer
$ANALYZER threads --db $DB --reviewer roger-cobalt --limit 10
```

## Useful Combinations for PR Review Context

```bash
# "What does the team flag on model changes?"
$ANALYZER query --db $DB --topic database --category bug --limit 5
$ANALYZER query --db $DB --topic validation --limit 5
$ANALYZER search --db $DB "path:models" --limit 10

# "What does the team flag on controller changes?"
$ANALYZER query --db $DB --topic api-design --limit 5
$ANALYZER query --db $DB --topic authorization --limit 5
$ANALYZER search --db $DB "path:controllers" --limit 10

# "What does the team flag on test changes?"
$ANALYZER query --db $DB --category testing --limit 10
$ANALYZER search --db $DB "factory mock stub" --limit 5

# "What are the team's hard stops?"
$ANALYZER query --db $DB --sentiment negative --category security --limit 5
$ANALYZER search --db $DB "do not merge" --limit 5
$ANALYZER search --db $DB "blocking" --limit 5
```
