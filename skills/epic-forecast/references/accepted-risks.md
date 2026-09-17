# Accepted risks

Things this skill knowingly does not guard against, why, and what would change the decision.

A risk recorded here has been considered and accepted. A risk *not* recorded here has not been
considered — if you find one, add it with a reason or fix it. "We never thought about it" and "we
decided it was fine" look identical in a codebase unless someone writes down which it was.

---

## 1. The capacity arithmetic has no automated regression test

**What.** `capacity-model.md` carries the formula, the input rules and a worked example in prose. If
somebody edits the worked example and gets it wrong, or reorders the steps so unplanned work is
applied before carryover, nothing fails.

**Why accepted.** The formula is guidance an agent follows against inputs a human supplies — three of
its five inputs (`non_working_weeks`, `engineers_it_draws`, and the window itself) cannot come from
any tracker. Moving it into a script to make it testable would create a second source of truth beside
the prose and a script with no caller, which is the failure the plan explicitly avoided when it left
`build_workbook.py` alone.

It was verified once, at the gate the plan called the real one: the documented formula and inputs
reproduce the source document exactly — 50 raw → 38 / 30.5 after carryover → 26.22 / 21.05 after the
measured 31%, published as 26.2 / 21.05, with demand ratios 0.8× and 1.9× against the optimistic
capacity. The wrong-order variant was computed alongside it and lands 3.72 engineer-weeks off, which
is why the ordering rule is stated three times in that file.

**Revisit when** a real run publishes capacity numbers that disagree with the worked example, or if
the arithmetic ever moves into a script — at which point it gets a test like any other code.

## 2. The skill was not run end to end against a live tracker

**What.** `build_graph.py` has 28 unit tests and was run end to end on a fixture; the capacity gate
was reconciled against the source document. The full pipeline — Phase 1 baseline through a nine-item
window-mode report and a single-item scope-mode report against live Jira — was not run.

**Why accepted.** An end-to-end run dispatches eight to ten research sub-agents against live tracker
data and produces a report only a human familiar with those items can grade. The parts that *can* be
wrong silently — the graph algorithms and the capacity arithmetic — are the parts that were
verified. The parts not verified are prose instructions, which fail visibly on first use.

**Revisit at the first real invocation.** Specifically check: whether the `graph` block is actually
assemblable from what the cluster agents return, whether the mode detection fires correctly on a bare
epic list, and whether the conditional sections render without stubs. Fix what breaks then rather
than guessing now.

## 3. A cycle can exclude most of the graph from the chain computation

**What.** `build_graph.py` excludes cycle members **plus their directed ancestors and descendants**
from the critical-chain computation. A cycle near the root of a well-connected graph can therefore
exclude nearly every node, leaving a chain section that reports very little.

**Why accepted.** The alternative is worse in a way that is hard to see: a longest path computed
through or around a cycle is not a longest path, and it would be published as a date. Every excluded
node is named in the output, and a cycle is itself a data-quality finding somebody has to fix — a
dependency recorded backwards, or two pieces of work that need splitting apart. A large exclusion
list is a loud signal, which is the right behaviour.

**Revisit if** a real run excludes more than roughly half its nodes and the cycle turns out to be
genuine rather than a data error. The fix would be per-component partial chains with an explicit
"this component has no defined chain" marker, rather than one global exclusion.

## 4. CI collects skill tests by shell glob

**What.** The workflow runs `pytest -q skills/*/scripts`. Tests that live anywhere other than a
skill's `scripts/` directory are not collected, and if no skill has a `scripts/` directory at all the
literal path reaches pytest and the step errors.

**Why accepted.** `scripts/` is the convention in this repo and two skills follow it; the glob is a
direct widening of the pinned path it replaced, which had the same shape and a narrower blind spot. A
hard error on zero matches is the correct failure — silent collection of nothing is what the widening
existed to fix.

**Revisit if** a skill ever ships tests outside `scripts/`, or if the repo grows enough skills that
collection time matters.

## 5. A per-engineer rate can still be misused, however it is labelled

**What.** The skill measures individual throughput and prints it. Every guardrail in
`estimation-model.md` §2g — the purpose limitation, aggregates by default, no ranking, no
fast/slow language — is an instruction to an agent and a sentence in a document. Neither stops a
reader copying the number into a different context where none of that text follows it.

**Why accepted.** The per-engineer rate is the feature: a team average is wrong for every individual
on a team with an 8× spread, and using one produces forecasts that are wrong in both directions at
once. Removing the measurement removes the skill's main improvement over guessing. The realistic
mitigations are the ones taken — keep the per-person breakdown out of the forwarded artefact, never
sort by it, never characterise a person, always print the denominator and the purpose limitation, and
require human confirmation before anyone is removed from a headcount.

Worth knowing rather than acting on now: individual productivity measurement is regulated in some
jurisdictions — GDPR Art. 88 on employment data, and works-council consultation in several European
countries. A team operating under those rules should check before circulating per-person output at
all.

**Revisit if** this skill is ever run by someone other than the team's own manager, if its output is
fed into a performance or calibration process, or if a reader asks for the per-person table to be
sorted. Any of those is a signal the guardrails are not holding and the per-person output should
become opt-in rather than default-suppressed.

## 6. `build_graph.py` overwrites its output file without asking

**What.** `main()` opens the output path with `"w"`, truncating whatever is there. There is no
`--force` flag and no parent-directory check.

**Why accepted.** A reviewer proposed refusing to overwrite without `--force`. Declined on balance:
the normal workflow is to regenerate `graph.md` repeatedly as the manifest firms up, and a confirm
prompt on every re-run trains the user to pass `--force` always, which is worse than no guard. The
output path comes from `argv`, never from manifest content, so tracker text cannot steer the write —
which is the failure that would have justified the guard.

**Revisit if** the path ever becomes derivable from the manifest, or if the skill starts invoking the
script on the user's behalf without showing the command.
