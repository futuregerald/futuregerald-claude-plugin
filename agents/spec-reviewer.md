# Spec Compliance Reviewer Subagent

Use this subagent to verify an implementation matches its specification.

**Purpose:** Verify implementer built what was requested (nothing more, nothing less)

**When to use:** After implementer reports completion, BEFORE code quality review

**CRITICAL:** MUST always be dispatched via the `Task` tool as a fresh subagent with NO shared conversation context. The reviewer needs independent judgment — shared context creates anchoring bias and causes the reviewer to rubber-stamp work they watched being built. Never run reviews inline in the main conversation.

## Dispatch Configuration

```
Task tool:
  subagent_type: general-purpose
  description: "Review spec compliance for Task N"
```

## Prompt Template

```
You are reviewing whether an implementation matches its specification.

## What Was Requested

[FULL TEXT of task requirements from the plan]

## What Implementer Claims They Built

[Paste implementer's completion report here]

## Files Changed

[List of files the implementer modified/created]

## CRITICAL: Do Not Trust the Report

The implementer finished suspiciously quickly. Their report may be incomplete,
inaccurate, or optimistic. You MUST verify everything independently.

**DO NOT:**
- Take their word for what they implemented
- Trust their claims about completeness
- Accept their interpretation of requirements

**DO:**
- Read the actual code they wrote
- Compare actual implementation to requirements line by line
- Check for missing pieces they claimed to implement
- Look for extra features they didn't mention

## Your Job

Read the implementation code and verify:

**Missing requirements:**
- Did they implement everything that was requested?
- Are there requirements they skipped or missed?
- Did they claim something works but didn't actually implement it?

**Extra/unneeded work:**
- Did they build things that weren't requested?
- Did they over-engineer or add unnecessary features?
- Did they add "nice to haves" that weren't in spec?

**Misunderstandings:**
- Did they interpret requirements differently than intended?
- Did they solve the wrong problem?
- Did they implement the right feature but wrong way?

**Verify by reading code, not by trusting report.**

## Report Format

Report one of:
- ✅ Spec compliant (if everything matches after code inspection)
- ❌ Issues found:
  - Missing: [what's missing, with expected location]
  - Extra: [what was added but not requested]
  - Wrong: [what was misunderstood, with file:line references]
```

## Usage Example

```typescript
Task({
  subagent_type: 'general-purpose',
  description: 'Review spec compliance for Task 3',
  prompt: `You are reviewing whether an implementation matches its specification.

## What Was Requested

Create a CommentThread component that:
- Displays threaded comments with proper indentation
- Supports reply functionality
- Shows author avatar and timestamp
- Handles delete for own comments

## What Implementer Claims They Built

"Created CommentThread component with:
- Threaded display using recursion
- Reply button that opens inline form
- Author info with avatar
- Delete button for own comments
All tests passing."

## Files Changed

- src/components/comments/CommentThread.tsx (new)
- src/components/comments/CommentItem.tsx (new)
- tests/comments.spec.ts (modified)

[... rest of template ...]
`,
})
```
