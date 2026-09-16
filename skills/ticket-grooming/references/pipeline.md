# Investigation Pipeline

Run these phases sequentially. After each phase, summarize findings (key facts, file paths, hypotheses) and carry ONLY the summary forward — not raw tool output. Follow the [accuracy rules](accuracy-rules.md) in every phase.

## Phase 0: Understand the ticket

**Read the ticket before touching the codebase.** Don't jump into code research before understanding what the reporter is actually asking.

1. **Classify:** `code-bug`, `code-feature`, `tech-debt`, `process-docs`, or `ambiguous`
2. **Summarize the reported problem in one sentence.** Use the reporter's words, not your interpretation.
3. **Identify the affected path:** write path (data stored wrong), read path (data displayed wrong), both, or unknown
4. **List open questions.** What's unclear? What assumptions would you need to make?
5. **Gate check:** If the ticket is too vague to investigate (no symptom, no repro steps, no affected area), recommend clarification before proceeding. Don't fabricate specificity.

If `process-docs`: run Phase 0 and Phase 2 (history only), then Phase 5 using the non-code template. Skip phases 1, 3, and 4.

## Phase 1: Investigate the code

**Budget: max 15 files deep-read, max 3-hop call path traces (per repo).**

### Step 0: Detect the framework

Before reading any application code, identify the project framework. Read [frameworks/README.md](frameworks/README.md) for the detection table, then read **only** the rules file for the framework you detected (one per repo in a multi-repo ticket). The framework determines how to interpret code — skipping this step leads to misdiagnoses. If the framework you detect has no rules file, follow the fallback in that README and say so in the notes.

### Step 1: Read the models FIRST

For any ticket involving database queries, data state, or associations: **read the relevant model files before forming any hypothesis.** This includes:
- All `belongs_to`, `has_many`, `has_one` declarations (association names differ between repos)
- All `include`'d concerns (methods and callbacks may be defined there)
- `enum` definitions (integer storage, generated scopes/predicates)
- `default_scope` (invisible WHERE clauses)
- `delegate` declarations (methods that appear to be missing)
- Callbacks (`before_save`, `after_commit`, etc.)

This is not optional. Do NOT skip to grep/search and start counting "affected files" before understanding the model definitions. See accuracy rule 6.

### Step 2: Search and trace

**With `codebase-memory-mcp` (preferred):**
1. Check index is current: `index_status`
2. Search the graph for entities related to the ticket: `search_graph`
3. Trace call paths for affected code (max 3 hops): `trace_call_path`
4. Check database schemas and migrations
5. Map the surface area: files, functions, models affected

**Without `codebase-memory-mcp` (fallback):**
1. Grep for class names, method names, keywords from the ticket
2. Glob for relevant files (`**/models/**`, `**/interactors/**`)
3. Read the most relevant files, trace call paths manually
4. Check `db/schema.rb` or `db/structure.sql` and recent migrations
5. Map the surface area

**Reachability is answered by grep, in both paths.** A call-path/trace tool tells you a symbol
exists and what it reaches — never who calls it. It cannot see dynamic dispatch: interactor and
organizer lists, `send`, `public_send`, `constantize`, job classes named by string, serializers,
delegation, callbacks, config-driven routing. **A zero-caller result from one is not evidence.**
Any negative or exhaustive claim ("nothing else calls this", "this is dead", "X is the only
caller") must name the scopes it grepped, and those scopes need index coverage too. A clean
coverage result means no recorded gap — never proof of completeness.

**Multi-repo (both paths):**
- For backend tickets: search EVERY repo listed under `Repos:` in the configuration, not just the first. Budget applies per repo.
- Where the configuration notes that one repo is mid-migration into another, code may exist in both — search both, fix only in the destination.

**Cosmetic tickets (shallow):** Reduce budget to max 3 files, no call path traces. Confirm the file and line that needs changing, verify no logic side effects, then skip to Phase 5.

### Verification mandate

Before carrying findings forward, verify each one individually:
- For "N files affected" claims: confirm each file is actually affected by reading it against its repo's model/config — not just by matching a grep pattern
- For column/association claims: confirm against `db/schema.rb` AND the model's association definitions
- For "method missing" claims: check concerns, delegation, and base classes

**Summarize before proceeding.** Carry forward: affected files with line numbers, key functions, schema details, call path summary. Include the verification trail for each finding (what you checked, what you found — see accuracy rule 7).

## Phase 2: Check the history

**Budget: max 50 git log entries, max 20 Jira results, max 20 PR results.**

0. **Keep the search narrow or it will not fit.** Always pass an explicit `fields` list
   (`key`, `summary`, `status`, `issuetype`, `updated` is usually enough) — the default returns
   every field on every hit. Never run a `text ~` query without a `project =` clause: an
   unscoped one returned 152,076 characters in a single call and had to be spilled to a file.
   Prefer `summary ~` over `text ~` when you know roughly what you are looking for.
1. Search past tickets (Jira: `searchJiraIssuesUsingJql`; GitHub: `gh issue list --search`)
2. Search PRs and commits (`git log --all --grep`, `gh pr list --state all --search`)
3. `git blame` on the most relevant files from Phase 1
4. Find related conversations, decisions, past fixes
5. Search for similar COMPLETED tickets for estimation grounding

**Summarize before proceeding.** Carry forward: related ticket keys with links, relevant PRs, key decisions.

### Step 3: Reach for a domain skill when the subject warrants one

The framework rules cover the *language and framework* — associations, callbacks, enums, dispatch.
They say nothing about the ticket's actual subject matter. When the ticket turns on a domain with
its own installed skill, load it:

| Ticket turns on | Load |
|---|---|
| Query plans, indexes, slow SQL, schema design | `sql-optimization-patterns` |
| A bug with a reproducible failure | `systematic-debugging` (phases 1-3 only — investigation, not fixes) |
| Frontend rendering, state, component structure | whichever frontend skill the repo's conventions name |

This was found by running the same index ticket through two prompts: both produced correct notes on
Rails mechanics and neither had any guidance on Postgres index selection, which was the entire
question. A framework rules file is not a substitute for a domain skill.

## Phase 3: Find the root cause

Apply systematic-debugging methodology (phases 1-3 ONLY — investigation, not implementation):

1. **Investigate:** Read error messages, check recent changes, trace data flow
2. **Find patterns:** Compare working examples against the broken path, identify differences
3. **Form hypotheses:** Rank by confidence (high/medium/low). Each must follow accuracy rule 2 ("Show the mechanism") — cite symptom, location, and mechanism. Apply rule 3 ("Challenge your own hypotheses") — challenge each hypothesis before writing it up.

**Do NOT implement fixes, write tests, or modify code.**

## Phase 4: Assess the risks

Using findings from phases 1-3:
1. Dependency analysis — what breaks if this changes?
2. Edge cases discovered during investigation
3. Blast radius — other features, services, or repos affected
4. Security implications
5. Performance implications

## Phase 5: Write the notes

**Before writing:** Apply accuracy rule 5 ("Stay on the reporter's actual problem") — re-read the ticket. Verify your findings address the reporter's actual problem.

Compile findings using the appropriate template from [output-templates.md](output-templates.md). Use estimation and priority tables from [estimation-priority.md](estimation-priority.md).

**The sub-agent WRITES the notes to a file and returns a short receipt. It does NOT post them,
and it does not paste the details block into its final message.** Write Markdown only — never JSON
or ADF; the orchestrator converts it. The file path and required marker are given in the dispatch
prompt (see [investigation-prompt.md](investigation-prompt.md)). Posting happens in the main
conversation after staff review.

### GitHub Permalinks

Every file/function/line reference MUST include a GitHub permalink:
`https://github.com/{ORG}/{REPO}/blob/{SHA}/{PATH}#L{LINE}`

For multi-repo: use each repo's own org/repo/SHA. Fallback: relative path `{repo}:{path}#L{line}` if SHA is not on remote.
