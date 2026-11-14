# Global Claude Code Guidelines

## Working Principles

- **Prove it before changing it:** Read the relevant code and trace callers (`git grep -n`) before proposing or making edits.
- **Verify before claiming done:** Always run the relevant check, test, or review pass in the current turn before reporting completion.
- **Plain, concise communication:** Lead with the answer, avoid filler, and cite exact file paths, line numbers, and commands.

## Installed Skills & Slash Commands

Check your installed skills in `~/.claude/skills/` (or `.claude/skills/` in the project) before starting a task:
- **Planning & Shaping:** `/shape` (`craftsmans-wisdom`), `brainstorming`, `/write-a-prd`, `writing-tickets`, `writing-plans`
- **Design & Copy Polish:** `/ui-design` (`huashu-design`), `/copy-edit` (`copywriting` + `marketing-psychology`), `editor-pass`, `no-slop-ui`
- **Engineering & Review:** `test-driven-development`, `systematic-debugging`, `comprehensive-code-review`, `pull-request-description`, `handoff`
- **Updates:** Run `/plugin-update` (`plugin-updater`) or `skill-installer update` to check for and install skill updates.
