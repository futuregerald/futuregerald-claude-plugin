---
name: team-pulse
description: Generate a concise team status report for an engineering manager before calls or check-ins. Covers progress, blockers, risks, individual workloads, PRs in flight, meeting context, and project health assessments. Default scope is the Delivery Domain (DL) team over the last 1.2 weeks. Use when the user says "team pulse", "team status", "what's my team working on", "prep me for standup", "what happened this week", "sprint update", "team report", "how is [project] going", "how is [person] doing", "prep me for 1:1", or any request for a team/project/person activity summary.
---

# Team Pulse

Generate a scannable status report an EM can read in 2 minutes before a call.

## Defaults

| Setting | Default | Override |
|---------|---------|----------|
| Team scope | Delivery Domain (DL) | User specifies team, project, epic, or person |
| Time window | 1.2 weeks (~8 days) | User specifies "last week", "last 2 weeks", "since Monday", etc. |
| Depth | Summary | User asks for "detailed" or "deep dive" |

## Architecture: Map-Reduce with Disk Intermediates

**The orchestrator (you) NEVER queries data sources directly.** All data gathering is delegated to sub-agents. The orchestrator stays lean — it resolves scope, dispatches agents, reads small digest files, and synthesizes.

### Why This Architecture

Many small agent contexts beat one mega-prompt. Each sub-agent keeps its own context small by:
- Querying only scoped, filtered data (never fetch-all-then-filter)
- Summarizing incrementally (one item at a time, not all at once)
- Writing a compressed digest to disk (not returning raw data via tool results)

This makes the skill fast and viable on local models with limited context windows.

### Flow

1. **Resolve scope** (orchestrator) — parse request, load team roster, compute date range
2. **Dispatch sub-agents in parallel** (orchestrator) — one agent per data source, each writes to `.updates/`
3. **Synthesize report** (orchestrator) — read the small digest files and produce the final report
4. **Deliver** (orchestrator) — output the report, clean up `.updates/`

### Sub-Agent Design Rules

- Each sub-agent gets a self-contained prompt with all context it needs (team roster, date range, exact queries)
- Sub-agents write their digest to a file in `.updates/` (e.g., `.updates/jira.md`, `.updates/github.md`, `.updates/meetings.md`)
- **Digest word limit scales with team size:** `50 words per person` in scope. A 7-person team = 350-word cap per daily digest. A single-person query = 50 words. Optional agents (D, E) that cover the full range get 200 words.
- Sub-agents summarize incrementally: process one PR, one ticket, or one meeting at a time. Never concatenate all raw data and summarize in one pass.
- The orchestrator reads ONLY the digest files — never raw JSON, full API responses, or large tool results
- If a sub-agent's tool call returns data too large to fit in its context, it must filter/summarize in chunks before writing the digest
- **Never read large files or raw JSON in the orchestrator** — if a sub-agent result is too large, dispatch another sub-agent to summarize it

### Context Efficiency Rules

These rules exist to minimize context usage in every agent, enabling fast execution on local models:

| Rule | Why |
|------|-----|
| **Scoped queries only** | Filter by date/repo/project/author at the source. Never fetch all then filter in context. |
| **Summarize incrementally** | Process items one at a time within each sub-agent. Never load all items, then summarize. |
| **Disk intermediates** | Sub-agents write to `.updates/` files. The synthesis step reads only these compressed digests. |
| **Parallel sub-agents** | Each agent keeps its own small context. No shared state between data-gathering agents. |
| **Digest cap scales with scope** | 50 words per person in scope. 7-person team = 350 words/day. Single person = 50 words/day. |
| **One agent per day per source** | 8-day window = 24 parallel agents (3 sources x 8 days). Each agent's context stays small. |
| **No duplicate data** | If Jira and GitHub both mention a PR, the orchestrator deduplicates during synthesis — not by loading both raw datasets. |

## Step 0: Prepare Workspace

Create the `.updates/` directory for intermediate digest files:
```bash
mkdir -p .updates
```

This directory is ephemeral — cleaned up after the report is delivered.

## Step 1: Resolve Scope

Parse the user's request for:
- **Team/project** — default: DL. Could be a Jira project key, epic, initiative, or person name.
- **Time window** — default: 8 days back from today. Convert relative dates to **a list of absolute dates** (e.g., `["2026-06-02", "2026-06-03", ..., "2026-06-09"]`).
- **Depth** — summary (default) or detailed.

Load team roster from [references/team.md](references/team.md). If scope is non-DL, ask the user for team members and Jira project key.

## Step 2: Dispatch Sub-Agents (Parallel — One Per Day Per Source)

**Key pattern: dispatch one agent per day per source.** For an 8-day window with 3 required sources, that's 24 agents running in parallel. Each agent queries exactly one day of data, keeping its context tiny.

Launch ALL agents in a **single message with multiple Agent tool calls**. Each agent prompt must include: team roster, GitHub handles, the **single date** it covers, and exact queries scoped to that date.

**Each agent writes its digest to `.updates/<source>-<date>.md`.** For example:
```
.updates/jira-2026-06-02.md
.updates/jira-2026-06-03.md
.updates/github-2026-06-02.md
.updates/github-2026-06-03.md
.updates/meetings-2026-06-02.md
.updates/meetings-2026-06-03.md
...
```

The orchestrator does NOT read the tool results for data — it reads the files in Step 3.

See [references/agent-prompts.md](references/agent-prompts.md) for the exact prompt templates for each agent.

### Required Agents (per day)

| Agent | Source | Tool | Per Day? |
|-------|--------|------|----------|
| A: Jira Activity | Atlassian MCP | `searchJiraIssuesUsingJql` | Yes — 1 agent per day |
| B: GitHub PRs | gh CLI | `gh pr list`, `gh search prs` | Yes — 1 agent per day |
| C: Krisp Meetings | Krisp MCP | `search_meetings`, `search_meeting_content` | Yes — 1 agent per day |

### Optional Agents (once, not per-day)

| Agent | Source | Tool | When? |
|-------|--------|------|-------|
| D: Datadog | Datadog MCP | `search_datadog_events` | User asks about deploys, incidents, reliability |
| E: GitHub Reviews | gh CLI | `gh search prs --reviewed-by` | Single-person deep dives |

### Why Per-Day?

A single agent querying 8 days of Jira/GitHub data fills its context fast, slows down requests, and is especially painful on local models. One agent per day means each agent handles a small slice — fast queries, tiny context, fast summarization. The parallelism makes the total wall-clock time shorter, not longer.

## Step 3: Synthesize Report

Read ONLY the digest files from `.updates/`. List them first:
```bash
ls .updates/
```

You'll see files like:
```
jira-2026-06-02.md    github-2026-06-02.md    meetings-2026-06-02.md
jira-2026-06-03.md    github-2026-06-03.md    meetings-2026-06-03.md
...
datadog.md            reviews.md              (if dispatched)
```

Each daily digest is max 100 words. Read them all — they're tiny. The orchestrator's synthesis context is just these digests + the report format — never raw data.

Follow the format in [references/report-format.md](references/report-format.md). Key rules:

- **Lead with the headline.** One sentence: are we on track or not?
- **Brevity over completeness.** Skip anything that's fine. Highlight what needs attention.
- **Name names.** "Paul has 2 PRs awaiting review for 4 days" not "some PRs are stale."
- **Assessments are required.** For each person and each project/epic, give a 1-line assessment.
- **Link everything.** Jira keys and PR numbers must be clickable.
- **No filler.** No "here's what I found" or "let me summarize." Just the report.
- **Meeting context enriches, not replaces.** Use Krisp data to add color (quotes, action items, sentiment) to Jira/GitHub findings. Don't create a separate "meetings" section for team-wide reports — weave it into the person's assessment. For single-person reports, a dedicated Meetings section is fine.
- **Deduplicate across sources.** If Jira and GitHub both reference the same work, merge into one mention.

## Step 4: Deliver

Output the report directly. If the user asked for Confluence or Slack format, adapt.

Clean up intermediates:
```bash
rm -rf .updates
```

## Scoping Variations

| User Says | Scope To |
|-----------|----------|
| "team pulse" | Full DL team, all active work |
| "team pulse on flywheel" | DL team members working on Flywheel only |
| "how is Paul doing" | Single person across all their work |
| "pulse on DL-2094" | Single initiative/epic and everyone assigned |
| "what did we ship this week" | Merged PRs + completed Jira issues only |
| "prep me for 1:1 with Molly" | Single person, deeper individual assessment |

## Assessment Scale

| Rating | Meaning |
|--------|---------|
| On Track | Progressing as expected, no concerns |
| Needs Attention | Minor risk, slipping, or blocked but recoverable |
| At Risk | Significant blocker, timeline threat, or capacity issue |
| Blocked | Cannot proceed without external input/decision |

## Anti-Patterns

- Do NOT query data sources directly from the orchestrator. Always use sub-agents.
- Do NOT read large tool results in the orchestrator. Dispatch a sub-agent to summarize.
- Do NOT dump raw Jira/GitHub/Krisp data. Synthesize.
- Do NOT include tickets that are Done unless user asks "what did we ship."
- Do NOT assess people you have no data on. Say "no activity in window" instead.
- Do NOT editorialize beyond the data. Assessments must cite specific evidence.
- Do NOT use more than 3 sentences for any single person's section (team report) or 5 sentences (individual report).
