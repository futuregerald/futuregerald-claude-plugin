# Active Context Template

```markdown
# ACTIVE_CONTEXT

Project: <project-key>
Branch: <branch-name>
Commit: <git-rev-parse-HEAD>  (Session B runs `git diff --stat` against this)
Turn / Token Depth: <turn-number> / <estimated-tokens>

## 1. Active File Schemas & Contracts

### File: <path/to/file_a.ext>
- Exported Types / Interfaces:
  `<symbol>(<arg>: <Type>) -> <ReturnType>`
- Data Shapes:
  `<field_name>: <Type>`

### File: <path/to/file_b.ext>
- Contract / Wire Protocol:
  `<format-details>`

## 2. Solved Issues, Architectural Decisions & Test Results

### Architectural Decisions
- Decision: <settled decision>
  Rationale: <why this approach was chosen>

### Solved Issues
- Issue: <defect description>
  Root Cause: <exact mechanism of failure>
  Resolution: <exact fix applied>

### Test & Verification Evidence
- Gate Command: `<test-command>`
  Exit Code: 0
  Output: `<minimal-summary-of-passing-output>`

## 3. Ordered Pending Steps

1. Step 1: <concrete action>
   Target Files: `<path/to/target>`
   Expected Contract: `<input -> output>`
   Verification Gate: `<exact command to verify>`

2. Step 2: <concrete action>
   Target Files: `<path/to/target>`
   Expected Contract: `<input -> output>`
   Verification Gate: `<exact command to verify>`

## 4. Strict Negative Constraints & Traps to Avoid

- Invariant: <system rule that must never be broken>
- Anti-Pattern: <pattern that was attempted and rejected>
- Reason for Rejection: <why it failed>
- Edge Case Trap: <subtle bug discovered during execution>
```
