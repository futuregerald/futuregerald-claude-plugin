---
name: team-pulse
description: Generate a team status report for an engineering manager - traffic lights and a plain-language description per epic, what shipped, what has not started, PRs in flight including work with no ticket at all, per-person assessments, and every blocker linked to the thing blocking it. Covers whole programs or a single person, and can publish a screen-shareable page to talk over on a call. Use when the user says "team pulse", "team update", "team status", "delivery update", "program update", "status update for leadership", "update for the call", "sprint update", "what's my team working on", "what happened this week", "what has my team been doing", "write up the last two weeks", "prep me for standup", "prep me for 1:1", "how is [person] doing", "how is [project] going", or names a program and asks for its update.
---

# Team Pulse

Generate a scannable status report an EM can read in 2 minutes before a call.

## First Run: Configure Your Team

`references/team.md` **ships empty on purpose** — this skill is published in a public repo, so
it carries no roster.

Before doing anything else, read `references/team.md`. If any value is still a bracketed
placeholder (`[your-org]`, `[Your Name]`, …), the skill is not configured yet. Stop and offer
to set it up:

> "team-pulse isn't configured yet — I need your tracker, GitHub org, repos, and roster before
> I can build a report. Want me to set that up now? I can read most of it off your git remotes
> and recent tickets, then show you the file to correct."

If they say yes, fill it in from what you can observe — `git remote -v` for the org and repos,
recent PR authors and ticket assignees for a first-draft roster — then **show the file and ask
them to correct it.** Never guess a person's role, and never invent a teammate. A wrong roster
produces a confidently wrong status report about real people.

If they say no, or ask you to continue anyway, run against whatever scope they name in the
request and say plainly in the report that the roster was not configured.

Once configured, skip this section entirely.

## Defaults

| Setting | Default | Override |
|---------|---------|----------|
| Team scope | The team in `references/team.md` | User specifies team, project, epic, or person |
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
- **Team/project** — default: the project key configured in `references/team.md`. Could be a
  tracker project key, epic, initiative, or person name.
- **Time window** — default: 8 days back from today. Convert relative dates to **a list of absolute dates** (e.g., `["2026-06-02", "2026-06-03", ..., "2026-06-09"]`).
- **Depth** — summary (default) or detailed.

Load team roster from [references/team.md](references/team.md). If the scope falls outside the configured project, ask the user for the team members and the project key.

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
| **F: Epic structure** | Atlassian MCP | `searchJiraIssuesUsingJql` | **No — once per run** |

Agent F is in [agent-prompts.md](references/agent-prompts.md) and supplies the descriptions,
child counts, deactivated-assignee flags, and the not-started list. Without it the report is a
list of ticket numbers.

### Required once, not per-day

**Epic descriptions and structure.** One agent, covering every epic in scope. It must return,
for each: a one-sentence plain-language description **from the `description` field**, the
done/total child count, the assignee **and whether that account's `active` flag is false**, and
whether the epic has started at all.

This is the highest-priority output. If an agent's budget runs short it delivers descriptions
and drops everything else — a report of ticket numbers with no descriptions is unusable to
anyone who does not live in the tracker.

**Epics that have NOT started.** Ask for these explicitly: Backlog or To Do with zero children
done. They are invisible to any activity-based query, so a report built only from "what
changed" will never contain them, and their absence makes a program look healthier than it is.

**Scope by discovery, not by a hand-written list.** Find the program's epics from labels,
summary matches, and links — never from a fixed set of keys in config. A hand-enumerated list
can only confirm what you already believed, and will silently miss whole workstreams.

### Reuse before you re-query

Before dispatching, check the run directory for digests from an earlier pass in this session.
When one exists, name its path in the agent's prompt, state exactly which facts it already
carries, and tell the agent to copy those through rather than re-fetch. Tracker queries are the
slowest part of this skill and a widened second pass overlaps the first by most of its rows.

### Optional Agents (once, not per-day)

| Agent | Source | Tool | When? |
|-------|--------|------|-------|
| D: Datadog | Datadog MCP | `search_datadog_events` | User asks about deploys, incidents, reliability |
| E: GitHub Reviews | gh CLI | `gh search prs --reviewed-by` | Single-person deep dives |

### Why Per-Day?

A single agent querying 8 days of Jira/GitHub data fills its context fast, slows down requests, and is especially painful on local models. One agent per day means each agent handles a small slice — fast queries, tiny context, fast summarization. The parallelism makes the total wall-clock time shorter, not longer.

## Step 2b: Scan PRs against the tracker

Run the script directly — this part is mechanical and must not go to an agent:

```bash
python3 scripts/pr_scan.py \
  --org ORG --repos repo-one,repo-two \
  --since YYYY-MM-DD --keys ABC,XYZ \
  --roster handle-one,handle-two,handle-three \
  --stale-days 3 --out "$RUN_DIR"
```

**`--roster` is not optional.** Repos are shared with other teams, so without it every count is
repo-wide and wildly overstates the team's output — measured at 83 repo-wide against 22 for the
team in one real week. Pass the roster's GitHub handles from `references/team.md` and read the
`*_team` counts (`merged_team`, `open_team`, `stale_team`, `no_ticket_team`), never the bare
totals. See [report-format.md](references/report-format.md#count-the-team-not-the-repo).

Writes `prs.json` and `prs.md`. Every PR lands in exactly one bucket: `linked_in_scope`,
`linked_out_of_scope`, `no_ticket`, `declared_no_ticket`.

`no_ticket` is the point of this step — work the tracker cannot see, invisible to any
Jira-only report. So is a PR approved months ago and never merged while its ticket reads Done.

**If the script exits with `TRUNCATED`, do not proceed.** Narrow the window and rerun. `gh pr
list` caps at 30 by default and at 1000 on the search path, both silently; a partial scan makes
the untracked section look complete while being empty.

**The window bounds merged PRs only.** Open PRs report as current state regardless of
`--since`, because a PR open five weeks is exactly what a status report should surface.

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
- **Every epic carries a one-sentence plain-language description** of what the work is, from
  its `description` field. Never a restatement of the title. Non-negotiable.
- **Traffic lights on every epic and every person** — RED / AMBER / GREEN, by the conditions in
  [report-format.md](references/report-format.md#traffic-lights).
- **Link the blocker, not just the blocked thing.** When something is blocked, say
  **"Blocked by:"** and link the specific open question, PR, decision ticket, or dependency.
  A blocker the reader cannot click is one they have to hunt for, and nobody hunts mid-meeting.
- **Every light carries a ticket key plus a date or a count.** No citation, no light.
- **Name names.** "Paul has 2 PRs awaiting review for 4 days" not "some PRs are stale."
- **Link everything.** Jira keys and PR numbers must be clickable.
- **Report what you could not measure as "not measured", never as zero.** A silent gap reads
  as a fact.
- **No filler.** No "here's what I found" or "let me summarize." Just the report.
- **Meeting context enriches, not replaces.** Use Krisp data to add color (quotes, action items, sentiment) to Jira/GitHub findings. Don't create a separate "meetings" section for team-wide reports — weave it into the person's assessment. For single-person reports, a dedicated Meetings section is fine.
- **Deduplicate across sources.** If Jira and GitHub both reference the same work, merge into one mention.

## Step 4: Deliver

Output the report directly. If the user asked for Confluence or Slack format, adapt.

**When the report will be shown to other people** — a leadership call, a screen-share, anything
read aloud rather than read alone — also publish it as an Artifact page and hand back the link.
Load the `artifact-design` skill first. Keep the same section order; the page earns its place by
being glanceable, not by being different:

- Traffic lights as a coloured dot **and** the word RED / AMBER / GREEN. Never colour alone —
  projectors shift hue and roughly 1 in 12 men cannot separate red from green.
- Each epic's description sits directly under its name, in the resting state, not behind a click.
- The unstarted pile collapses behind one summary line with its count.
- Every ticket key and PR number is a link; every blocker links to its blocker.
- A footer stating sources, window, and generation time. A status page with no provenance gets
  argued with; one that says where its numbers came from gets acted on.

Clean up intermediates:
```bash
rm -rf .updates
```

## Scoping Variations

| User Says | Scope To |
|-----------|----------|
| "team pulse" | The whole configured team, all active work |
| "team pulse on <project>" | Only the team members working on that project |
| "how is Paul doing" | Single person across all their work |
| "pulse on ABC-123" | Single initiative/epic and everyone assigned |
| "what did we ship this week" | Merged PRs + completed Jira issues only |
| "prep me for 1:1 with <name>" | Single person, deeper individual assessment |

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
