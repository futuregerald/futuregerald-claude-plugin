# Claim Ledger

Across 83 real adversarial reviews, **43% of gating findings were a false premise** — a
statement about the codebase that the code contradicted — and it was the sole cause of 12
rejections. Prose hides those claims. A table exposes them, to you first and the reviewer
second.

## The ledger

Every factual assertion the plan makes gets a row. Put it in the plan, above the Impact
Analysis.

| # | Claim | Evidence | What I actually did | Confidence |
|---|---|---|---|---|
| C1 | `Audit#retest_start_date` does not exist | `grep -rn "retest_start_date"` → 0 hits | ran it across the repo excluding `.git` | high |
| C2 | `audits` has `retest_end_date date` | `db/structure.sql:1285` | read the line | high |
| C3 | `Vulnerability::RETEST_SLA_DAYS` does not exist | `grep -rn "RETEST_SLA"` → `app/models/vulnerability.rb:44` | read it — **claim was wrong** | — |

C3 is the point. Writing the row is what makes you run the search.

Rules:

- One row per assertion. If a paragraph makes three claims, that is three rows.
- Evidence is a `file:line` or the exact command run. "I checked" is not evidence.
- "What I actually did" separates *read the file* from *inferred from the name*. The second
  is not verification.
- A row you cannot fill is not a row you delete. It goes to **Could Not Verify**.

## Absence claims carry their search

"Nothing else calls this", "there is no existing helper", "no test covers this" are the most
dangerous claims in any plan and the least verified. You cannot confirm an absence by finding
something — only by a search that came back empty.

So the row records **the searches you ran**, not the conclusion:

> C7 — `HTTPProfileLookup` has one production caller.
> `grep -rn "HTTPProfileLookup" --include='*.go'` → `main.go:267`, `profile.go:70`, plus 6
> test files. Also checked: interface satisfaction (`grep -n "ProfileLookup" *.go` for a
> named interface), struct tags, `//go:embed`, ldflags `-X` in `.goreleaser.yaml`, CI YAML,
> `scripts/*.sh`. **Test files are callers — 6 of them.**

Search deliberately for what a plain grep misses: reflection, `send`/`public_send`/
`constantize`, interface satisfaction, struct tags, embedded types, `method_missing`,
delegation, callbacks and observers, job class names stored as strings, config-driven
dispatch, `//go:embed`, build tags, ldflags, generated code, shell scripts and CI YAML that
grep source, docs shipped as release artifacts, and shared test helpers.

## Predictions are claims too

"This is behaviour-preserving." "Task 6 is move-only." "No fixture changes are expected."
"The existing tests still pass." "This compiles."

These are about code that does not exist yet, so they feel unfalsifiable. They are not — you
check them by reasoning from the actual code plus your own instructions, and in the measured
corpus this is where the false premises actually lived. A plan that said "no fixture changes
are expected" was wrong about 132 call sites that would have started returning 403.

Give them rows, and mark them `prediction`:

| # | Claim | Evidence | Confidence |
|---|---|---|---|
| C11 | Task 6 is move-only for `Simulator` (prediction) | traced every `&Simulator{` — 9 sites, 6 are bare literals relying on zero values | medium — a required-arg constructor breaks all 6 |

## Could Not Verify

A first-class section, and using it costs you nothing. It exists because the plan format
otherwise gives uncertainty nowhere to go, so it gets laundered into confident prose — which
is the mechanism behind the 43%.

> **Could Not Verify**
> - Whether the EU warehouse shows the same NULL `end_date` pattern. No EU dataset access.
>   **Not load-bearing** — the fix does not depend on the count.
> - Whether `perform_now` inside the transaction can deadlock under real concurrency.
>   **LOAD-BEARING** — Task 4's ordering depends on it.

The rule that gives the section teeth:

**An unverified claim that is load-bearing blocks dispatch.** Do not send the plan to review
with a load-bearing unknown; go and settle it, or restructure so the plan no longer depends
on it. Unverified and *not* load-bearing is fine to record and ship.

It must be cheaper to write "unverified" than to guess. If it is not, the ledger will fill
up with confident rows that were never checked, and you have rebuilt the problem.

## What good looks like

The reviewer independently rebuilds your Impact Analysis and checks your claims against the
code. Rows that hold cost it seconds instead of minutes; rows that do not are found by you
rather than by it. Either way the ledger is not what protects you — see
[[impact-analysis]] for the part that does. A plan whose ledger is entirely correct can
still be wrong in every way that matters, because the ledger only contains what you thought
to write down.
