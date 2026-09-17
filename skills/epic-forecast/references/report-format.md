# Report format

Two files, same numbers, generated from one source so they cannot drift: a Markdown report and a
single HTML page. Nothing else — no spreadsheet, no manifest, no diagram.

## One template, sections driven by the question

**The question table is in `SKILL.md`** — which phases each question needs. It is not repeated here;
one copy cannot drift from itself.

What this file adds is what each question does to the *report*: **conditional sections, not
different templates.** Every section below is marked with the question that turns it on. Two change
shape rather than appearing or disappearing: §3 (grouped by milestone when a window exists, by
dependency order otherwise) and §4's out-of-horizon block (which exists only with a window).

**The report header names the question it answered and lists the sections deliberately omitted**, so
nobody mistakes a skipped phase for an oversight — and so a reader who wanted a different question
knows to ask again.

## The Markdown report

Order matters — it is built for someone who will read the first screen and skim the rest.

### 1. What changed since the source document
**Lead with this.** A table of every discrepancy found: item, what the document said, what the
tracker says, why it matters. This is the part the reader cannot get anywhere else, and it is the
strongest evidence that the rest of the report was actually verified.

### 2. Headline
Three to six sentences. The bottom line, the one or two items that decide it, and the biggest risk.
In window mode that bottom line is a date; in scope mode it is the critical chain length and what
sits on it. No preamble.

### 3. TL;DR per item
2–4 sentences each, scannable. Grouped by milestone in window mode, by dependency order in scope
mode. Each carries: real status, what is genuinely left, the estimate range, and the one thing that
would change it. A reader who stops here should still be able to run the meeting.

### 4. Estimates — a table, not prose

**The report needs its own estimates table.** Numbers scattered through the TL;DR prose do not
satisfy a request for estimates: prose carries the one-engineer figure at best, and silently drops
the 2-engineer case, effort level, parallelisability and readiness. Generate the table from the same
manifest the workbook uses so the two cannot drift.

Columns: item · key · owner · **Done so far** · **Remaining (1 eng opt/pess)** · **TOTAL** ·
**Remaining (2 eng opt/pess)** · confidence.

**Always show Done, Remaining and TOTAL together, with `TOTAL = Done + Remaining`.** Remaining is the
headline for an epic in flight, but Remaining alone hides how much has already been spent: an epic
reading "0.1 – 0.4 weeks left" has very different meaning at 2.7 weeks spent than at zero. It also
exposes the rows with **Done = 0 and a large total** — the unstarted work that is easiest to
under-plan, and on a real run the largest of those turned out to be the critical chain.

Two things to state under it, because readers get both wrong:

- **The estimates do not sum.** Items run in parallel across N engineers, so the date is set by the
  longest *chain*, not the total. Cite the computed chain from section 6 — length and path — rather
  than naming a long pole in prose.
- **Which rows are out of scope for the current horizon.** *Window mode only:* work that falls
  outside the window belongs in its own clearly separated block that contributes nothing to the
  arithmetic — but keep it, sized, because "what's after this" is a real planning question. Deleting
  it is as wrong as counting it. **In scope mode there is no horizon to fall outside, so this block
  does not exist**; group by dependency order instead and do not invent an in/out split.

**Put the headline numbers on the landing view too.** Whatever sheet or section opens first must carry
Done / Remaining / TOTAL. A reader who opens the artefact, sees no numbers, and concludes there are no
estimates is not being careless — the artefact misled them.

### 4a. How these numbers were calculated — **always, and never as an appendix**

A reader who cannot see the arithmetic cannot disagree with one input without discarding the whole
estimate. Two short subsections, both with real numbers from this run:

**Effort.** The formula (`effective_rate = base_rate × size_factor × readiness_factor`), the base
rate **with its provenance** — measured, caller-supplied, or the default — and the size and
readiness tables as they were actually applied. Then **one worked example**, on the item whose number
is most surprising, showing the division line by line. Where a factor changed an item materially, say
what it would have been under the other factor.

**Engineering productivity.** How a rate was arrived at: the week classification (delivery /
planning / no-signal), what each bucket counted, the per-engineer ledgers, which rates were too thin
to use and what was used instead. State plainly that no week was attributed to time off. If review
latency is the floor on any item, show that arithmetic too.

Per item, print the division itself next to the estimate — `14 tickets ÷ (6 × 1.0 M × 1.4
prototyped) = 8.4/wk` — so the number is checkable where it appears rather than only in the method
section.

### 4b. Projected landing — **whenever a date is wanted**

The critical chain in engineer-weeks, divided by the engineers actually on it, from today. Give the
optimistic and pessimistic landing as a range, name the gates it is conditional on, and label it a
**projection** — what the measured rates imply — never a target or a commitment.

An unassigned epic on the chain has no divisor: report *"N weeks once someone picks it up"*, not a
date. State the assumptions: current staffing, no new blockers, gates clearing when asked.

*When a window was also supplied:* say whether the projection falls inside it. That comparison is
what a planning conversation turns on, and it is the bridge between "when does it land" and "does it
fit".

### 5. Capacity — **only when asked whether it fits**

Per `references/capacity-model.md`. Show the arithmetic, not just the result: productive weeks,
active engineers, carryover spill at both bounds, the measured unplanned rate with its method, and
the resulting band.

State the demand-versus-capacity ratio with **its denominator named** — both ratios divide by the
optimistic capacity — and state, every time, that the total is not a date.

If a human input is missing — non-working weeks, or which engineers the prior commitment draws —
**omit the section and say it was omitted for want of that input.** An omitted capacity section is
honest. A fabricated one is load-bearing and wrong.

`active_engineers` reports a count and a reason, **never a name**: `5 (6 on the roster; one removed —
see the EM)`. Absence of delivery signal is never reported as time off, here or anywhere else — it is
equally consistent with leave, a secondment, incident work, or a source this forecast did not query.
The removal is EM-confirmed before it is published; if it could not be confirmed, both figures are
published as a band and the removal is labelled unconfirmed. See `capacity-model.md`.

### 6. What blocks what — **always**

Three or four sentences, not a picture: what blocks what, the longest chain and its length, and what
is parallel and therefore safe to start. Name every non-ticket dependency — review queues, product
and legal decisions, another team's service — with its owner.

**No diagram.** A dozen-epic dependency picture is decoration; the three sentences carry the whole
decision.

### 7. Split suggestions — **both modes**

Where an epic could be cut, and **what the cut buys**. "This is too big" with no proposal attached is
an observation, not a recommendation. See `references/estimation-model.md` §8 for the candidate
triggers, the preference order for where to cut, and the arithmetic rules.

Each proposal carries a before/after table — one-engineer weeks per piece and in total, max useful
engineers, critical-chain contribution, earliest independently shippable piece, coordination cost —
and a sentence naming which trigger flagged the epic.

**Splitting is a proposal, never an action.** The skill does not create or restructure tickets.

With a single item in scope, this is often the most useful section in the whole report.

### 8. Commit tiers — **window mode only**

What the team should actually commit to, given the capacity in section 5. Four tiers, in order:

| Tier | What lands here |
|---|---|
| **Commit** | Ready now: scoped, ticketed, underway or unblocked, and it fits |
| **Commit, descoped** | Only the ready slice ships. **Name what is cut**, and where the remainder goes |
| **Spike only** | No design or no acceptance criteria, or it needs a capability that does not exist. **Name the question the spike answers** — a time-boxed spike is the only defensible commitment |
| **Hold, pending a decision** | Not blocked on engineering at all. **Name the decision and its owner** |

Table columns: **Tier | Items | Cost (eng-weeks, opt–pess) | Why**.

Reconcile against capacity underneath, in one line of the shape: *"Commit plus descoped plus spikes
totals X optimistic, Y pessimistic, against a capacity band of A–B eng-weeks."* Report the Hold
tier's cost **separately and outside that total** — it is not committed, and folding it in makes the
commitment look bigger than it is.

Two things must be said plainly:

- **This is a judgement the skill proposes and a human confirms.** The real tiering rested on calls
  like "the customer-facing send needs a product decision first", which no agent derives from a
  tracker. Present it as a proposal with its arithmetic.
- **A descope is often a readiness cut, not a capacity cut** — the item is not too big, it is not
  ready. Say which, because they point at different fixes.

Every low-confidence row carries `(placeholder — needs scoping)` inside the cell. A range is not a
commitment.

### 9. Who is working on what
Engineer → item table, plus who is unallocated and who is overloaded. Then competing non-initiative
load, because that is the honest reason forecasts slip.

The per-engineer rate table carries its **source** and its **week ledger**:
Engineer | Epic | Observed rate | **Source** | Delivery / planning / no-signal weeks | **Rate used in
this forecast**.

A rate without its denominator is not reportable — it is the difference between "slow" and "was not
doing delivery work that week". See `estimation-model.md` §2f.

**Print the purpose limitation directly under the table, every time**, in the words given in
`estimation-model.md` §2g: these are estimation inputs measured over a short window from partial
sources, and they must not be used in performance review, calibration, ranking, or any staffing
decision about an individual.

**The forwarded artefact carries the spread and the band, not the per-person table.** Give the
per-person breakdown to the EM, or produce it on request. Do not sort the table by rate, and do not
describe anyone as fast or slow. A week with no delivery signal is listed as exactly that — never as
time off, which the tracker cannot support.

### 10. Sequencing recommendation
What to do in what order, and why — dependency order first (from the graph, not from intuition), then
the risk-reduction argument. Name what to start now, what to start next, and **what to explicitly not
start**, which is usually the most useful half.

*Window mode:* if the target cannot hold, say so here with the arithmetic behind it, rather than
burying it.

### 11. Detailed justification per item
The thorough section. Per item: scope decomposition with child counts, code evidence with PR links,
the estimate derivation showing its arithmetic **and which rate source it divided by**, dependencies,
parallelisability with the seams named, and risks. This is where a skeptical reader goes to check a
number, so show the working.

### 12. Findings from the adversarial review
What the review caught and what changed as a result. Short, but present — it tells the reader which
claims were stress-tested.

### 13. Open questions
Only what a human can answer: undecided product calls, staffing, priority conflicts, ownership
disputes. Each with who owns it and what it blocks. A question you could have researched belongs in
the research, not here.

**Fold in every unanswered question found in the trackers' comment threads**, with its age and who
asked it. Those are already-articulated decision debt — somebody cared enough to ask and nobody
replied — and they are invisible in every roadmap view because no field changes when a question goes
unanswered. Sort by age × consequence: an old unanswered question blocking a keystone item is usually
the cheapest schedule fix in the whole report, because clearing it costs somebody five minutes.

Every `decision` node in the dependency graph appears here too, with its owner. A decision that is on
the critical chain and not in this section has been lost.

### Linking — every reference is clickable, in both artefacts

**Every ticket key, PR number and file path in the Markdown report is a link.** A reader who has to
copy a key into a search box to check your claim will not check it, which defeats the point of citing
evidence at all. Link them as you write, then verify with a pass that strips complete `[text](url)`
links and greps the remainder for bare keys — a naive "is it linked" check will match the key
*inside* its own URL and report false clean.

Two regex traps worth pre-empting, both of which silently skip real references: a key followed by `)`
— as in "Search reindex (ABC-101)" — and a key after a `/`, as in "ABC-101 / ABC-102". Also protect
fenced and inline code from linkification.


### 14. Method and limits
How pace was measured, **which of the three rate sources was used and why**, the constants used, and
what the estimates assume. In window mode, the capacity inputs and which of them were human-supplied.
State the limits plainly — a report that admits what it does not know is the one people trust the
second time.

## The HTML page — a primary deliverable

**This is the artefact people actually read.** Generate it from the same Markdown — never hand-built,
never a second copy of the numbers, because the one in the room would be the stale one.

One page, scannable, in this order:

1. **Headline** — the bottom line in two sentences.
2. **The work** — one row per item: status, owner, what is left, remaining estimate, confidence.
3. **Engineers** — how many the work needs, how many are on it, and who is unallocated. A reader's
   first question is always "do we have enough people", and a report that makes them derive it has
   buried its most useful number.
4. **What is left, per item** — specific and short.
5. **Risks** — what could make each item much longer than it looks.
6. **Open decisions** — each with an owner and what it blocks.

No diagram.

## Handing it over

**Two deliverables: the report and the workbook.** Everything else — the shared brief, the baseline,
the per-agent findings, the manifest, the raw graph output — lives in `research/` and is named once,
as a directory, not file by file. A virtualenv belongs in the scratchpad and never in the output
directory. A caller who opens the folder should see the two things they asked for.

Give clickable file links for both artefacts and the research directory. Lead the chat response with
section 1 — what changed — then the headline, then the open questions. Everything else is in the
files; do not re-narrate it in chat.
