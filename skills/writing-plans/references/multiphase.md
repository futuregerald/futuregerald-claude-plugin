# Multi-Task Plans

An Impact Analysis is a snapshot of the codebase at the base SHA. A six-task plan mutates
that codebase six times, so task 4's analysis is computed against pre-task-1 code and is
**stale by construction**. The plan's own diff is invisible to the plan.

In the measured corpus this was 8% of gating findings directly, and more once you count the
"specified mechanism cannot work" cases that trace back to it: *"Task 2 disarms the F3 safety
net Task 1 depends on"*, *"Task 3 Step 2 mispredicts the test's RED state — the test cannot
pass after Tasks 1–2"*.

## Contract delta

Each task declares what it changes about every symbol it touches — return shape, signature,
nil or zero case, error identity, ordering, side effect. One line each.

> **Task 3 contract delta:** `Authenticate` gains a `ctx` first parameter. Callers: 3
> production, 7 test. Error identity unchanged. `CheckM2M` closure now receives the request
> context rather than its own.

The delta is what later tasks read. Without it, task 6 is planned against the original tree.

## Moving baseline

Analyse task N against the tree as it stands **after tasks 1..N−1**, not against the base SHA.

Carry a running ledger: after each task, note which symbols now have a different contract,
which files moved, and which line numbers shifted. When you write task 5, read that ledger
first.

The tell that you skipped this: a task whose expected test output is only correct against the
original tree. "Expected: FAIL with `undefined method`" is wrong if task 2 already defined it.

## Hotspot symbols

Any symbol touched by two or more tasks is a hotspot. List them, and for each give the
sequencing rationale — why this order and not the other.

Hotspots are where task N breaks task M. If you cannot say why the order is what it is, the
order is arbitrary, and an arbitrary order is a coin flip on a defect.

## Citation drift

When an earlier task edits a file **above** a line you cite later, that citation is stale.
Re-check every `file:line` in tasks N+1.. after planning task N, or cite by symbol name
instead of line number — names do not drift.

Two specific traps:

- **A cited range wider than the replacement text supplied for it.** If the Files list says
  `foo.md:12–18` and the replacement block covers only what was at `:12–15`, an implementer
  following the citation silently deletes `:16–18`. Make the range and the replacement match
  exactly, or cite by anchor text.
- **A verbatim code quote that goes stale.** If task 2 edits the function you quoted in task
  7, the quote is now fiction. Either re-quote or stop quoting.

## Gate falsifiability

13% of gating findings were verification that cannot fail — the third-largest bucket, and the
only one **no downstream gate owns**. Code review does not check whether your test could
fail. This is the category that survives review and ships.

For every test, gate, command and pass criterion you write, ask: *would this detect the thing
it exists to detect, if that thing were broken?*

Specific failure modes, all observed in real plans:

- A "write the failing test" step whose test would actually **pass** at that point in the
  sequence, or fail for a different reason than stated. Work out the real state of the tree
  at that step and predict the actual output.
- A test that passes both with and without the change. Prove otherwise by reverting the fix
  and watching it go red — a test never watched failing has not been shown to test anything.
- A shell gate whose exit status cannot propagate: `| head`, `| grep`, `| tee` masking an
  earlier failure; missing `set -o pipefail`; a command that exits 0 on no matches; a loop
  that swallows status.
- A gate that runs in the wrong directory or against the wrong package — executing nothing
  and exiting 0.
- **A gate whose tool excludes the very thing it is meant to prove.** `go list -deps` ignores
  test imports, so it certifies an import boundary that test files violate.
- A pass criterion attached to a number the command cannot produce — `grep -c '^=== RUN'`
  counts subtests, so it never equals the test-function count.
- An assertion that does not bind to the value the change affects.
- **An existence check standing in for a substance check** — a gate that a one-line stub of
  the intended file would satisfy. Assert on required headings, minimum size, or distinctive
  content, not on the file being present.

Record the census honestly: how many gates, and how many of them cannot fail. If the answer
is not zero, say which, and why you are shipping it anyway. See [[claim-ledger]] for where
that admission belongs.
