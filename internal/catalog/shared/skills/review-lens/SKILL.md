---
name: review-lens
description: "Retrieve relevant past review situations from a corpus of PR review comments. Use when reviewing code, authoring PRs, writing plans, or investigating what reviewers historically flag. Surfaces the most similar past objections ranked by usefulness — no API tokens, no LLM calls at query time."
trigger: /review-lens
tags: [code-review, retrieval, embeddings, patterns, rag]
---

# review-lens

Token-free retrieval engine for past PR review comments. Given code or a
concept, it finds the most relevant things reviewers have said before —
ranked by similarity AND usefulness.

**Two query paths:**

| Query | What it matches |
|-------|-----------------|
| `--code <file or diff>` | Structurally similar code patterns (embedding-based) |
| `--concept "<text>"` | Conceptual matches in comment text (embedding-based) |
| `search "<text>"` | Keyword matches (FTS, no Python needed) |
| `query --topic X` | Structured filters by topic, category, reviewer, sentiment |

## Quick start

### /review-lens setup

Run this first. It checks prerequisites and walks you through setup.

**Step 1: Install the binary**

```bash
curl -sSfL https://raw.githubusercontent.com/futuregerald/review-lens/main/install.sh | bash
```

Or build from source:
```bash
git clone https://github.com/futuregerald/review-lens.git
cd review-lens && CGO_ENABLED=0 go build -o review-lens .
sudo mv review-lens /usr/local/bin/
```

Verify: `review-lens version`

**Step 2: Set up the Python embedder (optional — needed for `similar` and `embed`)**

The `search`, `query`, `stats`, and `export` commands need only the binary.
The `similar` and `embed` commands need Python + fastembed for local embeddings.

```bash
uv venv --python 3.12 /tmp/rlvenv
VIRTUAL_ENV=/tmp/rlvenv uv pip install fastembed==0.8.0
export PATH=/tmp/rlvenv/bin:$PATH
```

Verify: `review-lens setup --field both`

**Step 3: Build your review corpus**

You need a `reviews.db` with your team's review comments. Two ways to get one:

*Option A: Fetch from GitHub (recommended)*
```bash
# Fetch review comments for specific reviewers from a repo:
review-lens fetch --reviewer alice --repo myorg/myrepo --db reviews.db
review-lens fetch --reviewer bob --repo myorg/myrepo --db reviews.db

# Classify the comments:
review-lens analyze --db reviews.db

# Embed for semantic search (one-time, ~10 min for 5k comments):
review-lens embed --field both --db reviews.db
```

*Option B: Import from JSON files*
```bash
review-lens import --dir ./data/ --repo myorg/myrepo --db reviews.db
review-lens analyze --db reviews.db
review-lens embed --field both --db reviews.db
```

**Step 4: Query**
```bash
# Semantic search — find structurally similar code:
review-lens similar --code app/models/user.rb --k 5 --db reviews.db

# Concept search — find reviews about a pattern:
review-lens similar --concept "blanket rescue hides errors" --k 5 --db reviews.db

# Keyword search (no Python needed):
review-lens search "N+1 eager load" --db reviews.db

# Structured query:
review-lens query --topic error-handling --category bug --db reviews.db
```

---

## Workflows

### /review-lens check

**When:** Before pushing a PR. Check your diff against past review patterns.

```bash
# Check a file:
review-lens similar --code app/interactors/my_feature.rb --k 5 --db reviews.db

# Check a concept:
review-lens similar --concept "adding nil guard on association" --k 5 --db reviews.db

# Pipe a diff:
git diff HEAD~1 | review-lens similar --code - --k 5 --db reviews.db
```

Look for blocking comments and code suggestions in the results. If senior
reviewers have flagged this pattern before, address it proactively.

### /review-lens plan

**When:** After writing an implementation plan, BEFORE writing code.

Extract the plan's key approaches as concept queries:
```bash
review-lens similar --concept "blanket rescue in interactors" --k 5 --db reviews.db
review-lens similar --concept "eager loading associations" --k 5 --db reviews.db
```

Report which patterns have been flagged before and what reviewers said.

### /review-lens search

**When:** Looking for what the team has said about a specific topic.

```bash
# Full-text search:
review-lens search "race condition" --db reviews.db --limit 10

# Filter by category and topic:
review-lens query --topic database --category performance --db reviews.db

# Reviewer-specific patterns:
review-lens query --reviewer alice --topic testing --db reviews.db --verbose
```

### /review-lens stats

**When:** Understanding reviewer patterns and team review culture.

```bash
review-lens stats --db reviews.db
review-lens stats --db reviews.db --reviewer alice
```

---

## Commands

| Command | Needs Python | Description |
|---------|-------------|-------------|
| `similar` | Yes | Retrieve similar past reviews by code or concept |
| `embed` | Yes | Build the vector index (one-time, incremental) |
| `search` | No | FTS keyword search across comments |
| `query` | No | Filter by reviewer, category, topic, sentiment |
| `stats` | No | Reviewer statistics and patterns |
| `fetch` | No | Fetch review comments from GitHub via `gh` CLI |
| `import` | No | Import from JSON files |
| `analyze` | No | Run heuristic classification on comments |
| `pattern` | Mixed | Manage the distilled pattern library |
| `bench` | No | Benchmark embedding models (P@k, MRR) |
| `export` | No | Export comments as JSON |
| `threads` | No | View comment threads with replies |
| `setup` | No | Validate Python env and pre-download models |

## How it works

- **Two embedding indexes:** code (diff_hunk vectors, jina-v2-base-code 768d)
  and comment (body vectors, bge-small-en-v1.5 384d)
- **Model provenance chain:** the model used to embed the corpus is stamped
  in the DB. Queries always use that same model. You can't accidentally mix
  vectors from different models.
- **Usefulness reranking:** results are reranked by a composite of similarity
  and usefulness (blocking comments and code suggestions surface higher).
  A 0.1+ similarity gap is never overturned by usefulness.
- **CGO-free Go binary:** SQLite via wazero WASM. No C toolchain needed.
  Cross-compiles to linux/darwin/windows x amd64/arm64.

## JSON output

All query commands support `--json` for structured output:
```bash
review-lens similar --code app/models/user.rb --k 5 --json --db reviews.db
review-lens query --topic testing --json --db reviews.db
review-lens search "N+1" --json --db reviews.db
```
