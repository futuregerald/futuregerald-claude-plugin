# AdonisJS/Japa Testing Patterns (Reference)

Testing patterns specific to AdonisJS v6 with Japa test runner. See [../SKILL.md](../SKILL.md) for Jest/Vitest patterns and core concepts.

---

## Running Tests

```bash
# Run all tests
node ace test

# Run specific suite
node ace test functional
node ace test unit

# Run specific file
node ace test functional --files="user_auth"

# Run with coverage
node ace test --coverage
```

## Test Structure

```typescript
import { test } from '@japa/runner'

test.group('Feature | Description', (group) => {
  group.each.setup(() => {
    // runs before each test
  })

  group.each.teardown(() => {
    // runs after each test
  })

  test('specific behavior', async ({ assert }) => {
    const result = someFunction()
    assert.equal(result, expected)
  })
})
```

## Database Testing with Transactions

```typescript
import { test } from '@japa/runner'
import testUtils from '@adonisjs/core/services/test_utils'
import User from '#models/user'

test.group('Database tests', (group) => {
  // Wrap each test in a transaction that rolls back
  group.each.setup(() => testUtils.db().withGlobalTransaction())

  test('creates a record', async ({ assert }) => {
    const user = await User.create({ email: 'test@example.com' })
    assert.isNotNull(user.id)
    // Transaction rolls back - no cleanup needed
  })
})
```

## HTTP Testing

**Basic Request**

```typescript
test.group('API | Users', (group) => {
  group.each.setup(() => testUtils.db().withGlobalTransaction())

  test('GET /users returns list', async ({ client, assert }) => {
    const response = await client.get('/users')

    response.assertStatus(200)
    assert.isArray(response.body())
  })
})
```

**Authenticated Requests**

```typescript
test('authenticated endpoint', async ({ client }) => {
  const user = await User.create({
    /* ... */
  })

  // Web session auth
  const response = await client.get('/dashboard').loginAs(user)

  // API token auth
  const apiResponse = await client.get('/api/v1/me').loginAs(user, 'api')

  response.assertStatus(200)
})
```

**Testing Redirects**

```typescript
test('redirects after action', async ({ client }) => {
  const user = await User.create({
    /* ... */
  })

  const response = await client
    .post('/logout')
    .redirects(0) // Don't follow redirects
    .loginAs(user)

  response.assertStatus(302)
  response.assertHeader('location', '/login')
})
```

**Form and JSON Submissions**

```typescript
// Form data
const response = await client
  .post('/posts')
  .form({ title: 'My Post', description: 'A test post' })
  .loginAs(user)

// JSON API
const response = await client
  .post('/api/v1/posts')
  .json({ title: 'My Post', description: 'A test post' })
  .loginAs(user, 'api')

// AJAX request
const response = await client
  .post('/comments')
  .header('X-Requested-With', 'XMLHttpRequest')
  .form({ content: 'Test comment' })
  .loginAs(user)
```

## Japa Assertions

```typescript
test('assertions example', async ({ assert }) => {
  // Equality
  assert.equal(actual, expected)
  assert.deepEqual(obj1, obj2)

  // Truthiness
  assert.isTrue(value)
  assert.isFalse(value)
  assert.isNull(value)
  assert.isNotNull(value)

  // Types
  assert.isString(value)
  assert.isArray(value)
  assert.isObject(value)

  // Arrays/Objects
  assert.lengthOf(array, 3)
  assert.include(array, item)
  assert.property(obj, 'key')
  assert.containsSubset(obj, { key: 'value' })

  // Exceptions
  assert.throws(() => throwingFn(), Error)
  await assert.rejects(async () => asyncThrowingFn(), Error)
})
```

## Response Assertions

```typescript
response.assertStatus(200)
response.assertHeader('content-type', 'application/json')
response.assertHeader('location', '/dashboard')
response.assertBody({ success: true })
response.assertBodyContains({ id: 1 })
response.assertTextIncludes('Welcome')
```

## Testing with Sinon Mocks

```typescript
import sinon from 'sinon'
import EmailService from '#services/email_service'

test.group('With mocks', (group) => {
  group.each.teardown(() => {
    sinon.restore()
  })

  test('sends email on registration', async ({ assert }) => {
    const sendStub = sinon.stub(EmailService, 'send').resolves()

    await UserService.register({ email: 'test@example.com' })

    assert.isTrue(sendStub.calledOnce)
    assert.equal(sendStub.firstCall.args[0], 'test@example.com')
  })
})
```

## Common Test Patterns

**Auth Required Routes**

```typescript
test('requires authentication', async ({ client }) => {
  const response = await client.get('/dashboard').redirects(0)
  response.assertStatus(302)
  response.assertHeader('location', '/login')
})

test('API returns 401 without auth', async ({ client }) => {
  const response = await client.get('/api/v1/me')
  response.assertStatus(401)
})
```

**Validation Errors**

```typescript
test('validates required fields', async ({ client }) => {
  const user = await User.create({
    /* ... */
  })

  const response = await client.post('/api/v1/posts').json({}).loginAs(user, 'api')

  response.assertStatus(422)
  response.assertBodyContains({ code: 'E_VALIDATION' })
})
```

**Authorization**

```typescript
test('denies access to other user resources', async ({ client }) => {
  const owner = await User.create({ email: 'owner@test.com' })
  const other = await User.create({ email: 'other@test.com' })
  const resource = await Resource.create({ ownerId: owner.id })

  const response = await client
    .patch(`/api/v1/resources/${resource.id}`)
    .json({ title: 'Hacked' })
    .loginAs(other, 'api')

  response.assertStatus(403)
})
```
