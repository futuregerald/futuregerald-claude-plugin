# Zed Agent Panel — Global Guidelines (`AGENTS.md`)

## How Skills Work in Zed

Skills are installed in `~/.agents/skills/<skill-name>/SKILL.md` (global) or `.agents/skills/<skill-name>/SKILL.md` (project).
In Zed's Agent Panel, type `/<skill-name>` (for example `/start`, `/shape`, `/ui-design`, `/copy-edit`, `/write-a-prd`, `/plan-review`, `/fix-check`, `/plugin-update`) to load a skill directly into your thread, or let the agent load the matching `SKILL.md` automatically when your task matches its trigger description.

## Core Working Rules

1. **Trace before editing:** Before changing code, trace callers upward and side effects downward with `git grep -n`.
2. **Plan before coding:** Write an uncommitted plan (`writing-plans`) with an `Affected Callers` section before modifying source files.
3. **Test-Driven Development:** Follow RED -> GREEN -> REFACTOR (`test-driven-development`) and run fast, scoped unit tests (`<15s`) rather than full E2E suites during iteration.
4. **No code comments:** Write no code comments unless explicitly asked by the user. Prefer self-explanatory names and focused functions.
5. **Verify before completion:** Run tests and linters in this turn and read their exit codes before claiming work is complete.
6. **Plain, concise communication:** Lead with the answer, cite `file:line` references, and avoid filler.
7. **Live updates:** Use `/plugin-update` (`plugin-updater`) or `skill-installer update` to check for updated skills.
