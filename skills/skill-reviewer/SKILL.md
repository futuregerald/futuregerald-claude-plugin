---
name: skill-reviewer
description: Review skills for quality, size, progressive disclosure, encoded judgment, checkable standards, point of view and attribution. Use when auditing existing skills, reviewing skill changes in PRs, or when comprehensive-code-review detects SKILL.md files in a diff.
author: Gerald Onyango
tags: [quality, review, skills]
---

# Skill Reviewer

You are a **Staff Engineer** reviewing skills (SKILL.md files and their sibling resources) for quality and effectiveness. Skills are context that gets loaded into an AI agent's working memory — every line costs tokens and competes with the actual task.

## What good output looks like

- **For the skill's author:** every rating that isn't PASS cites `file:line` and says what to change; every PASS on a criterion that could have failed says what was checked.
- **The default to prevent:** a table of PASSes with no evidence — a rubber stamp.

**Vendored skills** are synced from an upstream repo (e.g. `impeccable`, `huashu-design`, `skill-creator`, the Matt Pocock set; provenance notes live in `docs/*-provenance.md` where present). Review them advisory-only: criteria 10-12 are this plugin's conventions, which upstream doesn't share, so rate them N/A, and remember any fix is wiped by the next sync. Fork the skill if a fix matters.

## Run First

Settle the mechanical checks with tools, not by eye, and cite their output:

```bash
wc -l <skill>/SKILL.md                                    # criterion 1
wc -c <skill>/SKILL.md                                    # criterion 1, tokens ≈ characters / 4
find <skill> -type f | sort                               # criteria 2 and 8
python3 <plugin>/skills/future-skill-creator/scripts/quick_validate.py <skill>   # criterion 3
ls <plugin>/skills/*/scripts/test_*.py                    # criterion 11, basename collisions
```

`quick_validate.py` checks the frontmatter keys, `name` against the directory, inline `tags`, and leftover TODO placeholders. It needs PyYAML; without it, run it with `uv run --with pyyaml python3 …`. Where `future-skill-creator` isn't installed, check those four by reading the frontmatter, and say so in the review.

## Review Checklist

For each skill, evaluate against these criteria and rate as PASS, WARN, or FAIL (N/A where a criterion says so):

### 1. Size Budget

| Metric | Target | WARN | FAIL |
|--------|--------|------|------|
| SKILL.md lines | < 300 | 300-500 | > 500 |
| SKILL.md tokens (estimate: characters / 4) | < 3000 | 3000-5000 | > 5000 |
| Total content (SKILL.md + references) | < 1500 lines | 1500-2500 | > 2500 |

### 2. Progressive Disclosure (Single-File Check)

- **FAIL** if SKILL.md > 500 lines — must be split into SKILL.md + reference files, no exceptions
- **FAIL** if SKILL.md > 300 lines with no reference files and contains clearly separable cold content (prompt templates, exhaustive examples, lookup tables)
- **WARN** if SKILL.md > 300 lines and some cold content could be split out
- Reference files must be clearly listed in SKILL.md with "when to read" guidance
- References should be one level deep (no `references/sub/sub/file.md`)
- Cold content = anything not needed on every invocation (detailed prompt templates, framework-specific examples, lookup tables, testing patterns)

### 3. Frontmatter Quality

- `name` — must match directory name, be kebab-case
- `description` — must clearly state WHEN the skill triggers and WHAT it does. This is the only thing Claude sees before deciding to load the skill. Vague descriptions like "best practices for X" are a WARN.
- `author` and `tags` are allowed, and so are the fields the installer and existing skills use: `model`, `effort`, `languages`, `argument-hint`, `trigger`, `version`, `user-invocable`, `license`, `allowed-tools`, `metadata`. `tags` must be an inline list (`tags: [a, b]`); the installer parses only that form, so a block list silently drops the skill from `--tags` filtering (FAIL).

### 4. Interface Over Internals

- **FAIL** if the skill explains how systems work internally when the model only needs to know the interface/API
- The model needs to know: what to write, what pattern to follow, what to avoid
- The model does NOT need to know: how the pipeline processes data, what happens after the log is emitted, internal implementation details of libraries

### 5. Example Efficiency

- Each code example must teach something distinct. Redundant examples are a WARN.
- Prefer 1-2 focused examples over 4+ verbose ones
- Anti-pattern examples: keep to the most common mistake (1-2), not an exhaustive list

### 6. No Duplication

- **WARN** if content duplicates what's in CLAUDE.md (team mappings, lifecycle phases, commit conventions)
- **WARN** if content duplicates another skill (e.g., testing patterns in both a Ruby skill and javascript-testing-patterns)
- Cross-reference instead of duplicating

### 7. Degrees of Freedom

- **Rigid patterns** (exact code, specific sequences): appropriate when operations are fragile, consistency is critical
- **Flexible patterns** (principles, heuristics): appropriate when multiple approaches are valid
- **WARN** if a skill is overly prescriptive for a flexible domain, or too loose for a fragile one

### 8. No Extraneous Files

- **FAIL** if the skill directory contains README.md, CHANGELOG.md, INSTALLATION_GUIDE.md, or other documentation not directly used by the agent
- Skills should only contain SKILL.md, `references/`, `scripts/`, and `assets/`
- Exempt: `LICENSE*` and `NOTICE*` files a source licence requires keeping (e.g. Apache-2.0 §4)

### 9. Actionability

- Every section should help the model produce correct output
- **WARN** for "nice to know" sections that don't change behavior (history, philosophy, "why we chose X")
- Tables and checklists are preferred over prose paragraphs

### 10. Encoded Judgment

- **WARN** where the skill makes a choice but gives advice an agent cannot act on ("use the appropriate X", "keep it clean") and a decision tree, do/don't pair or worked example would fit
- Decision trees for "which option when"; do/don't pairs where right and almost-right look alike; per-item reference files or a lookup script where the skill covers many items but any task needs only a few
- N/A for skills that make no choices (pure procedure or reference)

### 11. Checkable Standard

- **WARN** if part of the output standard is mechanically checkable (format, required sections, banned phrases, cross-references) but the skill ships no check script with a `test_*.py`
- **WARN** if a check script has no tests, or its test basename collides with another skill's (CI imports test files by basename)
- N/A for skills whose standard is purely judgment

### 12. Point of View and Attribution

- **WARN** if a skill that produces output never states what good output looks like, or the generic default it exists to prevent
- **FAIL** if the skill adapts third-party content (a talk, article, guide, another skill) with no credit line naming the source

## Output Format

For each skill reviewed:

```markdown
### skill-name (LINES lines)

| Criterion | Rating | Notes |
|-----------|--------|-------|
| Size | PASS/WARN/FAIL | {details} |
| Progressive disclosure | PASS/WARN/FAIL | {details} |
| Frontmatter | PASS/WARN/FAIL | {details} |
| Interface vs internals | PASS/WARN/FAIL | {details} |
| Example efficiency | PASS/WARN/FAIL | {details} |
| No duplication | PASS/WARN/FAIL | {details} |
| Degrees of freedom | PASS/WARN/FAIL | {details} |
| No extraneous files | PASS/WARN/FAIL | {details} |
| Actionability | PASS/WARN/FAIL | {details} |
| Encoded judgment | PASS/WARN/N/A | {details} |
| Checkable standard | PASS/WARN/N/A | {details} |
| Point of view & attribution | PASS/WARN/FAIL/N/A | {details} |

**Verdict:** APPROVED / NEEDS WORK
**Action items:** (numbered list of specific changes, if any)
```

**Verdict rule:** NEEDS WORK if any criterion is FAIL, or if any WARN would change what an agent following the skill actually does (a wrong instruction, a missing decision, a check that misfires). Otherwise APPROVED, with remaining WARNs listed as action items. Vendored skills are always reported as advisory.

End with a summary table:

```markdown
## Summary

| Skill | Lines | Verdict | Key Issue |
|-------|-------|---------|-----------|
| skill-name | N | APPROVED/NEEDS WORK | {one-line summary or "—"} |
```
