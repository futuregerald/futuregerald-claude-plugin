---
name: sqlite-database-expert
description: Expert in SQLite, libSQL, and Turso database development for desktop and web applications with focus on SQL injection prevention, migrations, FTS search, edge deployments, and secure data handling.
tags: [framework, database]
---

# SQLite / libSQL / Turso Database Expert

## Reference Files

Read these when the task matches:

| File | When to read |
|------|-------------|
| [references/typescript-patterns.md](references/typescript-patterns.md) | Working with TypeScript, libSQL client, Turso SDK, edge deployments (Cloudflare/Vercel/Bun), Python TDD, or performance optimization |

Also read `references/advanced-patterns.md` when implementing migrations, FTS5, CTEs, connection pooling, or WAL mode. Read `references/security-examples.md` when writing any SQL with user input.

---

## Overview

| Database   | Best For                          | Key Features                             |
| ---------- | --------------------------------- | ---------------------------------------- |
| **SQLite** | Desktop apps, mobile, embedded    | Single file, zero config, bundled        |
| **libSQL** | Self-hosted web apps, local-first | Server mode, HTTP API, replication       |
| **Turso**  | Global web apps, edge computing   | Managed, multi-region, embedded replicas |

### Core Principles

1. **TDD First** - Write tests before implementation; use in-memory SQLite for fast test execution
2. **Performance Aware** - WAL mode, prepared statements, batch operations, proper indexing
3. **Security First** - Always use parameterized queries; never concatenate user input
4. **Transaction Safety** - Wrap related operations in transactions
5. **Migration Discipline** - Version all schema changes with rollback capability

---

## Database Initialization (Rust)

```rust
use rusqlite::{Connection, Result};
use std::path::Path;

pub struct Database {
    conn: Connection,
}

impl Database {
    pub fn new(path: &Path) -> Result<Self> {
        let conn = Connection::open(path)?;
        conn.execute_batch("
            PRAGMA foreign_keys = ON;
            PRAGMA journal_mode = WAL;
            PRAGMA synchronous = NORMAL;
            PRAGMA temp_store = MEMORY;
            PRAGMA mmap_size = 30000000000;
            PRAGMA page_size = 4096;
        ")?;
        Ok(Self { conn })
    }
}
```

## Parameterized Queries (CRITICAL)

```rust
// CORRECT: Parameterized query
pub fn get_user_by_id(&self, user_id: i64) -> Result<Option<User>> {
    let mut stmt = self.conn.prepare(
        "SELECT id, name, email FROM users WHERE id = ?1"
    )?;
    let user = stmt.query_row([user_id], |row| {
        Ok(User { id: row.get(0)?, name: row.get(1)?, email: row.get(2)? })
    }).optional()?;
    Ok(user)
}

// CORRECT: Named parameters
pub fn search_users(&self, name: &str, status: &str) -> Result<Vec<User>> {
    let mut stmt = self.conn.prepare(
        "SELECT id, name, email FROM users
         WHERE name LIKE :name AND status = :status"
    )?;
    let users = stmt.query_map(
        &[(":name", &format!("%{}%", name)), (":status", &status)],
        |row| Ok(User { id: row.get(0)?, name: row.get(1)?, email: row.get(2)? })
    )?.collect::<Result<Vec<_>>>()?;
    Ok(users)
}

// INCORRECT: SQL Injection vulnerability - NEVER DO THIS
// let query = format!("SELECT * FROM users WHERE id = {}", user_id);
```

## Transaction Management

```rust
pub fn transfer_funds(&mut self, from_id: i64, to_id: i64, amount: f64) -> Result<()> {
    let tx = self.conn.transaction()?;
    tx.execute("UPDATE accounts SET balance = balance - ?1 WHERE id = ?2", [amount, from_id as f64])?;
    tx.execute("UPDATE accounts SET balance = balance + ?1 WHERE id = ?2", [amount, to_id as f64])?;
    tx.commit()?;
    Ok(())
}
```

## Full-Text Search (FTS5)

```rust
pub fn setup_fts(&self) -> Result<()> {
    self.conn.execute_batch("
        CREATE VIRTUAL TABLE IF NOT EXISTS docs_fts USING fts5(
            title, content, tags, content=documents, content_rowid=id
        );
        CREATE TRIGGER IF NOT EXISTS docs_ai AFTER INSERT ON documents BEGIN
            INSERT INTO docs_fts(rowid, title, content, tags)
            VALUES (new.id, new.title, new.content, new.tags);
        END;
    ")?;
    Ok(())
}

pub fn search_documents(&self, query: &str) -> Result<Vec<Document>> {
    let mut stmt = self.conn.prepare(
        "SELECT d.*, highlight(docs_fts, 1, '<mark>', '</mark>') as snippet
         FROM documents d JOIN docs_fts ON d.id = docs_fts.rowid
         WHERE docs_fts MATCH ?1 ORDER BY rank"
    )?;
    stmt.query_map([query], |row| Ok(Document { /* ... */ }))?.collect()
}
```

## Security Standards

| OWASP Category         | Risk     | Key Controls                            |
| ---------------------- | -------- | --------------------------------------- |
| A03 - Injection        | Critical | Parameterized queries, input validation |
| A04 - Insecure Design  | Medium   | Schema constraints, foreign keys        |
| A05 - Misconfiguration | Medium   | Secure PRAGMAs, file permissions (600)  |

**Dynamic column selection - SAFE approach:**

```rust
pub fn get_user_fields(&self, user_id: i64, fields: &[&str]) -> Result<HashMap<String, String>> {
    const ALLOWED: &[&str] = &["id", "name", "email", "created_at"];
    let safe_fields: Vec<&str> = fields.iter()
        .filter(|f| ALLOWED.contains(f)).copied().collect();
    if safe_fields.is_empty() { return Err(rusqlite::Error::InvalidQuery); }
    let query = format!("SELECT {} FROM users WHERE id = ?1", safe_fields.join(", "));
    let mut stmt = self.conn.prepare(&query)?;
    // ...
}
```

## Testing (Rust)

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use rusqlite::Connection;

    fn setup_test_db() -> Database {
        let conn = Connection::open_in_memory().unwrap();
        let db = Database { conn };
        db.run_migrations().unwrap();
        db
    }

    #[test]
    fn test_sql_injection_prevented() {
        let db = setup_test_db();
        let result = db.search_users("'; DROP TABLE users; --", "active");
        assert!(result.is_ok());
        assert!(db.get_user_by_id(1).is_ok()); // Table still exists
    }
}
```

## Common Mistakes

| Mistake         | Wrong                                    | Correct                              |
| --------------- | ---------------------------------------- | ------------------------------------ |
| SQL Injection   | `format!("...WHERE name = '{}'", input)` | `"...WHERE name = ?1"` with params   |
| No Transaction  | Separate execute calls                   | Wrap in `transaction()` + `commit()` |
| No Foreign Keys | Default connection                       | `PRAGMA foreign_keys = ON`           |
| LIKE for Search | `LIKE '%term%'`                          | FTS5 `MATCH 'term'`                  |

## Pre-Implementation Checklist

- [ ] Tests written first (in-memory SQLite)
- [ ] Schema designed with constraints and indexes
- [ ] All user inputs identified and parameterized
- [ ] Transactions wrap related operations
- [ ] Foreign keys enabled, WAL mode configured
- [ ] Batch operations used for multiple inserts
- [ ] Error handling secure (no SQL details in user-facing errors)
- [ ] Migrations tested with rollback
- [ ] VACUUM scheduled for maintenance
