# Report Format

The report is read by a product manager first and an engineer second. Write for that order.

## Structure

1. **How this was checked** — the methodology, first. See below.
2. **TL;DR** — one or two sentences. What works, what does not, how many defects, and the one to fix first.
3. **Acceptance criteria** — a table, one row per criterion, each Met / Partial / Failed with a one-line note. Include criteria taken from comments, visibly marked as such.
4. **Findings** — one entry each, ordered by severity. Title says what breaks. Then: what happens, the measured evidence, where in the code, and why it matters.
5. **Why the tests did not catch it** — per finding or grouped by reason.
6. **Open questions** — anything unresolved, as questions. No names attached.
7. **Next steps** — what to fix first and why, what to file, and any decision the team owes.

Full evidence — measured output, file and line references, per-finding detail — goes in a **collapsed block**, closed by default. The verdict, criteria table, and finding summaries stay visible.

## Methodology section

Near the top, so a reader knows how much to trust the rest. Cover, in a few lines:

- That every finding was verified **both ways**: traced in the code and reproduced against a running app.
- How the code was driven — real API requests as seeded users, real services, browser, or a mix.
- Anything stubbed, and why.
- Any finding **not** fully reproduced, named explicitly.
- That the environment was left as found.

Keep it about method, not machine. "Driven through real API requests as an authorized user" — not which database or port.

## Source-only reports

When the user has chosen a report without a running app, the content is the same but the labelling is not. A reader must never mistake it for a verified pass.

- **Title it as one.** "Source review" or "Unverified source review", not "QA verification".
- **First line of the methodology says so**, before anything else: findings come from reading the code and have not been confirmed by running it.
- **Every finding is a prediction.** Write "this would return X" rather than "this returns X", and drop the measured-evidence section — quote the source instead.
- **Say what is out of reach**, briefly: whether a path actually runs, what the logs emit, what a request returns, feature-switch state, timing and background work.
- **No verified/reproduced counts** in the TL;DR. Count findings, not confirmations.
- **Add a next step**: which findings most need confirming once an environment is available, ordered by how much rests on them.

The tracker and git-history searches still apply in full — they need no running app, and they are what stops a source-only report re-filing known work.

## Every finding needs

- **What breaks**, in plain words.
- **When** — the specific input or timing that triggers it.
- **What happens as a result** — the consequence someone would notice.
- **Measured evidence** — actual output from a run, not a description of it.
- **Where** — file and line.
- **Ticket status** — a link to the existing ticket, or marked new.

Severity reflects consequence, not effort: money, data loss, or work silently disappearing rank above cosmetic or observability issues.

## Language

- Plain words over jargon. Where a technical term is needed, use it and explain it in a clause: "the deadline is set by adding 48 hours and rolling to the next weekday, which skips over the weekend instead of stepping across it."
- Name what goes wrong, not its category. "A restaff on a Friday gets one working day instead of two" beats "SLA computation exhibits boundary-condition drift."
- Lead with the answer; reasoning after.
- No emoji. No filler.

## Evergreen output

The report should read the same in six months on someone else's machine. Leave out:

- database names, row counts, record identifiers, ports, file paths from your machine
- seed data specifics and anything about your local setup
- references to when you ran it, beyond a single date stamp

Keep the numbers that demonstrate the defect — a payout amount, a count of rounds, a day-of-week table. Those are properties of the code, not the environment.

## Open questions

Anything you could not resolve becomes a question here. Do not tag individuals or address anyone by name — the report may be read by people who were not part of the original thread, and a question aimed at one person reads as noise to everyone else. State the question and what turns on the answer.

## Posting to the ticket

Publish the full report as an artifact for sharing, then post a version as a comment on the ticket.

For Jira, post as ADF (Atlassian Document Format — Jira's own rich-text JSON) so tables and the collapsed block render. Markdown does not produce a collapsible section; an `expand` node does.

```bash
acli jira workitem comment create --key TICKET-123 --body-file report.json
```

Build the ADF document with these node types:

| Need | Node |
|---|---|
| Collapsed evidence | `expand` with `attrs.title` |
| Verdict callout | `panel` with `attrs.panelType` |
| Criteria table | `table` / `tableRow` / `tableHeader` / `tableCell` |
| Measured output | `codeBlock` |

**Always verify what actually rendered**, by reading the comment back as JSON and confirming the node types are present. A malformed document can post as a wall of raw text:

```bash
acli jira workitem view TICKET-123 --fields comment --json
```

Confirm the newest comment contains `expand`, `table`, and `codeBlock` nodes rather than one long `text` node.

## Tone

Describe gaps in delivered work factually, not as blame. The useful sentence is what is broken and what it costs — not who missed it. Where the ticket itself was ambiguous, say that; it is usually the real cause.

**Do not frame the report as a response to a request or a complaint.** QA is routine quality work and gets done whether or not anyone asked. Opening with "the comments asked for testing and none had been done" reads as pointing at a person, and it dates the report to one thread. Open with how it was checked and what was found.

The same applies inside the report. A criterion drawn from a comment is a requirement like any other — record what was asked for, not that someone had to ask. When a test locks in a defect or a criterion went untested, that is a coverage gap to close, not a lapse to attribute.
