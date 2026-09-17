# Agent prompts

Two files go into `research/` at the start of Phase 1 and every agent reads both. Keeping the shared
material in files rather than in each prompt is what makes it affordable to run 8–10 agents and what
makes their outputs comparable enough to assemble.

## `SHARED_BRIEF.md`

Adapt the bracketed parts to the team and systems in play.

```markdown
# Shared brief — <initiative> work-status + estimation research

Today is <date>. <One line of context: the goal and its target date.>

## Systems and access
- **Jira**: Atlassian MCP is connected. cloudId = `<uuid>`, site <site>.
  Issue URL form: https://<site>/browse/<KEY>-123
  Load the tools first, in one call:
  `ToolSearch("select:mcp__atlassian__searchJiraIssuesUsingJql,mcp__atlassian__getJiraIssue", 5)`
  Hierarchy: Initiative (level 2) > Epic (level 1) > Story/Task/Bug/Spike (level 0).
  Story points live on a custom field — pull `fields: ["*all"]` on ONE recently-closed story to find
  the id, then reuse it.
  **Jira MCP results are often too large.** A call that exceeds the token cap saves JSON to a file —
  parse that file with `python3`/`jq`, never read it whole. Request only the fields you need.
- **GitHub**: `gh` is authenticated for org `<org>`. Repos: <list, with one-line roles>.
  `gh search prs --owner <org> "<KEY>" --limit 50 --json number,title,state,repository,author,createdAt,url`
  Branch convention `<type>/<TICKET>/<summary>` means ticket keys appear in branches, PR titles, commits.

## The team whose capacity we are estimating
<Table: Name | GitHub handle | Role.>
So: **N ICs**. Other teams own their own rows and are NOT this team's capacity — but they are
frequently *dependencies*, and saying which is which is part of the job.

## Evidence rules
- **Never state a status from the source document — verify it in the tracker.** Known-stale examples
  from this dataset: <list 2–3 real ones found in Phase 1>. Expect more.
- Every claim gets a citation: a ticket key, a PR URL, a commit sha, or a file path.
- Distinguish **"no ticket found"** from **"work not needed"**, and state the query you ran.
- If you cannot verify something, write **UNVERIFIED** and say what you would need. Never guess.
- **Read the full comment thread on every item** — request `comment` explicitly, it is not a default
  field. Comments carry status updates, decisions and unanswered questions that never move a status
  field, so an item can be active while its counts sit still. Report the update cadence, every
  unanswered question with its age, and any comment that contradicts the item's own description.
- Count things. "Six of nine children Done, 21 points remaining" beats "mostly done".

## Output
Write to the path in your task prompt, as Markdown, using `ITEM_TEMPLATE.md`.
Also return a condensed version (<= 1200 words) as your final message.
```

## `ITEM_TEMPLATE.md`

```markdown
## <Item name> — `<primary key(s)>`

**Source document says:** <quoted status and note>
**Tracker says:** status, type, assignee, parent, last updated, resolution
**Verdict:** ACCURATE | STALE | WRONG — and precisely what differs

### Scope decomposition
Child table: key | type | summary | status | assignee | points | last updated.
Then: X of Y children Done, N points closed, M remaining, K children with no estimate.
No children? Say so and describe the real scope from the description.

### Code evidence
Merged PRs (url, title, merged date, +/-, author), grouped by repo.
Open PRs (url, age in days, draft?, review state, blocking comments quoted).
Nothing found? Name the `gh search` queries you ran.

### Comment thread (read ALL of it — request `comment` explicitly, it is not a default field)
- **Status updates**: author, date, verbatim quote, newest first. Note the **cadence** — weekly,
  sporadic, or stopped, and when it stopped.
- **Unanswered questions**: every question with no reply below it — asker, who it was aimed at, days
  silent, and what it blocks. *Decision debt with no ticket; rank by age × consequence.*
- **Decisions recorded here**, especially any that **contradict the item's own description or status
  field**. Quote both sides — a stale description the roadmap reads from is a finding.
- **Committed dates** anyone named ("end of sprint", "~2 weeks", a due date). Usually the only real
  dates in the system.
- If there are **no comments at all** on an actively-worked item, say so plainly: status is living
  outside the tracker and the roadmap is guessing.

### Signals of life — ACTIVE / WAITING / DORMANT
Judge on more than closed children: recent comments and who wrote them · children *created* recently ·
open PRs and branches · design docs or Figma links appearing · status transitions that did not reach
Done. State whether the item is **active in a way its ticket fields do not show** — an epic with no
closures but a PR in review and a status update from yesterday is ACTIVE, and looks identical to a
dormant one in a child-count table.

### Remaining work
Bulleted, specific, evidence-derived. **Flag undefined scope separately** — work nobody has written
down is the single biggest driver of the pessimistic bound.

### Dependencies and blockers
Upstream (with team name if external) · downstream · undecided decisions and who owns them.

### Parallelisability
Name the actual seams. State `max useful engineers` (1, 2, more) and why.

### Risk notes
Specifics only — what could make this much longer than it looks.
```

Then a batch summary: a status/%-complete/max-engineers table, an UNVERIFIED list, and open questions.

## Writing a good cluster prompt

The shared files carry the method. The cluster prompt carries the *suspicion*. A prompt that just
says "research these three epics" gets you a tidy restatement of Jira. What earns its cost is naming
what you already doubt:

- **Hand over the verified baseline** for the cluster's items and say "do not re-derive, build on it".
- **Quote the source document verbatim**, including its optimistic notes, so the agent has something
  concrete to contradict. "Sep 9 note says *finishing next week*" invites a test; a summary does not.
- **Ask the specific questions this cluster raises**, and say why each matters. "Why was ABC-260
  closed Won't Do right before the keystone epic ships — smart descope, or a deferred bill?" is a
  question an agent can answer well. "Investigate ABC-260" is not.
- **Point at non-obvious evidence**: local design docs, a sibling repo, a Datadog dashboard, an
  adjacent tracker project. Agents rarely find these on their own.
- **Pre-empt double-counting.** When clusters touch, say so: "another agent covers X in depth —
  confirm and defer in two lines."
- **Say what to lead the summary with.** You are assembling ten reports; ordering their front
  matter for you is free.

Cluster of 2–4 items, by dependency and subject. One agent per cluster. Dispatch every agent —
velocity, WIP, and all clusters — in a single message so they run concurrently.
