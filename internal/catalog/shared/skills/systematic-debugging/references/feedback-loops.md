# Building a Feedback Loop

**This is the heart of debugging.** A tight pass/fail signal that goes red on *this* bug turns diagnosis mechanical — bisection, hypothesis-testing, and instrumentation all just consume it. Without one, no amount of staring at code will save you. Spend disproportionate effort here. Be aggressive, be creative, refuse to give up.

## Ways to construct one — try roughly in this order

1. **Failing test** at whatever seam reaches the bug — unit, integration, e2e.
2. **Curl / HTTP script** against a running dev server.
3. **CLI invocation** with a fixture input, diffing stdout against a known-good snapshot.
4. **Headless browser** (Playwright / Puppeteer) — drives the UI, asserts on DOM / console / network.
5. **Replay a captured trace** — save a real request / payload / event log to disk, replay it through the code path in isolation.
6. **Throwaway harness** — a minimal subset of the system (one service, mocked deps) that exercises the bug path in a single call.
7. **Property / fuzz loop** — for "sometimes wrong output", run 1000 random inputs and look for the failure mode.
8. **Bisection harness** — if the bug appeared between two known states (commit, dataset, version), automate "boot at state X, check, repeat" so you can `git bisect run` it.
9. **Differential loop** — run the same input through old-version vs new-version (or two configs) and diff the outputs.
10. **HITL bash script** — last resort when a human must click. Drive *them* with `scripts/hitl-loop.template.sh` so the loop stays structured; captured output feeds back to you.

## Tighten the loop

Treat the loop as a product. Once you have one:

- **Faster** — cache setup, skip unrelated init, narrow the test scope. A 2-second deterministic loop beats a 30-second flaky one.
- **Sharper** — assert the specific symptom, not "didn't crash".
- **More deterministic** — pin time, seed RNG, isolate filesystem, freeze network.

## Non-deterministic bugs

The goal isn't a clean repro but a **higher reproduction rate**. Loop the trigger 100×, parallelise, add stress, narrow timing windows, inject sleeps. A 50%-flake bug is debuggable; a 1% one is not — keep raising the rate until it's debuggable.

## Performance regressions

Logs are usually the wrong tool. Establish a **baseline measurement** (timing harness, `performance.now()`, profiler, query plan), then bisect. Measure first, fix second.

## Red-capable completion criterion

The loop is ready when you can name **one command** — a script path, a test invocation, a curl — that you have **already run at least once** (paste the invocation and its output) and that is:

- **Red-capable** — drives the actual bug code path and asserts the user's exact symptom, so it can go red on this bug and green once fixed.
- **Deterministic** — same verdict every run (flaky bugs: a pinned, high reproduction rate).
- **Fast** — seconds, not minutes.
- **Agent-runnable** — you can run it unattended; a human in the loop only via `scripts/hitl-loop.template.sh`.

If you catch yourself reading code to build a theory before this command exists, **stop** — jumping straight to a hypothesis is the exact failure this prevents.

## When you genuinely cannot build a loop

Stop and say so explicitly. List what you tried, then ask your human partner for one of: (a) access to an environment that reproduces it, (b) a captured artifact (HAR file, log dump, core dump, timestamped screen recording), or (c) permission to add temporary production instrumentation. Do **not** proceed to hypothesise without a loop.

## Minimise the repro

Once it's red, shrink the repro to the **smallest scenario that still goes red**. Cut inputs, callers, config, data, and steps **one at a time**, re-running after each cut — keep only what's load-bearing. Done when removing any one remaining element makes the loop go green. A minimal repro shrinks the hypothesis space and becomes the clean regression test.
