# Running the Code

How to actually exercise the behavior, so a finding rests on observed output rather than a reading of the source.

## Start with a running app

This is the prerequisite for everything else here. Get it up before collecting criteria, so you find out early if you cannot.

Where projects record it: the README, `CONTRIBUTING`, `CLAUDE.md`, a `docker-compose.yml`, a `Procfile`, a `bin/setup` or `bin/dev` script, or a separate repository holding the local infrastructure for a group of services. Check the local machine before assuming nothing exists — the environment may already be set up, or partly running.

Confirm it genuinely responds before trusting it. Supporting services running is not the same as the app being up, and an app booting is not the same as it answering a request. Make one real request and look at the response.

Common things that block a start, worth checking before escalating: a required credential or token that is missing, a service the app depends on that is not running, or migrations that have not been applied since the last pull.

### If you cannot get it running

Stop and ask. Do not carry on and produce a report from reading the source alone — it will look exactly like a real QA report while being half-verified, which is worse than not writing one.

Keep the ask short:

- what you looked for and what you tried
- whether they already have the environment set up, and where it lives
- a link to the setup instructions or the local-environment repo, **taken from this project's own documentation** — never a guessed URL

Then wait. If the user says to proceed without it, say plainly in the report that findings were not reproduced and mark every one of them unverified.

## Pick the thinnest path that is still real

In order of preference, take the first one that genuinely exercises the change:

| Change | How to drive it |
|---|---|
| Backend / API only | Real HTTP requests against the running app, as a correctly seeded and authorized user |
| Service, job, interactor | Call the real class with real records; stub only what reaches outside the system |
| UI | Chrome against the running app |
| Scheduled or time-based | Freeze or travel time, then invoke the real entry point |

Never assert on a stub you installed yourself. If the only thing your scenario proves is that your own double was called, it proves nothing.

## Seed users properly

Most backend paths are gated by authorization, membership, or a role. Seed a user that legitimately holds the access under test, then make the request as that user. A request that 403s because the user was wrong is not evidence about the feature.

When testing whether someone *should not* be able to do something, run it as a user who genuinely holds that role. "The endpoint rejected me" means nothing if you were never entitled in the first place.

## Chrome

Drive the browser when the change is visible in the UI and Chrome is available. It catches the class of defect where both sides are individually correct and never meet — a field the frontend reads that the backend serializer does not send, for instance.

Do not block on it. If the browser is unavailable or the path is awkward to reach, fall back to the API and state in the report which you used.

## Positive controls

Before reporting that something does not happen, prove your setup can observe it happening.

The failure mode: a run shows nothing occurred, and you report the absence — when in fact your harness could never have seen it. Run the working path first and watch it produce the signal. Only then does the absence on the broken path mean anything.

```
Working path   -> signal observed      <- proves the harness sees it
Path under test -> signal absent        <- now this is evidence
```

Same idea in reverse for a guard: remove the guard and confirm the test goes red. A guard test that passes with the guard deleted was never testing the guard.

## Things that fake an absence

Any of these can make working behavior look missing. Rule each out before reporting an absence:

- **Feature flags off.** The path is gated and never ran. Prefer stubbing the flag check for the duration of a run over toggling the flag, which leaves residue and affects anyone else on that environment.
- **Deferred side effects.** Jobs and hooks that fire on transaction commit never fire inside a transaction you roll back. Either disable the deferral for the run or observe outside the transaction.
- **Async work.** A queued job has not run yet. Run it inline or wait on it.
- **Caching.** A stale cached read hides a fresh write.
- **A stub swallowing the call** you were trying to observe.

## Leave no residue

Prefer a transaction that rolls back. Where that is not possible, record what you created and remove it.

Before finishing, confirm the environment is as you found it: record counts unchanged, feature flags in their original state, no leftover test files, any service you started shut down. State this in the report's methodology section.

## Working notes vs the report

Keep raw environment detail — identifiers, counts, ports, paths, seed specifics — in your working notes. The report gets the behavior and the measured numbers that demonstrate it, never the coordinates of the machine you ran on.
