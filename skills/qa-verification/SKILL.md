---
name: qa-verification
description: "Verifies that shipped work actually meets a ticket's acceptance criteria, by both reading the code and running it. Every finding is confirmed twice — traced in the source and reproduced against a running app — then checked against the tracker and git history so existing tickets are linked rather than re-filed. Requires a runnable local environment; asks the user rather than falling back to a code-only review. Produces a QA report written for a product manager and posts it to the ticket. Use when asked to QA a ticket, test whether acceptance criteria were met, verify an epic or story was really delivered, or check whether shipped work matches what was asked for."
tags: [workflow, testing, project-management]
model: opus
---

# QA Verification

Confirm that what shipped matches what was asked for, and write it up so a product manager can act on it.

This is routine quality work. Do it because the work is worth checking, not because someone complained that it hadn't been — and write it that way.

**Announce at start:** "Using qa-verification to test [ticket] against its acceptance criteria."

## The rule that matters

**Every finding is verified twice: in the code, and by running the code.**

Reading the source tells you what *should* happen. Running it tells you what *does*. A claim resting on only one of those is not a finding yet — either finish it or label it unverified. Most defects that survive code review are caught by the second half, because the code reads correctly and still never runs the way the ticket assumed.

If you catch yourself writing "the code does not call X" without having watched it not call X, stop and go run it.

## 1. Get the app running first — this is a prerequisite, not a step

**Without a running local environment you cannot do this work.** Half of every finding comes from running the code, so a QA pass on an app you cannot start is not a QA pass.

Find how this project runs locally: its README, `CONTRIBUTING`, `CLAUDE.md`, a compose file, a setup script, or a separate repo the project points at for local infrastructure. Start it and confirm it actually responds before testing anything.

**If you cannot get it running, stop and ask the user.** Do not quietly fall back to a report based only on reading the source — that is the exact failure this skill exists to prevent, and a report that looks the same but is half-verified is worse than no report.

When you ask, keep it short and specific:

- what you looked for and what you tried
- whether they already have the environment set up, and where it lives
- a link to the setup instructions or local-environment repo **taken from this project's own docs** — never a guessed URL
- **whether they want a source-only report instead.** Offer it honestly: reading the source does find real problems, but nothing in it is confirmed by running code, logs, or actual request and response behavior. It is a weaker deliverable, not a worthless one — let them decide.

Then wait. If they choose source-only, `references/report-format.md` covers how to label it so no reader mistakes it for a verified one.

## 2. Collect the acceptance criteria

Three sources, all required:

- The ticket's own acceptance criteria list.
- **The comments.** Anything a PM or stakeholder asked for is an acceptance criterion, listed or not. Read the entire thread, including questions that were answered and questions that were not.
- Child issues: what shipped, what was dropped as Won't Do, what got filed as a bug afterwards.

Write them out and number them before testing anything, so the report can reference them and so a criterion cannot quietly go unchecked.

## 3. Trace each criterion in the code

Find the code implementing each one. Record file and line. Form a specific prediction — meets, fails, or partial, and under exactly what input.

A prediction is not a result. It tells you what to go run.

## 4. Run it

Full guidance in `references/running-the-code.md`. In short:

- Run it against a **real running app**, not a mental model.
- Backend-only change: drive the API directly with appropriately seeded users, or call the real services and jobs.
- UI change: drive Chrome when it is available. Never block on it — fall back to the API and say which you used.
- **Positive control first.** Before claiming something does not happen, prove your setup can observe it happening. Otherwise you cannot tell a real absence from a broken harness.
- Leave no residue: no rows, no flag changes, no stray files.

## 5. Check the tracker and the git history before calling anything new

Two searches, both required, before any finding is called new:

**The tracker.** Search for each finding. Label it **already filed** with a link, or **new**. A report that re-files known work wastes the team's time and makes the rest of it less trustworthy.

Read the near misses too, not just the exact matches. A closed ticket covering adjacent behavior usually sharpens the finding: it may show the requirement was met and yours is a *different* requirement, or that one code path was fixed and a sibling path was left behind. Either reframing is more useful than the finding on its own.

**The git history.** Check whether a later change already fixed it — a ticket marked Done tells you someone believed it was fixed, not that it is. Search merged pull requests and the log for the files involved. Testing against current `main` rather than the branch the ticket shipped on settles this by construction; say in the report which commit you tested. Check the reverse too: before citing an open ticket as covering a finding, confirm its fix has not already landed.

## 6. Check whether the existing tests catch it

Run the relevant suite. For each finding it misses, say *why* it misses — usually one of:

- the test asserts the current behavior as correct, locking the defect in
- the behavior is outside anything the tests assert
- there is no test for that path at all

This tells the team where to add coverage.

## 7. Write the report and post it

Structure, language rules, and posting mechanics: `references/report-format.md`.

Publish it as an artifact for sharing, and post a version to the ticket itself.

## Rules of thumb

- **Nothing environment-specific in the report.** No database names, row counts, ports, local paths, or seed data. They mean nothing to a reader on a different machine and nothing at all to a PM. Keep those in the working notes.
- **Never address individuals with questions.** Anything unresolved goes in a single Open questions section at the end.
- **Open with method and a TL;DR**, not with why the report exists. No framing the work as a response to someone's request.
- **Plain language.** Say what breaks, when, and what happens as a result. Where a technical term is genuinely needed, use it and explain it in a clause.
- **Full evidence collapsed by default.** Verdict and findings visible; measured output, file references and method behind a collapsed block.
- **Report faithfully.** If a scenario was not reproduced, say so plainly and say what you did instead. A single overclaim discredits the whole report.
