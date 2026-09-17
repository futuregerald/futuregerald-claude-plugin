# Team Pulse Report Format

## Contents

- [Structure](#structure)
- [Every epic gets a description](#every-epic-gets-a-description)
- [Traffic lights](#traffic-lights)
- [Link the blocker, always](#link-the-blocker-always)
- [Not started](#not-started)
- [PRs in flight](#prs-in-flight)
- [The citation rule](#the-citation-rule)
- [Length](#length)

## Structure

```
# Team Pulse: {scope} ({date range})

**Headline:** {one sentence — on track, at risk, or blocked, and why}

## Active Work

**{Epic name}** ({KEY}) — {LIGHT} · {done}/{total}
*{One plain-language sentence: what this work actually is.}*
- {progress this window, with keys and dates}
- **Blocked by:** {the specific thing, linked}

## Not started
{Epics nobody has begun, with descriptions and how long they have sat.}

## PRs in Flight
| PR | Author | Ticket | Age | Review |

## People
**{Name}** — {LIGHT}
{1-3 sentences. What they did, what is in their way.}

## Risks & Blockers
{Numbered. Each names what breaks, who owns it, and links the blocker.}

## Bottom Line
{2-3 sentences: overall health, and the highest-value next action.}
```

## Every epic gets a description

**Each epic named anywhere in the report carries one plain-language sentence saying what the
work actually is** — written for someone who has never opened the ticket. It comes from the
epic's `description` field, not from its title.

This is not garnish. The report is read by people who do not live in the tracker, and a line
reading "ABC-2091 — AMBER — 3/13" tells them nothing they can act on. The reader has to know
what the thing *is* before a light about it means anything.

A restatement of the title is not a description:

- ✗ "Detection History UI — the Detection History user interface"
- ✓ "Shows a customer every time a scanner has re-detected a finding, so they can see whether an issue keeps coming back"

When a ticket's `description` field is empty, say so — "no description on the ticket" is a
finding, not a blank to fill with a guess. An epic nobody can pick up as written is a real
problem worth surfacing.

## Traffic lights

RED / AMBER / GREEN, on every epic and every person. Assign by these conditions, in order;
first match wins.

**RED**
- A decision has been open past the threshold with no answer
- Every child of the epic is Blocked
- A customer-facing defect is unassigned
- An epic is marked Done but its acceptance criteria do not hold
- An epic is marked Done but its pull request was never merged
- Work is In Progress and assigned to a **deactivated account** — check the assignee's
  `active` flag, not just the name. A departed owner reads as "someone has this" on every
  board view, which is why it can sit for months without anyone noticing.

**AMBER**
- Real progress, but a load-bearing question is unanswered
- Under 25% complete against a committed date
- An open PR has gone past the stale threshold with nobody reviewing it
- Work is being descoped without written rationale
- Stalled: no status change in the window and under 25% of children done

**GREEN**
- Work merged in the window and no open decision

An initiative with no activity and no open decision is **GREEN**, not absent. Silence is a
finding: say "no movement in N weeks" in the line.

### Why RED leads with decisions

For an engineering manager's report the red items are usually unmade decisions rather than
technical blockers, and that is the most useful thing the report can say: the team is not stuck
on code, it is stuck waiting on a person. Order the RED conditions so a decision outranks a
technical symptom.

## Link the blocker, always

**Whenever the report says something is blocked, it links to the thing doing the blocking.**
Not just the blocked ticket — the blocker itself: the open question, the PR, the decision
ticket, the dependency, the person's name.

- ✗ "ABC-2764 is blocked on a product decision"
- ✓ "**Blocked by:** [Open Question #1 on ABC-2764](url) — does the `regressed` filter ship
  empty for V1? Holds [ABC-2769](url) and [ABC-2766](url) from merging."

A blocker the reader cannot click is a blocker they have to go hunting for, and in a meeting
nobody hunts. Use the phrase **"Blocked by:"** so the bullet is scannable, and name what
would unblock it where that is knowable.

### When the blocker is a person, name the person

The most actionable blocker in a status report is usually not a ticket — it is someone who has
been asked a question and has not answered. **Read the comment threads on blocked and stalled
items** and report it as: who asked, who they asked, what they asked, and how long ago.

- ✗ "XYZ-195 is open and unassigned"
- ✓ "**Blocked by: Joe Brinkley** — Eugene Revzin asked him on 08-27 whether the frequency
  discount applies test-by-test or averages evenly. No reply in 20 days."

The second version turns a stale ticket into one conversation someone can have today. The
first leaves the reader knowing something is stuck and not who can unstick it.

State it factually — who is waiting on whom, with dates. Never as blame; the person may not
know they are the blocker, and the report is read in front of people.

### Check the parent, not just the children

A parent that reads In Progress while its children are Won't Do or untouched for months is an
abandoned plan that nobody updated upward. It looks healthy on every board view. Flag the
contradiction with dates on both levels.

## Not started

Epics in scope that nobody has begun — Backlog or To Do with zero children done. Give each its
description and how long it has sat.

These are invisible to any activity-based query, so they have to be asked for separately or
they never appear at all. Leaving them out makes a program look smaller and healthier than it
is, and the unstarted pile is frequently the real story.

## Count the team, not the repo

**Every number in this report is about the team, not about the repositories they work in.**

Repos are shared. Scanning three repos for a week returns every PR any team merged there, and
reporting that total as the team's output is simply wrong — in one real run it was 83 repo-wide
against **22** for the team, a 4x overstatement made of other teams' work.

Always pass `--roster` to `pr_scan.py` with the team's GitHub handles, and read the
`*_team` counts:

| Use | Not |
|-----|-----|
| `counts.merged_team` | `counts.merged` |
| `counts.open_team` | `counts.open` |
| `counts.stale_team` | `counts.stale` |
| `counts.no_ticket_team` | `counts.no_ticket` |

The repo-wide figures stay in the JSON, and are worth one line of context — "the team merged 22;
60 further PRs in these repos this week belong to other teams" — but they are never the headline.

**Which filter for which question.** A PR counts as the team's when its author is on the roster
**or** its ticket key is in scope. Those answer different questions, so pick deliberately:

- **"Who on my team has a stale PR?"** → filter by **author**. A PR by another team that cites
  one of your tickets is not your engineer's stale PR.
- **"What work on our epics is in flight?"** → filter by **key**, which catches another team
  contributing to your ticket.

When the two disagree, say which you used. The same applies to the tracker side: the epics are
the team's epics, so the PR numbers beside them must be the team's PRs.

## PRs in flight

From `scripts/pr_scan.py`. Three things belong in the report:

**Stale PRs** — open past the threshold, bot PRs excluded. Oldest first. A PR approved months
ago and never merged is the single most common thing a tracker-only report misses.

**No ticket at all** — `bucket: no_ticket`, human authors only. This is work the tracker cannot
see. Bot PRs are counted, never listed: dependency bots open enough untracked PRs to outnumber
the human ones several times over, and listing them buries the one that matters.

**Another team's key** — `bucket: linked_out_of_scope`. Usually legitimate cross-team work in
shared repos; occasionally it is this team's work filed in the wrong place.

Frame untracked work as a question, not an accusation — "should this be tracked?" The author
may have a good reason, and the report is read in front of people.

## The citation rule

**Every light and every claim carries a ticket key plus a date or a count.**

"Slipping" is not a citation. "1/14 stories done, epic opened 09-01" is. "Stalled" is not a
citation. "Filed 08-26, not moved since" is.

If the evidence cannot be named, the sentence does not belong in a document someone will read
aloud to leadership.

**Say what you could not measure.** When a source came back empty for a reason you do not
trust — a `gh` qualifier that is unreliable, an API that returned nothing — report it as "not
measured", never as zero. A silent gap reads as a fact.

## Length

| Scope | Target |
|-------|--------|
| Full team, all programs | 900-1400 words |
| Single program | 500-800 words |
| Single person (1:1 prep) | 150-250 words |
| "What did we ship" | 100-200 words |

## Voice

- Plain language. The ordinary word when it says the same thing.
- Lead with the answer. Reasoning after, and only as much as changes what the reader does.
- Name what goes wrong, not the category it belongs to.
- Describe gaps in anyone's work factually, never as blame. "No written rationale on the
  ticket" — not "nobody bothered to explain".
- No emoji beyond the traffic lights. No filler openers.
