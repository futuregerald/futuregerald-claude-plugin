# {{PROJECT_NAME}} — OpenCode Configuration (`AGENTS.md`)

## Project Overview

{{PROJECT_DESCRIPTION}}

## Key Directories

{{KEY_DIRECTORIES}}

---

## OpenCode Skills, Commands & Subagents

- **Skills:** `.opencode/skills/<skill-name>/SKILL.md` (project) or `~/.config/opencode/skills/` (global)
- **Slash Commands:** `.opencode/commands/*.md` — `/start`, `/begin`, `/lifecycle`, `/shape`, `/ui-design`, `/write-a-prd`, `/plan-review`, `/fix-check`, `/copy-edit`, `/team-pulse`, `/plugin-update`
- **Subagents:** `.opencode/agents/*.md` (`implementer`, `adversarial-plan-reviewer`, `code-quality-reviewer`, `security-reviewer`, `sql-reviewer`, `code-simplifier`, `debugger`, `investigator`, `runner`)

---

## 12-Phase Development Lifecycle (`/start`, `/begin`, `/lifecycle`)

| Phase | Action | Command / Skill / Agent | Gate |
|---|---|---|---|
| 0. SHAPE | Interview for point-of-view brief on new/user-facing work; grill open decisions | `/shape`, `craftsmans-wisdom`, `future-grilling`, `/write-a-prd` | Brief confirmed |
| 0½. UI DESIGN | Generate 3 interactive HTML UI prototypes in `design-demos/index.html` | `/ui-design` (`huashu-design`) | Direction selected |
| 1. RECEIVE | Understand task and constraints | — | Task understood |
| 2. TRACE | Trace callers upward and side effects downward (`git grep -n`) | — | Callers & contracts mapped |
| 3. PLAN | Write uncommitted plan with `Affected Callers` and test gates | `writing-plans` | Plan file exists |
| 3½. PLAN REVIEW | One-shot review in fresh `adversarial-plan-reviewer` subagent | `/plan-review` | CRITICAL & IMPORTANT fixed |
| 4. IMPLEMENT | TDD (RED -> GREEN -> REFACTOR); check existing helpers first | `test-driven-development`, `implementer` | Fast unit tests pass; zero new comments |
| 5. TEST | Run fast unit tests scoped to affected specs | — | Zero failures (`exit code 0`) |
| 6. LINT | Run project typecheck and linter | — | Zero offenses (`exit code 0`) |
| 7. SIMPLIFY | Refine changed code (`code-simplifier`) and re-run scoped tests | `code-simplifier` | Simplified |
| 8. CODE REVIEW | Parallel `code-quality-reviewer` + `security-reviewer`; verify fixes via `/fix-check` | `comprehensive-code-review`, `/fix-check` | Approved |
| 9. SQL REVIEW | Audit changed queries/migrations with `sql-reviewer` | `sql-optimization-patterns` | Reviewed |
| 10. BLAST-RADIUS VERIFY | Re-verify gates and every `Affected Callers` row after all edits | — | Verified with evidence |
| 11. COMMIT | Conventional commit on a feature branch (never `main`) | — | Committed |
| 12. HANDOFF | Run `editor-pass` on user-facing work, open PR (`pull-request-description`), offer `/copy-edit` | `editor-pass`, `pull-request-description`, `/copy-edit` | User acknowledged |

---

## Quick Reference

```bash
{{TEST_COMMAND}}
{{TYPECHECK_COMMAND}}
{{BUILD_COMMAND}}
```

---

<!-- LANGUAGE_SPECIFIC -->
