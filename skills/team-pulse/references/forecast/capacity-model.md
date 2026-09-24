# Capacity model

**Only when the caller asked whether the work fits a window.** No quarter, no date range, no "next
N weeks" means there is nothing to fit into: skip this file entirely and do not invent a window to
enable it.

**This file is not how the skill produces a date.** Capacity answers *does the work fit in this
window*. The **projected landing** — *when does it land* — comes from the critical chain divided by
the engineers on it (`SKILL.md`, Phase 6), needs no window, and is produced whenever a date is
wanted. Skipping capacity never means skipping the projection.

Capacity answers a different question from estimation. Estimation asks *how big is this work*.
Capacity asks *how much work can this team absorb in this window*. Reporting one without the other
produces the two classic failures: a list of estimates nobody can fit into a plan, or a plan with no
evidence that the team can do it.

## The unit is engineer-weeks. Never hours.

One engineer-week is one engineer working for one calendar week. Hours are not used anywhere in this
skill, and converting to them is not an improvement — it invites a focus factor, and a focus factor
on top of an observed rate **discounts twice**. See `review-checklist.md` §5b. An observed closure
rate already contains that engineer's meetings, interrupts, on-call and review waits; multiplying it
by 0.7 "for overhead" removes the same time a second time.

## The arithmetic

```
productive_weeks  = calendar_weeks − non_working_weeks
raw               = productive_weeks × active_engineers
after_carryover   = raw − carryover_spill
available         = after_carryover − (unplanned_rate × after_carryover)
```

**Order matters.** `unplanned_rate` applies to the post-carryover remainder, not to `raw`. Carryover
is work already committed; the unplanned load lands on what is left after it, not on the whole
window. Applying the rate first overstates available capacity, and the two orderings differ by
`unplanned_rate × carryover_spill` — several engineer-weeks in any real window.

Run the whole chain twice, optimistic and pessimistic, and publish the pair as a band. The only input
that usually differs between the two runs is `carryover_spill`; say so, so a reader can see which
assumption moved the number.

## The inputs, and where each one comes from

### `calendar_weeks`

The window the caller gave. Supplied, never derived.

### `non_working_weeks`

Holidays, company shutdowns, an all-hands week, a planned offsite. **This is a human input and the
skill must ask for it.** There is no tracker field for "the week everybody was at the offsite", and
guessing it from a dip in closures confuses absence with slowness — the exact error §"Classify every
week" in `estimation-model.md` exists to prevent.

**If the caller cannot supply it, omit the capacity section** — do not default it to zero. Zero
produces the largest capacity the arithmetic can yield, which is the most oversubscribed commitment
available, and it arrives looking like a measurement. Say the section was omitted and which input was
missing.

### `active_engineers`

**Whole people who are actually producing. Not headcount, and not pro-rated.**

An engineer with no delivery signal across the measurement window is removed from the count
entirely — they are not 0.4 of an engineer. Pro-rating is wrong for a second reason too: half an
engineer cannot carry an epic, and capacity is consumed by whole people working on whole pieces of
work.

Illustrative: a team of six counted as **five** produces a capacity figure 20% lower than headcount
would, and a commitment built on the headcount figure is 20% oversubscribed. That is the whole reason
this input is not simply "how many people are on the team".

### Removing someone is a claim about a person. Three rules, all mandatory.

**1. Never assert time off, and never publish a name in the circulated artefact.**
Absence of signal is absence of signal. It is equally consistent with interviews, a support rotation,
incident response, parental or medical leave, a secondment to another tracker project, a window spent
entirely on code review in a repo you did not query, or someone who has already left. The bucket is
**"no delivery signal"** — the same rule as `estimation-model.md` §2f, and it applies here with more
force, because here it changes a number attached to a person.

The artefact says: `active_engineers = 5 (6 on the roster; one removed — see the EM)`.
The name and the evidence go to the EM **directly**, not into a document that will be forwarded.

**2. Confirm with the EM before removing anyone.** This is the highest-consequence judgement in the
model and it rests on the weakest evidence in it. If the EM is unavailable, do not choose: publish
**both** figures as a band (here, 6 and 5), say the removal is unconfirmed, and let the arithmetic
carry the uncertainty rather than a person.

**3. Apply the week classification first** (`estimation-model.md` §2f). A week with ticket-creation
activity and no closures is a **planning** week, not an absent one, and an engineer whose whole
window was planning is producing — just not closures. Check too that `non_working_weeks` was actually
supplied: if it defaulted to zero, you may be about to remove the person who was out during the
shutdown.

### `carryover_spill`

Work already committed from the previous window that will still be running inside this one.

```
carryover_spill = remaining_weeks_of_prior_commitment × engineers_it_draws
```

Both terms are estimates, so this input carries the widest uncertainty of any in the model, and it is
where the optimistic and pessimistic runs diverge:

- **Optimistic** — the prior commitment finishes at its optimistic bound, drawing its engineers for
  the shorter time.
- **Pessimistic** — it finishes at its pessimistic bound.

**`engineers_it_draws` is a human input.** Which engineers keep working on the old commitment while
the new window starts is an allocation decision, not a tracker fact. Ask for it. In the observed run
it was "roughly 3 of the 5", and that allocation was a call the EM made in conversation — nothing in
the tracker implied it.

Show the two spill figures separately in the output. They are the visible reason the capacity band
has two ends, and a reader who disagrees with the allocation can re-run the arithmetic themselves.

### `unplanned_rate`

**Measured from the tracker. Never assumed, never a round number pulled from memory.**

The method:

```
numerator   = closed tickets with no parent epic
            + bugs attached to an epic MORE THAN 30 DAYS after that epic was created
denominator = all closed tickets in the window
both sides    EXCLUDE issues owned by non-delivery roles (see below)
```

Each clause earns its place:

- **Unparented closed tickets** are work that arrived without a plan — the definition of unplanned.
- **Late-attached bugs** are the other half, and the half a naive count misses. A bug filed against a
  shipped epic two months later is unplanned work that a parent-based count records as planned,
  because by the time you look it *has* a parent. The 30-day threshold separates "found during the
  epic" from "arrived afterwards"; state the threshold you used, because it is a judgement.
- **Excluding non-delivery roles from both sides** matters because their tickets are overwhelmingly
  unparented by nature — planning, hiring, process, reporting. Whoever those roles are on this team
  (engineering manager, product manager, design, whoever files work that will never consume
  engineering capacity), leaving them in inflates the rate with work that was never engineering
  capacity in the first place. Excluding from the numerator only would deflate it; both sides, or
  neither. Name whom you excluded.

Record the two naive alternatives and why they are wrong, because somebody will propose them:

| Candidate | Observed | Why it is wrong |
|---|---|---|
| Unparented closed tickets only | 22% | **Understates.** Misses every bug that arrived late and got parented |
| All work types, non-delivery roles included | 59% | **Overstates.** Counts management and product tickets as engineering capacity |
| **Unparented + late bugs, non-delivery roles excluded** | **31%** | The measured figure this model uses |

A 22%-vs-59% spread on the same window is the whole argument for measuring rather than assuming: any
number in that range can be defended by choosing a definition, so the definition has to be stated.

### The default throughput rate

**4 tickets per engineer per week.** This constant is defined **here and nowhere else**. No other
file in this skill may embed a bare `4`.

It is the **lowest-precedence** source of a throughput rate and must be labelled a default wherever
it is used. `estimation-model.md` §2f gives the full precedence order: a per-engineer measured
cadence beats a caller-supplied rate, which beats this. The caller may override it; when they do,
record the supplied number as an input, not a measurement.

## Demand versus capacity

Demand is the sum of the one-engineer estimates for the work under consideration, expressed as a
range from the optimistic and pessimistic bounds of each item.

**Both ratios divide by the same denominator: the optimistic capacity.**

```
lower ratio = demand_optimistic  ÷ capacity_optimistic
upper ratio = demand_pessimistic ÷ capacity_optimistic
```

This is a demand *range* measured against **one reference capacity**, not two independent ratios.
Pairing the pessimistic demand with the pessimistic capacity compounds two worst cases and produces a
number the source document does not publish.

Worked, with the observed figures:

```
demand    21.5 – 50  eng-weeks
capacity  21.05 – 26.2 eng-weeks

21.5 / 26.2 = 0.8×      <- published
50   / 26.2 = 1.9×      <- published
50   / 21.05 = 2.4×     <- NOT published; compounds both worst cases
```

State the denominator explicitly every time the ratio appears, so the worked example and the rule
cannot drift apart.

## The sum is never a date

Say this wherever a capacity total appears. Capacity tells you whether the work **fits**. It never
tells you when it **finishes**, because items run concurrently across engineers.

**The date is the longest chain** — the critical path through the dependency graph, in engineer-weeks
along that path. See the ordering in `method.md` Phase 6. A portfolio that fits comfortably inside available
capacity can still miss its date, if one chain inside it is longer than the window; and that is the
common case, not an edge case.

Report both: *does it fit* (capacity) and *when does it land* (chain). A report with only the first
will be read as the second.

## Worked example

Inputs, from the observed run:

| Input | Value | Provenance |
|---|---|---|
| `calendar_weeks` | 13 | Supplied by the caller |
| `non_working_weeks` | 3 | Human input — holidays and a shutdown |
| `active_engineers` | 5 | 6 on the roster; one removed for no delivery signal, EM-confirmed |
| `carryover_spill` | 12 opt / 19.5 pess | Prior commitment's remaining weeks × ~3 of the 5 engineers |
| `unplanned_rate` | 31% | Measured; see the table above |

```
productive_weeks = 13 − 3                 = 10
raw              = 10 × 5                 = 50 eng-weeks

optimistic:  after_carryover = 50 − 12    = 38
             available       = 38 − 0.31 × 38    = 26.2

pessimistic: after_carryover = 50 − 19.5  = 30.5
             available       = 30.5 − 0.31 × 30.5 = 21.05

published band: 21 – 26 eng-weeks
```

Publish the band rounded, and the chain separately. Do not publish a midpoint: a single number
erases the uncertainty the band exists to show, and readers anchor on it — the same rule the two
estimate bounds follow in `estimation-model.md` §5.

## What the skill must never do here

- **Never invent `non_working_weeks`, `engineers_it_draws`, or the tier assignments.** These were
  human calls in every real run. Ask, or omit the section and say it was omitted for want of an
  input. An omitted capacity section is honest; a fabricated one is load-bearing and wrong.
- **Never pro-rate a non-producing engineer** into a fraction. Remove and report.
- **Never apply a focus factor** on top of a measured rate.
- **Never present the capacity total as a completion date.**
- **Never run this section at all in scope mode.** No window means no capacity.
