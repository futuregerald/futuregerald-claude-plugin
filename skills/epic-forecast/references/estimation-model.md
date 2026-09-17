# Estimation model

An estimate is a claim about the future that must cite a fact about the past. If you cannot say which
measurement a number came from, you have a guess wearing an estimate's clothes — write it down as
"needs scoping" instead.

**Every estimate has two halves, and they fail differently:**

```
weeks = (how many tickets are REALLY left)  ÷  (how fast this team REALLY ships)
         ^ §1 — where the big errors live       ^ §2 — where the systematic bias lives
```

§1 errors are arithmetic and can be off by 3×. §2 errors are bias, and they are almost always *slow*.
Both overstate. In one measured run the same epic went from "6–13 weeks" to "2 weeks" — and the
miscount was the larger of the two errors.

## 1. Count what is really left

### 1a. Strip moot and superseded children FIRST

**Before counting anything, remove children that are no longer real work.** This is the single biggest
source of inflated estimates and it is invisible in every tracker view, because a moot ticket still
reads `To Do`.

A child is moot when:

- **A pending PR deletes the code it describes.** A bug ticket against `foo_matcher.rb` is not work if
  the open PR removes `foo_matcher.rb`. Check it: `git diff --diff-filter=D --name-only main...<branch>`
  to list deleted files, then grep those files for the symbols the open tickets name. One measured epic
  had **4 of 13 "remaining" children** that were all bugs inside a component the pending PR deleted
  wholesale.
- **A rewrite supersedes it.** An epic that pivots to a new model orphans every ticket describing the
  old one, and nobody goes back to close them.
- **It closed while you were working.** Re-pull statuses before publishing — on an active epic they
  move during the exercise.

Then apply the team's **historical Won't Do rate** to speculative newly-written children. If 27% of
resolved issues are Won't Do, a freshly-written batch of 25 is ~18 substantive. State the rate and that
you applied it.

Finally, **cross-check survivors against open PRs** — a child already covered by an open PR is work
that is done but not yet counted, and it comes out of remaining scope too.

### 1b. Count DESCENDANTS, not children

A one-level child query returns **one level only**. Trackers nest deeper — Epic → Story → Sub-task.
(In Jira the query is `parent = EPIC` and the sub-tasks carry `hierarchyLevel: -1`; every tracker has
its own spelling for the same trap.)

Measured on one real epic: the one-level query returns **3**. One of those has **6 sub-tasks**, five
Done. The real tree is **9 nodes, 8 complete (~89%)** — but the one-level count reports "2 of 3"
and makes a nearly finished epic look barely started.

Recurse: ask for the item's children, then for those keys' children, until a level returns nothing.
Report the depth walked and the count at each level.

### 1c. Then pick a sizing method

1. **Remaining real tickets ÷ the effective rate** (§2–3). The default.
2. **Remaining story points**, only if most closed children in that epic carried estimates.
3. **Analogy to a completed epic** — name it, and say why it is comparable.
4. **Bottom-up decomposition** when there are no children: migration, model, service, endpoint,
   serializer, view, flag, tests, backfill — each sized from comparable code already in the repo. The
   only defensible way to size an unticketed row, and far better than a blank cell.

## 2. The rate — measure it, then confirm it with a human

### 2a. Measure per engineer, and never take the bottom

Expect a **5–10× spread**. Measure merged PRs per week per engineer over ~8 weeks, convert with the
measured tickets-per-PR ratio (§2d).

The failure to avoid: taking the **bottom** of the measured range and applying it to everyone. On a
team whose fastest engineer runs roughly 5× the slowest, planning at the slowest inflates every
estimate by roughly 3×. A team-wide average is nearly as bad — it is dragged down by whoever is
stuck, and it describes nobody.

**These are estimation inputs, not a ranking.** See §2g before any per-engineer number reaches a
document.

### 2b. A low rate is usually a queue, not a capacity

Before planning at someone's low observed rate, ask *why* it is low. If their PRs sit 16–30 days
awaiting review, **that is their queue, not their throughput.** Planning at it bakes a solvable review
problem into the schedule as a permanent staffing fact — and points the reader at hiring when the fix
was a code review.

Plan at the **unblocked** rate — what the team's unblocked engineers actually achieve — and record the
queue as a risk to clear rather than a constraint to absorb.

### 2c. A changed toolchain invalidates the baseline

**Historical throughput is a lagging indicator whenever how the team works has changed.** If the team
adopted AI coding tools during or after the measurement window, the historical rate understates current
capability, and extrapolating from it produces confidently wrong, uniformly pessimistic numbers.

Two consequences:

- **Authoring stops being the constraint; review becomes the floor.** Implementation compresses, human
  review does not. Say so — it changes the recommendation from "add people" to "clear the queue".
- **Ask.** Whether the toolchain changed, and what rate to plan at, is something the EM knows and the
  tracker cannot tell you.

### 2d. Do not assume PRs batch tickets — measure the ratio

Compute tickets-per-PR from merged PRs. Worth checking, but the common case is the opposite of
batching: one measured team ran **median 1.0 tickets/PR** while averaging **1.27 PRs per ticket** —
tickets *split across* PRs, so counting PRs *overstated* throughput by ~27%.

### 2e. Confirm the rate with the EM before deriving anything

**The rate is one number that every estimate multiplies.** Getting it wrong is not one error, it is
every error at once — worth one question rather than a fourth silent guess.

Present the measurement, show what each candidate rate implies for the headline items, and let the EM
choose. `AskUserQuestion` with the implied per-epic numbers in the option previews works well: they
see the consequence of each choice instead of picking an abstract number.

**If you have already revised estimates twice against the same feedback, stop revising and ask.**
Repeated patching of individual numbers when the complaint is "they're all off" means the rate is
wrong, not the arithmetic.

### 2f. Three rate sources, in strict precedence — and say which you used

The rate is the denominator of every estimate. Where it came from is a **stated choice**, never an
invisible default.

| # | Source | When it applies | Confidence |
|---|---|---|---|
| 1 | **Per-engineer measured cadence** | An engineer is named for the item and has closure history | Highest — use it |
| 2 | **Caller-supplied rate** | The caller passed one (`--rate 6`, "assume 3 a week") | As stated. Record it as an **input**, not a measurement |
| 3 | **Team-average default** | Nothing better is available | Lowest — **label it a default in the output** |

The team-average default constant is defined **once**, in `capacity-model.md`. No other file embeds
a bare number for it, and the caller may override it.

**Source 1 is the point of this section.** Measured spreads across a single team routinely reach
**8×**, and routinely include a **zero**. Against a team average, a forecast that hands every engineer
the same rate is wrong for every individual in that spread, and wrong in both directions at once —
too slow for the fastest, too fast for whoever is stuck.

Measure it as tickets closed per **delivery** week on **that engineer's own item**, over a stated
lookback, and print the window and the sample size next to the rate.

#### Classify every week before you divide by it

A rate is `tickets ÷ weeks`, and the denominator is where this goes wrong. A week with no closures is
not evidence of slowness — the engineer may have been away, or scoping, or writing an ADR. Counting
those weeks as delivery weeks manufactures a low rate and then forecasts from it.

Classify each engineer-week from evidence, and report which bucket each fell into:

| Bucket | Evidence | In the numerator | In the denominator |
|---|---|---|---|
| **Delivery** | Tickets closed, or PRs merged or reviewed | Yes | Yes |
| **Planning / discovery** | Tickets **created or refined on an item**, ADR or design-doc commits, spike or research tickets worked, heavy comment activity with no closures | No | **No** — report separately as planning load |
| **No delivery signal** | Nothing across any source: no closures, no ticket creation, no commits, no comments, no reviews | No | **No** — excluded |

**Ticket-creation events are the key signal, and they are already available.** When tickets appear on
an item, somebody was decomposing it. An engineer with zero closures and twelve tickets created on
their item that week was working, and the report says so rather than recording a 0.

**Do not assert time off.** Absence of signal is absence of signal — equally consistent with
interviews, a support rotation, incident response, or a week of meetings. Label the bucket
**"no delivery signal"**, list the weeks, and ask the EM to confirm. Never print "on PTO" as a fact
the tracker cannot support.

Report a **week ledger** per engineer beside the rate: `N delivery · M planning · K no-signal`. A
rate computed from 2 delivery weeks is a far weaker claim than one from 8, and the ledger makes that
visible without extra prose.

#### Rules that hold whichever source is used

- **A 0/wk rate is a blocker, not a duration.** Report it as "N weeks after the block clears" and
  **name the block**. Never divide by zero, and never silently substitute the team average. Week
  classification tells you *which kind* of zero — blocked, planning, or absent — and the three imply
  completely different actions.
- **Thin evidence is disclosed, not smoothed.** Fewer than ~3 closed tickets, or fewer than ~3
  delivery weeks after classification, is not a rate. Fall back a level and say which level you used
  and why.
- **Print the denominator, always.** A rate without its week ledger is not reportable.
- **Never mix sources silently.** If two items in one report use different sources, the per-item rate
  table carries a **Source** column.
- **Still no focus factor on top.** The double-discounting rule applies to all three sources.

### 2g. What a per-engineer rate may not be used for

**A rate produced here is an estimation input. It is not a productivity measure.**

State this wherever a per-engineer number appears — in the report, and on the workbook's Method
sheet — in these terms:

> These rates are estimation inputs, measured over a short window from partial sources. They are not
> a productivity measure and must not be used in performance review, calibration, ranking, or any
> staffing decision about an individual. A rate here can be low because of review latency, a queue, a
> planning week, work in a repo or project this forecast did not query, or a window that was mostly
> discovery.

Three behaviours follow, and they are not optional:

- **Default to aggregates in anything that gets forwarded.** The circulated artefact carries the
  *spread* and the *band*. The per-person table goes to the EM, or is produced on request — not
  because the numbers are secret, but because a sortable column of names and rates is a stack rank
  whether or not anyone intended one.
- **Never rank.** Do not sort the per-engineer table by rate, and do not describe an engineer as fast
  or slow anywhere in the output. Report the rate, its source, its sample size and its week ledger,
  and let those speak.
- **A zero is a question, never a verdict.** §2f already says a 0/wk rate is a blocker rather than a
  duration. It is also the number most likely to be read as a judgement, so it carries its cause or
  it does not get printed.

## 3. Adjust for what a flat rate hides

A single rate implies every ticket is the same size and every epic equally ready. Neither holds. Make
both adjustments **visible as columns**, not buried in the rate, so a reader can disagree with one
without discarding the estimate.

```
effective_rate = base_rate × size_factor × readiness_factor
weeks          = tickets ÷ effective_rate  +  discovery_increment
```

**There are exactly two factors, and the rate gap (§3a) is not a third.** This formula is a
ready-made multiplication site, and the next plausible-looking multiplier added here is
double-discounting — the thing `review-checklist.md` §5b exists to catch. If you are about to
multiply by something that is not `size_factor` or `readiness_factor`, stop.

**Ticket size** — measure median PR lines/files per epic, don't eyeball it:

| Size | Factor | Shape |
|---|---|---|
| S | ×1.35 | surgical fixes, ~120 lines |
| M | ×1.0 | a feature slice, ~280 lines |
| L | ×0.7 | multi-surface or cross-repo |

**Scope readiness** — *once planning is done, shipping is much faster*:

| Readiness | Factor | Discovery increment |
|---|---|---|
| Ticketed & scoped | ×1.15 | 0 |
| Ticketed, unsized | ×1.0 | ~0.25 wk |
| **Not ticketed** | ×0.85 | **~1 wk before any build starts** |

The readiness column earns its place by explaining *why* a row is expensive. An unticketed epic with a
modest ticket count lands near the top of the list and the reason is legible: **its cost is that nobody
has written the tickets.** That points at a decision — scope it now, or defer it deliberately — where a
bare week-count would not.

### 3a. The rate gap — report it, never apply it

An engineer nominally assigned to one item delivers roughly **1.1–1.5 of that item's tickets per
week**, against a team headline rate several times higher. The gap is real: the rest of their week
goes to other items, reviews, incidents and meetings.

**It is descriptive. It is never a factor.**

The gap is **already inside every per-item measured rate**, because §2f measures children closed per
week on each item by its actual owner. Applying it again on top of a source-1 rate discounts the same
time twice, which is precisely the failure `review-checklist.md` §5b names — one level down, and
harder to spot because the multiplier has a plausible story attached.

It is reported for one reason: so a reader who sees a headline per-engineer rate does not multiply
it by a ticket count and conclude the team has roughly twice the capacity it has. Print it beside
the headline rate as a caution, not inside the arithmetic.

This section was previously named "concurrency haircut". The name was changed because it invited
exactly the error it exists to prevent — a "haircut" reads as something you apply.

## 4. Report Done, Remaining and TOTAL — and make them tie out

Always give **three** numbers per item:

```
TOTAL = Done so far + Remaining
```

Derive TOTAL that way rather than computing it independently. Independent computation produces the
embarrassing, easy-to-miss case where **remaining exceeds total** — which happens as soon as the
remaining figure includes review or discovery time the total didn't.

`Done so far` = completed tickets ÷ the same effective rate, and it is worth showing for its own sake.
An epic that has quietly consumed six engineer-weeks on incident work is the largest real spend in a
portfolio, and a remaining-only view hides that entirely. It also exposes rows with `Done = 0` and a
large total — the unstarted, unscoped work that is easiest to under-plan.

## 5. The two bounds

Both bounds describe the **same** scope. If pessimistic silently includes work optimistic excludes, the
range is meaningless.

**Optimistic** — dependencies land on time, undefined scope resolves small, the owner is uninterrupted,
review at the measured median.

**Pessimistic** — effective rate at roughly 0.6×, **every undefined-scope item at the large end** (the
dominant term), review at p75 plus one fix cycle, and one integration surprise per epic that crosses a
repo or team boundary.

Do **not** publish a single "most likely" number alongside them; readers anchor on it and it erases the
uncertainty the range exists to show.

### 5a. Two anchors, reported side by side and deliberately not reconciled

Run the estimate twice, from two independent bases, and publish both with the gap between them:

| Anchor | Basis | Use it for |
|---|---|---|
| **Measured rate** | `tickets ÷ effective_rate` from §2f | Items with real children and a real owner |
| **Median completed item** | The team's median item end-to-end — measure it: expect something like six weeks and ten children, over a range several times wider at both ends | Items with no children, or as a sanity check on the first |

**Do not average them and do not pick a winner.** The two anchors sit roughly 2× apart by
construction, because the ticket rate measures an engineer's throughput while the median-item figure
measures calendar elapsed time including every wait. Reporting a reconciled midpoint hides exactly
the uncertainty a reader needs.

Where they disagree by more than ~2×, that is a finding worth a sentence: usually it means the item's
children are unusually large or unusually small relative to the team's norm.

A third check worth stating when the data supports it: the **review ceiling**. Median and p75 hours
per PR, multiplied by the expected PR count, gives a floor on calendar time that is independent of
authoring speed entirely. Where that floor exceeds the measured-rate estimate, **review is the
constraint and more engineers will not move the date** — say so, because it changes the
recommendation from "add people" to "clear the queue".

## 6. One engineer vs two

**Never estimate two engineers as half of one.** Compute from the item's seams:

- Backend/frontend across an agreed contract — genuinely separable, ~1.6–1.8×.
- Independent children under one epic — near-linear if they share no migration or core model.
- One migration, one model, one service object — **not** separable. `max useful = 1`.
- Blocked on a dependency or a decision — headcount changes nothing. `max useful = 1`, and say the
  constraint is the dependency.

```
two_engineer_weeks = one_engineer_weeks / speedup + coordination
```

Speedup 1.0 / 1.5 / 1.8 by seam; coordination 0.5–1 day for the contract plus 0.5 day per overlapping
week. **Ramp-up is frequently the whole answer** — a second engineer joining a 70%-done epic in
unfamiliar code will not pay back inside two weeks; say so rather than reporting a speedup nobody will
observe. Where the second engineer only frees up later, **model the lag**: the wait often eats most of
the gain, which makes "move them now" a materially different recommendation from "add them".

## 7. Effort level and confidence

| Level | One-engineer calendar time |
|---|---|
| XS | < 3 days |
| S | 3–10 days |
| M | 2–4 weeks |
| L | 1–2 months |
| XL | > 2 months |

**Confidence**: High = ticketed, estimated, underway, no external dependency. Medium = scope written
but not decomposed, or one external dependency. Low = no ticket, an undecided input, or scope that is
one sentence in a spreadsheet.

**Low-confidence estimates are placeholders and must say so inside the cell**, because a number in a
spreadsheet loses its caveat the moment someone copies it.

An estimate supplied by the EM is **High confidence and used verbatim** — label it as theirs.
They hold information the tracker does not, and reconciling their number against your derivation is
how you find out which of your factors is wrong.

## 8. Where to split, and what the split buys

**Both modes.** Splitting needs no window, and with a single item in scope it is often the most
useful output in the whole report.

"This epic is too big" is an observation. A split proposal with its arithmetic is a recommendation.
This section is the difference.

### 8a. Which items are candidates

Flag an item when any of these hold, and **say which trigger fired** — the trigger is half the
argument:

- The one-engineer estimate exceeds ~4 weeks. It will cross a reporting boundary and resist tracking.
- `max useful engineers = 1` while the estimate is large. The **shape** is the limit, not the
  headcount, and no amount of staffing fixes it.
- Children span more than one repo or surface — backend and frontend, or two services.
- Part of the tree is blocked by an external `decision` or `queue` node while the rest is not.
- Readiness is mixed: some children ticketed and scoped, others a single line in a spreadsheet.
- A `Won't Do` cluster suggests the item absorbed unrelated work.

### 8b. Where to cut, in preference order

The best cut is the one that frees work that is already unblocked.

1. **Along the blocking boundary.** Separate the blocked sub-tree from the unblocked one. Highest
   value by a distance: it converts "the whole item waits" into "half of it ships now".
2. **Along a repo or contract seam.** Backend/frontend across an *agreed* contract is the only split
   §6 credits with a 1.8 speedup. Anything vaguer gets 1.5 or 1.0.
3. **Along readiness.** Scoped children into a deliverable item, unscoped ones into a discovery item
   carrying the ~1 week discovery increment — so the unknown stops inflating the known.
4. **Along deliverable value.** One thin independently shippable slice; the remainder deferred.

Never propose a split that produces a piece with no independent value, and never split merely to make
a number smaller.

### 8c. Quantify it — this is the point of the section

For each proposal, state before and after:

| Measure | Before | After |
|---|---|---|
| One-engineer weeks — each piece, and the total | | |
| Max useful engineers | | |
| Critical-chain contribution | | |
| Earliest independently shippable piece | | |
| Coordination cost added | | |

Four rules the arithmetic must obey:

- **Splitting does not reduce total work.** The post-split one-engineer total must be **≥** the
  pre-split total, and usually slightly more. A proposal showing *less* total work is an arithmetic
  error, not a win. The gain is in parallelism and earlier delivery, and the table has to show it as
  such.
- **Charge the coordination cost** from §6 — 0.5–1 day of contract alignment plus 0.5 day per
  overlapping week — as a visible debit, not a footnote.
- **Re-derive the critical chain with the split applied** (`dependency-graph.md`). A split that
  shortens the longest path is the strongest argument available and should be stated first. A split
  that does *not* shorten it is still legitimate for trackability — but say plainly that the date
  does not move, so nobody reads the proposal as a schedule fix.
- **A split is a proposal, never an action.** The skill does not create, move, or restructure
  tickets. Output is a recommendation with its arithmetic, for a human to accept or reject.
