## Node.js/TypeScript Rules

### TypeScript Best Practices

- Avoid `any` - use proper types or `unknown`
- Prefer `const` over `let`, never use `var`
- Use optional chaining (`?.`) and nullish coalescing (`??`)
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

### Error Handling

```typescript
async function apiCall() {
  try {
    const response = await fetch('/api/endpoint')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }
    return await response.json()
  } catch (error) {
    console.error('API call failed:', error)
    throw error
  }
}
```

### Module Structure

- Use ES modules with proper import sorting
- Prefer `function` keyword for top-level functions
- Use arrow functions for callbacks
