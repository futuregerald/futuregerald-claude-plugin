---
name: sql-optimization-patterns
description: Master SQL query optimization, indexing strategies, and EXPLAIN analysis to dramatically improve database performance and eliminate slow queries. Use when debugging slow queries, designing database schemas, or optimizing application performance.
---

# SQL Optimization Patterns

Transform slow database queries into lightning-fast operations through systematic optimization, proper indexing, and query plan analysis.

## When to Use This Skill

- Debugging slow-running queries
- Designing performant database schemas
- Optimizing application response times
- Reducing database load and costs
- Improving scalability for growing datasets
- Analyzing EXPLAIN query plans
- Implementing efficient indexes
- Resolving N+1 query problems

## Core Concepts

### 1. Query Execution Plans (EXPLAIN)

```sql
-- Basic explain
EXPLAIN SELECT * FROM users WHERE email = 'user@example.com';

-- With actual execution stats
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'user@example.com';

-- Full diagnostics
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT u.*, o.order_total
FROM users u
JOIN orders o ON u.id = o.user_id
WHERE u.created_at > NOW() - INTERVAL '30 days';
```

**Key Metrics:**

| Metric | Meaning |
|--------|---------|
| Seq Scan | Full table scan (usually slow for large tables) |
| Index Scan | Using index (good) |
| Index Only Scan | Using covering index, no table access (best) |
| Nested Loop | Join for small datasets |
| Hash Join | Join for larger datasets |
| Merge Join | Join for sorted data |
| Cost | Estimated query cost (lower is better) |
| Actual Time | Real execution time |

### 2. Index Strategies

| Type | Use Case | Database |
|------|----------|----------|
| B-Tree | Equality and range queries (default) | All |
| Hash | Equality (=) only | PostgreSQL |
| GIN | Full-text search, arrays, JSONB | PostgreSQL |
| GiST | Geometric data, full-text search | PostgreSQL |
| BRIN | Very large tables with natural ordering | PostgreSQL |

```sql
-- Standard B-Tree index
CREATE INDEX idx_users_email ON users(email);

-- Composite index (column order matters!)
CREATE INDEX idx_orders_user_status ON orders(user_id, status);

-- Partial index (index subset of rows)
CREATE INDEX idx_active_users ON users(email)
WHERE status = 'active';

-- Expression index
CREATE INDEX idx_users_lower_email ON users(LOWER(email));

-- Covering index (avoid table lookups) -- PostgreSQL
CREATE INDEX idx_users_email_covering ON users(email)
INCLUDE (name, created_at);

-- JSONB index -- PostgreSQL
CREATE INDEX idx_metadata ON events USING GIN(metadata);
```

### 3. Query Optimization Patterns

**Avoid SELECT \*:** Fetch only the columns you need.

```sql
-- Bad
SELECT * FROM users WHERE id = 123;
-- Good
SELECT id, email, name FROM users WHERE id = 123;
```

**Avoid functions on indexed columns in WHERE:**

```sql
-- Bad: prevents index usage
SELECT * FROM users WHERE LOWER(email) = 'user@example.com';
-- Good: create expression index, or store normalized data
CREATE INDEX idx_users_email_lower ON users(LOWER(email));
```

**Filter before JOIN:**

```sql
-- Good: explicit JOIN with WHERE
SELECT u.name, o.total
FROM users u
JOIN orders o ON u.id = o.user_id
WHERE u.created_at > '2024-01-01';

-- Better: filter both sides early
SELECT u.name, o.total
FROM (SELECT id, name FROM users WHERE created_at > '2024-01-01') u
JOIN orders o ON u.id = o.user_id;
```

## Optimization Patterns

### Pattern 1: Eliminate N+1 Queries

**Problem:** N+1 queries execute one query per row instead of batching.

```python
# Bad: N+1 queries
users = db.query("SELECT * FROM users LIMIT 10")
for user in users:
    orders = db.query("SELECT * FROM orders WHERE user_id = ?", user.id)
```

**Solutions:**

```sql
-- Solution 1: JOIN
SELECT u.id, u.name, o.id as order_id, o.total
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.id IN (1, 2, 3, 4, 5);

-- Solution 2: Batch query
SELECT * FROM orders WHERE user_id IN (1, 2, 3, 4, 5);
```

### Pattern 2: Cursor-Based Pagination

**Problem:** OFFSET-based pagination degrades on large tables.

```sql
-- Bad: slow for large offsets
SELECT * FROM users ORDER BY created_at DESC
LIMIT 20 OFFSET 100000;
```

**Solution:** Use a cursor (last seen value).

```sql
-- Fast: cursor-based
SELECT * FROM users
WHERE (created_at, id) < ('2024-01-15 10:30:00', 12345)
ORDER BY created_at DESC, id DESC
LIMIT 20;

-- Supporting index
CREATE INDEX idx_users_cursor ON users(created_at DESC, id DESC);
```

## ORM-Specific Anti-Patterns

### N+1 in ActiveRecord (Rails)

```ruby
# Bad: N+1
users = User.all
users.each { |u| u.orders }  # N queries!

# Good: Eager loading
users = User.includes(:orders).all

# Good: With constraints
users = User.includes(:orders).where(orders: { status: 'completed' })
```

### Sequential Queries in Loops

```ruby
# Bad
user_ids.each { |id| User.find(id) }  # N queries!

# Good: Batch fetch
users = User.where(id: user_ids)
```

### Missing Eager Loading in Nested Relations

```ruby
# Bad: loads posts, then N queries for comments
users = User.includes(:posts)
users.each { |u| u.posts.each { |p| p.comments } }

# Good: Nested eager loading
users = User.includes(posts: :comments)
```

## Best Practices

1. **Index Selectively** -- too many indexes slow down writes
2. **Monitor Query Performance** -- use slow query logs
3. **Keep Statistics Updated** -- run ANALYZE regularly
4. **Use Appropriate Data Types** -- smaller types = better performance
5. **Normalize Thoughtfully** -- balance normalization vs performance
6. **Cache Frequently Accessed Data** -- use application-level caching
7. **Connection Pooling** -- reuse database connections
8. **Regular Maintenance** -- VACUUM, ANALYZE, rebuild indexes

## Common Pitfalls

- **Over-Indexing**: Each index slows down INSERT/UPDATE/DELETE
- **Unused Indexes**: Waste space and slow writes
- **Missing Indexes**: Slow queries, full table scans
- **Implicit Type Conversion**: Prevents index usage
- **OR Conditions**: Can't use indexes efficiently
- **LIKE with Leading Wildcard**: `LIKE '%abc'` can't use index
- **Function in WHERE**: Prevents index usage unless functional index exists

## Reference Material

| Topic | File |
|-------|------|
| Aggregate optimization, subquery patterns, batch operations | [references/advanced-patterns.md](references/advanced-patterns.md) |
| Materialized views, partitioning, query hints | [references/advanced-patterns.md](references/advanced-patterns.md) |
| Monitoring queries, maintenance commands | [references/advanced-patterns.md](references/advanced-patterns.md) |
