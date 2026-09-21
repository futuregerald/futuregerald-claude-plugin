---
name: team-pulse
description: Generate a concise team status report for an engineering manager before calls or check-ins. Covers progress, blockers, risks, individual workloads, PRs in flight, meeting context, and project health assessments. Default scope is the team configured in references/team.md over the last 1.2 weeks. Use when the user says "team pulse", "team status", "what's my team working on", "prep me for standup", "what happened this week", "sprint update", "team report", "how is [project] going", "how is [person] doing", "prep me for 1:1", or any request for a team/project/person activity summary.
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

This keeps the orchestrator small. **On a local model, read the batching argument below with
care:** its saving is measured in tokens, which are free locally, while the binding constraint
there is the context window — and batching makes each agent's window 8x larger. On a local
model, split sources aggressively or keep the window short.

### Flow

1. **Resolve scope** (orchestrator) — parse request, load team roster, compute date range
2. **Dispatch sub-agents in parallel** (orchestrator) — one agent per data source, each writes to `.updates/`
3. **Synthesize report** (orchestrator) — read the small digest files and produce the final report
4. **Deliver** (orchestrator) — output the report, clean up `.updates/`

### Sub-Agent Design Rules

- Each sub-agent gets a self-contained prompt with all context it needs (team roster, date range, exact queries)
- Sub-agents write their digest to a file in `.updates/` (e.g., `.updates/jira.md`, `.updates/github.md`, `.updates/meetings.md`)
- **Digest word limit scales with team size AND window length.** The budget is
  `50 words x people in scope x days in window`, and the orchestrator computes it and passes it
  as `{WORD_LIMIT}`. A 7-person team over 8 days = **2,800 words** for that source's digest.
  A single person over 8 days = 400. Optional agents (D, E) get a flat 200.
  **Do not pass a per-day cap to an agent covering a whole range** — it silently truncates the
  report to an eighth of the work and nothing flags it.
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
| **Digest cap scales with scope AND window** | `50 x people x days`. 7-person team, 8 days = 2,800 words per source. Single person, 8 days = 400. The orchestrator computes it — an agent covering 8 days must not be handed a one-day budget. |
| **One agent per SOURCE, not per day** | 3 agents for a 3-source window, not 24. Each dispatch costs ~57,000 tokens of floor before it does any work (~31,000 with a restricted `tools:` grant), and that floor is charged per agent, not per token of data. Splitting one source across 8 days pays the floor 8 times to move the same digest volume. |
| **No duplicate data** | If Jira and GitHub both mention a PR, the orchestrator deduplicates during synthesis — not by loading both raw datasets. |

## Step 0: Prepare Workspace

Create the `.updates/` directory for intermediate digest files:
```bash
mkdir -p .updates
```

This directory is ephemeral — cleaned up after the report is delivered.

## Step 1: Resolve Scope

Parse the user's request for:
- **Team/project** — default: the team in `references/team.md`. Could be a tracker project key, epic, initiative, or person name.
- **Time window** — default: 8 days back from today. Resolve to an absolute **start and end date** (e.g., `2026-06-02` to `2026-06-09`) and pass that range to each agent.
- **Depth** — summary (default) or detailed.
- **Word budget** — compute `{WORD_LIMIT} = 50 x (people in scope) x (days in window)` and pass
  it to every required agent. Optional agents D and E take a flat 200.
- **Meeting result cap** — compute `{WINDOW_MEETING_LIMIT} = 10 x (days in window)` and pass it to
  Agent C, so batching does not shrink its capacity below what per-day agents had.

**Date convention — both bounds are INCLUSIVE.** `{START_DATE}` is the first day of the window
and `{END_DATE}` is the last, so an 8-day window ending today is `2026-06-02`..`2026-06-09`.
Every query template is written to include `{END_DATE}` itself. Getting this wrong is silent and
costly: an exclusive upper bound drops today's activity, which is the day an EM most needs, and
it desynchronises the tracker from GitHub so today's PRs appear with no matching ticket movement
— which reads exactly like the "code shipped, tickets not moved" red flag the report format
treats as a finding.

Load team roster from [references/team.md](references/team.md). If the scope falls outside the configured team, ask the user for its members and tracker project key.

## Step 2: Dispatch Sub-Agents (Parallel — One Per Source)

**Default pattern: one agent per source, covering the whole window.** Three required sources
means **three agents**. Fan out per day instead when attribution accuracy or speed matters more
than tokens — see the trade table below; the prompts in `references/agent-prompts.md` take a
date range, so a per-day agent is the same prompt with `{START_DATE} == {END_DATE}`. Each agent queries its own source across the full
date range and writes a single digest.

Launch ALL agents in a **single message with multiple Agent tool calls**. Each agent prompt must
include: team roster, GitHub handles, the **full date range** it covers, and exact queries scoped
to that range.

**Restrict each agent's tool grant if your harness lets you.** `tools:` is a field in an *agent
definition file*, not a dispatch parameter — you cannot pass it on an `Agent(...)` call. So this
is only available if you define agent types for these three sources (each granted its own data
source plus `Write`) and name them as the `subagent_type`. **This skill does not ship them.**
Out of the box each agent inherits the full tool surface and costs the unrestricted floor.

Note also that a sub-agent does not necessarily inherit the parent's MCP servers, so agents A and
C need definitions that explicitly grant theirs.

**Each agent writes its digest to `.updates/<source>.md`:**
```
.updates/jira.md
.updates/github.md
.updates/meetings.md
.updates/metrics.md     (if dispatched)
.updates/reviews.md     (if dispatched)
```

The orchestrator does NOT read the tool results for data — it reads the files in Step 3.

See [references/agent-prompts.md](references/agent-prompts.md) for the exact prompt templates for each agent.

### Required Agents — one each, full window

| Agent | Source | Tool | Grant it needs |
|-------|--------|------|----------------|
| A: Tracker Activity | Tracker MCP | issue search by JQL or equivalent | that MCP server + `Write` |
| B: GitHub PRs | `gh` CLI | `gh pr list`, `gh search prs` | `Bash` + `Write` |
| C: Meetings | Meeting-notes MCP | meeting + content search | that MCP server + `Write` |

### Optional Agents

| Agent | Source | Tool | When? |
|-------|--------|------|-------|
| D: Metrics | Metrics MCP | event search | User asks about deploys, incidents, reliability |
| E: GitHub Reviews | `gh` CLI | `gh search prs --reviewed-by` | Single-person deep dives |

### One per source is the default. Per day is a supported choice.

**The per-day split bought something real, and it was not context headroom.** It bought
**attribution isolation.** An agent that holds only Tuesday cannot report Tuesday's work under
Wednesday, cannot merge two people's tickets into one summary, and cannot carry a stray detail
from an adjacent day into a sentence about this one. An agent holding eight days for seven people
can do all three, and **nothing downstream will flag it** — the digest will be fluent,
well-formed, and wrong in a way only the person it describes would catch.

| | One per source (default) | One per source per day |
|---|---|---|
| Dispatch floors | 3 | 24 |
| Tokens | ~171,000 | ~1,365,000 |
| Wall clock | slower | faster — fan-out parallelises |
| Raw input per agent | 8x larger | bounded at one day |
| Attribution risk | **real, and silent** | structurally prevented |

**Default to one per source.** Take the per-day fan-out when the report will be acted on
personally — a 1:1, a performance conversation, anything where a name attached to the wrong piece
of work is the expensive failure — or when you need the report fast.

**Why this skill is the risky shape, when batching is usually safe.** Batching many *lookups*
onto one agent is measured safe even on deliberately confusable material: eight sibling handlers
with adjacent line numbers came back 8/8 with zero cross-attribution, twice, because every answer
had a unique key tying it to one source line. **These digests have no such key.** The agent reads
prose about seven people across eight days and emits a narrative; nothing structurally binds a
sentence to its author or its date, so a merge leaves no trace. That is the difference between
"list these eight line numbers" and "summarise what everyone did", and it is why the fan-out
stays on the table here even though the token arithmetic dislikes it.

The instinct to split by day is usually stated as context headroom. That is the wrong worry —
misattribution is the right one.

**The digest volume is identical either way.** At 50 words per person per day, a 7-person,
8-day window produces ~2,800 words per source whether one agent writes it or eight do. What
changes is how many times you pay the startup cost.

**What batching genuinely costs, stated plainly: each agent's raw INPUT multiplies by the window
length.** The digest is the output; the tool responses the agent reads to produce it are not, and
eight per-day agents bounded that pull at one day each. **So batching is safe exactly where the
source lets you cap the response at the query — a result limit, a field list, a date bound — and
only there.** Every template in `references/agent-prompts.md` does cap its query, which is what
makes this change safe; if you add a source that cannot, do not batch it.

**That cost is per agent, not per token.** A dispatch costs ~57,000 tokens before the agent does
anything — mostly tool-definition schema — or ~31,000 with a restricted `tools:` grant. So:

| Shape | Agents | Floor paid |
|---|---|---|
| One per source per day (what this skill used to do) | 24 | ~1,365,000 |
| One per source, grants inherited — **what you get by default** | 3 | ~171,000 |
| One per source, with restricted agent definitions | 3 | ~74,000 (3 x the measured 24,561 null-task cost) |

Read the middle row as the actual payoff of this change: **8x**, from batching alone. The third
row needs agent definitions this skill does not ship, and its extra saving is the grant change,
not the batching — do not credit one with the other.

**Batching is a token saving, not a latency one — and it is a latency cost.** Nine questions
answered by one agent cost 76,993 tokens against 165,052 split across two agents. But the only
wall-clock A/B in the router skill has a five-way fan-out finishing 19% faster than doing the
work inline, so a wide fan-out is genuinely quicker. **This skill trades that latency for the
token saving deliberately.** If you need the report in the next sixty seconds more than you need
the tokens, fan out and accept the cost. Answer quality did not degrade at ~100,000 tokens of
accumulated context, so the batched agent's larger context is not the concern.

**When to split a source anyway.** Two different triggers, and they want different splits:

- **For capacity** — the source's response cannot be capped at the query, or a normal week turns
  out unusually busy. The agent's first defence is rule 3 in `agent-prompts.md`: summarise one
  item at a time and discard it, never accumulate. Where that is not enough, split into **halves
  or thirds**. Days are the wrong unit here; each split costs another floor and capacity does not
  need that granularity.
- **For attribution** — the report will be acted on personally and a name against the wrong work
  is the expensive failure. Here **days are exactly the right unit**, because the boundary you
  want the agent unable to cross is the day. See the trade table above.

Whichever the trigger, **a split source must write numbered digests** —
`.updates/<source>-1.md`, `.updates/<source>-2.md`, or `.updates/<source>-<date>.md` for a
per-day split. Otherwise the second agent silently overwrites the first and Step 3 reports on
part of the window from a file that looks complete.

## Step 3: Synthesize Report

Read ONLY the digest files from `.updates/`. List them first:
```bash
ls .updates/
```

You'll see files like:
```
jira.md    github.md    meetings.md    metrics.md    reviews.md
```

Each digest is capped at `50 x people x days` words. For a 7-person, 8-day window that is up to
~2,800 words per source — roughly 3,500 tokens, so three sources is ~10,000 tokens of synthesis
context. Read them all. The orchestrator's context is these digests plus the report format,
never raw data. This total is the same whether the digests arrived from 3 agents or 24; only
the number of dispatch floors paid differs.

Follow the format in [references/report-format.md](references/report-format.md). Key rules:

- **Lead with the headline.** One sentence: are we on track or not?
- **Brevity over completeness.** Skip anything that's fine. Highlight what needs attention.
- **Name names.** "<person> has 2 PRs awaiting review for 4 days" not "some PRs are stale."
- **Assessments are required.** For each person and each project/epic, give a 1-line assessment.
- **Link everything.** Jira keys and PR numbers must be clickable.
- **No filler.** No "here's what I found" or "let me summarize." Just the report.
- **Meeting context enriches, not replaces.** Use meeting data to add color (action items, decisions, sentiment) to tracker and GitHub findings. **Do not quote transcripts and do not name the meeting tool in the report** — say "on a call". Don't create a separate "meetings" section for team-wide reports — weave it into the person's assessment. For single-person reports, a dedicated Meetings section is fine.
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
| "team pulse" | Full team from `references/team.md`, all active work |
| "team pulse on <project>" | Team members working on that project only |
| "how is <person> doing" | Single person across all their work |
| "pulse on ABC-123" | Single initiative/epic and everyone assigned |
| "what did we ship this week" | Merged PRs + completed Jira issues only |
| "prep me for 1:1 with <person>" | Single person, deeper individual assessment |

## Assessment Scale

| Rating | Meaning |
|--------|---------|
| On Track | Progressing as expected, no concerns |
| Needs Attention | Minor risk, slipping, or blocked but recoverable |
| At Risk | Significant blocker, timeline threat, or capacity issue |
| Blocked | Cannot proceed without external input/decision |

## Anti-Patterns

- Do NOT query data sources directly from the orchestrator. Always use sub-agents.
- Do NOT fan out per day by reflex — and do not refuse to when attribution matters. One per
  source is the default; per day is the deliberate choice for 1:1s, performance conversations,
  and anything needing speed. See the trade table.
- Do NOT dispatch an agent without a `tools:` grant. It doubles the floor for no benefit.
- Do NOT read large tool results in the orchestrator. Dispatch a sub-agent to summarize.
- Do NOT dump raw tracker, GitHub or meeting data. Synthesize.
- Do NOT include tickets that are Done unless user asks "what did we ship."
- Do NOT assess people you have no data on. Say "no activity in window" instead.
- Do NOT editorialize beyond the data. Assessments must cite specific evidence.
- Do NOT use more than 3 sentences for any single person's section (team report) or 5 sentences (individual report).
