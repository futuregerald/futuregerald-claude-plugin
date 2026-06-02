# Team Review Brief — Gathering Steps

How the orchestrator builds `{TEAM_REVIEW_BRIEF}` from the review-style-analyzer database.
Only applies to cobalt repos (cobalthq/*).

## Graceful Failure

**If the analyzer binary or DB is missing**, do NOT fail the review. Instead:
1. Log in the review report header: `Note: Team review brief unavailable — analyzer not found at <path>`
2. Set `{TEAM_REVIEW_BRIEF}` to `"(skipped — analyzer not available)"`
3. Proceed with the review normally — the brief enriches but is not required

**If individual queries fail** (e.g., empty results, command errors), log the failure and continue with whatever results you have. Partial data is better than no data.

```bash
ANALYZER=~/Documents/dev/cobalt-review-action/analyzer/analyzer
DB=~/Documents/dev/cobalt-review-action/analyzer/reviews.db
```

## Step 1: Team precedent for this type of change

Map changed file paths to topics and query for what the team has historically flagged:

```bash
# Map changed files to topics:
# app/models/      -> database, naming, validation
# app/controllers/ -> api-design, authorization, error-handling
# app/interactors/ -> error-handling, architecture, naming
# app/policies/    -> authorization, security
# app/serializers/ -> api-design, naming
# app/jobs/        -> error-handling, performance
# spec/            -> testing
# db/migrate/      -> database, performance

# For EACH topic relevant to the diff:
$ANALYZER query --db $DB --topic <topic> --limit 5 --verbose

# Also search by file path patterns found in the diff:
$ANALYZER search --db $DB "path:<directory_name>" --limit 5
```

## Step 2: Code suggestions for similar patterns

```bash
# Search for suggestions related to the diff's key concepts
# (class names, method patterns, gem/library names)
$ANALYZER search --db $DB "<key concept from diff>" --limit 10
$ANALYZER search --db $DB "<another concept>" --limit 10

# Example: if the diff adds an interactor with error rescue:
$ANALYZER search --db $DB "rescue interactor" --limit 5
$ANALYZER search --db $DB "context.fail" --limit 5
```

## Step 3: What makes this team block a PR

```bash
$ANALYZER query --db $DB --category security --sentiment negative --limit 5
$ANALYZER query --db $DB --category bug --sentiment negative --limit 5
$ANALYZER search --db $DB "blocking" --limit 3
$ANALYZER search --db $DB "do not merge" --limit 3
```

## Step 4: Reviewer-specific concerns for this area

Query reviewers whose strengths match the diff's areas:

```bash
# Roger: API design (10.9%), code suggestions (16.6%)
# If diff touches controllers/serializers:
$ANALYZER query --db $DB --reviewer roger-cobalt --topic api-design --limit 5 --verbose

# Paul: Questions (26.6%), error handling (13.7%)
# If diff touches interactors/services:
$ANALYZER query --db $DB --reviewer Lucianolo --curiosity question --topic error-handling --limit 5

# Mauricio: Database (19.5%)
# If diff touches models/migrations:
$ANALYZER query --db $DB --reviewer mauricio-reis --topic database --limit 5 --verbose

# David: Security (2.3%), authorization (8.0%), testing (23.4%)
# If diff touches policies/auth/controllers:
$ANALYZER query --db $DB --reviewer davidgm0 --topic authorization --limit 5
$ANALYZER query --db $DB --reviewer davidgm0 --category security --limit 5
```

## Step 5: Questions the team would ask

```bash
# Question-style comments for relevant topics — these surface design
# concerns that directive-style LLM reviews miss
$ANALYZER query --db $DB --curiosity question --topic <relevant_topic> --limit 5
```

## Output format

Combine the query results into `{TEAM_REVIEW_BRIEF}` with this structure:

```markdown
## Team Review Brief

### Top concerns for this type of change
[Synthesize from steps 1 + 3: what the team flags and blocks on for these file types]

### Real examples of team feedback on similar code
[Include 3-5 actual review comments from query results, verbatim with reviewer attribution]

### Code suggestions the team has made
[Include 2-3 actual code suggestions from step 2, showing what the team recommends]

### Questions the team would ask
[Include 2-3 actual question-style comments from step 5]

### Reviewer strengths relevant to this PR
[Brief note on which reviewer's expertise is most relevant and what they'd focus on]
```

## Sub-agent self-serve access

Sub-agents also get the `ANALYZER` and `DB` paths in their prompts so they can run
their own queries during the review. They should query when:

- They find a pattern deviation and want to check team precedent
- They're unsure about severity and want to see if the team blocks on it
- They spot a simplification and want to see if the team has suggested it
- They want to frame feedback as a question and see how the team phrases things
- They find a security issue and want to check if David has flagged it before

See the correctness-prompt.md and safety-prompt.md templates for the full
self-serve query section that gets included in each sub-agent's prompt.
