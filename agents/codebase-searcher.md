# Codebase Searcher Subagent

Use this subagent for comprehensive codebase exploration and search tasks.

**Purpose:** Find files, patterns, implementations, and gather context across the codebase

**When to use:**

- Before implementation to understand existing patterns
- When investigating how something works
- Finding all usages of a function/component/pattern
- Discovering related files for a feature
- Answering architectural questions

## Dispatch Configuration

```
Task tool:
  subagent_type: Explore
  description: "Search: [what you're looking for]"
```

## Prompt Template

```
Search the codebase for: [SPECIFIC SEARCH GOAL]

## Search Context

[Why you need this information - what task or decision it supports]

## Search Scope

Project: [Project name and stack]

Key directories to check:
[List relevant directories for your project, e.g.:]
- src/controllers/ - API controllers
- src/models/ - Data models
- src/services/ - Business logic
- src/components/ - UI components
- tests/ - Test files

## What I Need

[Specific questions to answer, patterns to find, or files to locate]

Examples:
- "Find all places where comments are handled"
- "How does the reaction system work end-to-end?"
- "What components exist for user display?"
- "Find the pattern used for API responses"

## Report Format

Provide:
1. **Files Found** - Relevant files with brief description of each
2. **Patterns Discovered** - How the codebase handles similar things
3. **Key Code Sections** - Important snippets with file:line references
4. **Gaps/Missing** - What doesn't exist yet that might be needed
5. **Recommendations** - Suggestions based on findings
```

## Thoroughness Levels

Specify in the Task description:

- **quick** - Basic pattern matching, first few results
- **medium** - Moderate exploration, follows some references
- **very thorough** - Comprehensive analysis, multiple search strategies

## Usage Examples

### Finding existing patterns

```typescript
Task({
  subagent_type: 'Explore',
  description: 'Search: comment handling patterns (medium)',
  prompt: `Search the codebase for: How comments are currently handled

## Search Context

I need to implement a CommentThread component and want to understand
the existing comment infrastructure.

## What I Need

1. Where is the Comment model defined?
2. What API endpoints exist for comments?
3. Are there any existing comment-related components?
4. How are comments serialized for the frontend?
5. What validation exists for comment input?

[... rest of template ...]
`,
})
```

### Finding all usages

```typescript
Task({
  subagent_type: 'Explore',
  description: 'Search: Button component usages (quick)',
  prompt: `Search the codebase for: All usages of the Button component

## Search Context

Planning to update Button variants and need to understand impact.

## What I Need

1. Which pages/components import Button?
2. What variants are currently used?
3. Are there any custom styling overrides?

[... rest of template ...]
`,
})
```

### Architectural investigation

```typescript
Task({
  subagent_type: 'Explore',
  description: 'Search: notification system architecture (very thorough)',
  prompt: `Search the codebase for: Complete notification system implementation

## Search Context

Need to add UI for notifications. Want complete picture of backend.

## What I Need

1. Notification model and relationships
2. How notifications are created (triggers)
3. API endpoints for notifications
4. Any existing frontend components
5. Email notification integration
6. Preference management

[... rest of template ...]
`,
})
```

## Best Practices

1. **Be specific** - "Find comment components" vs "Find files"
2. **Provide context** - Why you need this helps focus the search
3. **Set thoroughness** - Don't over-search for simple lookups
4. **Use for unknowns** - Don't guess, search first
5. **Follow up** - If initial search is incomplete, refine and search again
