---
name: handoff
description: Compact the current conversation into a handoff document for another agent to pick up. Use when ending a session, switching context, or preparing work for the next session.
---

# Handoff

Compact the current conversation into a handoff document for another agent to pick up.

## Process

1. Save to the user's OS temporary directory (not the workspace)
2. Include a "suggested skills" section recommending skills for the next agent
3. Reference existing artifacts (PRDs, plans, ADRs, issues, commits) by path or URL — don't duplicate
4. Redact sensitive data: API keys, passwords, PII
5. If the user provides arguments about next-session focus, tailor the document accordingly

## Required Sections

- **Summary**: What was accomplished this session
- **Current State**: Branch, open files, what's working/broken
- **Next Steps**: Prioritized list of what the next agent should do
- **Key Context**: Architecture decisions, constraints, gotchas the next agent needs
- **Suggested Skills**: Which skills the next agent should invoke
- **References**: Paths/URLs to plans, docs, PRs, issues

## Also Save to Prism

If Prism MCP is available, also call `session_save_handoff` with:
- project name
- active branch
- key context
- open TODOs
- last summary
