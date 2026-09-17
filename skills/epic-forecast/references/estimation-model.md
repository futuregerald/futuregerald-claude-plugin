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

`parent = EPIC` returns **one level only**. Trackers nest deeper — Epic → Story → Sub-task, where
sub-tasks carry `hierarchyLevel: -1`.

Measured on one real epic: `parent = <epic>` returns **3**. One of those has **6 sub-tasks**, five Done. The real tree
is **9 nodes, 8 complete (~89%)** — but the one-level count reports "2 of 3" and makes a nearly
finished epic look barely started.

Recurse: `parent = <epic>`, then `parent in (<those keys>)`, until a level returns nothing. Report the
depth walked and the count at each level.

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

In one run the per-engineer rates were 7.8, 6.4, 2.4, 2.0 and 1.6 merged PRs/week. Planning at
"1.1–2.2" — the bottom of that range, applied to everyone — inflated every estimate by roughly 3×.
A team-wide average is nearly as bad: it is dragged down by whoever is stuck.

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

## 3. Adjust for what a flat rate hides

A single rate implies every ticket is the same size and every epic equally ready. Neither holds. Make
both adjustments **visible as columns**, not buried in the rate, so a reader can disagree with one
without discarding the estimate.

```
effective_rate = base_rate × size_factor × readiness_factor
weeks          = tickets ÷ effective_rate  +  discovery_increment
```

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
