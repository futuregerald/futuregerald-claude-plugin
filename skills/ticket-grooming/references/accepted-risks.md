# Accepted risks

Things this skill knowingly does not guard against, why, and what would change the decision.

A risk recorded here has been considered and accepted. A risk *not* recorded here has not been
considered — if you find one, add it with a reason or fix it. "We never thought about it" and "we
decided it was fine" look identical in a codebase unless someone writes down which it was.

---

## 1. The orchestrator posts a document it has not read in full

**What.** The investigation sub-agent writes `notes.md`; the orchestrator reads only the visible
summary before posting. The collapsed details block is posted without the orchestrator reading it.

**Why accepted.** Reading the full notes back into the orchestrator's context is the single largest
cost this design removes — it is most of the reason the change exists. Three things bound the risk:

- On every path except cosmetic, a staff reviewer has read the details block in full and attacked it.
- `visible.md` is generated from the same bytes that get converted, so the orchestrator reads the
  real input to the posted document, not a sub-agent's summary of it.
- `scripts/md2adf.py` warns on local paths, which is the concrete leak this would otherwise enable.

**Not accepted on the cosmetic path**, where no reviewer runs. There the orchestrator reads the
whole file — it is short by construction.

**Revisit if.** A note is ever posted containing content nobody intended, or the cosmetic path grows
long enough that reading it in full stops being cheap.

---

## 2. `{TICKET_KEY}` validation is an instruction, not an executable gate

**What.** The key is interpolated into a shell command and a filesystem path. `SKILL.md` states the
pattern it must match and says to reject anything else, but nothing enforces that.

**Why accepted.** The key comes from the operator or a URL they pasted — it is not derived from
ticket content, a webhook, or any attacker-influencable source. Enforcing it at every call site
would mean wrapping each tracker command in a validating script: a real layer of indirection for an
input a human typed.

**Removed, for the filesystem half.** A SAST scan flagged the key reaching the filesystem, and the
answer turned out to be deletion rather than validation: `scripts/md2adf.py` now takes **no path
argument at all**. It reads `notes.md` and writes `notes.visible.md` — both module constants — in
the directory it is started in. The ticket key lives in the caller's `cd`, so no externally-supplied
value travels into a file path. What remains accepted is the *shell* interpolation in the tracker
command, which no script mediates.

It took four attempts, and the failures are the useful part. `realpath` plus a same-directory check
was genuinely weak — it followed a symlink and then compared the result to the symlink's own parent.
A `--base` flag contained better, but the base was itself an argument, so untrusted input still
chose the safe directory. An allowlist regex plus a working-directory base was actually safe, and
the scanner still flagged it, because a validated string is still a string that reached `open()`.
Only removing the parameter removed the finding. **Two lessons: a SAST finding that survives your
fix is evidence the fix is weak, not that the scanner is noisy — and the strongest fix for "is this
input sanitised?" is usually to stop taking the input.**

**Revisit if.** The skill ever takes the key from ticket content, a queue, a webhook, or any
automated feed. At that point the source becomes untrusted and the remaining instruction is not
enough.

---

## 3. There is no automated drift check between the two plugin copies

**What.** This skill is maintained in two repos. Nothing verifies they stay in sync.

**Why accepted, narrowly.** This is the failure that caused the re-sync this file ships with — one
copy fell three months and seven reference files behind. It is accepted only because the fix
removed the *cause*: company-specific values now live in `references/config.md`, so the copies no
longer diverge by design and a sync is a copy rather than a merge. That lowers the likelihood; it
does not detect a recurrence.

**Revisit if.** The copies diverge again in any file other than `config.md`. A CI job diffing the
two trees, excluding that one file, is the obvious guard and is worth building the first time this
recurs.

---

## 4. Instructions have no test suite

**What.** Most of this skill is prose an agent follows. Nothing automatically verifies that a
change to the prose still produces correct behaviour.

**Why accepted.** No good mechanism exists. What is done instead: cross-references are checked
mechanically, `scripts/md2adf.py` has fixture tests, and prompt changes are validated by running a
real ticket through and asking the agent to self-report what it read and what confused it. That
self-report has caught genuine defects — a template that both required and banned headings, a
length rule that became uncountable, a name the agent invented because none was supplied.

**Revisit if.** A prose change ever ships a defect that a cheap check would have caught. The
cross-reference and placeholder-definition checks already run; extend those before inventing
anything heavier.

---

## 5. `md2adf.py` handles a subset of Markdown

**What.** Link hrefs stop at the first `)`. Backticks inside `**bold**` render literally. An
unclosed backtick pairs with the next one on the line.

**Why accepted.** All three produce *odd text*, never invalid ADF — the document still posts and
still renders. A full CommonMark parser is a large dependency for a converter whose input is
written by an agent following a template that does not use these constructs.

**Revisit if.** A posted note is ever mangled by one of them, or the templates start calling for
Markdown the converter does not cover. The limits are listed in the script's docstring; keep that
list current.

**Not accepted: regex backtracking.** A subset parser is fine; a parser that hangs on its own input
is not. CodeQL found catastrophic backtracking in the trailing-rule stripper — 24 repetitions took
5.6 seconds, growing exponentially — caused by nesting `\s*` inside a `+` group where `\s` also
matched the newline the group consumed. Both that and the table-separator test are now
character-scans rather than regexes. When adding a pattern here, do not put a repeated whitespace
class inside another repetition; the input is agent-written and nobody reads it before it is parsed.

---

## 6. Deferred: splitting the staff review across two models

**What.** The reviewer's mechanical checks (does this entity exist, is there a verification trail)
would run fine on a cheaper model, concurrently, leaving only the judgement calls on the expensive
one.

**Why deferred, not rejected.** Worth roughly 40% of review wall-clock, but it requires merging two
verdicts into one, and a split verdict is a new way for a review to be wrong. The cost it removes is
not currently the bottleneck.

**Revisit if.** Review latency becomes the thing people complain about.
