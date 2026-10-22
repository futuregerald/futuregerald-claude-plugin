# Global Agent Guidelines (Antigravity & Gemini)

Mechanics (Prism signatures, review-lens commands, search hierarchy, model routing, breaker rules, baton-pass) live in the `antigravity-orchestration` skill. Load it when you need them.

## 0. Execution Discipline (Read First)

1. **Thought cap:** `<thought>` selects the next action, 1–5 sentences. Never draft prose, reports, tables or code there, and never walk item-by-item through data. Write content once, directly in the tool call.
2. **Thinking level:** Low or Medium by default. The level changes depth per step, not the number of steps, so a low setting does not excuse long runs; the check-in rule (§1) still applies.
3. **Search before read:** `grep -rn` first, then read the hit range ±40 lines. Read whole files only when editing them. Never re-read a range already in context. When invoking a skill, read `SKILL.md` first and a `references/` file only when the active step requires it; never read `scripts/test_*.py` while running a skill.
4. **Answer first** for advice, opinion, planning and estimation: lead with the answer, then only the reasoning that changes what the user does. Claims about code or runtime behaviour still follow "Prove it" (§3).
5. **Scripts over markup:** never hand-write >150 lines of repetitive HTML, Markdown or boilerplate when a script can generate it.
6. **Intercepted output:** when tool output lands in a file (`output.txt`, a JSON dump), write one `/tmp/parse_<name>.py` and run it. Never probe the same file with repeated `python3 -c` one-liners.
7. **Few shell round-trips:** chain deterministic commands in one `run_command` (`a && b`) and filter bulky output at the shell: `cmd > /tmp/out.log 2>&1; rc=$?; tail -20 /tmp/out.log; echo $rc`.
8. **Large documents:** over 150 lines without a renderer, write section 1 with `Overwrite: true`, then append the rest section by section.
9. **Adapt to the task:** project management, planning, writing and review are not coding. Run the lifecycle (§2) only for code changes; for reports, Jira lookups, 1:1 prep and questions, execute directly.

## 1. Check-in vs Authorized Runs

- **Authorized** means the user's message gives a multi-step mandate: implement phases or streams, continue from Pending Steps, ingest an ACTIVE_CONTEXT handoff, act as lead orchestrator, work autonomously, keep going, `/goal`. In an authorized run, execute every planned stream to completion without stopping between them.
- **Unauthorized turns:** before work needing more than ~5 tool calls, write one line: goal · done when · estimate (calls). At 2× the estimate, or when the hook sends a checkpoint, send found / unknown / options and wait.
- A hook deny is a signal, not an obstacle: change approach or ask. Never retry the same call.

## 2. Development Lifecycle (Code Changes Only)

Choose the tier first. If unsure, go one tier up.

| Tier | Examples | Process |
|---|---|---|
| Trivial | rule/doc prose edits, obvious one-line fix | edit → check → commit |
| Medium | contained feature or bugfix; hook code under `~/.gemini/config/hooks/` | short plan → one review → TDD → fast tests |
| High | shared contracts, DB, money/auth, multi-repo | full lifecycle below |

| Phase | Action | Gate |
|---|---|---|
| 1. Receive | Understand task and constraints; offer a Jira ticket if appropriate (ask first) | Understood |
| 2. Trace | Trace callers up and callees down for every symbol touched, including invisible edges (`send`, `constantize`, serializers, callbacks, string-named jobs) | Affected and untested callers identified |
| 3. Plan | `writing-plans`; never commit the plan | Plan has Affected Callers |
| 3½. Plan review | `plan-review` in a fresh `pro` subagent, once. Fix every CRITICAL and IMPORTANT. work-org repos: also the review-lens corpus check | Fixed |
| 4. Implement | TDD (RED → GREEN → REFACTOR); only planned files; reuse check before adding any symbol. Orchestrator dispatches `flash` implementers with a `/tmp/subagent-context-<project>-<stream>.md` brief | Fast tests pass |
| 5. Test & lint | Fast unit tests for affected specs plus linter/typecheck. Never run E2E or `pretest` multi-builds locally between fixes | Zero failures and offenses |
| 6. Simplify | Reuse, clarity, dead code | Done |
| 7. Code review | Fresh read-only `pro` reviewer reading the context file (never re-running tests); work-org repos: corpus check on the diff. Delete the context file once approved | Approved |
| 7½. SQL review | If queries or mutations changed: fresh `pro` audit for performance, indexes, security | Reviewed |
| 8. Blast radius | Walk the Affected Callers list plus callers of every symbol the final diff touched that the plan missed. Runs last | Verified with evidence |
| 9. Commit | Conventional commit; summarize; on `futuregerald/*` push and open the PR | the user acknowledged |

**Independence:** never review a plan, code or SQL in the context that wrote it. A plan review reads and checks premises against code; it never builds the plan. After fixing a reviewer's findings, re-dispatch only that reviewer.

**Plans:** sections Goal · Affected Callers · Approach · TDD tasks with exact paths · Risks · Rollback · Out of scope. Tasks give signatures, test cases and gates, not function bodies. Affected Callers lists callers the change breaks (with consequence) plus unbroken callers with no test. Keep the status, as-built notes and departures current in the turn they change.

**Reuse and exemplars:** before adding a symbol, search its name and body tokens and read candidates. Match the newest sibling file in the same directory and cite `file:line` in the plan. On a new repo, check `docs/adr/`, `CONTRIBUTING.md`, `*GUIDELINES*.md`, `*PATTERNS*.md`.

## 3. Core Rules

**Prove it, cheapest evidence first:** trace with graph/grep → read the source (lockfile, schema, migration) → spike only as a last resort, in a scratch dir, one fact, two attempts max, never booting the full app or DB.

**Verify before done:** never say done, fixed or passing without running the check in this turn and reading its output and exit code.

**Review findings:** fix every CRITICAL and IMPORTANT finding from plan reviews, code reviews, CI bots and humans. Fix trivial MINOR findings; surface non-trivial ones with evidence. Explain factually wrong findings to the user rather than skipping them. Genuine Cycode/CodeQL false positives: reply on the PR (`#cycode_sast_false_positive <reason>` or `False positive. <reason>`) and report the links.

**Code comments:** write none. Ask before adding any, every time. Prefer longer names that say what the code does, never ticket keys. Leave existing comments alone unless your change makes them wrong.

**Branches, commits, PRs:** never commit to main. Work branches: `<type>/<TICKET>/<summary>` (`feature`, `fix`, `chore`, `maintenance` + uppercase Jira key, or `docs/NO-TICKET/...`). Conventional commits. Always use `pull-request-description` for PRs (`Closes PROJ-XXXX` on work repos). Never auto-merge. On `futuregerald/*`, push and open the PR after commit; no real names or personal data in public repos.

**Tickets:** never create GitHub issues. Jira at `your-site.atlassian.net` (cloud ID and project keys in your local config). Reads are free; ask before writes.

**CI fails 3+ times:** stop pushing, reproduce locally. Blocked over 30 minutes: ask the user with options A/B/C.

**Context size:** the breaker hook estimates context per segment and warns at ~105k tokens, blocking at ~130k. On the warn, hand off at the next natural break with `baton-pass` (mechanics in `antigravity-orchestration`), then give the user the one-line Session B prompt. On turn 1 of Session B, read the handoff and continue without re-reading the codebase.

**Memory:** Prism `session_load_context` on turn 1 only for repo engineering, debugging or resuming work. After coding, planning, grooming or review, save the ledger (and handoff when state changes).

## 4. Stack Conventions

- **Ruby/Rails (work repos):** 2-space indent, `snake_case`, `&&`/`||`, `present?`/`blank?`, `find_by`, thin controllers, Interactors, Pundit, Packwerk, RSpec. Structured logging is mandatory: `Rails.logger.info('event_name', key: value)` (`structured-logging`).
- **Go:** use `go-api-structure` during trace and plan. Consumer-declared interfaces, domain decoupled from SQL, DTOs at the HTTP edge, `context.Context` first, table-driven tests. Gates: `test -z "$(gofmt -l .)"`, `go vet ./...`, `golangci-lint run`, `CGO_ENABLED=0 go test ./... -race -shuffle=on`, `govulncheck ./...`.

## 5. Communication

Plain language, no filler, no emoji unless asked. Lead with the answer; say it once. Link files and symbols (`[file.rb](file:///abs/path#L10-L20)`), and give ticket keys, PRs, dates, names and exact commands.
