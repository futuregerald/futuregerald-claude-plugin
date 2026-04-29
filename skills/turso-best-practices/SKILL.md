---
name: turso-best-practices
description: Turso and libSQL best practices for SQLite-compatible cloud database development with edge distribution, embedded replicas, and vector search.
tags: [framework, database]
---

# Turso & libSQL Best Practices

## Reference Files

| File | When to read |
|------|-------------|
| [references/advanced-features.md](references/advanced-features.md) | Working with embedded replicas, vector search/AI embeddings, Drizzle ORM, database branching, encryption at rest, or SQLite extensions |

---

## Overview

Turso is a fully managed SQLite-compatible database platform built on libSQL, a fork of SQLite. It provides edge distribution, embedded replicas, native vector search, branching, and point-in-time recovery. Core principle: **SQLite simplicity with cloud-scale distribution**.

## When to Use

- Building applications needing SQLite with cloud features
- Implementing embedded replicas for offline-first apps
- Adding vector search/AI embeddings to applications
- Setting up local development with Turso
- Managing database migrations and branching

## Quick Reference

| Task                  | Command/Pattern                                                             |
| --------------------- | --------------------------------------------------------------------------- |
| Install CLI (macOS)   | `brew install tursodatabase/tap/turso`                                      |
| Install CLI (Linux)   | `curl -sSfL https://get.tur.so/install.sh \| bash`                          |
| Login                 | `turso auth login`                                                          |
| Create database       | `turso db create my-db`                                                     |
| Connect to shell      | `turso db shell my-db`                                                      |
| Get credentials       | `turso db show my-db --url` and `turso db tokens create my-db`              |
| Local dev server      | `turso dev`                                                                 |
| Local with file       | `turso dev --db-file local.db`                                              |

## Installation & Setup

### CLI Installation

```bash
# macOS
brew install tursodatabase/tap/turso

# Linux / Windows (WSL)
curl -sSfL https://get.tur.so/install.sh | bash
```

### Authentication

```bash
turso auth signup     # Sign up (opens browser)
turso auth login      # Login (opens browser)
turso auth login --headless  # Headless mode (WSL/CI)
```

### Create Your First Database

```bash
turso db create my-db          # Auto-detects closest region
turso db show my-db            # Show database info
turso db show my-db --url      # Get connection URL
turso db tokens create my-db   # Create auth token
turso db shell my-db           # Connect to shell
```

## SDK Usage (TypeScript/JavaScript)

### Installation

```bash
npm install @libsql/client
```

### Basic Connection

```typescript
import { createClient } from '@libsql/client'

const client = createClient({
  url: process.env.TURSO_DATABASE_URL!,
  authToken: process.env.TURSO_AUTH_TOKEN,
})
```

### Execute Queries

```typescript
// Simple query
const result = await client.execute('SELECT * FROM users')

// Positional placeholders
const result = await client.execute({
  sql: 'SELECT * FROM users WHERE id = ?',
  args: [1],
})

// Named placeholders (:, @, or $)
const result = await client.execute({
  sql: 'INSERT INTO users (name, email) VALUES (:name, :email)',
  args: { name: 'Alice', email: 'alice@example.com' },
})
```

### Response Structure

```typescript
interface ResultSet {
  rows: Array<Row>         // Row data (empty for writes)
  columns: Array<string>   // Column names
  rowsAffected: number     // Affected rows (writes)
  lastInsertRowid?: bigint // Last inserted row ID
}
```

### Batch Transactions

```typescript
const results = await client.batch(
  [
    { sql: 'INSERT INTO users (name) VALUES (?)', args: ['Alice'] },
    { sql: 'INSERT INTO users (name) VALUES (?)', args: ['Bob'] },
  ],
  'write' // Transaction mode: "write" | "read" | "deferred"
)
```

### Interactive Transactions

```typescript
const transaction = await client.transaction('write')

try {
  const balance = await transaction.execute({
    sql: 'SELECT balance FROM accounts WHERE id = ?',
    args: [userId],
  })

  if (balance.rows[0].balance >= amount) {
    await transaction.execute({
      sql: 'UPDATE accounts SET balance = balance - ? WHERE id = ?',
      args: [amount, userId],
    })
    await transaction.commit()
  } else {
    await transaction.rollback()
  }
} catch (e) {
  await transaction.rollback()
  throw e
}
```

### Transaction Modes

| Mode       | SQLite Command               | Description                                      |
| ---------- | ---------------------------- | ------------------------------------------------ |
| `write`    | `BEGIN IMMEDIATE`            | Read/write, serialized on primary                |
| `read`     | `BEGIN TRANSACTION READONLY` | Read-only, can run on replicas in parallel       |
| `deferred` | `BEGIN DEFERRED`             | Starts as read, upgrades to write on first write |

## Local Development

### Option 1: SQLite File (Simplest)

```typescript
const client = createClient({ url: 'file:local.db' })
```

### Option 2: Turso Dev Server (Full Features)

```bash
turso dev                    # Start local libSQL server
turso dev --db-file local.db # With persistent file
```

```typescript
const client = createClient({ url: 'http://127.0.0.1:8080' })
```

### Environment Variables Pattern

```env
# Production
TURSO_DATABASE_URL=libsql://my-db-org.turso.io
TURSO_AUTH_TOKEN=eyJ...

# Development
TURSO_DATABASE_URL=file:local.db
# No auth token needed
```

## Common Mistakes

| Mistake                                    | Fix                                         |
| ------------------------------------------ | ------------------------------------------- |
| Using `@libsql/client/web` with file URLs  | Use `@libsql/client` for local files        |
| Long-running write transactions            | Keep writes short, they block other writes  |
| Opening local file during sync             | Wait for sync to complete                   |
| Forgetting to sync embedded replicas       | Call `sync()` or use `syncInterval`         |
| Hardcoding credentials                     | Use environment variables                   |
| Not using transactions for related writes  | Use `batch()` or `transaction()`            |

## Performance Tips

- Use `batch()` for multiple related operations
- Use `read` transactions for read-only queries (parallel on replicas)
- Set appropriate `syncInterval` for embedded replicas
- Use vector indexes for tables with >1000 rows
- Use positional placeholders for frequently executed queries
