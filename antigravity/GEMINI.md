# Global Agent Guidelines (Antigravity & Gemini)

Mechanics (Prism signatures, review-lens commands, search hierarchy, model routing, breaker rules, baton-pass) live in the `antigravity-orchestration` skill. Organization, tracker and repo specifics live in local rule files under `~/.gemini/config/rules/` and `antigravity-orchestration/references/config.md`.

## 0. Execution Discipline (Read First)

1. **No drafting in `<thought>`:** Use `<thought>` only to reason about decisions, edge cases, and the next action—never to draft prose, reports, tables, or code, and never to walk item-by-item through data. On routine tool selection and lookups, keep `<thought>` to 1–5 sentences; on non-trivial design, debugging, or review, spend the budget on verifying assumptions and failure modes, then write the output once.
2. **Thinking level:** The UI level sets the ceiling per step, not the number of steps, so a low setting does not excuse long runs; the check-in rule (§1) still applies.
3. **Search before read:** `grep -rn` first, then read the hit range ±40 lines. Read whole files only when editing them. Never re-read a range already in context. When invoking a skill, read `SKILL.md` first and a `references/` file only when the active step requires it; never read `scripts/test_*.py` while running a skill.
4. **Answer first** for advice, opinion, planning and estimation: lead with the answer, then only the reasoning that changes what the user does. Claims about code or runtime behaviour still follow "Prove it" (§3).
5. **Scripts over markup:** never hand-write >150 lines of repetitive HTML, Markdown or boilerplate when a script can generate it.
6. **Intercepted output:** when tool output lands in a file (`output.txt`, a JSON dump), write one `/tmp/parse_<name>.py` and run it. Never probe the same file with repeated `python3 -c` one-liners.
7. **Few shell round-trips:** chain deterministic commands in one `run_command` (`a && b`) and filter bulky output at the shell: `cmd > /tmp/out.log 2>&1; rc=$?; tail -20 /tmp/out.log; echo $rc`.
8. **Large documents:** over 150 lines without a renderer, write section 1 with `Overwrite: true`, then append the rest section by section.
9. **Adapt to the task:** project management, planning, writing and review are not coding. Run the lifecycle (§2) only for code changes; for reports, tracker lookups, 1:1 prep and questions, execute directly.

## 1. Check-in vs Authorized Runs

- **Authorized** means the user's message gives a multi-step mandate: implement phases or streams, continue from Pending Steps, ingest an ACTIVE_CONTEXT handoff, act as lead orchestrator, work autonomously, keep going, `/goal`. In an authorized run, execute every planned stream to completion without stopping between them.
- **Unauthorized turns:** before work needing more than ~5 tool calls, write one line: goal · done when · estimate (calls). At 2× the estimate, or when the hook sends a checkpoint, send found / unknown / options and wait.
- A hook deny is a signal, not an obstacle: change approach or ask. Never retry the same call.

## 2. Development Lifecycle (Code Changes Only)

Choose the tier first. If unsure, go one tier up.

| Tier | Examples | Process |
|---|---|---|
| Trivial | rule/doc prose edits, obvious one-line fix | edit → check → commit |
| Medium | contained feature or bugfix; hook code under `~/.gemini/config/hooks/` | short plan → fast read-only `plan-review` (no spikes) → TDD → scoped fast tests → commit (`comprehensive-code-review` at PR time) |
| High | shared contracts, DB, money/auth, multi-repo | full lifecycle below (`comprehensive-code-review` between phases and at PR time) |

| Phase | Action | Gate |
|---|---|---|
| 1. Receive | Understand task and constraints; offer a tracker ticket if appropriate (ask first) | Understood |
| 2. Trace | Trace callers up and callees down for every symbol touched, including invisible edges (`send`, `constantize`, serializers, callbacks, string-named jobs) | Affected and untested callers identified |
| 3. Plan | `writing-plans`; never commit the plan | Plan has Affected Callers |
| 3½. Plan review | `plan-review` in a fresh `pro` subagent, once — **short and read-only, zero spikes / zero `run_command`**. Fix every CRITICAL and IMPORTANT. Where configured in `antigravity-orchestration/references/config.md`, also run the `review-lens` corpus check | Fixed |
| 4. Implement | TDD (RED → GREEN → REFACTOR); reuse check before adding any symbol. Implement inline for single-stream tasks; dispatch `flash` implementers for 2+ parallel streams or multi-phase plans | Fast tests pass |
| 5. Test & lint | Fast unit tests scoped to affected specs (`-run <AffectedTests>` in Go; <15s target) plus linter/typecheck. Never run unscoped `go test ./...`, slow DB integration suites, E2E, or `pretest` multi-builds locally—leave full-suite sweeps to CI | Zero failures and offenses |
| 6. Simplify | Reuse, clarity, dead code | Done |
| 7. Code review | Run **for the PR when ready, or between phases in large multi-phase projects** (never per commit): fresh read-only `pro` reviewer that trusts the written `/tmp/subagent-context-<project>-<stream>.md` file (never re-fetching its diff/files or running tests/builds); where configured, run the corpus check on the diff. Delete the context file once approved | Approved |
| 7½. SQL review | If queries or mutations changed: fresh `pro` audit for performance, indexes, security | Reviewed |
| 8. Blast radius | Walk the Affected Callers list plus callers of every symbol the final diff touched that the plan missed. Runs last | Verified with evidence |
| 9. Commit | Conventional commit; summarize; on personal repos configured for auto-PR, push and open the PR | User acknowledged |

**Independence:** never review a plan, code or SQL in the context that wrote it. A plan review reads and checks premises against code; it never builds the plan. After fixing a reviewer's findings, re-dispatch only that reviewer.

**Plans:** sections Goal · Affected Callers · Approach · TDD tasks with exact paths · Risks · Rollback · Out of scope. Tasks give signatures, test cases and gates, not function bodies. Affected Callers lists callers the change breaks (with consequence) plus unbroken callers with no test. Keep the status, as-built notes and departures current in the turn they change.

**Reuse and exemplars:** before adding a symbol, search its name and body tokens and read candidates. Match the newest sibling file in the same directory and cite `file:line` in the plan. On a new repo, check `docs/adr/`, `CONTRIBUTING.md`, `*GUIDELINES*.md`, `*PATTERNS*.md`.

## 3. Core Rules

**Prove it, cheapest evidence first:** trace with graph/grep → read the source (lockfile, schema, migration) → spike only as a last resort, in a scratch dir, one fact, two attempts max, never booting the full app or DB.

**Verify before done:** never say done, fixed or passing without running the check in this turn and reading its output and exit code.

**Review findings:** fix every CRITICAL and IMPORTANT finding from plan reviews, code reviews, CI bots and humans. Fix trivial MINOR findings; surface non-trivial ones with evidence. Explain factually wrong findings to the user rather than skipping them. Genuine Cycode/CodeQL false positives: reply on the PR (`#cycode_sast_false_positive <reason>` or `False positive. <reason>`) and report the links.

**Code comments:** write none. Ask before adding any, every time. Prefer longer names that say what the code does, never ticket keys. Leave existing comments alone unless your change makes them wrong.

**Branches, commits, PRs:** never commit to main. Follow the branch and PR conventions in your local org rule file (`~/.gemini/config/rules/`). Conventional commits. Always use `pull-request-description` for PRs. Never auto-merge. No real names, company details or personal data in public repos.

**Tickets:** never create GitHub issues unless the repo uses them. Use the tracker configured in your local org rule file (`~/.gemini/config/rules/`). Reads are free; ask before writes.

**CI fails 3+ times:** stop pushing, reproduce locally. Blocked over 30 minutes: ask the user with options A/B/C.

**Context size:** the breaker hook estimates context per segment and warns at ~105k tokens, blocking at ~130k. On the warn, hand off at the next natural break with `baton-pass` (mechanics in `antigravity-orchestration`), then give the user the one-line Session B prompt. On turn 1 of Session B, read the handoff and continue without re-reading the codebase.

**Memory:** Prism `session_load_context` on turn 1 only for repo engineering, debugging or resuming work. After coding, planning, grooming or review, save the ledger (and handoff when state changes).

## 4. Stack Conventions

- **Ruby/Rails:** 2-space indent, `snake_case`, `&&`/`||`, `present?`/`blank?`, `find_by`, thin controllers, Interactors, Pundit, Packwerk, RSpec. Structured logging: `Rails.logger.info('event_name', key: value)`.
- **Go:** use `go-api-structure` during trace and plan. Consumer-declared interfaces, domain decoupled from SQL, DTOs at the HTTP edge, `context.Context` first, table-driven tests. Local gates: `test -z "$(gofmt -l .)"`, `go vet ./...`, `golangci-lint run`, `CGO_ENABLED=0 go test ./<affected-pkg> -run <AffectedTests> -race` (never run unscoped `go test ./...` locally on DB/migration-backed suites; leave full-repo test and `govulncheck` sweeps to CI unless dependencies changed).

## 5. Communication

Plain language, no filler, no emoji unless asked. Lead with the answer; say it once. Link files and symbols (`[file.rb](file:///abs/path#L10-L20)`), and give ticket keys, PRs, dates, names and exact commands.
