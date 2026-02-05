# Code Quality Reviewer Subagent

Use this subagent to review code quality after spec compliance is verified.

**Purpose:** Verify implementation is well-built (clean, tested, maintainable)

**When to use:** ONLY after spec compliance review passes

**CRITICAL:** MUST always be dispatched via the `Task` tool as a fresh subagent with NO shared conversation context. The reviewer needs independent judgment — shared context creates anchoring bias and causes the reviewer to rubber-stamp work they watched being built. Never run reviews inline in the main conversation.

## Dispatch Configuration

```
Task tool:
  subagent_type: superpowers:code-reviewer
  description: "Code quality review for Task N"
```

## Prompt Template

```
Review the implementation for code quality.

## What Was Implemented

[Summary from implementer's report]

## Plan/Requirements Reference

Task N from plan: [plan file path or inline requirements]

## Git Context

- Base SHA: [commit before task started]
- Head SHA: [current commit after implementation]

## Description

[Brief description of what the task accomplishes]

## Quality Standards

Check for:

**Architecture:**
- Controllers/handlers are thin (business logic in services/models)
- Proper separation of concerns
- Appropriate use of design patterns

**Code Quality:**
- Clear, descriptive naming
- Functions are focused and small
- No code duplication
- Consistent with codebase patterns

**Error Handling:**
- Errors are handled gracefully
- User-facing errors are friendly
- Errors are logged appropriately

**Testing:**
- Tests verify actual behavior
- Edge cases covered
- No mocked behavior tests
- Tests are readable and maintainable

**Performance:**
- Queries are efficient (no N+1)
- Appropriate caching where needed
- No obvious performance issues

**Security:**
- Input validation present
- Authorization checks in place
- No obvious vulnerabilities
```

## Review Criteria

The code reviewer evaluates:

**Strengths** - What was done well

**Issues** - Categorized by severity:

- **Critical:** Security issues, data loss risks, broken functionality
- **Important:** Performance problems, maintainability concerns, missing error handling
- **Minor:** Style inconsistencies, naming suggestions, documentation gaps

**Assessment** - Overall verdict:

- ✅ Approved
- ⚠️ Approved with suggestions
- ❌ Changes required

## Usage Example

```typescript
Task({
  subagent_type: 'superpowers:code-reviewer',
  description: 'Code quality review for Task 3',
  prompt: `Review the implementation for code quality.

## What Was Implemented

CommentThread component with threaded display, reply functionality,
author info, and delete capability.

## Plan/Requirements Reference

Task 3 from plan: "Create CommentThread component for displaying and interacting with comments"

## Git Context

- Base SHA: abc1234
- Head SHA: def5678

## Description

Adds the frontend component for displaying threaded comments with
full CRUD operations.

## Quality Standards

[... quality standards from template ...]
`,
})
```

## Review Loop

If code quality review returns issues:

1. **Critical issues:** Implementer MUST fix before proceeding
2. **Important issues:** Implementer SHOULD fix, reviewer re-reviews
3. **Minor issues:** Can be deferred or fixed at implementer's discretion

After fixes, dispatch another code quality review to verify.
