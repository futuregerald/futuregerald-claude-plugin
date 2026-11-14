---
name: fix-check
description: Light verification after a fix round. Takes the findings a review already raised and the fix diff, and reports per finding whether the code now fixes it, where, and which test covers it. Use after an implementer addresses review findings, between fix rounds, or before merging a fix. Use it INSTEAD of a fresh comprehensive review or a re-review; it does not hunt for new issues.
author: Gerald Onyango <gerald.onyango@gmail.com>
tags: [review, quality, verification, code-review]
---

# Fix Check

A review already found the problems. This skill answers one question for each of them: is it fixed? It is a ledger, not a review. A fresh comprehensive review after every fix round costs a full review each time and keeps surfacing new, smaller findings, so the loop never closes. Fix check closes it.

## What good output looks like

- **Who reads it:** the orchestrator deciding whether to merge, and the person who raised the findings.
- **Good:** one row per original finding with a verdict, the `file:line` of the change, and the test that would fail without it. A reader can merge, or send back exactly the unfixed rows, without reading the diff.
- **The default to prevent:** a second full review that re-reads everything, finds new MINORs and starts another round. Also prevented: "all fixed" with no location and no test.
- **The detail a careless version misses:** a fix that changes the code but not the behaviour the finding named, with a test that asserts the new code path instead of the failure the finding described. Check the test against the finding's failure scenario, not against the diff.

## Inputs

1. The findings list, verbatim from the review: severity, `file:line`, what breaks.
2. The fix range (`<base>..<head>`) or the diff.
3. The gate output from the orchestrator's own run (tests, lint). Read it; never re-run it.

No findings list → stop and ask for it. Without one, this turns into a review.

## Procedure

For each finding, in order:

1. Find the change in the diff that addresses it. None → **NOT FIXED**.
2. Read only that change and the lines around it. Under the finding's own input, is the described failure gone?
   - Yes, fully → step 3.
   - Only for some cases or some of the call sites the finding listed → **PARTIAL**, naming what is left.
   - The code changed but the failure still happens → **NOT FIXED**.
3. Find the test. Would it fail on the old code, for the reason the finding gave?
   - Yes → **FIXED**.
   - No test, and the finding is testable → **FIXED, UNTESTED**.
   - The test checks only structure (a function exists, a mock was called), not the behaviour → **FIXED, UNTESTED**, with the reason.
4. A finding the orchestrator chose not to fix (accepted, deferred, judged wrong) → **NOT ADDRESSED (by decision)**, quoting the decision. Never FIXED.

Then two passes over the diff, and only these two:

- **Unrequested changes:** each hunk no finding asked for, one line each. Don't judge them; the reader decides.
- **Obvious regression in the touched lines:** only what plainly breaks on its face: a removed guard, an inverted condition, a deleted or skipped test, a call site the change missed. One line each, marked `REGRESSION?`. Anything that needs analysis beyond the touched lines is out of scope: say so and stop.

## Do / don't

| Don't | Do |
|---|---|
| "I also noticed the error message could be clearer." | Nothing. It is not a finding and not a regression. |
| "Fixed." | "FIXED: `site.js:23` pushes the returned function; `site.test.js:345` fails on the old `.stop` push." |
| Re-read the whole module to judge overall quality | Read each finding's hunk and a few lines around it |
| Re-run the test suite | Read the gate output the orchestrator ran |
| Mark FIXED because a test was added | Check the test reproduces the finding's failure scenario |

## Output

```
## Fix check: <range>

| # | Finding (severity) | Verdict | Where | Test |
|---|---|---|---|---|
| 1 | <one line> (IMPORTANT) | FIXED | path:line | test-file:line |
| 2 | <one line> (MINOR) | PARTIAL: <what is left> | path:line | — |

Unrequested changes: <hunks, or "none">
Possible regressions: <REGRESSION? lines, or "none">
Verdict: <ready to merge | send back rows N, M>
```

"Ready to merge" needs every CRITICAL and IMPORTANT row FIXED (UNTESTED only where the finding cannot be tested) and no `REGRESSION?` line. Otherwise send back exactly the failing rows, to the implementer, and run fix check again on the next fix diff.

## Where it runs

- Inline in the orchestrator when the fix diff is small (a few hundred lines): cheaper than a dispatch.
- Otherwise one read-only sub-agent, given the findings list, a diff file and the gate output. Never parallel reviewers, never the highest effort.
- `comprehensive-code-review` stays for the one review per stream or phase and the final pass. Fix check replaces every review after those.
