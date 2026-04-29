---
name: error-handling-patterns
description: Master error handling patterns across languages including exceptions, Result types, error propagation, and graceful degradation to build resilient applications. Use when implementing error handling, designing APIs, or improving application reliability.
tags: [quality]
---

# Error Handling Patterns

Build resilient applications with robust error handling strategies that gracefully handle failures and provide excellent debugging experiences.

## When to Use This Skill

- Implementing error handling in new features
- Designing error-resilient APIs
- Debugging production issues
- Improving application reliability
- Creating better error messages for users and developers
- Implementing retry and circuit breaker patterns
- Handling async/concurrent errors
- Building fault-tolerant distributed systems

## Core Concepts

### Error Handling Philosophies

| Approach | When to Use |
|----------|-------------|
| **Exceptions** (try-catch) | Unexpected errors, exceptional conditions |
| **Result Types** (Ok/Err) | Expected errors, validation failures |
| **Error Codes** | C-style APIs requiring discipline |
| **Option/Maybe Types** | Nullable values, absent data |
| **Panics/Crashes** | Unrecoverable errors, programming bugs |

### Error Categories

**Recoverable:** Network timeouts, missing files, invalid user input, API rate limits -- retry, degrade, or prompt the user.

**Unrecoverable:** Out of memory, stack overflow, null-pointer bugs -- fail fast, log, and crash cleanly.

## Universal Patterns

### Pattern 1: Circuit Breaker

Prevent cascading failures by tracking consecutive errors and short-circuiting calls to a failing dependency.

- **CLOSED** -- normal operation; failures increment a counter.
- **OPEN** -- failure threshold reached; all calls rejected immediately.
- **HALF-OPEN** -- after a timeout, allow a probe call; if it succeeds, reset to CLOSED.

```typescript
class CircuitBreaker {
  private state: 'CLOSED' | 'OPEN' | 'HALF_OPEN' = 'CLOSED'
  private failures = 0
  private lastFailure = 0

  constructor(
    private threshold = 5,
    private timeoutMs = 60_000,
    private successesToReset = 2
  ) {}

  async call<T>(fn: () => Promise<T>): Promise<T> {
    if (this.state === 'OPEN') {
      if (Date.now() - this.lastFailure > this.timeoutMs) {
        this.state = 'HALF_OPEN'
      } else {
        throw new Error('Circuit breaker is OPEN')
      }
    }

    try {
      const result = await fn()
      this.onSuccess()
      return result
    } catch (e) {
      this.onFailure()
      throw e
    }
  }

  private onSuccess() {
    this.failures = 0
    if (this.state === 'HALF_OPEN') this.state = 'CLOSED'
  }

  private onFailure() {
    this.failures++
    this.lastFailure = Date.now()
    if (this.failures >= this.threshold) this.state = 'OPEN'
  }
}

// Usage
const breaker = new CircuitBreaker()
const data = await breaker.call(() => externalApi.getData())
```

### Pattern 2: Error Aggregation

Collect multiple errors instead of failing on the first one -- essential for form validation and batch processing.

```typescript
class ErrorCollector {
  private errors: Error[] = []

  add(error: Error): void { this.errors.push(error) }
  hasErrors(): boolean { return this.errors.length > 0 }

  throw(): never {
    if (this.errors.length === 1) throw this.errors[0]
    throw new AggregateError(this.errors, `${this.errors.length} errors occurred`)
  }
}

// Usage: validate multiple fields at once
function validateUser(data: any): User {
  const errors = new ErrorCollector()

  if (!data.email)                errors.add(new ValidationError('Email is required'))
  else if (!isValidEmail(data.email)) errors.add(new ValidationError('Email is invalid'))

  if (!data.name || data.name.length < 2)
    errors.add(new ValidationError('Name must be at least 2 characters'))

  if (errors.hasErrors()) errors.throw()
  return data as User
}
```

### Pattern 3: Graceful Degradation

Provide fallback functionality when the primary path errors.

```python
from typing import Optional, Callable, TypeVar

T = TypeVar('T')

def with_fallback(
    primary: Callable[[], T],
    fallback: Callable[[], T],
    log_error: bool = True
) -> T:
    """Try primary function, fall back on error."""
    try:
        return primary()
    except Exception as e:
        if log_error:
            logger.error(f"Primary function failed: {e}")
        return fallback()

# Usage
profile = with_fallback(
    primary=lambda: fetch_from_cache(user_id),
    fallback=lambda: fetch_from_database(user_id)
)
```

## Best Practices

1. **Fail Fast** -- validate input early, fail quickly
2. **Preserve Context** -- include stack traces, metadata, timestamps
3. **Meaningful Messages** -- explain what happened and how to fix it
4. **Log Appropriately** -- error = log; expected failure = don't spam logs
5. **Handle at Right Level** -- catch where you can meaningfully handle
6. **Clean Up Resources** -- use try-finally, context managers, defer
7. **Don't Swallow Errors** -- log or re-throw, don't silently ignore
8. **Type-Safe Errors** -- use typed errors when possible

## Common Pitfalls

- **Catching Too Broadly** -- `except Exception` hides bugs
- **Empty Catch Blocks** -- silently swallowing errors
- **Logging and Re-throwing** -- creates duplicate log entries
- **Not Cleaning Up** -- forgetting to close files, connections
- **Poor Error Messages** -- "Error occurred" is not helpful
- **Returning Error Codes** -- use exceptions or Result types instead
- **Ignoring Async Errors** -- unhandled promise rejections

## Reference Files

| File | Contents |
|------|----------|
| [references/language-patterns.md](references/language-patterns.md) | Python, TypeScript, Rust, and Go error-handling idioms with full examples |
