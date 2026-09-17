# Report format

Two artefacts, same numbers. The Markdown carries the reasoning; the workbook carries the data
someone will sort, filter, and paste into a deck.

## The Markdown report

Order matters — it is built for someone who will read the first screen and skim the rest.

### 1. What changed since the source document
**Lead with this.** A table of every discrepancy found: item, what the document said, what the
tracker says, why it matters. This is the part the reader cannot get anywhere else, and it is the
strongest evidence that the rest of the report was actually verified.

### 2. Headline
Three to six sentences. The bottom line on the target date, the one or two items that decide it, and
the biggest risk. No preamble.

### 3. TL;DR per item
2–4 sentences each, scannable, grouped by phase or milestone. Each carries: real status, what is
genuinely left, the estimate range, and the one thing that would change it. A reader who stops here
should still be able to run the meeting.

### 4. Estimates — a table, not prose

**The report needs its own estimates table.** Numbers scattered through the TL;DR prose do not
satisfy a request for estimates: prose carries the one-engineer figure at best, and silently drops
the 2-engineer case, effort level, parallelisability and readiness. Generate the table from the same
manifest the workbook uses so the two cannot drift.

Columns: item · key · readiness · size · **Done so far** · **Remaining (1 eng opt/pess)** ·
**Remaining (2 eng opt/pess)** · **TOTAL** · max useful engineers · parallelisability · confidence.

Two things to state under it, because readers get both wrong:

- **The estimates do not sum.** Items run in parallel across N engineers, so the date is set by the
  longest *chain*, not the total. Name the chain and its length.
- **Which rows are out of scope for the target date.** Post-GA or Q4 work belongs in its own clearly
  separated block that contributes nothing to the arithmetic — but keep it, sized, because "what's
  after this" is a real planning question. Deleting it is as wrong as counting it.

**Put the headline numbers on the landing view too.** Whatever sheet or section opens first must carry
Done / Remaining / TOTAL. A reader who opens the artefact, sees no numbers, and concludes there are no
estimates is not being careless — the artefact misled them.

### 5. Who is working on what
Engineer → item table, plus who is unallocated and who is overloaded. Then competing non-initiative
load, because that is the honest reason forecasts slip.

### 6. Sequencing recommendation
What to do in what order, and why — dependency order first, then the risk-reduction argument.
Name what to start now, what to start next, and **what to explicitly not start**, which is usually
the most useful half. If the target date cannot hold, say so here with the arithmetic behind it,
rather than burying it.

### 7. Detailed justification per item
The thorough section. Per item: scope decomposition with child counts, code evidence with PR links,
the estimate derivation showing its arithmetic, dependencies, parallelisability with the seams named,
and risks. This is where a skeptical reader goes to check a number, so show the working.

### 8. Findings from the adversarial review
What the review caught and what changed as a result. Short, but present — it tells the reader which
claims were stress-tested.

### 9. Open questions
Only what a human can answer: undecided product calls, PTO and staffing, priority conflicts,
ownership disputes. Each with who owns it and what it blocks. A question you could have researched
belongs in the research, not here.

**Fold in every unanswered question found in the trackers' comment threads**, with its age and who
asked it. Those are already-articulated decision debt — somebody cared enough to ask and nobody
replied — and they are invisible in every roadmap view because no field changes when a question goes
unanswered. Sort by age × consequence: an eight-week-old unanswered question blocking a Beta item is
usually the cheapest schedule fix in the whole report, because clearing it costs somebody five minutes.

### Linking — every reference is clickable, in both artefacts

**Every ticket key, PR number and file path in the Markdown report is a link.** A reader who has to
copy `ABC-101` into a search box to check your claim will not check it, which defeats the point of
citing evidence at all. Link them as you write, then verify with a pass that strips complete
`[text](url)` links and greps the remainder for bare keys — a naive "is it linked" check will match
the key *inside* its own URL and report false clean.

Two regex traps worth pre-empting, both of which silently skip real references: a key followed by `)`
— as in "Search reindex (ABC-101)" — and a key after a `/`, as in "ABC-101 / ABC-102". Also protect
fenced and inline code from linkification.

In the workbook, key and PR columns use the `link` style so they are clickable there too.

### 10. Method and limits
How pace was measured, the constants used, and what the estimates assume. State the limits plainly —
a report that admits what it does not know is the one people trust the second time.

## The workbook

Build it with `scripts/build_workbook.py` from a JSON manifest. Sheets, in order:

| Sheet | Contents |
|---|---|
| **Summary** | One row per item: area, item, key(s), tracker status, source-doc status, verdict, % complete + basis, owner, confidence, link |
| **Estimates** | Item, remaining scope, 1-eng optimistic/pessimistic (weeks), 2-eng optimistic/pessimistic, max useful engineers, parallelisability note, effort level, confidence |
| **Current Assignments** | Engineer, item, key, status, since, open PRs, notes |
| **Sequencing** | Order, item, start-when, why, blocked-by, parallel-with |
| **Dependencies** | Item, depends on, owning team, status of the dependency, date if any, impact if it slips |
| **Open Questions** | Question, owner, blocks, why it matters |
| **Method** | The planning constants with provenance, and the estimate assumptions |

Rules for the cells:

- **Every estimate cell carries its confidence**, because a number in a spreadsheet loses its
  caveat the moment someone copies it. Low-confidence numbers say so in the cell itself.
- Never write a bare number for an unticketed item. Use the range plus `(placeholder — needs scoping)`.
- Status cells use the `status` style so the colour matches the tracker's own vocabulary.
- Ticket and PR columns use the `link` style so they are clickable.
- Keep the source document's own status in its own column next to the verified one. Seeing them side
  by side is the point.

## Handing it over

Give clickable file links for both artefacts and the research directory. Lead the chat response with
section 1 — what changed — then the headline, then the open questions. Everything else is in the
files; do not re-narrate it in chat.
