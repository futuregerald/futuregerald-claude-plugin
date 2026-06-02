# Data Collection Reference

How to fetch, store, and maintain the review comment database.

```bash
ANALYZER=~/Documents/dev/cobalt-review-action/analyzer/analyzer
DB=~/Documents/dev/cobalt-review-action/analyzer/reviews.db
```

## Prerequisites

- `gh` CLI installed and authenticated (`gh auth login`)
- The analyzer binary built: `cd ~/Documents/dev/cobalt-review-action/analyzer && go build -tags fts5 -o analyzer .`

## Fetching Data

### Fetch for a single reviewer

```bash
$ANALYZER fetch --db $DB --reviewer Lucianolo --repo cobalthq/cobalt-pentest-api
```

This:
1. Calls `gh api repos/cobalthq/cobalt-pentest-api/pulls/comments` with pagination
2. Filters to comments by the specified reviewer
3. Saves raw JSON to `data/<reviewer>_reviews.json`
4. Auto-imports into the database with classification

### Fetch with date filter

```bash
# Only fetch comments since January 2026
$ANALYZER fetch --db $DB --reviewer roger-cobalt --since 2026-01-01
```

Use `--since` for incremental updates. The database deduplicates by `github_id`, so re-fetching is safe.

### Fetch with reply threads

```bash
$ANALYZER fetch --db $DB --reviewer davidgm0 --all-threads
```

The `--all-threads` flag also collects replies to the reviewer's comments from other engineers, enabling thread analysis.

### Bulk fetch for all tracked reviewers

```bash
for user in Lucianolo roger-cobalt davidgm0 mauricio-reis; do
  echo "Fetching $user..."
  $ANALYZER fetch --db $DB --reviewer $user --repo cobalthq/cobalt-pentest-api --since 2026-01-01
done
```

### Fetch from a different repo

```bash
$ANALYZER fetch --db $DB --reviewer jchayan --repo cobalthq/cobalt-web
```

The database stores comments from any repo. The `pr_url` field preserves the source.

## Adding New Reviewers

To track a new team member:

```bash
# Fetch their full history
$ANALYZER fetch --db $DB --reviewer <github_login> --repo cobalthq/cobalt-pentest-api

# Verify they're in the database
$ANALYZER stats --db $DB --reviewer <github_login>
```

Current tracked reviewers:
- `Lucianolo` -- Paul Lucian Ursache (DL team)
- `roger-cobalt` -- Roger (DL team)
- `davidgm0` -- David Garcia Mora (CX team)
- `mauricio-reis` -- Mauricio Reis (DL team)

To add: `jchayan` (Jorge), `learodrigo` (Leandro), `falvariza-cobalt` (Fabian), `futuregerald` (Gerald).

## Importing from JSON Files

If you have review data in JSON format (e.g., collected via sub-agents or scripts):

```bash
# Single file
$ANALYZER import --db $DB --file data/custom_reviews.json

# Directory of files
$ANALYZER import --db $DB --dir data/

# Reply data
$ANALYZER import --db $DB --replies data/replies.json
```

### JSON format

```json
[
  {
    "github_id": 1234567890,
    "pr_number": 42,
    "pr_url": "https://github.com/org/repo/pull/42#discussion_r1234",
    "commit_sha": "abc123",
    "diff_hunk": "@@ -10,6 +10,8 @@\n context\n+added",
    "body": "The review comment text.",
    "path": "app/models/user.rb",
    "line": 15,
    "side": "RIGHT",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z",
    "reviewer": "github_login",
    "reviewer_name": "Display Name",
    "in_reply_to_id": 0,
    "suggestion_state": "none"
  }
]
```

## Collecting via Sub-Agents

For large-scale collection (thousands of comments across many pages), dispatch parallel sub-agents:

```
Agent({
  description: "Fetch review comments for <reviewer>",
  prompt: "Fetch all PR review comments by <reviewer> from cobalthq/cobalt-pentest-api using gh api.
    Run: gh api 'repos/cobalthq/cobalt-pentest-api/pulls/comments?per_page=100&sort=created&direction=desc' --paginate
    Filter to comments where user.login == '<reviewer>'.
    Save to ~/Documents/dev/cobalt-review-action/analyzer/data/<reviewer>_reviews.json
    in the format documented above.
    Then import: ~/Documents/dev/cobalt-review-action/analyzer/analyzer import --db ~/Documents/dev/cobalt-review-action/analyzer/reviews.db --file <output_path>"
})
```

Launch one agent per reviewer in parallel for fastest collection.

## Re-Classification

After updating the classifier heuristics in `internal/classifier.go`:

```bash
# Re-run classification on all stored comments
$ANALYZER analyze --db $DB
```

This reads every comment, re-runs `Classify()`, and updates all analysis fields in a single transaction.

## Database Maintenance

The database uses SQLite with WAL mode. No special maintenance needed.

```bash
# Check database size
ls -lh $DB

# Verify FTS index integrity
sqlite3 $DB "INSERT INTO review_fts(review_fts) VALUES('integrity-check');"

# Rebuild FTS index if needed
sqlite3 $DB "INSERT INTO review_fts(review_fts) VALUES('rebuild');"

# Vacuum to reclaim space after large deletes
sqlite3 $DB "VACUUM;"
```

## Data Freshness

| Reviewer | Last Fetched | Comments | Coverage |
|----------|-------------|----------|----------|
| davidgm0 | 2026-05-14 | 3,871 | 2021-04 to 2026-05 |
| roger-cobalt | 2026-05-14 | 1,250 | 2022-07 to 2026-05 |
| Lucianolo | 2026-05-14 | 1,107 | 2020-02 to 2026-05 |
| mauricio-reis | 2026-05-14 | 766 | 2021-12 to 2026-05 |

Check current state: `$ANALYZER stats --db $DB`
