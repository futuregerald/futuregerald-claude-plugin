# JavaScript/TypeScript Best Practices

- Use ES modules with proper import sorting
- Prefer `function` keyword for top-level functions (hoisting, clearer stack traces)
- Use arrow functions for callbacks and inline functions
- Explicit return type annotations for public APIs
- Avoid `any` - use proper types or `unknown`
- Prefer `const` over `let`, never use `var`
- Use optional chaining (`?.`) and nullish coalescing (`??`)
- Destructure objects/arrays when it improves clarity
- Prefer `async/await` over raw Promises
- Use early returns to reduce nesting

```typescript
// Before
function processUser(user: User | null) {
  if (user) {
    if (user.isActive) {
      return user.name.toUpperCase()
    } else {
      return 'inactive'
    }
  } else {
    return 'unknown'
  }
}

// After
function processUser(user: User | null): string {
  if (!user) return 'unknown'
  if (!user.isActive) return 'inactive'
  return user.name.toUpperCase()
}
```
