# SQLite TypeScript/libSQL Patterns (Reference)

TypeScript, Bun, and edge deployment patterns for libSQL/Turso. See [../SKILL.md](../SKILL.md) for core patterns and overview.

---

## libSQL / Turso Connection Setup

**Local libSQL (Development)**

```typescript
import { createClient } from '@libsql/client'

// Local file database
const db = createClient({
  url: 'file:local.db',
})

// In-memory for testing
const testDb = createClient({
  url: ':memory:',
})
```

**Turso (Production)**

```typescript
import { createClient } from '@libsql/client'

const db = createClient({
  url: process.env.TURSO_DATABASE_URL!,
  authToken: process.env.TURSO_AUTH_TOKEN!,
})
```

**Turso with Embedded Replica (Low Latency)**

```typescript
import { createClient } from '@libsql/client'

// Syncs from remote, reads from local replica
const db = createClient({
  url: 'file:local-replica.db',
  syncUrl: process.env.TURSO_DATABASE_URL!,
  authToken: process.env.TURSO_AUTH_TOKEN!,
  syncInterval: 60, // Sync every 60 seconds
})

// Manual sync when needed
await db.sync()
```

## Parameterized Queries (CRITICAL)

```typescript
// CORRECT: Parameterized query - ALWAYS use this pattern
async function getUserById(id: string) {
  const result = await db.execute({
    sql: 'SELECT id, name, email FROM users WHERE id = ?',
    args: [id],
  })
  return result.rows[0]
}

// CORRECT: Named parameters for clarity
async function searchUsers(name: string, status: string) {
  const result = await db.execute({
    sql: 'SELECT * FROM users WHERE name LIKE :name AND status = :status',
    args: { name: `%${name}%`, status },
  })
  return result.rows
}

// INCORRECT: SQL Injection vulnerability - NEVER do this
async function getUserUnsafe(id: string) {
  // DANGER: SQL injection risk!
  const result = await db.execute(`SELECT * FROM users WHERE id = '${id}'`)
  return result.rows[0]
}
```

## Batch Operations

```typescript
// Batch insert with transaction
async function createUsers(users: Array<{ name: string; email: string }>) {
  const statements = users.map((user) => ({
    sql: 'INSERT INTO users (name, email) VALUES (?, ?)',
    args: [user.name, user.email],
  }))

  // Executes all in single round-trip, atomic transaction
  const results = await db.batch(statements, 'write')
  return results
}
```

## Transactions

```typescript
async function transferFunds(fromId: string, toId: string, amount: number) {
  const tx = await db.transaction('write')

  try {
    await tx.execute({
      sql: 'UPDATE accounts SET balance = balance - ? WHERE id = ?',
      args: [amount, fromId],
    })

    await tx.execute({
      sql: 'UPDATE accounts SET balance = balance + ? WHERE id = ?',
      args: [amount, toId],
    })

    await tx.commit()
  } catch (error) {
    await tx.rollback()
    throw error
  }
}
```

## Repository Pattern (Recommended)

```typescript
// repositories/user-repository.ts
import { Client } from '@libsql/client'

export class UserRepository {
  constructor(private db: Client) {}

  async findById(id: string) {
    const result = await this.db.execute({
      sql: 'SELECT * FROM users WHERE id = ?',
      args: [id],
    })
    return result.rows[0] ?? null
  }

  async create(data: { name: string; email: string }) {
    const id = crypto.randomUUID()
    await this.db.execute({
      sql: 'INSERT INTO users (id, name, email, created_at) VALUES (?, ?, ?, ?)',
      args: [id, data.name, data.email, new Date().toISOString()],
    })
    return { id, ...data }
  }

  async update(id: string, data: Partial<{ name: string; email: string }>) {
    const sets: string[] = []
    const args: unknown[] = []

    if (data.name !== undefined) {
      sets.push('name = ?')
      args.push(data.name)
    }
    if (data.email !== undefined) {
      sets.push('email = ?')
      args.push(data.email)
    }

    if (sets.length === 0) return

    args.push(id)
    await this.db.execute({
      sql: `UPDATE users SET ${sets.join(', ')} WHERE id = ?`,
      args,
    })
  }

  async delete(id: string) {
    await this.db.execute({
      sql: 'DELETE FROM users WHERE id = ?',
      args: [id],
    })
  }
}
```

## Migrations for libSQL/Turso

```typescript
// migrations/001_create_users.ts
export const up = async (db: Client) => {
  await db.execute(`
    CREATE TABLE IF NOT EXISTS users (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      email TEXT NOT NULL UNIQUE,
      created_at TEXT NOT NULL,
      updated_at TEXT
    )
  `)
  await db.execute(`
    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)
  `)
}

export const down = async (db: Client) => {
  await db.execute('DROP TABLE IF EXISTS users')
}

// migrate.ts - Simple migration runner
import { createClient } from '@libsql/client'
import * as migration001 from './migrations/001_create_users'

const migrations = [migration001]

async function migrate() {
  const db = createClient({ url: process.env.DATABASE_URL! })

  await db.execute(`
    CREATE TABLE IF NOT EXISTS _migrations (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL UNIQUE,
      applied_at TEXT NOT NULL
    )
  `)

  const applied = await db.execute('SELECT name FROM _migrations')
  const appliedNames = new Set(applied.rows.map((r) => r.name))

  for (const [index, migration] of migrations.entries()) {
    const name = `${String(index + 1).padStart(3, '0')}`
    if (!appliedNames.has(name)) {
      console.log(`Applying migration ${name}...`)
      await migration.up(db)
      await db.execute({
        sql: 'INSERT INTO _migrations (name, applied_at) VALUES (?, ?)',
        args: [name, new Date().toISOString()],
      })
    }
  }

  console.log('Migrations complete')
}
```

## Edge Deployment Patterns

**Cloudflare Workers**

```typescript
import { createClient } from '@libsql/client/web'

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const db = createClient({
      url: env.TURSO_DATABASE_URL,
      authToken: env.TURSO_AUTH_TOKEN,
    })

    const users = await db.execute('SELECT * FROM users LIMIT 10')
    return Response.json(users.rows)
  },
}
```

**Vercel Edge Functions**

```typescript
import { createClient } from '@libsql/client/web'

export const config = { runtime: 'edge' }

export default async function handler(request: Request) {
  const db = createClient({
    url: process.env.TURSO_DATABASE_URL!,
    authToken: process.env.TURSO_AUTH_TOKEN!,
  })

  const result = await db.execute('SELECT COUNT(*) as count FROM users')
  return Response.json({ count: result.rows[0].count })
}
```

**Bun with libSQL**

```typescript
import { createClient } from '@libsql/client'

const db = createClient({
  url: process.env.TURSO_DATABASE_URL!,
  authToken: process.env.TURSO_AUTH_TOKEN!,
})

Bun.serve({
  port: 3000,
  async fetch(req) {
    const url = new URL(req.url)

    if (url.pathname === '/users') {
      const result = await db.execute('SELECT * FROM users')
      return Response.json(result.rows)
    }

    return new Response('Not found', { status: 404 })
  },
})
```

## TypeScript/libSQL Testing

```typescript
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { createClient, Client } from '@libsql/client'
import { UserRepository } from './user-repository'

describe('UserRepository', () => {
  let db: Client
  let repo: UserRepository

  beforeEach(async () => {
    db = createClient({ url: ':memory:' })
    await db.execute(`
      CREATE TABLE users (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        created_at TEXT NOT NULL
      )
    `)
    repo = new UserRepository(db)
  })

  afterEach(() => {
    db.close()
  })

  it('creates user with valid data', async () => {
    const user = await repo.create({ name: 'Test', email: 'test@example.com' })
    expect(user.id).toBeDefined()
    expect(user.name).toBe('Test')
  })

  it('prevents SQL injection in search', async () => {
    await repo.create({ name: 'Test', email: 'test@example.com' })
    const result = await repo.search("'; DROP TABLE users; --")
    const count = await db.execute('SELECT COUNT(*) as count FROM users')
    expect(count.rows[0].count).toBe(1)
  })
})
```

## Python TDD Patterns

### Write Failing Test First

```python
# tests/test_user_repository.py
import pytest
import sqlite3

@pytest.fixture
def db():
    """In-memory SQLite for fast testing."""
    conn = sqlite3.connect(":memory:")
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON")
    yield conn
    conn.close()

class TestUserRepository:
    def test_create_user_returns_id(self, db):
        repo = UserRepository(db)
        repo.initialize_schema()
        user_id = repo.create_user("test@example.com", "Test User")
        assert user_id > 0

    def test_sql_injection_prevented(self, db):
        repo = UserRepository(db)
        repo.initialize_schema()
        malicious = "'; DROP TABLE users; --"
        user_id = repo.create_user(malicious, "Hacker")
        assert repo.get_by_id(user_id)["email"] == malicious
```

### Implement Minimum Code to Pass

```python
# app/repositories/user.py
class UserRepository:
    def __init__(self, conn):
        self.conn = conn

    def initialize_schema(self):
        self.conn.execute("""
            CREATE TABLE IF NOT EXISTS users (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                email TEXT NOT NULL UNIQUE,
                name TEXT NOT NULL
            )""")
        self.conn.commit()

    def create_user(self, email: str, name: str) -> int:
        cursor = self.conn.execute(
            "INSERT INTO users (email, name) VALUES (?, ?)", (email, name))
        self.conn.commit()
        return cursor.lastrowid

    def get_by_id(self, user_id: int):
        return self.conn.execute(
            "SELECT * FROM users WHERE id = ?", (user_id,)).fetchone()
```

## Performance Patterns

### WAL Mode

```python
conn.execute("PRAGMA journal_mode = WAL")
conn.execute("PRAGMA synchronous = NORMAL")
conn.execute("PRAGMA cache_size = -64000")  # 64MB
```

### Batch Inserts

```python
# Good: Single transaction for batch
conn.executemany("INSERT INTO items (name) VALUES (?)", records)
conn.commit()

# Bad: Commit per row (100x slower)
```

### Connection Pooling

```python
from queue import Queue
class ConnectionPool:
    def __init__(self, db_path, size=5):
        self.pool = Queue(size)
        for _ in range(size):
            conn = sqlite3.connect(db_path, check_same_thread=False)
            conn.execute("PRAGMA journal_mode = WAL")
            self.pool.put(conn)
```

### Index Optimization

```python
conn.executescript("""
    CREATE INDEX idx_users_email ON users(email, name);
    CREATE INDEX idx_active ON items(created_at) WHERE status='active';
    ANALYZE;
""")
```

### VACUUM Scheduling

```python
def nightly_maintenance(conn):
    conn.execute("PRAGMA optimize")
    freelist = conn.execute("PRAGMA freelist_count").fetchone()[0]
    if freelist > 1000:
        conn.execute("VACUUM")
```
