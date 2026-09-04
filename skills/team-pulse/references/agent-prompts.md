# Sub-Agent Prompt Templates

Use these templates when dispatching sub-agents. The orchestrator resolves the date range and dispatches **one agent per day per source**. Replace `{VARIABLES}` with resolved values from Step 1.

## Per-Day Dispatch Pattern

For an 8-day window with 3 required sources, the orchestrator dispatches 24 agents in a single message:

```
Day 1: Agent A (Jira, Jun 2) + Agent B (GitHub, Jun 2) + Agent C (Meetings, Jun 2)
Day 2: Agent A (Jira, Jun 3) + Agent B (GitHub, Jun 3) + Agent C (Meetings, Jun 3)
...
Day 8: Agent A (Jira, Jun 9) + Agent B (GitHub, Jun 9) + Agent C (Meetings, Jun 9)
+ Optional: Agent D (Datadog, full range) + Agent E (Reviews, full range)
```

Each agent receives a **single date** (`{DATE}`) and the **next date** (`{NEXT_DATE}`) to bound its query. The orchestrator computes these — agents never calculate dates themselves.

## Context Efficiency Contract (applies to ALL agents)

Every sub-agent MUST follow these rules to keep context usage minimal:

1. **Single-day scope** — each agent queries exactly one day. The date is passed by the orchestrator.
2. **Scoped queries only** — filter by date, author, project at the API/CLI level. Never fetch everything and filter in-context.
3. **Summarize incrementally** — process one PR, ticket, or meeting at a time. Never load all results into context simultaneously.
4. **Write digest to disk** — write your compressed findings to `.updates/<source>-<date>.md` using the Write tool.
5. **Word limit scales with team size** — `50 words per person` in scope. The orchestrator passes `{WORD_LIMIT}` in your prompt. Stay within it.
6. **No raw data** — never include raw JSON, full API responses, or unprocessed tool output in the digest.
7. **Return a 1-line summary** — after writing the file, return only a brief confirmation (e.g., "Wrote .updates/jira-2026-06-03.md — 3 tickets moved, 1 blocker").
8. **Empty days are fine** — if no activity found, write "No activity." to the file and return. Don't waste context searching harder.

## Agent A: Jira Activity (per day)

```
Search Jira for {SCOPE_DESCRIPTION} on {DATE} using Atlassian MCP tools.

Cloud ID: {CLOUD_ID}   # from references/team.md

Run this JQL:
{JQL_QUERY}
Fields: summary, status, issuetype, assignee, priority, updated, labels

CONTEXT EFFICIENCY: This agent covers ONE DAY only ({DATE}). Process results incrementally —
summarize each ticket as you encounter it. If no results, write "No activity." and return.

Write your digest to `.updates/jira-{DATE}.md` using the Write tool. Format:
- Group by person: what they completed, what's in progress, what's stuck
- Flag: issues In Progress >5 days, unassigned work, blocked items
- Max {WORD_LIMIT} words

After writing the file, return only: "Wrote .updates/jira-{DATE}.md — {brief 1-line summary}"
```

### JQL Templates (single day)

**Full team, single day:**
```
project = DL AND updated >= "{DATE}" AND updated < "{NEXT_DATE}" ORDER BY updated DESC
```

**Single person, single day:**
```
project = DL AND assignee = "{JIRA_ACCOUNT_ID}" AND updated >= "{DATE}" AND updated < "{NEXT_DATE}" ORDER BY updated DESC
```

**Single epic/initiative, single day:**
```
project = DL AND (parent = {EPIC_KEY} OR key = {EPIC_KEY}) AND updated >= "{DATE}" AND updated < "{NEXT_DATE}" ORDER BY updated DESC
```

**Topic search, single day:**
```
project = DL AND (summary ~ "{TOPIC}" OR labels in ("{TOPIC}")) AND updated >= "{DATE}" AND updated < "{NEXT_DATE}" ORDER BY updated DESC
```

---

## Agent B: GitHub PRs (per day)

```
Search GitHub for PR activity by {SCOPE_DESCRIPTION} on {DATE} (from {DATE} to {NEXT_DATE}).

Team GitHub handles: {HANDLES_LIST}
Repos: {ORG}/{REPO} for each repo listed in references/team.md

CONTEXT EFFICIENCY: This agent covers ONE DAY only ({DATE}). Use --limit and --search filters
to scope at the source. Process each repo independently — summarize before moving to the next.

For each repo, run:
gh pr list --repo {ORG}/{REPO} --state all {AUTHOR_FLAG} --limit 20 \
  --json number,title,author,state,createdAt,mergedAt,reviewDecision,additions,deletions,headRefName \
  --search "created:{DATE}..{NEXT_DATE} OR merged:{DATE}..{NEXT_DATE}" | cat

Summarize this repo's results immediately, then move to the next repo.
If no results across all repos, write "No activity." and return.

Write your digest to `.updates/github-{DATE}.md` using the Write tool. Format:
- PRs merged (with +/- lines)
- PRs opened or updated
- Stale PRs touched this day (>3 days without review) — flag explicitly
- Max {WORD_LIMIT} words

After writing the file, return only: "Wrote .updates/github-{DATE}.md — {brief 1-line summary}"
```

### Author flag

- **Full team:** omit `--author` flag, then filter results by team handles from the output
- **Single person:** `--author {HANDLE}`

---

## Agent C: Krisp Meetings (per day)

```
Search Krisp for meetings on {DATE} involving {PERSON_OR_TEAM}.

CONTEXT EFFICIENCY: This agent covers ONE DAY only ({DATE}). Process one meeting at a time.
Use search_meetings (structured data) first — you likely don't need full transcripts for a
single day's meetings. If no meetings found, write "No meetings." and return.

1. Search meetings:
   Use mcp__krisp__search_meetings with:
   - search: "{SEARCH_TERM}"
   - after: "{DATE}"
   - before: "{NEXT_DATE}"
   - limit: 10
   - fields: ["name", "date", "attendees", "speakers", "key_points", "action_items", "detailed_summary"]

   Summarize each meeting's findings as you process it. Move on.

2. Full transcripts — ONLY if a meeting needs deeper context:
   Use mcp__krisp__get_multiple_documents with the specific meeting ID.
   Process, extract, summarize, discard.

Write your digest to `.updates/meetings-{DATE}.md` using the Write tool. Extract:
- What was discussed and committed to
- Action items with owners
- Blockers or concerns raised
- Max {WORD_LIMIT} words

After writing the file, return only: "Wrote .updates/meetings-{DATE}.md — {brief 1-line summary}"
```

### Search terms

- **Single person:** use their first name (e.g., "Leandro", "Jorge")
- **Full team:** run one search per person, or search for the team lead name and look at attendee lists
- **Topic:** search for the topic name (e.g., "Flywheel", "CAP", "VulnCheck")

### Important: Krisp is scoped to Gerald's account

Krisp returns meetings Gerald attended or that were shared with him. It won't show meetings between other team members that Gerald wasn't part of. Note this limitation in findings if relevant.

---

## Agent D: Datadog (Optional — full range, NOT per-day)

```
Search Datadog for recent deploy and incident activity related to {SCOPE}.

Use mcp__datadog__search_datadog_events with:
- query: "source:deploy OR source:incident {SERVICE_FILTER}"
- from: "now-{DAYS}d"

Write your digest to `.updates/datadog.md` using the Write tool. Format:
- Deploys by DL team members (count, services affected)
- Any incidents or alerts triggered
- Max 200 words

After writing the file, return only: "Wrote .updates/datadog.md — {brief 1-line summary}"
```

---

## Agent E: GitHub Reviews Given (Optional — full range, NOT per-day)

```
Search GitHub for PRs reviewed by {HANDLE} across the configured repos since {START_DATE}.

gh search prs --reviewed-by {HANDLE} --owner {ORG} --updated ">={START_DATE}" \
  --json repository,number,title,author,state --limit 30 | cat

Write your digest to `.updates/reviews.md` using the Write tool. Format:
- How many PRs reviewed
- Whose PRs they reviewed (pattern: reviewing one person vs. spread across team)
- Any review given on non-DL repos
- Max 200 words

After writing the file, return only: "Wrote .updates/reviews.md — {brief 1-line summary}"
```

---

## Combining Results for Large Tool Outputs

If any sub-agent tool call returns a result that is too large (saved to file), dispatch a follow-up sub-agent:

```
Read the file {FILE_PATH} in sequential chunks using offset/limit until you have
read 100% of it. This contains {DESCRIPTION}.

Extract and return:
- {SPECIFIC_EXTRACTION_INSTRUCTIONS}
- Max {WORD_LIMIT} words
```

Never read these files directly in the orchestrator.
