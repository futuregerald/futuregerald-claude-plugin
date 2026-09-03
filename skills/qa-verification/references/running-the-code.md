# Running the Code

How to actually exercise the behavior, so a finding rests on observed output rather than a reading of the source.

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
