---
name: epic-forecast
description: Verify the true state of a set of epics, initiatives, or roadmap rows and forecast the remaining work with evidence-based optimistic/pessimistic estimates. Produces a Markdown report plus an Excel workbook covering per-item status, who is working on what, 1-engineer vs 2-engineer estimates, parallelisability, sequencing, and open questions. Use when the user asks to "verify the state of these epics", "estimate this initiative", "how long will this take", "status and estimates for the roadmap", "go through this tracker and tell me where things really are", "forecast GA", or hands over a status spreadsheet and asks what is real.
author: Gerald Onyango
---

# Epic Forecast

Take a set of epics, initiatives, or roadmap rows. Find out what is *actually* true about each one,
then forecast the rest against the team's *measured* pace. Deliver a Markdown report and an Excel
workbook.

The whole skill exists to defeat one failure mode: **a status document that reflects what people
said in a meeting rather than what the tracker and the code say.** Every status in the output is
verified independently, and every estimate is anchored to a number somebody can check.

## The five rules

1. **Never restate a status from the input document.** Verify it in the tracker. Status drift is
   the norm, not the exception — expect to find items marked "To Do" that are Done, items marked
   "In Development" whose epic was closed Won't Do, and spikes marked "Research" that shipped.
2. **Estimates are anchored to measured throughput**, never to intuition. Measure the team's pace
   first (Phase 2), then express every estimate as a multiple of it. An estimate with no stated
   basis is not an estimate.
3. **Undefined scope is the estimate.** When nobody has written down what an item is, the honest
   output is a range with "needs scoping" attached and a note that the range is a placeholder — not
   a confident number. Say which items those are, loudly.
4. **All research runs in sub-agents.** The orchestrator stays a verifier and an assembler. It
   spot-checks agent claims rather than absorbing every file the agents read.
5. **Adversarially review the agents' output before it reaches the report.** Sub-agents are
   confidently wrong in predictable ways (Phase 5). Their findings are input, not truth.
6. **The people you are reporting on know things the tracker does not.** An EM's own estimate, or
   "that dependency isn't a blocker", is evidence — usually better evidence than your derivation.
   Reconcile against it rather than defending the model, and use their numbers verbatim where given.

## Phase 1 — Scope and baseline

Establish what you are forecasting and get an independent baseline before any agent runs.

- **Resolve the scope.** The user gives you epics, initiatives, a spreadsheet, a Jira filter, or a
  label. If it is a spreadsheet, parse it with `openpyxl` (create a venv in the scratchpad if the
  module is missing) and dump every row with its row number, so citations are checkable.
  Ask which rows are in scope only if genuinely ambiguous — "the DL rows" is not ambiguous.
- **Pull the baseline yourself, in one bulk query.** For Jira:
  `key in (K1, K2, ...)` with `fields: ["summary","status","issuetype","assignee","updated","created","labels","resolution","parent"]`.
  Parse the saved JSON with python if it exceeds the token cap. Keep the resulting table — it is
  what you check the agents against in Phase 5.
- **Diff the baseline against the input document immediately** and note every discrepancy. These are
  usually the most valuable findings in the whole exercise and they cost one query.
- Create the output directory (`<repo-or-home>/<project>-status-<date>/` with a `research/`
  subdirectory) and write the two shared files below into it.

Write `research/SHARED_BRIEF.md` and `research/ITEM_TEMPLATE.md` — see
`references/agent-prompts.md` for both, and adapt the roster/systems sections to this team.
Every agent reads them, which keeps the individual prompts short and the outputs comparable.

## Phase 2 — Measure the pace (this agent runs first and matters most)

Dispatch a **velocity agent**. Everything downstream is calibrated against what it returns, so it
gets the most specific prompt. It must produce, with the query behind each number:

- **Children closed per week on each in-scope epic, by its actual owner, over the last 2–4 weeks.**
  This is the number every estimate is built from — not a team average. Expect a 5–10× spread, and
  expect a team-wide mean to understate the healthy engineers badly, because it is dragged down by
  epics that are stalled on dependencies. See `references/estimation-model.md` §1a; getting this
  wrong is the method's biggest failure mode and it always errs slow.
- **Which epics are closing 0/week, and why.** A zero rate is a *cause to name*, never a slower
  duration to multiply out. Distinguish four: awaiting approval · author action (CI red / changes
  requested) · genuinely not started · **an artefact of batching**, where a big PR closes several
  tickets at once so resolution-date velocity reads zero while the work is reviewable.
- **The tickets-per-PR ratio, and every ticket key in open PR titles and bodies.** Titles here often
  carry several (`[KEY-1/KEY-2/KEY-3]`). Children already covered by an open PR are done-but-uncounted
  and come out of remaining scope. Never call an epic slow without reading its open PRs first.
- Issues resolved per week and story points per week, team-wide and per engineer, over a long
  window (~12–14 weeks) *and* a short one (~6 weeks) so a trend is visible. Use these for context
  and for unowned epics only.
- **What fraction of resolved issues carry an estimate at all.** If most do not, story points are
  not a planning unit for this team and issue count is — say so and switch units.
- **Median PR size (lines and files) per epic**, so a rate can be transferred between epics with a
  size adjustment rather than assumed to carry over unchanged.
- **Median elapsed calendar time for a comparable completed epic**, measured end to end on 3–5 real
  examples — a sanity check on the per-epic rates, not the primary input.
- PR lead time (open → merged, median and p75). Where an engineer's review latency *is* their
  closure rate, say so — that makes review the fix, not headcount.

Output is a labelled block of **planning constants**, led by the per-epic rate table. Refuse to
proceed to estimates on vibes if the data is thin; state the bound and the reason instead.

## Phase 3 — Map current WIP

Dispatch a **WIP agent** to answer "who is working on what, right now". It must separate genuinely
active work (In Progress / In Review, with a recent update) from parked backlog items, list each
engineer's open PRs with ages, identify who has **no** active work on this initiative, and quantify
**competing non-initiative load** — the usual reason forecasts slip. Stalled signals (PR open >7 days
unreviewed, ticket In Progress >14 days untouched) are findings, not noise.

## Phase 4 — Research the items in clusters

Group the items into **coherent clusters of 2–4 related items**, one agent each. Cluster by
dependency and subject matter, not alphabetically — an agent that holds a whole dependency chain
gives better answers than four agents each holding a fragment.

Each cluster prompt carries: the verified baseline for its items, the input document's claims quoted
verbatim (so the agent can contradict them), and **the specific questions that cluster raises**.
Generic prompts produce generic findings. Name the thing you suspect and ask them to check it.

**Every cluster agent must read each item's full comment thread**, not just its fields. Comments are
where status updates, decisions and unanswered questions live, and none of them moves a status field —
so an epic can be genuinely active while its child counts sit still. Comments are not returned by
default; they must be requested explicitly. The two things to come back with are the **cadence and
content of status updates** and **every question with no answer below it**, aged. An unanswered
question is decision debt with no ticket and is invisible to every roadmap view; it is routinely the
cheapest schedule fix available. Zero comments on an actively-worked epic is itself a finding.

Every agent follows `research/ITEM_TEMPLATE.md`, whose load-bearing sections are:
**Verdict on the source document** (ACCURATE / STALE / WRONG), **Signals of life** (ACTIVE / WAITING /
DORMANT, judged on more than closures), **Remaining work** with undefined scope flagged separately,
**Dependencies**, and **Parallelisability with the actual seams named**.

Run every cluster agent in parallel in one message, together with Phases 2 and 3.

## Phase 5 — Adversarially review the agent output

**Mandatory. Do not skip it, and do not let an agent's confident prose into the report unchecked.**

First, spot-check yourself against the Phase 1 baseline: pick the highest-stakes claims — anything
that changes a date, a status, or an owner — and verify them directly. A claim you carry into the
report is a claim you own.

Then dispatch a fresh reviewer over the research directory. Give it the baseline table and
`references/review-checklist.md`. It hunts the known failure modes:

- A status asserted without a citation, or citing the source document rather than the tracker.
- "No ticket exists" with no query stated — absence of evidence dressed as evidence.
- A percentage-complete with no basis (child counts? story points? PRs?).
- Two agents contradicting each other — they overlap at cluster boundaries, and the contradictions
  are where the real ambiguity lives.
- An estimate that ignores the Phase 2 constants, or an item whose "remaining work" list is thinner
  than its own dependency list.
- Optimism inherited from the source document instead of tested against it.

Fix every finding: re-query, send the agent back with `SendMessage`, or downgrade the claim to
UNVERIFIED in the report. Record what changed — the review's own findings are worth a short section.

## Phase 6 — Estimate

Read `references/estimation-model.md` and apply it. The order matters, because the two halves of an
estimate fail differently:

1. **Count what is really left** — strip moot children (a bug ticket against code a pending PR
   *deletes* is not work), apply the historical Won't Do rate, subtract anything already covered by an
   open PR, and count **descendants** not children. Errors here run to 3×.
2. **Get the rate right, then confirm it with the EM.** Measure per engineer, never plan at the bottom
   of the range, and treat a low rate caused by review latency as a queue to clear rather than a
   capacity to plan around. If the team's toolchain changed — AI coding tools especially — the
   historical baseline is a lagging indicator and review, not authoring, becomes the floor.
3. **Adjust for ticket size and scope readiness**, as visible columns rather than a fudged rate.
4. **Report Done / Remaining / TOTAL**, with `TOTAL = Done + Remaining` so they tie out.
5. **Bounds and the 2-engineer case** from the item's real seams — never by halving.

**If the EM says the estimates are wrong twice, stop adjusting individual numbers and ask for the
rate.** Two rounds of patching against the same feedback means the multiplier is wrong, not the
arithmetic, and a third guess costs more credibility than the question would have.

## Phase 7 — Deliver

Per `references/report-format.md`, produce:

1. **The Markdown report** — TL;DR per item up top (2–4 sentences each, scannable), the sequencing
   recommendation, then the thorough justifications, then open questions.
2. **The Excel workbook** — via `scripts/build_workbook.py`, which takes a JSON manifest. Sheets:
   Summary, Estimates, Current Assignments, Sequencing, Dependencies, Open Questions, Method.
3. **Clickable links for everything** — Jira and PR URLs, and a file link for each artefact.
4. **Open questions** — only what a human can answer. A question you could have researched is a
   research failure, not an open question.

Lead the chat response with what changed versus the source document. That is what the reader wants
first, and it is what they cannot get anywhere else.

## References

- `references/agent-prompts.md` — the shared brief, the per-item template, and cluster prompt patterns
- `references/estimation-model.md` — sizing, the two bounds, the 1-vs-2 engineer rule, parallelisability
- `references/review-checklist.md` — the adversarial review pass
- `references/report-format.md` — report structure and workbook schema
- `scripts/build_workbook.py` — JSON manifest → styled .xlsx
