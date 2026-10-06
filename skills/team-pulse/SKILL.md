---
name: team-pulse
description: Engineering-manager reports in two modes. PULSE — a scannable status report before calls or check-ins, covering progress, blockers, risks, individual workloads, PRs in flight and meeting context, for a team, project, epic or one person (1:1 prep). FORECAST — verify the true state of epics or roadmap rows against the tracker and code, then forecast remaining work against measured throughput, giving 1-engineer vs 2-engineer estimates, dependency ordering and critical chain, projected landing, and, given a window, capacity arithmetic and commit tiers ("what fits in Q4"). Use for "team pulse", "team status", "what's my team working on", "prep me for standup", "prep me for 1:1", "how is [person] doing", "how is [project] going", "sprint update", and also "forecast", "how long will this take", "estimate these epics", "does this fit in Q4", "what can we commit to this quarter", "when will X land", "sequence this work", "verify the state of this initiative".
---

# Team Pulse

Two modes. **Pulse:** a scannable status report an EM can read in 2 minutes before a call. **Forecast:** a verified state-and-estimate report for epics or a roadmap, including whether a list fits in a window.

## Pick the Mode First

| The request | Mode | Follow |
|---|---|---|
| Status, activity, "what happened", standup or 1:1 prep | **Pulse** | This file |
| "How long", "when will it land", "does it fit in <window>", "what can we commit to", estimates, sequencing, splits, "what is really the state of these epics" | **Forecast** | `references/forecast/method.md`, entirely |

If it is unclear which mode, ask.

**Forecast mode follows its own method.** It has its own phases (baseline, measured pace, WIP,
clustered research, adversarial review, estimates, delivery), its own agent prompts and its own report
format, all under `references/forecast/`. The pulse sections below (map-reduce digests, word budgets,
the pulse report format) do not apply to it. What the two modes share: the roster and tracker
configuration in the team config, the linking and plain-language rules, and the
visual style. A forecast's HTML page is built on `assets/forecast.html`.

A pulse can point at a forecast ("the checkout rebuild is at risk — see the forecast") but never runs
the forecast method inline: a full forecast costs several research agents, and a pulse is meant to
be read in two minutes.

## First Run: Configure Your Team

`references/team.md` **ships empty on purpose** — this skill is published in a public repo, so
it carries no roster. Local team configurations live in git-ignored `references/team.local.md`.

**The team config** is `references/team.local.md` when it exists, else `references/team.md`. Every
other section of this file says "the team config" and means whichever of the two applies.

Before doing anything else, check the team config. If neither file exists or any value is still a
bracketed placeholder (`[your-org]`, `[Your Name]`, …), the skill is not configured yet. Stop and
offer to set it up:

> "team-pulse isn't configured yet — I need your tracker, GitHub org, repos, and roster before
> I can build a report. Want me to set that up now? I can read most of it off your git remotes
> and recent tickets, then show you the file to correct."

If they say yes, fill it in from what you can observe — `git remote -v` for the org and repos,
recent PR authors and ticket assignees for a first-draft roster, and on Jira each roster member's
Jira Account ID from the `assignee.accountId` of their recent tickets (leave it blank when none
turns up) — then save to `references/team.local.md` and **show the file and ask them to correct
it.** Never guess a person's role, and never invent a teammate. A wrong roster produces a
confidently wrong status report about real people. Never write a real roster into the tracked
`references/team.md`.

If they say no, or ask you to continue anyway, run against whatever scope they name in the
request and say plainly in the report that the roster was not configured.

Once configured, skip this section entirely.

## Defaults

| Setting | Default | Override |
|---------|---------|----------|
| Team scope | The team in the team config | User specifies team, project, epic, or person |
| Time window | 1.2 weeks (~8 days) | User specifies "last week", "last 2 weeks", "since Monday", etc. |
| Depth | Summary | User asks for "detailed" or "deep dive" |

## Architecture: Map-Reduce with Disk Intermediates

**The orchestrator (you) NEVER queries data sources directly — except Steps 2a and 2b.**
`jira_scan.py` (Step 2a) and `pr_scan.py` (Step 2b) are deterministic scripts, not queries into
context: each writes a size-bounded digest that the orchestrator reads like any other. Every other
data gathering step is delegated to sub-agents. The orchestrator stays lean — it resolves scope,
runs the two scripts, dispatches agents, reads small digest files, and synthesizes.

### Why This Architecture

Many small agent contexts beat one mega-prompt, and a script beats an agent wherever collection is
mechanical: Jira and GitHub are collected by scripts in seconds, with no agent floor and no
tool call per item. Each sub-agent keeps its own context small by:
- Querying only scoped, filtered data (never fetch-all-then-filter)
- **Work in bulk.** One query per source covering the whole range, then process every result in a single pass — with a short script when the result is large. Never spend one tool call per ticket, PR or meeting.
- Writing a compressed digest to disk (not returning raw data via tool results)

This keeps the orchestrator small. **On a local model, read the batching argument in
[references/batching-rationale.md](references/batching-rationale.md) with care:** its saving is
measured in tokens, which are free locally, while the binding constraint there is the context
window — and batching makes each agent's window 8x larger. On a local model, split sources
aggressively or keep the window short.

### Flow

1. **Resolve scope** (orchestrator) — parse request, load team roster, compute date range
2. **Collect** (orchestrator) — run `jira_scan.py`, then dispatch the agents in one message and run `pr_scan.py` while they work; everything lands in `.updates/`
3. **Synthesize report** (orchestrator) — read the small digest files and produce the final report
4. **Deliver** (orchestrator) — output the report, clean up `.updates/`

### Sub-Agent Design Rules

- Each sub-agent gets a self-contained prompt with all context it needs (team roster, date range, exact queries)
- Sub-agents write their digest to a file in `.updates/` (e.g., `.updates/meetings.md`, `.updates/docs.md`); the two scripts write theirs the same way
- **Digest word limit scales with team size AND window length.** The budget is
  `50 words x people in scope x days in window`, and the orchestrator computes it and passes it
  as `{WORD_LIMIT}`. A 7-person team over 8 days = **2,800 words** for that source's digest.
  A single person over 8 days = 400. Optional agents (D, E) get a flat 200.
  **Do not pass a per-day cap to an agent covering a whole range** — it silently truncates the
  report to an eighth of the work and nothing flags it.
- **Work in bulk.** One query per source covering the whole range, then process every result in a single pass — with a short script when the result is large. Never spend one tool call per ticket, PR or meeting.
- The orchestrator reads ONLY the digest files — never raw JSON, full API responses, or large tool results. `jira.json` and `prs.json` are inputs for scripts, not for the orchestrator
- If a sub-agent's tool call returns data too large to fit in its context, it must filter/summarize in chunks before writing the digest
- **Never read large files or raw JSON in the orchestrator** — if a sub-agent result is too large, dispatch another sub-agent to summarize it

### Context Efficiency Rules

These rules exist to minimize context usage in every agent, enabling fast execution on local models:

| Rule | Why |
|------|-----|
| **Scoped queries only** | Filter by date/repo/project/author at the source. Never fetch all then filter in context. |
| **Work in bulk** | One query per source covering the whole range, then process every result in a single pass — with a short script when the result is large. Never spend one tool call per ticket, PR or meeting. |
| **Disk intermediates** | Sub-agents write to `.updates/` files. The synthesis step reads only these compressed digests. |
| **Parallel sub-agents** | Each agent keeps its own small context. No shared state between data-gathering agents. |
| **Digest cap scales with scope AND window** | `50 x people x days`. 7-person team, 8 days = 2,800 words per source. Single person, 8 days = 400. The orchestrator computes it — an agent covering 8 days must not be handed a one-day budget. |
| **One agent per SOURCE, not per day** | One meeting agent for an 8-day window, not 8. Each dispatch costs ~57,000 tokens of floor before it does any work (~31,000 with a restricted `tools:` grant), and that floor is charged per agent, not per token of data. Splitting one source across 8 days pays the floor 8 times to move the same digest volume. |
| **No duplicate data** | If Jira and GitHub both mention a PR, the orchestrator deduplicates during synthesis — not by loading both raw datasets. |

## Step 0: Prepare Workspace

Create the `.updates/` directory for intermediate digest files:
```bash
mkdir -p .updates
```

This directory is ephemeral — cleaned up after the report is delivered.

**Two files persist across runs**, in the reports directory (Step 4), never in `.updates/`:

- `.doc-cache.json`: one entry per document, keyed by doc id plus its last-modified time, holding
  the digest Agent G wrote. An unchanged doc is never read twice.
- `.source-prefs.json`: which tool worked for each kind of source (wiki, docs, sheets), and which
  failed and how. Agent G reads it first and updates it at the end.

Both are local conveniences and hold no secrets; if either is missing or unreadable, start empty.

## Step 1: Resolve Scope

Parse the user's request for:
- **Team/project** — default: the team in the team config. Could be a tracker project key, epic, initiative, or person name.
- **Time window** — default: 8 days back from today. Resolve to an absolute **start and end date** (e.g., `2026-06-02` to `2026-06-09`) and pass that range to each agent.
- **Depth** — summary (default) or detailed.
- **Word budget** — compute `{WORD_LIMIT} = 50 x (people in scope) x (days in window)` and pass
  it to every required agent. Optional agent D takes a flat 200; optional agent E takes 200 and
  agent F takes `{WORD_LIMIT_F}` (below).
- **Meeting result cap** — compute `{WINDOW_MEETING_LIMIT} = 10 x (days in window)` and pass it to
  Agent C, so batching does not shrink its capacity below what per-day agents had.
- **Excluded resolutions** — read the team config's Excluded resolutions line: the resolutions that
  mean "not delivered" (for example Won't Do, Declined, Duplicate). They go to `jira_scan.py` as
  `--excluded-resolutions` and to Agent A as `{EXCLUDED_RESOLUTIONS}`, each value double-quoted and
  comma-separated (`"Won't Do", "Declined", "Duplicate"`). The values are site-specific, and one the
  site lacks fails every query, so if the line is missing, ask the user; never guess.
- **Epic start, for Agent F** — when Agent F is dispatched, compute `{EPIC_START}`: the earliest
  `Created` date among the in-scope epics in `jira.md`'s `## Active epics` and `## Not started`. Agent F's queries
  describe the epic's own work, not the reporting window, so they need the epic's start date, not
  `{START_DATE}`.
- **Word budget for Agent F** — compute `{WORD_LIMIT_F} = 250 x (epics in scope)` and pass it
  instead of `{WORD_LIMIT}`.

**Date convention — both bounds are INCLUSIVE.** `{START_DATE}` is the first day of the window
and `{END_DATE}` is the last, so an 8-day window ending today is `2026-06-02`..`2026-06-09`.
Every query template is written to include `{END_DATE}` itself. Getting this wrong is silent and
costly: an exclusive upper bound drops today's activity, which is the day an EM most needs, and
it desynchronises the tracker from GitHub so today's PRs appear with no matching ticket movement
— which reads exactly like the "code shipped, tickets not moved" red flag the report format
treats as a finding.

Load the team roster from the team config. If the scope falls outside the configured team, ask the user for its members and tracker project key.

## Step 2: Collect (Scripts First, Then One Agent Per Source)

**The order is fixed:**

1. **Step 2a: run `jira_scan.py`** when the team config's tracker type is `jira`. It takes seconds.
2. **Right after it returns, in one message:** dispatch Agent C, Agent G and any optional agents
   (D, E, F) the scope calls for, and run Step 2b (`pr_scan.py --jira-map .updates/jira.json`)
   while they work.
3. **Fallback path** (the tracker is not Jira, or Step 2a exited 1): dispatch Agent A, Agent C and
   any optional agents in one message. Once Agent A returns, dispatch Agent G and run Step 2b,
   because both read what Agent A writes (`jira.md`'s Doc links, `jira.json`'s child-to-epic map).

**Default: always one agent per source, covering the whole window — including 1:1s.** Fan out per
day only when the user explicitly asks for it, and only for agents A and C: see
[references/batching-rationale.md](references/batching-rationale.md) for the trade-off. The
prompts in `references/agent-prompts.md` take a date range, so a per-day agent is the same prompt
with `{START_DATE} == {END_DATE}`, and each per-day agent writes its own dated digest
(`.updates/<source>-<YYYY-MM-DD>.md`) so the files do not overwrite each other. Otherwise, each
agent queries its own source across the full date range and writes a single digest.

Each agent prompt must include: team roster, GitHub handles, the **full date range** it covers,
and exact queries scoped to that range.

**Restrict each agent's tool grant if your harness lets you.** `tools:` is a field in an *agent
definition file*, not a dispatch parameter — you cannot pass it on an `Agent(...)` call. So this
is only available if you define agent types for these sources (each granted its own data
source plus `Write`) and name them as the `subagent_type`. **This skill does not ship them.**
Out of the box each agent inherits the full tool surface and costs the unrestricted floor.

Note also that a sub-agent does not necessarily inherit the parent's MCP servers, so agents A and
C need definitions that explicitly grant theirs.

**Everything collected lands in `.updates/`:**
```
.updates/jira.json      (Step 2a; on the fallback path Agent A writes only its child_to_epic map)
.updates/jira.md        (Step 2a, or Agent A on the fallback path)
.updates/prs.json       (Step 2b)
.updates/prs.md         (Step 2b)
.updates/meetings.md    (Agent C)
.updates/docs.md        (Agent G)
.updates/metrics.md     (if Agent D dispatched)
.updates/reviews.md     (if Agent E dispatched)
.updates/breakdown.md   (if Agent F dispatched)
```

The orchestrator does NOT read the tool results for data — it reads the `.md` files in Step 3.

See [references/agent-prompts.md](references/agent-prompts.md) for the exact prompt templates for each agent.

### Required Sources — one each, full window

| Source | Collected by | Tool | Grant it needs |
|--------|--------------|------|----------------|
| Tracker (Jira) | Step 2a, `jira_scan.py` | `acli` | none: the orchestrator runs it |
| Tracker (not Jira, or Step 2a exited 1) | Agent A: Tracker Activity (fallback) | issue search by JQL or equivalent | that MCP server + `Write` |
| GitHub PRs | Step 2b, `pr_scan.py` | `gh` CLI | none: the orchestrator runs it |
| Meetings | Agent C | meeting + content search | that MCP server + `Write` |

If the meeting source needs re-authentication, do not authenticate from a sub-agent: write
"Meeting source unavailable" to the digest and say so in the report.

### Optional Agents

| Agent | Source | Tool | When? |
|-------|--------|------|-------|
| D: Metrics | Metrics MCP | event search | User asks about deploys, incidents, reliability |
| E: GitHub Reviews | `gh` CLI | `gh search prs --reviewed-by` | Single-person scope (always, for 1:1s) |
| F: Work Breakdown | Tracker MCP + `gh` CLI | epic children, PR bodies | Single-person or single-epic scope only |
| G: Docs | Whatever wiki, doc and spreadsheet tools the session has | metadata first, then read only what changed | Every run where `jira.md` lists doc links or the team config lists standing docs; forecast mode always |

**Agent G is cheap by design, and gets cheaper each run.** It never searches first. Its candidates
are the doc links `jira.md` lists for in-scope epics plus any standing docs in the team config. It
checks each doc's last-modified time and reads only docs changed inside the window, at most 8, on
the cheapest model, 60 words per doc. Summaries are cached by doc id and modified time, so an
unchanged doc costs one metadata call. Search is a fallback for epics with no linked docs, capped
at 5 results by title and date. A run where nothing changed costs a few thousand tokens.

**It picks its own tools.** Do not name a connector in the prompt. The agent uses whatever doc,
wiki and spreadsheet tools the session exposes, and when more than one could serve the same
source (two accounts, two connectors), it tries them and records which one returned the
document. That choice is saved in the source preferences file (Step 0) and tried first next time.
A source that is not connected, or needs a login, is reported as "not checked"; the agent never
authenticates or retries.

### Batching vs Fan-Out

- **Default: One agent per source across the full window**, always — including 1:1s. Scoped
  queries cap raw input. The two scripts are not agents and never fan out.
- **Fan-out by day, for agents A and C only:** Use only when the user explicitly asks for
  it — not by default for 1:1s or performance reviews. See
  [references/batching-rationale.md](references/batching-rationale.md) for full benchmarks and
  trade-off analysis.

## Step 2a: Scan Jira (Jira Tracker Only)

Run it first, before any agent, and only when the team config's tracker type is `jira`; any other
tracker goes straight to Agent A. This is one of the two named exceptions to "the orchestrator
never queries data sources directly" — `jira_scan.py` is a deterministic script, not a query into
context, so it is mechanical and must not go to an agent:

```bash
python3 <skill-dir>/scripts/jira_scan.py \
  --keys ABC,XYZ --since {START_DATE} --until {END_DATE} \
  --config <team config path> \
  --excluded-resolutions "Won't Do,Declined,Duplicate" \
  --word-limit {WORD_LIMIT} --out .updates
```

- `--config` is the team config file (Step 1). The script reads its roster table and matches
  assignees by the optional Jira Account ID column first, by name otherwise.
- `--excluded-resolutions` takes the values on the team config's Excluded resolutions line
  (Step 1), comma-separated. The values in the example above are one site's, not a default; never
  pass them without reading the config.
- `--word-limit` is `{WORD_LIMIT}`. Single-person scope adds `--scope person --person "<roster
  name>"`; single-epic scope adds `--scope epic --epic ABC-123`. The default scope is the team.
- It needs `acli`, installed and authenticated. Never authenticate it on the user's behalf.

It writes `jira.json` (input for Step 2b's `--jira-map`; never read it yourself) and `jira.md`, the
tracker digest. `jira.md` puts the sections that are never cut first: `## Not measured` (only when
something failed), `## Active epics` (in team scope, only epics with work done or in progress:
key, title, status, priority, parent, assignee with an `(inactive)` flag, created date,
done/total/in-progress, a 120-character description excerpt, `Refs` for other tickets the
description names, and up to 2 open children with review or acceptance first; person and epic
scopes give every epic a block), `## Closed or ongoing epics` (team scope: one line per Done,
Won't Do, Released or Ongoing epic, with its owner and, when 3 or fewer, its open children),
`## Flagged` (Blocked, or In Progress with no status change for 5 days: one line each with its age,
the newest comment for the first 15 lines, and one closing `Long-stalled` line listing every
non-epic item stalled over 30 days with its owner's first name; stalled epics are left out because
their own sections cover them), `## Unassigned`
(unassigned bugs and unassigned In Progress work), `## High priority outside epics` (not-done
P0/P1, Highest, Blocker or Critical window items with no parent at all), `## Question candidates`,
`## Unmatched assignees` and `## Doc links`. Then the cuttable detail: `## By person` (team),
`## Issues` (person) or `## Children` (epic), and `## Not started` (epics in scope with no work done or in
progress come first and are never cut, with a description excerpt and refs). The count JQL is given once with an
`{EPIC}` placeholder; substitute the epic key to link a count to its query, and add
`AND statusCategory = Done` or `AND statusCategory = "In Progress"` for the done and in-progress
counts. The last line,
`Omitted: …`, counts everything cut to fit the word limit.

**Exit codes:**
- **0** — continue with step 2 of the order above.
- **1** (`acli unavailable: …` or `cannot read --config …`) — dispatch Agent A as the fallback.
  When the message is `cannot read --config`, the roster table is the problem and Agent A needs the
  same roster, so fix the team config first.
- **2 with `INCOMPLETE: <keys>`** — continue with step 2 of the order above, and report those
  keys as "not measured", the same rule as Step 2b's `INCOMPLETE`. Both files are still written,
  with the failed keys under `## Not measured`, so `--jira-map .updates/jira.json` still works. The
  acli error for each key is printed before the `INCOMPLETE` line. Only a rate limit (`429`) is
  worth a rerun first, with a lower `--workers` (default 8).
- **2 with a `usage:` message** — a required flag is missing. Fix the command and rerun.

## Step 2b: Scan PRs Against the Tracker

Run it after Step 2a, alongside the agents — on the fallback path, after Agent A returns. This is
the other named exception to "the orchestrator never queries data sources directly" —
`pr_scan.py` is a deterministic script, not a query into context, so it is mechanical and must not
go to an agent:

```bash
python3 <skill-dir>/scripts/pr_scan.py \
  --org ORG --repos repo-one,repo-two \
  --since {START_DATE} --until {END_DATE} --keys ABC,XYZ \
  --roster handle-one,handle-two,handle-three \
  --frontend-repos repo-one --backend-repos repo-two \
  --jira-map .updates/jira.json \
  --stale-days 3 --out .updates
```

`<skill-dir>` is this skill's base directory (the directory containing this `SKILL.md`).

- `--frontend-repos` and `--backend-repos` come from the team config's Frontend repos and Backend
  repos lines.
- `--jira-map .updates/jira.json` goes in when Step 2a exited 0, or when Agent A wrote `jira.json`
  on the fallback path. It rolls each PR up to its epic through the child-to-epic map. Without it,
  `## By epic` rows are keyed by the ticket key on the PR, not by epic: say so, and never present
  those rows as epics. An unreadable map makes the script exit 1 with `cannot read --jira-map`.

**`--roster` is not optional.** Repos are shared with other teams, so without it every count is
repo-wide and overstates the team's output, measured at 83 repo-wide against 22 for the team in
one real week. Pass the roster's GitHub handles from the team config and read the `*_team`
counts (`merged_team`, `open_team`, `stale_unreviewed_team`, `no_ticket_team`,
`opened_in_window_team`), never the bare totals. See "Count the team, not the repo" in
[report-format.md](references/report-format.md).

It writes `prs.json` and `prs.md`. Every PR lands in exactly one bucket: `linked_in_scope`,
`linked_out_of_scope`, `no_ticket`, `declared_no_ticket`. `no_ticket` is the point of the step:
work the tracker cannot see, invisible to any tracker-only report. So is a PR approved months ago
and never merged while its ticket reads Done.

`prs.md` is the whole GitHub picture; there is no GitHub agent. Every row links its PR:
- `## Not on the board`, `## Another team's ticket`, `## Stale, unreviewed open PRs`
- `## Merged in window, in scope`, with Opened, Merged and Size columns
- `## Open PRs (team)`: every team PR open now, whatever its age, drafts included, oldest first,
  with its review state; an approved PR reads `approved, not merged`
- `## Frontend vs backend` and `## By epic` (merged and open, per side), when the stack flags or
  the map are given
- `## Closed without merging`, with the closed date

**If the script exits with `TRUNCATED`, do not proceed.** Narrow the window and rerun. `gh pr list`
caps results silently, and a partial scan makes the untracked section look complete while empty.

**If the script exits with `INCOMPLETE: <repos>`, stop the same way.** One or more repos could not
be read. Report those repos' counts as "not measured", never as zero — a silent gap reads as a
fact.

**The window bounds merged and closed PRs only.** Open PRs report as current state regardless of
`--since`, because a PR open five weeks is exactly what a status report should surface.

## Step 3: Synthesize Report

Read ONLY the digest files from `.updates/`. List them first:
```bash
ls .updates/
```

Tracker facts come from `jira.md` and PR facts from `prs.md`; never open `jira.json` or
`prs.json`. The judgment work an agent used to do on the tracker is yours now, from those two
digests:

- **Write each epic's one sentence yourself**, from the description excerpt in its `jira.md` block:
  what it delivers, for someone who has never opened the ticket, never a restatement of the title.
  An epic whose block has no description is a finding: report it as "no description on the
  ticket" rather than guessing.
- **Judge each question candidate.** `jira.md` lists the flagged items and active epics whose
  newest comment asks a question or @mentions someone, with nobody commenting after; other items'
  comments are not read. `Author → Name` says who the comment waits on.
  Decide which are real open questions; report each with who asked, who it was aimed at, the date
  and how long it has been silent. Drop the ones that are rhetorical or answered in another digest
  (a PR, a meeting).
- **Build What's Left from each epic block's open children in `jira.md` plus `prs.md`**: in-review
  PRs from `## Open PRs (team)` and `## By epic` first, then in-progress assigned items, then the
  next unblocked tickets. A block ending `+N more open` has more work than it lists; say so.
- **Give every item in `## High priority outside epics` a line** in the risk register or its
  person's card: a P0/P1 with no epic appears nowhere else in the digest.
- **Put every `Long-stalled` entry on that person's card** (key and days), and treat a **closed**
  line in `## Closed or ongoing epics` that still has open children as a tracker-hygiene finding.
  An ongoing line with open children is not one: ongoing epics exist to hold open work.
- **Read `## Unassigned` for the At Risk rule** "a customer-facing defect is unassigned": an
  unassigned bug there is the evidence. Unassigned In Progress work is a finding too: name it.

On the fallback path, `jira.md` is Agent A's digest and has none of these headings: take the same
facts from Agent A's own sections, and say in the report whether unassigned work was checked.

Follow the format in [references/report-format.md](references/report-format.md). Key rules:

- **Lead with the headline.** One sentence: are we on track or not?
- **Quantitative epic progress.** Report exact completion percentage (`Done / Total` non-cancelled issues) for every active epic. If `Total == 0`, report `N/A`.
- **Explain exactly why when flagged.** If an epic, initiative, or teammate is rated *Needs Attention*, *At Risk*, or *Blocked*, explicitly detail **exactly why** (specific root cause, dependency, failure mode, or idle duration). If tracker commentary is silent, use empirical fallback (e.g. "No commits/transitions for N days").
- **What's Left (TL;DR).** Every active epic must include a 2–4 bullet list of remaining tasks and PRs required to reach 100%, prioritized by in-review PRs and active assigned tasks.
- **Link the blocker, not just the blocked thing.** When something is blocked, write
  **"Blocked by:"** and link the specific open question, PR, decision ticket or dependency. When the
  blocker is a person who has not answered, name who asked whom, what, and how long ago, stated as
  fact, never as blame.
- **Every assessment carries a ticket key plus a date or a count.** "Slipping" is not a citation;
  "1/14 stories done, epic opened 09-01" is. No citation, no rating.
- **Report what you could not measure as "not measured", never as zero.** A silent gap reads as a
  fact.
- **Count the team, not the repo.** PR numbers come from `pr_scan.py`'s `*_team` counts.
- **Brevity over completeness.** Skip anything that's fine. Highlight what needs attention.
- **Name names.** "<person> has 2 PRs awaiting review for 4 days" not "some PRs are stale."
- **Assessments are required.** For each person and each project/epic, give a 1-line assessment.
- **Match the detail to the scope.** Team and multi-initiative reports stay short: one line
  per epic for the frontend/backend split, no per-PR tables. Single-person and single-epic
  reports add section 02b (per-PR tables, remaining work described). See "Detail Depends on
  Scope" in `references/report-format.md`.
- **Say what each epic is.** Every epic card carries one sentence from its description, its
  priority and its linked parent initiative.
- **Date every PR.** PR tables carry Opened and Merged columns.
- **Link every count, person and repo** as well as every ticket and PR: a count links to the
  query that produced it.
- **Link 100% of tickets and PRs.** Every single ticket key (e.g. ABC-123) and PR reference (e.g. #xxxx) must be hyperlinked across all surfaces (card titles, card metadata, bullet text, table titles, action items, why callouts, what's left lists, and person cards) — no plain-text references where a reader would have to manually search.
- **No filler.** No "here's what I found" or "let me summarize." Just the report.
- **Meeting context enriches, not replaces.** Use meeting data to add color (action items, decisions, sentiment) to tracker and GitHub findings. **Do not quote transcripts and do not name the meeting tool in the report** — say "on a call". Don't create a separate "meetings" section for team-wide reports — weave it into the person's assessment. In a 1:1, meeting context goes into Talking Points; there is no separate Meetings section.
- **Deduplicate across sources.** If Jira and GitHub both reference the same work, merge into one mention.

## Step 4: Deliver

**Reports hold per-person data, so they default to a directory outside any git repository** — for
example `~/team-pulse-reports/` — rather than the current working tree. Create it if it does not
exist, and use it for both files below unless the user names another location.

**Pick the templates by scope.** **Any single-person scope** ("prep me for 1:1 with <person>" and
"how is <person> doing" alike) uses `assets/template-1on1.md` and `assets/template-1on1.html`, saved as
`team-pulse-<person-slug>-<END_DATE>.md` / `.html`, where `<person-slug>` is the person's first name or
GitHub handle, lowercased, with only `a-z`, `0-9` and `-`. Every other scope uses `assets/template.md`
and `assets/template.html`, saved as `team-pulse-<END_DATE>.md` / `.html`.
Fill the template; never copy an earlier report as the starting point. `<report-file>` below is the
chosen name.

1. Output the scannable markdown report directly in chat.
2. Save a publication-grade markdown document (`<report-file>.md`) using the bundled template for the scope. A team or epic report (`assets/template.md`) is ready to paste into GitHub issues, Jira tickets, or Confluence docs, with 100% hyperlinked keys, Unicode progress meters, and callouts. **A single-person report (`assets/template-1on1.md`) is a private local file:** it holds ratings, talking points and meeting context about one person, so never post it to a tracker, wiki, issue, chat channel or repository.
3. Generate a publication-grade standalone HTML dashboard (`<report-file>.html`) using the bundled template for the scope (`assets/template.html` or `assets/template-1on1.html`) (zero external CDN or font dependencies, offline-safe, matching the design system in [references/report-format.md](references/report-format.md)).
4. Open the dashboard in default browser (macOS / Linux, safe in headless):
```bash
open "<report-dir>/<report-file>.html" 2>/dev/null || xdg-open "<report-dir>/<report-file>.html" 2>/dev/null || true
```
5. Always print the clickable local file links in the chat response: `file://<report-dir>/<report-file>.md` and `file://<report-dir>/<report-file>.html`.
6. If the user asked for Confluence or Slack format, adapt chat output accordingly.
7. Clean up intermediates:
```bash
rm -rf .updates
```

## Scoping Variations

| User Says | Scope To |
|-----------|----------|
| "team pulse" | Full team from the team config, all active work |
| "team pulse on <project>" | Team members working on that project only |
| "how is <person> doing" | Single person across all their work |
| "pulse on ABC-123" | Single initiative/epic and everyone assigned |
| "what did we ship this week" | Merged PRs + completed Jira issues only |
| "prep me for 1:1 with <person>" | Single person, full depth: Steps 2a and 2b plus agents C, E and F; statistics for the window only (default: the past week); epic progress covers the whole epic; wins, reviews given, talking points and questions; laid out by `assets/template-1on1.md` / `.html` |

## Assessment Scale

| Rating | Meaning |
|--------|---------|
| On Track | Progressing as expected, no concerns |
| Needs Attention | Minor risk, slipping, or blocked but recoverable |
| At Risk | Significant blocker, timeline threat, or capacity issue |
| Blocked | Cannot proceed without external input/decision |

Assign by these conditions, in order; first match wins. Decisions outrank technical symptoms: an
EM's red items are usually unmade decisions, and saying so tells the reader the team is waiting on
a person, not stuck on code.

**The threshold for an open decision is 5 working days with no answer.** Past that, the item is
Blocked if it cannot proceed without that decision, and At Risk if work can continue around it.

- **Blocked** (cannot proceed without external input/decision):
  - a decision has been open past the 5-working-day threshold with no answer, and no other work on
    the item can proceed until it is answered
  - every child of the epic is Blocked
  - work is In Progress and assigned to a **deactivated account**; check the assignee's `active`
    flag, not just the name
- **At Risk** (significant blocker, timeline threat, or capacity issue, but work can continue):
  - a decision has been open past the 5-working-day threshold with no answer, but other work on the
    item is still moving
  - a customer-facing defect is unassigned
  - an epic is marked Done but its acceptance criteria do not hold, or its PR never merged
- **Needs Attention:**
  - real progress, but a load-bearing question is unanswered
  - under 25% complete against a committed date
  - an open PR that is `stale_unreviewed` (stale, with nobody reviewing it — see
    [report-format.md](references/report-format.md)); an approved-but-unmerged PR is reported
    separately and does not by itself trigger this
  - scope cut without written rationale
  - stalled: no status change in the window and under 25% of children done
- **On Track:** work merged in the window and no open decision. An initiative with no activity and
  no open decision is On Track, not absent. Say "no movement in N weeks" in its line.

Always show the word as well as the colour. Projectors shift hue, and about 1 in 12 men cannot
separate red from green.

## Anti-Patterns

- Do NOT query data sources directly from the orchestrator, except Steps 2a and 2b
  (`jira_scan.py`, `pr_scan.py`) — deterministic scripts, not queries into context. Everything
  else goes through sub-agents.
- Do NOT fan out per day by reflex, including for 1:1s. One agent per source, covering the whole
  window, is always the default. Per-day fan-out, for agents A and C only, is a deliberate choice made
  only when the user explicitly asks for it. See
  [references/batching-rationale.md](references/batching-rationale.md).
- Prefer agent definitions with a restricted `tools:` grant where you have them; without them,
  dispatch anyway and accept the higher per-agent cost.
- Do NOT read large tool results in the orchestrator. Dispatch a sub-agent to summarize.
- Do NOT dump raw tracker, GitHub or meeting data. Synthesize.
- Do NOT include tickets that are Done unless user asks "what did we ship."
- Do NOT assess people you have no data on. Say "no activity in window" instead.
- Do NOT editorialize beyond the data. Assessments must cite specific evidence.
- Do NOT use more than 3 sentences for any single person's section in a team report. A 1:1 has no person section; keep each talking point to 2 sentences.
