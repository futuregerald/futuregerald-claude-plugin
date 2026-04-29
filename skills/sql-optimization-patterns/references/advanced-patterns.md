# SQL Optimization Patterns -- Advanced Reference

> Cold reference for the `sql-optimization-patterns` skill. See [../SKILL.md](../SKILL.md) for core patterns.

## Pattern 3: Aggregate Efficiently

### Optimize COUNT Queries

```sql
-- Bad: Counts all rows
SELECT COUNT(*) FROM orders;  -- Slow on large tables

-- Good: Use estimates for approximate counts (PostgreSQL)
SELECT reltuples::bigint AS estimate
FROM pg_class
WHERE relname = 'orders';

-- Good: Filter before counting
SELECT COUNT(*) FROM orders
WHERE created_at > NOW() - INTERVAL '7 days';

-- Better: Use index-only scan
CREATE INDEX idx_orders_created ON orders(created_at);
SELECT COUNT(*) FROM orders
WHERE created_at > NOW() - INTERVAL '7 days';
```

### Optimize GROUP BY

```sql
-- Bad: Group by then filter
SELECT user_id, COUNT(*) as order_count
FROM orders
GROUP BY user_id
HAVING COUNT(*) > 10;

-- Better: Filter first, then group (if possible)
SELECT user_id, COUNT(*) as order_count
FROM orders
WHERE status = 'completed'
GROUP BY user_id
HAVING COUNT(*) > 10;

-- Best: Use covering index
CREATE INDEX idx_orders_user_status ON orders(user_id, status);
```

## Pattern 4: Subquery Optimization

### Transform Correlated Subqueries

```sql
-- Bad: Correlated subquery (runs for each row)
SELECT u.name, u.email,
    (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) as order_count
FROM users u;

-- Good: JOIN with aggregation
SELECT u.name, u.email, COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
GROUP BY u.id, u.name, u.email;

-- Better: Use window functions (PostgreSQL)
SELECT DISTINCT ON (u.id)
    u.name, u.email,
    COUNT(o.id) OVER (PARTITION BY u.id) as order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id;
```

### Use CTEs for Clarity

```sql
WITH recent_users AS (
    SELECT id, name, email
    FROM users
    WHERE created_at > NOW() - INTERVAL '30 days'
),
user_order_counts AS (
    SELECT user_id, COUNT(*) as order_count
    FROM orders
    WHERE created_at > NOW() - INTERVAL '30 days'
    GROUP BY user_id
)
SELECT ru.name, ru.email, COALESCE(uoc.order_count, 0) as orders
FROM recent_users ru
LEFT JOIN user_order_counts uoc ON ru.id = uoc.user_id;
```

## Pattern 5: Batch Operations

### Batch INSERT

```sql
-- Bad: Multiple individual inserts
INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com');
INSERT INTO users (name, email) VALUES ('Carol', 'carol@example.com');

-- Good: Batch insert
INSERT INTO users (name, email) VALUES
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com'),
    ('Carol', 'carol@example.com');

-- Better: Use COPY for bulk inserts (PostgreSQL)
COPY users (name, email) FROM '/tmp/users.csv' CSV HEADER;
```

### Batch UPDATE

```sql
-- Bad: Update in loop
UPDATE users SET status = 'active' WHERE id = 1;
UPDATE users SET status = 'active' WHERE id = 2;

-- Good: Single UPDATE with IN clause
UPDATE users
SET status = 'active'
WHERE id IN (1, 2, 3, 4, 5);

-- Better: Use temporary table for large batches (PostgreSQL)
CREATE TEMP TABLE temp_user_updates (id INT, new_status VARCHAR);
INSERT INTO temp_user_updates VALUES (1, 'active'), (2, 'active');

UPDATE users u
SET status = t.new_status
FROM temp_user_updates t
WHERE u.id = t.id;
```

## Advanced Techniques

### Materialized Views (PostgreSQL)

Pre-compute expensive queries.

```sql
-- Create materialized view
CREATE MATERIALIZED VIEW user_order_summary AS
SELECT
    u.id,
    u.name,
    COUNT(o.id) as total_orders,
    SUM(o.total) as total_spent,
    MAX(o.created_at) as last_order_date
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.name;

-- Add index to materialized view
CREATE INDEX idx_user_summary_spent ON user_order_summary(total_spent DESC);

-- Refresh materialized view
REFRESH MATERIALIZED VIEW user_order_summary;

-- Concurrent refresh (no lock)
REFRESH MATERIALIZED VIEW CONCURRENTLY user_order_summary;

-- Query (very fast)
SELECT * FROM user_order_summary
WHERE total_spent > 1000
ORDER BY total_spent DESC;
```

### Partitioning

Split large tables for better performance.

```sql
-- Range partitioning by date (PostgreSQL)
CREATE TABLE orders (
    id SERIAL,
    user_id INT,
    total DECIMAL,
    created_at TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create partitions
CREATE TABLE orders_2024_q1 PARTITION OF orders
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');

CREATE TABLE orders_2024_q2 PARTITION OF orders
    FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');

-- Queries automatically use appropriate partition
SELECT * FROM orders
WHERE created_at BETWEEN '2024-02-01' AND '2024-02-28';
-- Only scans orders_2024_q1 partition
```

### Query Hints and Optimization

```sql
-- Force index usage (MySQL)
SELECT * FROM users
USE INDEX (idx_users_email)
WHERE email = 'user@example.com';

-- Parallel query (PostgreSQL)
SET max_parallel_workers_per_gather = 4;
SELECT * FROM large_table WHERE condition;

-- Join hints (PostgreSQL)
SET enable_nestloop = OFF;  -- Force hash or merge join
```

## Monitoring Queries

```sql
-- Find slow queries (PostgreSQL)
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- Find missing indexes (PostgreSQL)
SELECT
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    seq_tup_read / seq_scan AS avg_seq_tup_read
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_tup_read DESC
LIMIT 10;

-- Find unused indexes (PostgreSQL)
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;
```

## Maintenance Commands

```sql
-- Update statistics
ANALYZE users;
ANALYZE VERBOSE orders;

-- Vacuum (PostgreSQL)
VACUUM ANALYZE users;
VACUUM FULL users;  -- Reclaim space (locks table)

-- Reindex
REINDEX INDEX idx_users_email;
REINDEX TABLE users;
```
