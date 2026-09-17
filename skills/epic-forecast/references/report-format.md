# Report format

Two artefacts, same numbers. The Markdown carries the reasoning; the workbook carries the data
someone will sort, filter, and paste into a deck.

## Two modes, one template

The report has **conditional sections, not two templates.** Which sections appear depends on whether
the caller supplied a window.

| | **Window mode** | **Scope mode** |
|---|---|---|
| Trigger | A window was supplied: a quarter, a date range, or "the next N weeks" | No window given — the default for a bare list of items |
| Capacity | Yes | **Omitted entirely** |
| Commit tiers | Yes | Omitted — there is nothing to fit into |
| Grouping | By target or milestone if one exists | By dependency order |
| Everything else | Yes | Yes |

**Never infer a window.** If the caller gave no quarter and no dates, the report contains no dates,
no quarter, and no target-date framing — and its header says so explicitly, so a reader never reads
the absence of dates as an oversight. Inferring a quarter from today's date is the single easiest way
for this skill to publish a commitment nobody made.

One item is a valid input. With one item, sequencing degenerates to its internal order and the graph
shows its external edges. Say that, rather than emitting empty sections.

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

Columns: item · key · readiness · size · **Done so far** · **Remaining (1 eng opt/pess)** ·
**Remaining (2 eng opt/pess)** · **TOTAL** · max useful engineers · parallelisability · confidence.

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

### 5. Capacity — **window mode only**

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

### 6. Dependency graph — **both modes**

Per `references/dependency-graph.md`, rendered by `scripts/build_graph.py`. The Mermaid diagram, the
edge table, and then the three computed results that a flat table cannot give:

- **The critical chain** — its length in epic weeks, the unweighted gates on it by name, and the
  path. This is the date-bearing number and the report should say so where it appears.
- **What is blocked transitively** — blocked by something that is itself blocked.
- **Orphans** — no edges either way, so parallelisable now.

Report **link coverage** underneath: n of m **epics** carry any edge — gates are excluded from the
denominator, since a gate exists only to be an edge. A sparse graph means either
genuinely independent work or thin dependency research, the two look identical, and they have
opposite consequences.

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

In the workbook, key and PR columns use the `link` style so they are clickable there too.

### 14. Method and limits
How pace was measured, **which of the three rate sources was used and why**, the constants used, and
what the estimates assume. In window mode, the capacity inputs and which of them were human-supplied.
State the limits plainly — a report that admits what it does not know is the one people trust the
second time.

## The workbook

Build it with `scripts/build_workbook.py` from a JSON manifest. Sheets, in order:

| Sheet | Contents | Mode |
|---|---|---|
| **Summary** | One row per item: area, item, key(s), tracker status, source-doc status, verdict, % complete + basis, **Done so far**, **Remaining**, **TOTAL**, owner, confidence, link | Both |
| **Estimates** | Item, remaining scope, 1-eng optimistic/pessimistic (weeks), 2-eng optimistic/pessimistic, max useful engineers, parallelisability note, effort level, rate source, confidence | Both |
| **Current Assignments** | Engineer, item, key, status, since, open PRs, rate source, week ledger, notes | Both |
| **Sequencing** | Order, item, start-when, why, blocked-by, parallel-with | Both |
| **Dependencies** | **Node id**, item, depends on, node kind, owning team, status of the dependency, edge source, impact if it slips | Both |
| **Dependency Graph** | Edge list: from, to, type, source, note — the same data `build_graph.py` renders | Both |
| **Splits** | Epic, trigger, proposed cut, before/after one-eng weeks, max useful engineers, chain effect, coordination cost | Both |
| **Capacity** | The arithmetic line by line, each input with its provenance and whether it was measured or supplied | **Window only** |
| **Commit Tiers** | Tier, item, cost opt/pess, why, what is cut or which decision is pending | **Window only** |
| **Open Questions** | Question, owner, blocks, why it matters | Both |
| **Method** | The planning constants with provenance, the rate source, and the estimate assumptions | Both |

The Dependencies sheet carries the **node id** so its rows join to the graph block. The human-readable
labels differ between sheets and cannot be joined on — that is why the id column exists.

Rules for the cells:

- **Every estimate cell carries its confidence**, because a number in a spreadsheet loses its
  caveat the moment someone copies it. Low-confidence numbers say so in the cell itself.
- Never write a bare number for an unticketed item. Use the range plus `(placeholder — needs scoping)`.
- Status cells use the `status` style so the colour matches the tracker's own vocabulary.
- Ticket and PR columns use the `link` style so they are clickable.
- Keep the source document's own status in its own column next to the verified one. Seeing them side
  by side is the point.

## The HTML artefact — offered, not assumed

Markdown is the primary artefact. HTML is a **second** artefact, produced only if the caller wants a
page to screen-share or send on, and generated **from the same Markdown** — never hand-built and
never maintained as a second source of truth. Two hand-kept copies of the same numbers diverge, and
the one in the room is the one that is wrong.

Offer it; do not produce it unasked.

## Handing it over

Give clickable file links for both artefacts and the research directory. Lead the chat response with
section 1 — what changed — then the headline, then the open questions. Everything else is in the
files; do not re-narrate it in chat.
