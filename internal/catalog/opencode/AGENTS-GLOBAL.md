# OpenCode — Global Agent Guidelines (`AGENTS.md`)

## Installed Skills, Commands & Subagents

- **Skills:** `~/.config/opencode/skills/<skill-name>/SKILL.md` (or `.opencode/skills/` per project)
- **Slash Commands:** `~/.config/opencode/commands/<command>.md` (or `.opencode/commands/` per project) — type `/start`, `/begin`, `/shape`, `/ui-design`, `/write-a-prd`, `/plan-review`, `/fix-check`, `/copy-edit`, `/team-pulse`, `/plugin-update`
- **Subagents:** `~/.config/opencode/agents/<agent>.md` (or `.opencode/agents/` per project)

## Core Discipline

1. **Filter bulky shell output:** `cmd > /tmp/out.log 2>&1; rc=$?; tail -20 /tmp/out.log; echo $rc`.
2. **Trace & Plan before touching code:** Map callers upward and side effects downward with `git grep -n`, then write an uncommitted plan (`writing-plans`) including `Affected Callers`.
3. **Fresh review context:** Run `/plan-review` (`adversarial-plan-reviewer`) and `comprehensive-code-review` (`code-quality-reviewer`, `security-reviewer`, `sql-reviewer`) in fresh subagents, and verify fixes with `/fix-check`.
4. **Strict TDD & zero comments:** Follow RED -> GREEN -> REFACTOR (`test-driven-development`) and write no code comments unless explicitly asked.
5. **Verify before completion:** Run fast scoped tests and linters in the current turn before claiming work is done.
6. **One upfront kickoff update (no per-tool narration):** Give one concise high-level kickoff update when given an ask, then execute without narrating individual file reads or tool calls.
7. **Keep skills updated:** Run `/plugin-update` (`plugin-updater`) or `skill-installer update` to pull new skills from `main`.
