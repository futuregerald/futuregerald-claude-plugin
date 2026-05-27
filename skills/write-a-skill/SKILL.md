---
name: write-a-skill
description: Create new agent skills with proper structure, progressive disclosure, and bundled resources. Use when user wants to create, write, or build a new skill.
---

# Writing Skills

## Process

1. **Gather requirements** — ask about: task/domain, use cases, need for scripts, reference materials
2. **Draft the skill** — create SKILL.md (under 100 lines), split to REFERENCE.md if needed
3. **Review with user** — present draft, iterate

## Structure

```
skill-name/
├── SKILL.md           # Main instructions (required, <100 lines)
├── REFERENCE.md       # Detailed docs (if >100 lines)
├── EXAMPLES.md        # Usage examples (if needed)
└── scripts/           # Utility scripts (if deterministic ops needed)
```

## SKILL.md Template

```md
---
name: skill-name
description: Brief capability description. Use when [specific triggers].
---

# Skill Name

## Quick start
[Minimal working example]

## Workflows
[Step-by-step processes]

## Advanced features
[Link to REFERENCE.md if needed]
```

## Description Rules

- Max 1024 chars, third person
- First sentence: what it does
- Second sentence: "Use when [triggers]"

## When to Add Scripts

Add when: operation is deterministic, same code generated repeatedly, errors need handling.

## When to Split Files

Split when: SKILL.md exceeds 100 lines, content has distinct domains, advanced features rarely needed.

## Review Checklist

- [ ] Description includes "Use when..." triggers
- [ ] SKILL.md under 100 lines
- [ ] No time-sensitive info
- [ ] Concrete examples included
