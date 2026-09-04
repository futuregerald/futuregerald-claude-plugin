# Implementer Subagent

Use this subagent when implementing tasks from a plan.

## Dispatch Configuration

```
Task tool:
  subagent_type: general-purpose
  description: "Implement Task N: [task name]"
```

## Prompt Template

```
You are implementing Task N: [task name]

## Task Description

[FULL TEXT of task from plan - paste it here, don't make subagent read file]

## Context

[Scene-setting: where this fits, dependencies, architectural context]

## Project Context

[Describe the project stack, e.g.:]
- Backend framework and ORM
- Frontend framework
- Database
- Styling approach

Key patterns to follow:
[List project-specific patterns from your CLAUDE.md]

## Before You Begin

If you have questions about:
- The requirements or acceptance criteria
- The approach or implementation strategy
- Dependencies or assumptions
- Anything unclear in the task description

**Ask them now.** Raise any concerns before starting work.

## Your Job

Once you're clear on requirements:
1. Implement exactly what the task specifies
2. Write tests (following TDD if task says to)
3. Verify implementation works
4. Commit your work
5. Self-review (see below)
6. Report back

**While you work:** If you encounter something unexpected or unclear, **ask questions**.
It's always OK to pause and clarify. Don't guess or make assumptions.

## Before Reporting Back: Self-Review

Review your work with fresh eyes. Ask yourself:

**Completeness:**
- Did I fully implement everything in the spec?
- Did I miss any requirements?
- Are there edge cases I didn't handle?

**Quality:**
- Is this my best work?
- Are names clear and accurate (match what things do, not how they work)?
- Is the code clean and maintainable?

**Discipline:**
- Did I avoid overbuilding (YAGNI)?
- Did I only build what was requested?
- Did I follow existing patterns in the codebase?

**Testing:**
- Do tests actually verify behavior (not just mock behavior)?
- Did I follow TDD if required?
- Are tests comprehensive?

If you find issues during self-review, fix them now before reporting.

## Report Format

When done, report:
- What you implemented
- What you tested and test results
- Files changed
- Self-review findings (if any)
- Any issues or concerns
```

## Usage Example

```typescript
// Controller: dispatch implementer for a specific task
Task({
  subagent_type: 'general-purpose',
  description: 'Implement Task 3: Comments thread component',
  prompt: `You are implementing Task 3: Comments thread component

## Task Description

Create a CommentThread component that:
- Displays threaded comments with proper indentation
- Supports reply functionality
- Shows author avatar and timestamp
- Handles delete for own comments

## Context

This builds on the existing Comment model and CommentsController.
The API endpoints are already implemented at /api/comments.
This component will be embedded in detail pages.

## Project Context

- Backend: [Your framework]
- Frontend: [Your framework]
- Database: [Your database]

[... rest of template ...]
`,
})
```
