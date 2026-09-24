# Sub-Agent Prompt Templates

Use these templates when dispatching sub-agents. The orchestrator resolves the date range and dispatches **one agent per source, covering the whole range**. Replace `{VARIABLES}` with resolved values from Step 1.

## Dispatch Pattern

Three required sources means **three agents**, launched in a single message:

```
Agent A (Tracker,  {START_DATE} .. {END_DATE})
Agent B (GitHub,   {START_DATE} .. {END_DATE})
Agent C (Meetings, {START_DATE} .. {END_DATE})
+ Optional: Agent D (Metrics) + Agent E (Reviews)
+ Optional: Agent F (Work Breakdown) — single-person or single-epic scope only
```

Each agent receives the **full range** — `{START_DATE}` and `{END_DATE}`. The orchestrator
computes these; agents never calculate dates themselves.

**Give every agent a `tools:` grant if your harness supports it.** `tools:` is a field in an
agent *definition file*, not something you can pass on a dispatch — so this requires defining
agent types for these sources and naming them as `subagent_type`. This skill ships none, so by
default each agent inherits the session's whole tool surface and pays the unrestricted
~57,000-token floor. A sub-agent also does not necessarily inherit the parent's MCP servers, so
any definition you write must grant them explicitly.

**Do not split a source by day, by default — including for 1:1s.** One agent per source, covering
the whole window, is always the default; every extra agent pays the floor again — 24 agents cost
roughly 1,365,000 tokens of floor against ~171,000 for 3 with inherited grants. Per-day fan-out is
a deliberate exception, taken only when the user explicitly asks for it, and only for agents A–C —
see [batching-rationale.md](batching-rationale.md). A per-day agent writes a dated digest
(`.updates/<source>-<YYYY-MM-DD>.md`) so the days do not overwrite each other.

Batching multiplies each agent's raw *input* by the window length, so covering the whole range in
one agent is safe only where the query can cap the response, as every template below does. If one
source cannot be capped, split it in halves — never into days — and write numbered digests
(`.updates/<source>-1.md`, `-2.md`) so the halves do not overwrite each other.

## Context Efficiency Contract (applies to ALL agents)

Every sub-agent MUST follow these rules to keep context usage minimal:

1. **Whole-range scope** — each agent queries its own source across `{START_DATE}`..`{END_DATE}`, passed by the orchestrator. **Both bounds are INCLUSIVE** — `{END_DATE}` is the last day of the window and must appear in your results. An exclusive upper bound drops the most recent day, which is the one that matters most.
2. **Scoped queries only** — filter by date, author, project at the API/CLI level. Never fetch everything and filter in-context.
3. **Summarize incrementally** — process one PR, ticket, or meeting at a time. Never load all results into context simultaneously.
4. **Write digest to disk** — write your compressed findings to `.updates/<source>.md` using the Write tool.
5. **Word limit scales with team size and window** — the orchestrator computes `50 x people x days` and passes it as `{WORD_LIMIT}`. It already accounts for the full range you are covering. Stay within it; do not scale it down yourself.
6. **No raw data** — never include raw JSON, full API responses, or unprocessed tool output in the digest.
7. **Return a 1-line summary** — after writing the file, return only a brief confirmation (e.g., "Wrote .updates/jira.md — 3 tickets moved, 1 blocker").
8. **Empty ranges are fine** — if no activity found, write "No activity." to the file and return. Don't waste context searching harder.

## Agent A: Tracker Activity (whole range)

```
Search the tracker for {SCOPE_DESCRIPTION} between {START_DATE} and {END_DATE} using the tracker MCP tools.

Cloud ID: {CLOUD_ID}   # from the team config

Run this JQL:
{JQL_QUERY}
Fields: summary, status, issuetype, assignee, priority, updated, labels, parent

EPIC COMPLETION AGGREGATION:
For each active epic with activity, also query its child issues:
- Jira Cloud: `parent in ({ACTIVE_EPIC_KEYS})`
- Jira Server/DC fallback: `"Epic Link" in ({ACTIVE_EPIC_KEYS})`
Fields: summary, status, issuetype, parent, resolution

For each active epic, also fetch the epic itself (fields: summary, description, priority, parent,
assignee) and record:
- ONE plain sentence on what it delivers, from its `description` field, written for someone who has
  never opened the ticket. Never a restatement of the title. This is the highest-priority field: if
  budget runs short, deliver these and drop the rest. An empty description is a finding; say "no
  description on the ticket" rather than guessing.
- Its priority, and its parent initiative's key and name.
- Its assignee AND whether that account's `active` flag is false. A departed owner reads as
  "someone has this" on every board view.
- Whether its own status contradicts its children: a parent reading In Progress while its children
  are Won't Do or untouched for months is an abandoned plan. Flag it with dates on both levels.
For each open child, record one line on what it is, from its description.

DISCOVER the epics in scope rather than working only from a supplied list: search by label, by
summary match, by links from known roots, and by the parents of issues that moved this window. A
hand-enumerated list can only confirm what someone already believed.

List epics that have NOT STARTED (Backlog or To Do with zero children done) in their own section,
with how long each has sat. They are invisible to any activity-based query.

COMMENTS, FOR BLOCKED/STALLED/QUESTIONED ITEMS ONLY: for any item that is Blocked, In Progress
for more than 5 days, or has an open question, fetch its last 10 comments (add `comment` to the
field list for that item only) and report any unanswered question: who asked, who it was aimed
at, the date, and how long it has been silent.

PAGINATION & MATH RULES:
- Read `total` from response metadata for the denominator. If results are capped, use `total`, never `results.length`.
- If `Total == 0`, report `N/A (No child issues logged)`.
- Exclude cancelled/won't do issues from both numerator and denominator
  (`(resolution is EMPTY OR resolution not in ("Won't Do", "Declined", "Cancelled"))`). JQL's
  `not in` never matches an empty field, so a bare `resolution not in (...)` silently drops every
  open child from the denominator too and makes epics read as far more complete than they are.
- Compute: `% Complete = (Done delivering issues) / (Total active scope issues) * 100`.

CONTEXT EFFICIENCY: Process results incrementally across the range —
summarize each ticket as you encounter it, then discard it. If no results, write "No activity." and return.

Write your digest to `.updates/jira.md` using the Write tool. Format:
- Group by person: what they completed, what's in progress, what's stuck
- Active Epics Progress Breakdown:
  - What it is: one sentence from the description, priority, parent initiative (key + name).
  - Quantitative progress: Total issues, Done count, In-Progress count, and % complete.
  - The JQL behind each count, so the report can link the number to the query that produced it.
  - Exactly Why It Needs Attention / At Risk: Root cause, upstream dependency, failure mode, or idle duration if not On Track. (If tracker is silent, note empirical observation: e.g. "No code pushed or ticket movement for N days").
  - What's Left (TL;DR): Select 2–4 items strictly prioritized by: (1) In-review PRs, (2) Active assigned in-progress tasks, (3) Next unblocked milestone tickets. If >4 items remain, summarize as "- [Top 3 items] and N other open tickets".
- Flag: issues In Progress >5 days, unassigned work, blocked items
- Unanswered questions found in comments, each with asker, addressee, date, and age
- Max {WORD_LIMIT} words

After writing the file, return only: "Wrote .updates/jira.md — {brief 1-line summary}"
```

### JQL Templates (whole range)

**Full team, whole range:**
```
project = {PROJECT_KEY} AND updated >= "{START_DATE}" AND updated <= "{END_DATE} 23:59" ORDER BY updated DESC
```

**Single person, whole range:**
```
project = {PROJECT_KEY} AND assignee = "{JIRA_ACCOUNT_ID}" AND updated >= "{START_DATE}" AND updated <= "{END_DATE} 23:59" ORDER BY updated DESC
```

**Single epic/initiative, whole range:**
```
project = {PROJECT_KEY} AND (parent = {EPIC_KEY} OR key = {EPIC_KEY}) AND updated >= "{START_DATE}" AND updated <= "{END_DATE} 23:59" ORDER BY updated DESC
```

**Topic search, whole range:**
```
project = {PROJECT_KEY} AND (summary ~ "{TOPIC}" OR labels in ("{TOPIC}")) AND updated >= "{START_DATE}" AND updated <= "{END_DATE} 23:59" ORDER BY updated DESC
```

---

## Agent B: GitHub PRs (whole range)

```
Search GitHub for PR activity by {SCOPE_DESCRIPTION} from {START_DATE} to {END_DATE}.

Team GitHub handles: {HANDLES_LIST}
Repos: {ORG}/{REPO} for each repo listed in the team config

CONTEXT EFFICIENCY: Process results incrementally across the range. Use --limit and --search filters
to scope at the source. Process each repo independently — summarize before moving to the next.

For each repo, run:
gh pr list --repo {ORG}/{REPO} --state all {AUTHOR_FLAG} --limit 20 \
  --json number,title,author,state,createdAt,mergedAt,closedAt,reviewDecision,additions,deletions,headRefName,url \
  --search "created:{START_DATE}..{END_DATE} OR merged:{START_DATE}..{END_DATE}" | cat

Summarize this repo's results immediately, then move to the next repo.
If no results across all repos, write "No activity." and return.

Write your digest to `.updates/github.md` using the Write tool. Format:
- PRs merged (with +/- lines, opened date and merged date)
- PRs opened or updated (with opened date)
- PRs closed without merging (with closed date)
- Every PR open right now, whatever its age, including drafts: read this from `pr_scan.py`'s
  `.updates/prs.json` (the `open` list) rather than this agent's own windowed query, which cannot
  see a PR opened before `{START_DATE}` — an old open PR is easy to forget
- Every PR as a link, with its tracker key if the title or branch names one
- Stale, unreviewed PRs in the range (`stale_unreviewed`: stale and nobody reviewing it) — flag
  explicitly. An approved-but-unmerged PR is reported separately, not as this
- For each epic key found in a PR title or branch: count merged and open PRs by side —
  frontend/backend, from `{FRONTEND_REPOS}` / `{BACKEND_REPOS}` in the team config
- Max {WORD_LIMIT} words

After writing the file, return only: "Wrote .updates/github.md — {brief 1-line summary}"
```

### Author flag

- **Full team:** omit `--author` flag, then filter results by team handles from the output
- **Single person:** `--author {HANDLE}`

---

## Agent C: Meetings (whole range)

```
Search the meeting source for meetings between {START_DATE} and {END_DATE} involving {PERSON_OR_TEAM}.

CONTEXT EFFICIENCY: Process one meeting at a time, summarising as you go — never load them
all at once. Use search_meetings (structured data) first; you rarely need full transcripts.
If no meetings found, write "No meetings." and return.

1. Search meetings:
   Use the meeting-notes MCP's search tool (for example `mcp__krisp__search_meetings`) with:
   - search: "{SEARCH_TERM}"
   - after: "{START_DATE}"
   - before: "{END_DATE}"
   - limit: {WINDOW_MEETING_LIMIT}   # orchestrator passes 10 x days in window; default 80
   - fields: ["name", "date", "attendees", "speakers", "key_points", "action_items", "detailed_summary"]

   Summarize each meeting's findings as you process it. Move on.

   **If the number of results equals the limit, the window is truncated** — say so explicitly in
   the digest so the orchestrator knows the report covers only part of the range.

2. Full transcripts — ONLY if a meeting needs deeper context:
   Use the meeting-notes MCP's document-fetch tool (for example `mcp__krisp__get_multiple_documents`)
   with the specific meeting ID. Process, extract, summarize, discard.

Write your digest to `.updates/meetings.md` using the Write tool. Extract:
- What was discussed and committed to
- Action items with owners
- Blockers or concerns raised
- Max {WORD_LIMIT} words

After writing the file, return only: "Wrote .updates/meetings.md — {brief 1-line summary}"
```

### Search terms

- **Single person:** use their first name
- **Full team:** run one search per person, or search for the team lead name and look at attendee lists
- **Topic:** search for the topic or project name

### Important: the meeting source is scoped to one account

It returns only meetings the account holder attended or that were shared with them — not
meetings between other team members. Note this limitation in the findings when it matters.

---

## Agent F: Work Breakdown (single-person and single-epic scopes only)

Dispatch this only when the scope is one person or one epic. A team report gets the one-line
frontend/backend split from agents A and B instead.

```
Explain what {PERSON_OR_EPIC} built in epics {EPIC_KEYS}, split into frontend and backend, and
what is done versus left.

Frontend repos: {FRONTEND_REPOS}   Backend repos: {BACKEND_REPOS}   # from the team config

1. For each epic, list its children (fields: summary, status, assignee, resolutiondate).
2. Find the matching PRs: gh pr list --repo {ORG}/{REPO} {AUTHOR_FLAG} --state all --limit 40 \
     --search "updated:>={EPIC_START}" \
     --json number,title,state,createdAt,mergedAt,closedAt,additions,deletions,headRefName,url
   Match a PR to a ticket by the key in its title or branch name.
3. For each matched PR, one at a time:
   gh pr view {N} --repo {ORG}/{REPO} --json body --jq '.body[0:1500]'
   Write ONE plain sentence on what it changes for the user. Say whether it runs on real data
   or stubbed/mock data, and name any endpoint or permission check it adds.
4. For each open child, write one line on what it is, from its description. If it has no
   description, say so and describe it from the title.
5. If the epic's parent has sibling epics the work depends on (for example, a backend epic
   feeding a frontend one), list them: key, title, status, owner, one line each.

Write .updates/breakdown.md. Per epic:
- 2–4 sentence summary: which side of the stack, what is real vs stubbed, what it depends on
- Backend table and Frontend table: Ticket | Status | PR | Opened | Merged | What it does
  (Merged is the date, "open", or "closed unmerged <date>")
- Still to do: Ticket | Status / owner | What it is
- PRs closed without merging: why, from the last comments (paraphrase), and whether a
  replacement exists
Every ticket and PR as a link. Max {WORD_LIMIT} words.

Return only: "Wrote .updates/breakdown.md — {1-line summary}"
```

---

## Agent D: Metrics (Optional)

```
Search the metrics source for recent deploy and incident activity related to {SCOPE}.

Use the metrics MCP's event-search tool (for example `mcp__datadog__search_datadog_events`) with:
- query: "source:deploy OR source:incident {SERVICE_FILTER}"
- from: "{START_DATE}"
- to: "{END_DATE}"

Write your digest to `.updates/metrics.md` using the Write tool. Format:
- Deploys by team members (count, services affected)
- Any incidents or alerts triggered
- Max 200 words

After writing the file, return only: "Wrote .updates/metrics.md — {brief 1-line summary}"
```

---

## Agent E: GitHub Reviews Given (Optional)

```
Search GitHub for PRs reviewed by {HANDLE} across the configured repos since {START_DATE}.

This search finds candidate PRs updated in the window — it does not mean the review itself was
submitted in the window. Confirm that separately:

1. gh search prs --reviewed-by {HANDLE} --owner {ORG} --updated ">={START_DATE}" \
     --json repository,number,title,author,state --limit 30 | cat

2. For each candidate PR, list {HANDLE}'s actual review timestamps:
     gh api repos/{ORG}/{REPO}/pulls/{N}/reviews \
       --jq '[.[] | select(.user.login=="{HANDLE}") | .submitted_at]'
   Count only reviews whose `submitted_at` falls within {START_DATE}..{END_DATE} inclusive.

Write your digest to `.updates/reviews.md` using the Write tool. Format:
- How many reviews were submitted within {START_DATE}..{END_DATE} (not how many PRs were merely updated in it)
- Whose PRs they reviewed (pattern: reviewing one person vs. spread across team)
- Any review given on repos outside the configured list
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
