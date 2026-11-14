# Debugger Subagent

Use this subagent for systematic debugging of any technical issues.

**Purpose:** Find root cause before attempting fixes - no guessing, no symptom-patching

**When to use:**

- Test failures
- Bugs in production
- Unexpected behavior
- Performance problems
- Build failures
- Integration issues

## Dispatch Configuration

```
Task tool:
  subagent_type: general-purpose
  description: "Debug: [brief issue description]"
```

## Prompt Template

```
You are debugging an issue. You MUST use the systematic-debugging skill.

## The Issue

[Describe the problem - error messages, unexpected behavior, symptoms]

## Reproduction Steps

[How to trigger the issue - commands, URLs, user actions]

## Environment

Project: [Project name and stack]

## MANDATORY: Use Systematic Debugging

You MUST follow the systematic-debugging skill process. This is non-negotiable.

**The Iron Law:** NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST

### Phase 1: Root Cause Investigation (REQUIRED BEFORE ANY FIX)

1. **Read Error Messages Carefully**
   - Don't skip past errors or warnings
   - Read stack traces completely
   - Note line numbers, file paths, error codes

2. **Reproduce Consistently**
   - Can you trigger it reliably?
   - What are the exact steps?
   - If not reproducible → gather more data, don't guess

3. **Check Recent Changes**
   - What changed that could cause this?
   - Git diff, recent commits
   - New dependencies, config changes

4. **Gather Evidence**
   - Add diagnostic logging at component boundaries
   - Log what data enters/exits each layer
   - Run once to gather evidence showing WHERE it breaks

5. **Trace Data Flow**
   - Where does bad value originate?
   - What called this with bad value?
   - Keep tracing up until you find the source

### Phase 2: Pattern Analysis

1. Find working examples in the codebase
2. Compare against references
3. Identify differences between working and broken
4. Understand dependencies

### Phase 3: Hypothesis and Testing

1. Form single hypothesis: "I think X is the root cause because Y"
2. Test minimally - SMALLEST possible change
3. Verify before continuing
4. If didn't work, form NEW hypothesis (don't stack fixes)

### Phase 4: Implementation

1. Create failing test case FIRST
2. Implement single fix addressing root cause
3. Verify fix - test passes, no regressions
4. If 3+ fixes failed: STOP and question the architecture

## Red Flags - STOP If You Think:

- "Quick fix for now, investigate later"
- "Just try changing X and see if it works"
- "I don't fully understand but this might work"
- "Let me add multiple changes and run tests"

ALL of these mean: STOP. Return to Phase 1.

## Report Format

When done, report:

1. **Root Cause:** What was actually causing the issue
2. **Evidence:** How you confirmed this was the root cause
3. **Fix Applied:** The specific change made
4. **Verification:** How you confirmed it's fixed
5. **Regression Test:** Test added to prevent recurrence
6. **Files Changed:** List of modified files
```

## Usage Example

```typescript
Task({
  subagent_type: 'general-purpose',
  description: 'Debug: 404 after creating resource',
  prompt: `You are debugging an issue. You MUST use the systematic-debugging skill.

## The Issue

After creating a new resource, the redirect to the detail page returns a 404.
The resource appears in the database but the page doesn't load.

Error in logs:
"Row not found"

## Reproduction Steps

1. Navigate to /resources/new
2. Fill in required fields
3. Click "Create"
4. Observe 404 error instead of detail page

## Environment

Project: [Your project name and stack]

## MANDATORY: Use Systematic Debugging

[... rest of template ...]
`,
})
```

## Debugging Session Flow

```
1. Subagent receives issue description
2. Phase 1: Investigate root cause
   - Read errors carefully
   - Reproduce issue
   - Check recent changes
   - Add diagnostic logging
   - Trace data flow
3. Phase 2: Pattern analysis
   - Find working examples
   - Compare differences
4. Phase 3: Hypothesis testing
   - Form single hypothesis
   - Test minimally
   - Verify or form new hypothesis
5. Phase 4: Fix implementation
   - Create failing test
   - Apply single fix
   - Verify fix works
6. Report findings
```

## When Subagent Gets Stuck

If the debugger subagent reports:

- "I've tried 3+ fixes without success"
- "Each fix reveals new problems"
- "This requires architectural changes"

**STOP** - This indicates an architectural problem, not a bug.
Discuss with your human partner before continuing.
