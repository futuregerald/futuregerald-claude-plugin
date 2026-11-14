# Global Antigravity Guidelines

## 1. Execution & Communication Discipline

- **One upfront kickoff update, answer first & stay clear:** When the user gives an ask, start with one concise high-level kickoff update (pointing to `task.md` on multi-step work), then do the work without narrating individual file reads or tool calls. Lead with the answer in plain language and avoid filler.
- **Search before reading whole files:** Use `git grep -n` in git repositories to locate exact `file:line` hits, then read ±40 lines around matches.
- **Maintain `task.md` on multi-step work:** Keep `<appDataDir>/brain/<conversation-id>/task.md` updated as steps complete.
- **Verify before claiming done:** Run the check or review pass in the current turn before saying a task is complete.

## 2. Installed Skills & Workflows

Use the skills installed in `~/.gemini/config/skills/` (or `.agents/skills/` in the workspace) whenever they match the task:
- **Product, Strategy & Shaping:** `craftsmans-wisdom` (`/shape`), `brainstorming`, `future-grilling`, `future-wayfinder`, `grilling-page`, `write-a-prd`, `writing-tickets`, `ticket-grooming`, `team-pulse`, `meeting-debrief`, `research`
- **Design, Copy & Quality Review:** `ui-design` (`huashu-design`), `copy-edit` (`copywriting` + `marketing-psychology`), `impeccable`, `no-slop-ui`, `animation-principles`, `editor-pass`, `visualize-plan`, `prototype`
- **Core Engineering:** `writing-plans`, `test-driven-development`, `systematic-debugging`, `comprehensive-code-review`, `pull-request-description`, `handoff`
- **Live Updates:** Invoke `plugin-updater` (`/plugin-update`) or run `skill-installer update` to pull the latest skills from `main`.
