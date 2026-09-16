---
name: ticket-grooming
description: "Investigate and groom tickets by dispatching sub-agents for codebase research, history, root cause analysis, and risk assessment. Use when the user says 'groom', 'triage', or asks to investigate, dig into, or work up a ticket. Posts structured Triaging Notes as a comment on the ticket. Default is short-form (what is happening, root cause, fix, estimate, risks, priority) with the full investigation collapsed. Use --full for the complete report. Works with Jira, GitHub Issues, or any ticketing system."
tags: [workflow, project-management, debugging]
model: opus
---

# Ticket Grooming

Investigate one or more tickets, then post structured "Triaging Notes" as a comment. Notes are written for two audiences: PMs who need to understand the issue and prioritize it, and engineers who need to know where to start.

**Announce at start:** "Using ticket-grooming to investigate [ticket(s)]."

## Inputs

Extract from the user's message:
- **Ticket key(s) or URL(s)** (e.g., `ABC-1234`, `https://your-site.atlassian.net/browse/ABC-1234`, `#42`)
- OR a **verbal description** of the issue
- **Flags:**
  - `--dry-run` — preview without posting
  - `--full` — post the full report (no collapsed section)
- **Mode:** `--full` flag > `grooming-mode` in CLAUDE.md > default (`short`)

## Depth Calibration

Not every ticket needs the full pipeline. Match investigation depth to the ticket's complexity.

| Signal | What to run | Staff review? |
|--------|-------------|---------------|
| **Cosmetic** (label, copy, UI text) | Phases 0, 1 (shallow), 5. Skip root cause and risk assessment. | No |
| **Pinned bug** — one symptom, plus a named error, stack trace, or failing assertion that already identifies the code | Full pipeline (phases 0-5) | Yes, but capped: 5 tool calls, restricted to "is this the right problem?" and fix completeness |
| **Standard bug or feature** | Full pipeline (phases 0-5) | Yes, full checklist |
| **Security / data integrity** | Full pipeline + security checklist in staff review | Yes, full checklist |

**Who classifies:** the orchestrator, from the ticket text, **before dispatching**. The depth
decision controls what gets dispatched (whether a reviewer runs at all, and with what budget), so
it cannot wait for the sub-agent's Phase 0. Pass the result through `{REVIEW_BUDGET}` and
`{REVIEW_SCOPE}` on the review prompt.

**How to detect:**

- **Cosmetic** — no logic change, no data change, no authorization change. Use the shallow path.
- **Pinned** — the ticket hands you the failure site. The expensive half of staff review is hunting
  alternative root causes, and a stack trace has already settled that, so the reviewer only needs to
  confirm the notes answer the reported problem and that the fix is complete. A ticket with two
  symptoms is NOT pinned, even if one of them carries a stack trace — divergent symptoms are exactly
  where one plausible cause misleads.
- When in doubt, use the standard path. Misclassifying downward costs a wrong note; misclassifying
  upward costs a few thousand tokens.

## Pre-Flight

Before dispatching sub-agents:

### 0. Resolve configuration

Read [config-resolution.md](references/config-resolution.md) and resolve the config once. It gives
you the ticket system, site, org, repo list with paths and slugs, grooming mode, and whether field
writes may be auto-applied. Do not rediscover these per run.

If nothing resolves, offer to write a `.ticket-grooming.json` cache rather than asking the same
questions on every future run.

### 1. Check the codebase index — and record whether it exists

If `codebase-memory-mcp` is available: call `index_status`, re-index if stale. Sub-agents verify via
`index_status` but do NOT re-index.

**If the server failed to connect, or you are at a directory that indexes many unrelated repos, say
so explicitly in every dispatch prompt.** Both sub-agents branch on this. An unavailable index that
is not declared gets treated as an empty one, and "the graph found no callers" then reads as
evidence when it is nothing of the kind.

### 1b. Validate the ticket key

`{TICKET_KEY}` is interpolated into a shell command and into a filesystem path, so constrain it
before it reaches either: it must match `^[A-Za-z][A-Za-z0-9]*-[0-9]+$` (a tracker key) or
`^[a-z0-9][a-z0-9-]*$` (a slug for a verbal request). Anything else — path separators, `..`, shell
metacharacters, whitespace — is rejected, not sanitised. Ask the user for a valid key instead.

### 2. Detect ticket system

The resolved config (step 0) names the ticket system. If it did not, infer from the URL pattern (`*.atlassian.net` = Jira, `github.com` = GitHub), then the key pattern (`XX-1234` = Jira, `#1234` = GitHub), then ask.

| System | Read | Search | Post |
|--------|------|--------|------|
| Jira | `getJiraIssue` | `searchJiraIssuesUsingJql` | `addCommentToJiraIssue` |
| GitHub | `gh issue view` | `gh issue list`, `gh pr list` | `gh issue comment` |

**acli fallback:** Use `acli` CLI when MCP lacks a capability (deleting comments, bulk edits). `acli --body-file` accepts ADF JSON only.

### 3. Resolve GitHub remote info

For each repo: extract org/repo from `git remote get-url origin`, get HEAD SHA via `git rev-parse HEAD`, verify pushed. Fallback: relative paths instead of permalinks.

### 4. Backend tickets: resolve every configured repo

Take the repo list from the resolved configuration (Pre-Flight step 0) and confirm each path and GitHub slug. Refresh each HEAD SHA — that is the one field worth re-resolving every run.

Search **all** of them, not just the first — a ticket whose code lives in the second repo otherwise comes back as "no code found". If the configuration notes that one repo is migrating into another, search both and propose fixes only in the destination. Skip backend repos entirely for frontend-only tickets.

### 5. Multi-ticket: detect shared context

When grooming 2+ tickets: read all, check for overlapping components. If overlap found, run shared investigation once and pass as context. Otherwise dispatch independently.

### 6. Dispatch sub-agents

- 1 sub-agent per ticket (two-level only — sub-agents don't spawn their own)
- Max 3 concurrent. Queue additional as slots free.
- Each gets fresh context — no shared state between tickets.
- **Pass absolute file paths, never file contents.** Resolve this skill's own directory and
  substitute it for `{SKILL_DIR}`. Never use `${CLAUDE_PLUGIN_ROOT}` — it is unset outside an
  installed plugin, expands to empty, and the sub-agent silently reads nothing.
- **Tell the sub-agent where to write** — `{SCRATCHPAD}/{TICKET_KEY}/notes.md`. Use the session
  scratchpad directory, not `/tmp`.

## Investigation

Dispatch each sub-agent using the template in **[references/investigation-prompt.md](references/investigation-prompt.md)**.

**Read that one file; do not read the rest.** The sub-agent reads them itself from the absolute
paths you pass. Inlining them made the orchestrator pay for the same content twice — once reading
it, once re-emitting it into the prompt — which was the single largest cost in a grooming run.

The sub-agent's own required reading is listed in the Reference Files table at the
bottom of this file — you do not need to read those files yourself.

## Staff Engineer Review

After the investigation sub-agent returns, dispatch a review sub-agent using
**[references/staff-review-prompt.md](references/staff-review-prompt.md)** with `model: "opus"`.

Pass it the **path** to `notes.md`, not the notes themselves — relaying several thousand tokens of
notes through your own context to hand them to another agent is pure waste. Tell it whether the
code index is available (see Pre-Flight step 1).

The review checks:
- Is the note about the reporter's actual problem? (highest priority)
- Are all named entities verified in the codebase?
- Are hypotheses backed by evidence with counterarguments?
- Are visible sections PM-readable? (no unexplained jargon, plain language)
- Correctness, security, and pattern adherence

**Skip staff review only for cosmetic tickets** (per depth calibration above).

## Post the Notes

After staff review:

| Verdict | Action |
|---------|--------|
| **PASS** | Post as-is |
| **PASS WITH NOTES** | Post as-is, mention notes to user |
| **NEEDS FIXES** | Apply the corrections to `notes.md`, regenerate, post, report what changed |

The investigation sub-agent wrote `{SCRATCHPAD}/{TICKET_KEY}/notes.md` and returned a receipt. The
notes live in that file; the receipt is not the notes.

**On NEEDS FIXES:** edit `notes.md` — it is Markdown, so corrections are a text edit, and the
reviewer supplies the replacement text verbatim. Never hand-edit generated JSON. Then regenerate.
If the corrections are extensive, send them back to the investigation sub-agent instead; it still
has the context.

### Build, check, post

Convert the notes. The script **warns on any absolute local path** it finds — those leak your
username and private repo names to everyone who can read the ticket, and the details block is the
part nobody re-reads before it goes out. Fix them in `notes.md` and regenerate; do not post over the
warning. (It splits the file on its marker and writes `notes.visible.md` beside it
— named from the input stem, so two notes files in one directory cannot clobber each other):

```
# run from the ticket's scratchpad directory; the script takes no path
cd {SCRATCHPAD}/{TICKET_KEY}

# short mode (details collapsed behind an expand node)
python3 {SKILL_DIR}/scripts/md2adf.py --title "Full Investigation Details" > notes.adf.json

# full mode (details inline, nothing collapsed)
python3 {SKILL_DIR}/scripts/md2adf.py --no-expand > notes.adf.json
```

**The script takes no path argument.** It reads `./notes.md` and writes `./notes.visible.md`, both
fixed names in whatever directory you run it from. The ticket key lives in the `cd`, so no
externally-supplied value ever reaches a file path — there is nothing to validate and nothing to
escape. If you see `no notes.md in ...`, you are in the wrong directory.

Redirect its stdout to `{SCRATCHPAD}/{TICKET_KEY}/notes.adf.json`.

**Then, in this order:**

1. **Read the notes.** Read `<stem>.visible.md` — the file, not the sub-agent's receipt, because
   the two can disagree and the file is what gets converted. You are about to write to a ticket
   other people read; never post text you have not seen.

   On the **standard, pinned and security paths** the details block may stay unread: a staff
   reviewer has already read it, and it sits behind a collapsed node.

   On the **cosmetic path there is no staff review**, so nothing else has read the details block.
   Read the whole of `notes.md` before posting — it is short by construction on that path.
2. **Assert the document shape.** `jq -e .` only proves the file is JSON. A file that is valid JSON
   but the wrong shape posts as a flat wall of text, or is rejected outright. Run exactly this,
   with `==1` for short mode and `==0` for full mode:
   ```
   jq -e '.version==1 and .type=="doc" and ([..|objects|select(.type=="expand")]|length)==1' \
     {SCRATCHPAD}/{TICKET_KEY}/notes.adf.json > /dev/null
   ```
   A non-zero exit means do not post.
3. **Post:**
   - **Jira, both modes:** `acli jira workitem comment create --key {TICKET_KEY} --body-file {SCRATCHPAD}/{TICKET_KEY}/notes.adf.json`
   - **GitHub:** `gh issue comment` with `notes.md` as-is (GitHub renders Markdown correctly, including the link form Jira strips)

Full mode differs only by the `--no-expand` flag on the converter. **Do not post raw Markdown to
Jira in either mode** — its Markdown converter silently strips the ``[`code`](url)`` link form the
details block is built out of, which is most of the evidence in the note.

Posting details and the ADF node reference live in [adf-posting.md](references/adf-posting.md),
including why short mode needs ADF at all and why HTML `<details>` never works.

**After posting — ask first, do not assume:**

These are writes to a ticket other people own, so they are proposed, not applied:

1. Offer to add the `has_notes` label — and check what the project actually uses first; some teams
   label by area instead.
2. Offer to set the priority field to the value in the Priority section, quoting the severity and
   urgency that produced it. A priority change tells other people what to work on; that is the
   author's call, not the groomer's.

Skip asking **only** when the resolved config sets `auto_apply_fields: true`.

Story points appear in the note text and are **not** written to a Jira field. Most Jira screens do
not expose one — verify before promising otherwise.

**`--dry-run`:** Show the visible summary in conversation, say where `notes.md` is, and ask
"Post to ticket?" before doing anything.

**Multi-ticket progress:**
```
Grooming 3 tickets...
  - ABC-1234: Posted (review: PASS)
  - ABC-1235: Posted (review: NEEDS FIXES -- 2 corrections applied)
  - ABC-1236: In progress -- staff review
```

## No Ticket? No Problem.

When the user describes an issue verbally (no ticket key):
1. Run the investigation pipeline (default to standard depth unless the description is clearly cosmetic)
2. Present findings in conversation
3. Ask: "Should I create a ticket with these notes?" (respect project rules about ticket creation)

## Reference Files

| File | Contents |
|------|----------|
| [investigation-prompt.md](references/investigation-prompt.md) | Sub-agent prompt template with placeholder assembly |
| [accuracy-rules.md](references/accuracy-rules.md) | 7 investigation accuracy rules |
| [pipeline.md](references/pipeline.md) | Investigation phases 0-5 |
| [output-templates.md](references/output-templates.md) | Writing style rules and output templates (short, full, non-code) |
| [estimation-priority.md](references/estimation-priority.md) | T-shirt estimation table and P1-P3 priority matrix |
| [staff-review-prompt.md](references/staff-review-prompt.md) | Staff engineer review prompt and checklist |
| [adf-posting.md](references/adf-posting.md) | ADF expand node reference and posting details |
| [error-handling.md](references/error-handling.md) | What to do when things fail |
| [accepted-risks.md](references/accepted-risks.md) | What this skill knowingly does not guard against, and why |
| [frameworks/README.md](references/frameworks/README.md) | Framework detection table + what to do when a framework has no rules file |
| [frameworks/rails.md](references/frameworks/rails.md) | Rails rules (associations, callbacks, enums, default scopes, STI, Pundit, Packwerk) |
| [frameworks/go.md](references/frameworks/go.md) | Go investigation rules |
| [frameworks/javascript.md](references/frameworks/javascript.md) | JavaScript / TypeScript investigation rules |
| [config-resolution.md](references/config-resolution.md) | **Start here for config** — the 4-tier precedence order |
| [config.example.md](references/config.example.md) | Template to copy to `references/config.md` (tier 3) |
| [scripts/md2adf.py](scripts/md2adf.py) | Markdown to ADF converter used by the posting step |
| [configuration.md](references/configuration.md) | CLAUDE.md configuration options |

## Skills Referenced

| Skill | When |
|-------|------|
| `systematic-debugging` | Root cause analysis (phases 1-3 only, no implementation) |
| `dispatching-parallel-agents` | Multi-ticket invocations (max 3 concurrent) |
