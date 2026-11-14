# Global Claude Code Guidelines (Opinionated Dev Workflow)

## 1. Execution & Delegation Discipline

- **Filter bulky output at the shell, never into context:** `cmd > /tmp/out.log 2>&1; rc=$?; tail -20 /tmp/out.log; echo $rc`. Capture the exit code before the pipe.
- **Triage inline before dispatching sub-agents:** A sub-agent costs ~31k–57k tokens before it does anything. Identify exact `file:line` entry points with `git grep -n` first, then dispatch one focused sub-agent rather than fanning out blindly.
- **Never delegate:** synthesis of this conversation, writing in the user's voice (PR descriptions, tickets, messages), or the final verification gate in your own shell.

## 2. 12-Phase Development Lifecycle (`/start`, `/begin`, `/lifecycle`)

Every code change follows this 12-phase workflow. Only pure doc updates and `git revert` skip planning; only Phase 3½ (`plan-review`) may be skipped for a trivial change with an obvious, self-contained blast radius.

| Phase | Action | Skill / Command | Gate |
|---|---|---|---|
| 0. SHAPE | Interview for point-of-view brief on new/user-facing work; optionally grill open decisions or write PRD | `/shape`, `craftsmans-wisdom`, `future-grilling`, `/write-a-prd` | Brief confirmed (or skipped for internal technical work) |
| 0½. UI DESIGN | For net-new pages, new apps, or major UI changes: generate 3 interactive HTML prototypes on `design-demos/index.html` | `/ui-design` (`huashu-design`) | Direction chosen in browser |
| 1. RECEIVE | Understand the task and create a task list (`TaskCreate`) | — | Task understood |
| 2. TRACE | Trace callers upward and side effects downward (`git grep -n`) | — | Every caller named with contract expectations |
| 3. PLAN | Write the uncommitted plan (`writing-plans`) with `Affected Callers`, sibling exemplar `file:line`, and test gates | `writing-plans` | Plan file exists |
| 3½. PLAN REVIEW | Run one-shot adversarial review in a fresh `adversarial-plan-reviewer` sub-agent | `/plan-review` | CRITICAL & IMPORTANT findings fixed |
| 4. IMPLEMENT | Strict TDD (`test-driven-development`): RED -> GREEN -> REFACTOR. Check existing helpers first (`Reuse Before You Build`) | `test-driven-development`, `subagent-driven-development` | Fast unit tests pass; zero new comments |
| 5. TEST | Run fast unit tests scoped to affected specs only | — | Zero failures (`exit code 0`) |
| 6. LINT | Run project linter and typecheck | — | Zero offenses (`exit code 0`) |
| 7. SIMPLIFY | Refine recently changed code (`code-simplifier`), then re-run scoped tests | `code-simplifier` | Simplified |
| 8. CODE REVIEW | Parallel read-only `code-quality-reviewer` and `security-reviewer` sub-agents; verify fixes with `/fix-check` | `comprehensive-code-review`, `/fix-check` | Approved |
| 9. SQL REVIEW | Audit changed SQL queries or migrations with `sql-reviewer` | `sql-optimization-patterns` | Reviewed |
| 10. BLAST-RADIUS VERIFY | Re-run gates and verify every `Affected Callers` row after all code mutations | — | Every caller verified with evidence |
| 11. COMMIT | Conventional commit on a feature branch (never commit to `main`) | — | Commit created |
| 12. HANDOFF | Run `editor-pass` against the brief on user-facing work, open PR (`pull-request-description`), and offer `/copy-edit` | `editor-pass`, `pull-request-description`, `/copy-edit` | User acknowledged |

## 3. Core Rules

- **Evidence before claims:** Never claim tests pass or a bug is fixed without running the verification command in the current turn and reading its exit code.
- **No code comments:** Write no code comments unless explicitly asked. Prefer clear names, small functions, and tests that state the case.
- **Communication:** Give one concise high-level kickoff update when the user gives an ask (updating the task list so the user can track phase progress), then do the work without narrating individual file reads or tool calls. Plain language, concise, lead with the answer, cite exact `file:line` paths and commands.
- **Keep skills current:** Run `/plugin-update` (`plugin-updater`) anytime to check for updated skills from `futuregerald/futuregerald-claude-plugin`.
