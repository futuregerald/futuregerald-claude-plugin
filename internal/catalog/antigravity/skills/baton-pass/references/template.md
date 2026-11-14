# Active Context Template

```markdown
# ACTIVE_CONTEXT — <Project / Stream> (<YYYY-MM-DD HH:MM>)

- **Repo / Worktree:** `<abs-path>` (e.g. `/Users/dev/project`)
- **Branch & HEAD SHA:** `<branch-name>` (`<git-rev-parse-HEAD>`)
- **Execution Mode:** `swarm | subagent | inline`
- **Anchored WIP Ref / Swarm Wind-Down:** `<none | merged-to-branch | baton/<stream> (<sha>) — all swarm workers killed>`
- **Plan Path:** `<abs-path-to-PLAN.md>`
- **Active Phase:** `Phase N (<Name>)`
- **Turn / Token Depth:** `<turn-number> / <estimated-tokens>`

## 1. Active Files & Contracts

### File: `<path/to/file_a.ext:start-end>`
- Exported Types / Signatures:
  `<symbol>(<arg>: <Type>) -> <ReturnType>`
- Data Shapes / Invariants:
  `<field_name>: <Type>`

### File: `<path/to/file_b.ext:start-end>`
- Contract / Wire Protocol:
  `<format-details>`

## 2. Decisions, Evidence & Test Results

### Architectural & Review Decisions
- Decision: <settled decision from user, plan review, or peer review>
  Rationale: <why this approach was chosen>

### Solved Issues
- Issue: <defect description>
  Root Cause: <exact mechanism of failure>
  Resolution: <exact fix applied>

### Test & Verification Evidence
- Gate Command: `<test-command>`
  Exit Code: `0`
  Output: `<concise summary of passing output>`

## 3. Pending Steps (Ordered)

1. [ ] <concrete action>
   - Target Files: `<path/to/target>`
   - Expected Contract: `<input -> output>`
   - Verification Gate: `<exact command to verify>`

2. [ ] <concrete action>
   - Target Files: `<path/to/target>`
   - Expected Contract: `<input -> output>`
   - Verification Gate: `<exact command to verify>`

## 4. Negative Constraints (Do Not Repeat)

- Invariant: <system or repo rule that must never be broken>
- Anti-Pattern: <pattern that was attempted and rejected, with reason>
- Edge Case Trap: <subtle bug or harness constraint discovered during execution>
```
