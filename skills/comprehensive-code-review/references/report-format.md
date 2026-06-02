# Unified Report Format

Use this template for the consolidated report in Phase 4.

`{DATE}` = current date (YYYY-MM-DD). `{COUNT}` = number of files in `{FILE_LIST}`.

```markdown
# Comprehensive Code Review Report

**PR:** {PR_URL or branch name} | **Reviewed:** {DATE} | **Range:** {BASE_SHA}..{HEAD_SHA} | **Files:** {COUNT}

## CRITICAL ({count})
### 1. [Short title]
- **File:** `path/to/file:line_number` | **Dimension:** {Correctness | Safety}
- **Problem:** [Clear description]
- **Recommendation:** [Specific fix]
- **Impact:** [What happens if not fixed]

## IMPORTANT ({count})
[Same format as CRITICAL]

## MINOR ({count})
[Same format, omit Impact]

## Simplification Opportunities
[From Correctness agent Section C. Mark each "Approved — implement" or "Deferred — follow-up".]

## Out-of-Scope Changes (Advisory)
[From Correctness agent. Omit if all in scope or no requirements provided.]

## Strengths
[What was done well — specific file:line references]

## Overall Assessment

| Dimension | Verdict | Critical | Important | Minor |
|-----------|---------|----------|-----------|-------|
| Correctness | {verdict} | {n} | {n} | {n} |
| Safety | {verdict} | {n} | {n} | {n} |

**Final Verdict:** {APPROVED | CHANGES REQUIRED}
**Action Required:** CRITICAL: {n} must fix | IMPORTANT: {n} must fix | MINOR: {n} at discretion
```

## Skill File Reviews

If `{FILE_LIST}` contains any `SKILL.md` files or files under a `skills/` directory, dispatch an **additional** sub-agent:

```
Agent tool:
  subagent_type: "code-quality-reviewer"
  description: "Skill quality review"
  prompt: |
    You are reviewing skill files for quality. First, read the skill-reviewer
    skill by invoking: Skill tool with skill: "skill-reviewer"

    Then apply its checklist to every SKILL.md and reference file in this diff.

    ## Diff
    ```diff
    {DIFF}
    ```

    ## File List
    {FILE_LIST}

    Read each SKILL.md file in the diff (use the Read tool — the diff may
    truncate large files). For each skill, produce the review table and
    verdict from the skill-reviewer checklist.
```

Include skill review findings in the consolidated report under a `## Skill Quality` section after Simplification Opportunities.
