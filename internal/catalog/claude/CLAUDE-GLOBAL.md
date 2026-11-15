# Global Claude Code Guidelines (Opinionated Dev Workflow)

## 1. Execution & Delegation Discipline

- **Filter bulky output at the shell, never into context:** `cmd > /tmp/out.log 2>&1; rc=$?; tail -20 /tmp/out.log; echo $rc`. Capture the exit code before the pipe.
- **Turn batching (reads, edits & fast gates):** Batch `Read` calls in parallel (`5–12` files/turn, `400–800` lines/read; zero `30–50` line micro-slices or 1-file tail loops). Batch `Edit` calls across **distinct files** (`4–8` files/turn), while keeping same-file edits to 1 contiguous replacement (or full `Write` when `<300` lines; never `3+` separate `Edit` calls on the same file in one turn). Never mutate source files with ad-hoc `node -e` / `python3 -c` regex one-liners. Run 1 combined fast verification gate command at the end.
- **Pre-packed context bundles & 5-section handoff:** Pack target slices before Phase 4 and chunked `git diff -U3` + gate summaries before Phase 8 with `python3 ~/.claude/skills/subagent-driven-development/scripts/build_subagent_context.py` (`pre-code` / `post-code`) into `/tmp/subagent-context-<project>-<stream>.md` (reading all `## Read Chunk Index` windows in parallel on Turn 1 when `>750` lines). Every `implementer` returns the **5-section handoff** (`Changes Made`, `Acceptance Criteria & Success Metrics Met`, `Flawed Assumptions Surfaced`, `Carried Risks & Edge Cases`, `Verification Evidence`).
- **Fresh one-shot `Agent` dispatches & effort routing:** Triage inline (`git grep -n`, `<=3` calls) before dispatching. Always prefer fresh one-shot `Agent` dispatches over warm `Agent` `resume` (which replays prior transcript history at `+58%` token cost), and reserve experimental `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` (`~7x` multi-session token multiplier) strictly for when the user explicitly requests `3+` independent parallel workstreams. Route `>=8`-query caller scouting to `context-finder` and noisy build/test reproduction to `runner` (`model: "sonnet"`, `effort: "low"`); keep `haiku` strictly for pre-fed bounded input (`<=3` turns, zero hunting).
- **Never delegate:** synthesis of this conversation, writing in the user's voice (PR descriptions, tickets, messages), or the final verification gate in your own shell.

## 2. 12-Phase Development Lifecycle (`/start`, `/begin`, `/lifecycle`)

Every code change follows this 12-phase workflow. Only pure doc updates and `git revert` skip planning; only Phase 3½ (`plan-review`) may be skipped for a trivial change with an obvious, self-contained blast radius.

| Phase | Action | Skill / Command | Gate |
|---|---|---|---|
| 0. SHAPE | Interview for point-of-view brief on new/user-facing work; optionally grill open decisions or write PRD | `/shape`, `craftsmans-wisdom`, `future-grilling`, `/write-a-prd` | Brief confirmed (or skipped for internal technical work) |
| 0½. UI DESIGN | For net-new pages, new apps, or major UI changes: generate 3 interactive HTML prototypes on `design-demos/index.html` | `/ui-design` (`huashu-design`) | Direction chosen in browser |
| 1. RECEIVE | Understand the task and create a task list (`TaskCreate`) | — | Task understood |
| 2. TRACE | Trace callers upward and side effects downward (`git grep -n`) | — | Every caller named with contract expectations |
| 3. PLAN | Write the uncommitted plan (`future-writing-plans`) with `Acceptance Criteria & Success Metrics` (mapped to mounted `file:Symbol`), `Affected Callers`, sibling exemplar `file:line`, and test gates, then **STOP and present the proposed plan to the user for approval** before writing code (unless user explicitly requested autonomous execution) | `future-writing-plans` | Plan file exists & approved by user |
| 3½. PLAN REVIEW | Run one-shot adversarial review in a fresh `adversarial-plan-reviewer` sub-agent verifying both code premises and `Acceptance Criteria & Success Metrics` coverage | `/plan-review` | CRITICAL & IMPORTANT findings fixed |
| 4. IMPLEMENT | Strict TDD (`test-driven-development`): RED -> GREEN -> REFACTOR. Check existing helpers first (`Reuse Before You Build`) | `test-driven-development`, `subagent-driven-development` | Fast unit tests pass; all Acceptance Criteria wired in runtime code; zero new comments |
| 5. TEST | Run fast unit tests scoped to affected specs only | — | Zero failures (`exit code 0`) |
| 6. LINT | Run project linter and typecheck | — | Zero offenses (`exit code 0`) |
| 7. SIMPLIFY | Refine recently changed code (`code-simplifier`), then re-run scoped tests | `code-simplifier` | Simplified |
| 8. CODE REVIEW | Parallel read-only `code-quality-reviewer` (auditing `Acceptance Criteria & Success Metrics`, Brief, and Approved UI Design against mounted runtime code) and `security-reviewer` sub-agents; verify fixes with `/fix-check` | `comprehensive-code-review`, `/fix-check` | 100% of Acceptance Criteria PASS; approved |
| 9. SQL REVIEW | Audit changed SQL queries or migrations with `sql-reviewer` | `sql-optimization-patterns` | Reviewed |
| 10. BLAST-RADIUS VERIFY | Re-run gates and verify every `Affected Callers` row after all code mutations | — | Every caller verified with evidence |
| 11. COMMIT | Conventional commit on a feature branch (never commit to `main`) | — | Commit created |
| 12. HANDOFF | Run `editor-pass` against the brief on user-facing work, open PR (`pull-request-description`), and offer `/copy-edit` | `editor-pass`, `pull-request-description`, `/copy-edit` | User acknowledged |

## 3. Core Rules

- **Evidence before claims:** Never claim tests pass or a bug is fixed without running the verification command in the current turn and reading its exit code.
- **No code comments:** Write no code comments unless explicitly asked. Prefer clear names, small functions, and tests that state the case.
- **Communication:** Plain language, concise, lead with the answer, cite exact `file:line` paths and commands.
- **Keep skills current:** Run `/plugin-update` (`plugin-updater`) anytime to check for updated skills from `futuregerald/futuregerald-claude-plugin`.
