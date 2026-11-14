# {{PROJECT_NAME}} — Zed Agent Configuration (`AGENTS.md`)

## Project Overview

{{PROJECT_DESCRIPTION}}

## Key Directories

{{KEY_DIRECTORIES}}

---

## Using Installed Skills in Zed

Project skills live in `.agents/skills/<skill-name>/SKILL.md` (and global skills in `~/.agents/skills/`).
Invoke any skill in Zed's Agent Panel with `/<skill-name>`:

| Goal | Zed Slash Command / Skill |
|---|---|
| Full 12-phase engineering workflow (`Shape` -> `Handoff`) | `/start`, `/begin`, `/lifecycle` |
| Shape product/feature point-of-view brief & grill decisions | `/shape`, `/craftsmans-wisdom`, `/future-grilling`, `/grilling-page`, `/write-a-prd` |
| 3-direction interactive HTML UI prototypes (`design-demos/index.html`) | `/ui-design` (`/huashu-design`, `/no-slop-ui`) |
| Write & review implementation plans | `/writing-plans`, `/plan-review` |
| TDD implementation & debugging | `/test-driven-development`, `/systematic-debugging`, `/code-simplifier` |
| Code review, SQL audit & fix verification | `/comprehensive-code-review`, `/sql-optimization-patterns`, `/fix-check` |
| Final quality critique, copy audit & PR description | `/editor-pass`, `/copy-edit`, `/pull-request-description`, `/handoff` |
| Check for and apply skill updates | `/plugin-update` (`plugin-updater`) |

---

## Development Workflow & Quality Gates

1. **Trace & Plan first:** Never edit source files before tracing callers (`git grep -n`) and writing a plan with `Affected Callers`.
2. **TDD (RED -> GREEN -> REFACTOR):** Write a failing unit test first, make it pass with minimal code, then simplify.
3. **No code comments:** Write no code comments unless explicitly requested by the user.
4. **Verify before done:** Run the scoped unit tests and linter in the current turn before claiming completion.

---

## Quick Reference

```bash
{{TEST_COMMAND}}
{{TYPECHECK_COMMAND}}
{{BUILD_COMMAND}}
```

---

<!-- LANGUAGE_SPECIFIC -->
