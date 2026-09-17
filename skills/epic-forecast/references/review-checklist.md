# Adversarial review of the research output

Runs in a **fresh sub-agent** that did not do the research. Give it the Phase 1 baseline table and
the whole `research/` directory. Its job is to find what is wrong before the report launders it into
something that looks authoritative.

The orchestrator also spot-checks the highest-stakes claims itself, directly against the tracker.
Anything that changes a date, a status, or an owner gets verified by hand. A claim you carry into the
report is a claim you own.

## The failure modes, in rough order of how often they bite

**1. Status asserted without a citation.** Or worse, citing the source document — the very thing
being audited. Every status needs a tracker field or a PR behind it.

**2. "No ticket exists" with no query stated.** Absence of evidence dressed as evidence. An
unticketed Beta blocker is a big claim; it needs the JQL, the projects searched, and both a label
search and a text search.

**3. A percentage-complete with no basis.** Ask: child count, story points, or merged PRs? Each
gives a different answer and they routinely disagree by 30 points. Whichever is used, it gets named.

**4. Contradictions between agents.** Clusters overlap at their boundaries and that is exactly where
ambiguity lives. Diff every item two agents both touched. A contradiction is a finding to resolve,
not a formatting problem to smooth over.

**5. Estimates that ignore the measured constants — in either direction.** An estimate implying
throughput the team has never achieved is wrong. So is one built on a **team-wide average rate when
per-epic rates were available**: that is the single most common defect here, it inflates every
estimate, and it is invisible unless you check the rate an estimate actually divided by. For each
sizing, ask *which engineer's measured rate is this, on which epic?* "The team average" is a wrong
answer whenever the epic has an owner with history.

**5a. A blocked epic priced as a slow one.** An epic closing 0 children/week must be estimated as
"N weeks after the block clears", with the calendar date reported unbounded. If you see a long
duration where the evidence says *blocked*, the estimate is answering the wrong question and will
send the reader to hire people instead of clearing the blocker.

**5b. Double-discounting.** An observed closure rate already contains interrupts, review waits and
on-call. Applying a focus factor on top of it discounts twice. Check whether any estimate did.

**5c. Mixed units.** Engineer-days quoted in a table of calendar weeks reads as roughly half its
real length. Flag every estimate whose unit differs from the artefact's.

**5d. An item judged without its comment thread.** If an agent called something stalled, dormant or
on-track without quoting a comment, the judgement rests on fields alone and is unreliable. Check that
each item reports its status-update cadence and its unanswered questions — and that any epic reported
as having *zero* comments really does, because that is a finding in its own right, not an absence of
one.

**5e. Unanswered questions dropped on the floor.** Questions sitting unanswered in comment threads are
blockers with no ticket. If the research surfaced them but the report has no home for them, they will
vanish exactly as they did in the tracker. Every one belongs in the open-questions output with its age.

**5f. Moot children counted as remaining work.** For every epic with an open PR, check whether any
"remaining" child describes code that PR deletes or replaces. This is the largest single inflator of
estimates and it is invisible in the tracker, because a moot ticket still reads To Do.

**5g. Remaining exceeds Total.** If both are reported and TOTAL was computed independently rather than
as Done + Remaining, this happens whenever remaining includes review or discovery time. It is an
obvious tell that the two numbers came from different models.

**5h. The rate is the bottom of the measured range.** Check which engineer's rate each estimate divided
by. A rate depressed by review latency is a queue, not a capacity, and planning at it points the reader
at hiring when the fix was a code review.

**6. Inherited optimism.** The source document says "finishing next week"; the agent repeats it with
new formatting. Test it: do the open children, their assignees, and the PR activity actually support
the claim? Say what the tickets support, not what the note asserts.

**7. Undefined scope silently priced as small.** "Fit and finish items" with no tickets are not
zero. If the agent gave a tight range to an item whose scope is one sentence, the range is fiction.

**8. Dependency asymmetry.** An item whose "remaining work" list is shorter than its dependency list
has not been thought through. Both directions get named: what blocks it, what it blocks.

**8a. A dependency called "blocked" that is actually provisioned.** A "DO NOT MERGE" draft PR, a
local snapshot, a service running on someone's laptop — these are frequently a deliberate test double
that *unblocks* the consuming team, not evidence of a stalled dependency. The tracker cannot tell you
which; it looks identical either way. **Before writing "blocked", ask the owning team, and check
whether the consumer has actually built against it.** Getting this backwards inverts the recommendation
— it sends the reader to chase another team when the real constraint is inside their own.

Corollary: when an epic shows 0 closures but is *not* blocked, the cause is usually review, merge
strategy, or the owner being on something else. Check which before pricing it as slow.

**9. A completed spike with no successor ticket.** Reliably becomes forgotten work. So does an epic
closed Won't Do whose scope was "absorbed" somewhere unnamed — check that the absorbing ticket really
contains it.

**10. Stale assignees.** Epic-level assignees are frequently months out of date. Cross-check against
who is actually moving child tickets and merging PRs.

## Output

For each finding: severity (CRITICAL / IMPORTANT / MINOR), the file and section, what is wrong, the
evidence, and the corrected claim if determinable.

- **CRITICAL** — would change a date, an owner, or a go/no-go decision.
- **IMPORTANT** — a wrong or unsupported number, or a missed dependency.
- **MINOR** — imprecision that does not change a conclusion.

Fix every CRITICAL and IMPORTANT before writing the report: re-query, send the agent back with
`SendMessage`, or downgrade the claim to UNVERIFIED. Record what changed — the review's own findings
belong in a short section of the report, because *the source document was wrong in these specific
ways* is one of the most useful things the exercise produces.
