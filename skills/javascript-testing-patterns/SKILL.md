---
name: javascript-testing-patterns
description: Comprehensive JavaScript/TypeScript testing patterns for Jest, Vitest, and AdonisJS/Japa. Use when writing tests, reviewing test code, or debugging test failures.
tags: [testing, javascript]
---

# JavaScript Testing Patterns

## Reference Files

| File | When to read |
|------|-------------|
| [references/adonisjs-japa.md](references/adonisjs-japa.md) | Writing or reviewing AdonisJS tests with Japa runner, database transaction tests, HTTP client tests, Sinon mocks |

---

## Core Principle: AAA Pattern

Every test follows **Arrange, Act, Assert**:

```typescript
test('calculates total with discount', () => {
  // Arrange - Set up test data
  const cart = { items: [{ price: 100 }], discount: 0.1 }

  // Act - Execute the code under test
  const total = calculateTotal(cart)

  // Assert - Verify the result
  expect(total).toBe(90)
})
```

## Framework Quick Reference

| Framework     | Run Tests       | Watch Mode            | Coverage                   |
| ------------- | --------------- | --------------------- | -------------------------- |
| Jest          | `npm test`      | `npm test -- --watch` | `npm test -- --coverage`   |
| Vitest        | `npx vitest`    | `npx vitest --watch`  | `npx vitest --coverage`    |
| AdonisJS/Japa | `node ace test` | N/A                   | `node ace test --coverage` |

---

## Jest/Vitest Patterns

### Configuration

**Jest (jest.config.js)**

```javascript
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  coverageThreshold: {
    global: { branches: 80, functions: 80, lines: 80, statements: 80 },
  },
  setupFilesAfterEnv: ['./jest.setup.ts'],
}
```

**Vitest (vitest.config.ts)**

```typescript
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    coverage: { provider: 'v8', reporter: ['text', 'json', 'html'] },
  },
})
```

### Unit Testing Pure Functions

```typescript
import { describe, it, expect } from 'vitest' // or jest

describe('calculateDiscount', () => {
  it('returns 0 for amounts below threshold', () => {
    expect(calculateDiscount(50)).toBe(0)
  })

  it('applies 10% discount for amounts over 100', () => {
    expect(calculateDiscount(200)).toBe(20)
  })

  it('throws for negative amounts', () => {
    expect(() => calculateDiscount(-50)).toThrow('Amount cannot be negative')
  })
})
```

### Mocking Strategies

**Module Mocking**

```typescript
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { sendEmail } from './email-service'
import { UserService } from './user-service'

vi.mock('./email-service', () => ({
  sendEmail: vi.fn().mockResolvedValue({ sent: true }),
}))

describe('UserService with mocked email', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('sends welcome email on registration', async () => {
    const service = new UserService()
    await service.register({ email: 'new@test.com' })

    expect(sendEmail).toHaveBeenCalledWith({
      to: 'new@test.com',
      template: 'welcome',
    })
  })
})
```

**Dependency Injection (Preferred)**

```typescript
interface EmailSender {
  send(to: string, template: string): Promise<void>
}

class UserService {
  constructor(private emailSender: EmailSender) {}

  async register(data: { email: string }) {
    await this.emailSender.send(data.email, 'welcome')
  }
}

// In tests
describe('UserService', () => {
  it('sends welcome email', async () => {
    const mockSender = { send: vi.fn().mockResolvedValue(undefined) }
    const service = new UserService(mockSender)

    await service.register({ email: 'test@example.com' })

    expect(mockSender.send).toHaveBeenCalledWith('test@example.com', 'welcome')
  })
})
```

### Async Testing

```typescript
describe('API client', () => {
  it('fetches data successfully', async () => {
    const data = await fetchUser(123)
    expect(data.id).toBe(123)
  })

  it('retries on failure', async () => {
    const mockFetch = vi
      .fn()
      .mockRejectedValueOnce(new Error('Network error'))
      .mockRejectedValueOnce(new Error('Network error'))
      .mockResolvedValueOnce({ data: 'success' })

    const result = await fetchWithRetry(mockFetch, 3)

    expect(result.data).toBe('success')
    expect(mockFetch).toHaveBeenCalledTimes(3)
  })
})
```

### Integration Testing with Supertest

```typescript
import request from 'supertest'
import { app } from '../app'
import { db } from '../database'

describe('POST /api/users', () => {
  beforeAll(async () => { await db.migrate.latest() })
  afterEach(async () => { await db('users').truncate() })
  afterAll(async () => { await db.destroy() })

  it('creates user and returns 201', async () => {
    const response = await request(app)
      .post('/api/users')
      .send({ email: 'test@example.com', password: 'secure123' })
      .expect(201)

    expect(response.body).toMatchObject({
      id: expect.any(Number),
      email: 'test@example.com',
    })
  })
})
```

### React Component Testing

```typescript
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LoginForm } from './LoginForm'

describe('LoginForm', () => {
  it('submits with valid credentials', async () => {
    const onSubmit = vi.fn()
    render(<LoginForm onSubmit={onSubmit} />)

    await userEvent.type(screen.getByLabelText('Email'), 'test@example.com')
    await userEvent.type(screen.getByLabelText('Password'), 'password123')
    await userEvent.click(screen.getByRole('button', { name: /login/i }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123'
      })
    })
  })

  it('shows validation error for empty email', async () => {
    render(<LoginForm onSubmit={vi.fn()} />)
    await userEvent.click(screen.getByRole('button', { name: /login/i }))
    expect(screen.getByText('Email is required')).toBeInTheDocument()
  })
})
```

### Test Factories with Faker

```typescript
import { faker } from '@faker-js/faker'

export const createTestUser = (overrides = {}) => ({
  id: faker.string.uuid(),
  email: faker.internet.email(),
  name: faker.person.fullName(),
  createdAt: faker.date.past(),
  ...overrides
})
```

---

## Anti-Patterns to Avoid

### Don't Test Implementation Details

```typescript
// BAD - Test observable behavior, not internal methods
test('calls internal method', async () => {
  const spy = vi.spyOn(service, '_internalHelper')
  await service.doThing()
  expect(spy).toHaveBeenCalled()
})

// GOOD
test('produces correct output', async () => {
  const result = await service.doThing()
  expect(result).toEqual(expected)
})
```

### Don't Over-Mock

```typescript
// BAD - Testing mock, not real code
test('calls database', async () => {
  const mockDb = { query: vi.fn().mockResolvedValue([]) }
  const service = new UserService(mockDb)
  await service.getUsers()
  expect(mockDb.query).toHaveBeenCalled()
})

// GOOD - Test real behavior with test database
test('returns users from database', async () => {
  await User.create({ name: 'Test' })
  const users = await service.getUsers()
  expect(users).toHaveLength(1)
})
```

### Don't Forget Cleanup

Use transaction rollback or `afterEach` cleanup to prevent test pollution.

---

## File Organization

```
tests/
  functional/           # HTTP/integration tests
  unit/                 # Unit tests
  factories/            # Test data factories
  bootstrap.ts          # Test setup
```

## Quick Reference

| Action         | Jest/Vitest                    | AdonisJS/Japa                  |
| -------------- | ------------------------------ | ------------------------------ |
| Run tests      | `npm test`                     | `node ace test`                |
| Run file       | `npm test -- path/to/file`     | `node ace test --files="name"` |
| Mock function  | `vi.fn()` / `jest.fn()`        | `sinon.stub()`                 |
| Spy            | `vi.spyOn()`                   | `sinon.spy()`                  |
| Auth request   | N/A (manual)                   | `.loginAs(user)`               |
| Assert status  | `expect(res.status).toBe(200)` | `response.assertStatus(200)`   |
