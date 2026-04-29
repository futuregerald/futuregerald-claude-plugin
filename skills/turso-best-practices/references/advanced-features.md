# Turso Advanced Features (Reference)

Embedded replicas, vector search, Drizzle ORM integration, branching, encryption, and extensions. See [../SKILL.md](../SKILL.md) for core SDK usage and setup.

---

## Embedded Replicas

Local database that syncs with remote Turso database. Reads are instant (local), writes go to remote.

### Configuration

```typescript
const client = createClient({
  url: 'file:replica.db', // Local file
  syncUrl: 'libsql://my-db.turso.io', // Remote primary
  authToken: '...',
  syncInterval: 60, // Auto-sync every 60 seconds
})
```

### Manual Sync

```typescript
await client.sync()
```

### Offline Mode

```typescript
const client = createClient({
  url: 'file:replica.db',
  syncUrl: 'libsql://my-db.turso.io',
  authToken: '...',
  offline: true, // Writes go to local, sync later
})
```

### Important Notes

- Reads always from local replica
- Writes go to remote primary (unless offline mode)
- Read-your-writes guaranteed after successful write
- Don't open local file while syncing (corruption risk)
- One frame = 4KB (minimum write unit)

## Vector Search (AI & Embeddings)

Native vector search without extensions.

### Create Table with Vector Column

```sql
CREATE TABLE movies (
  id INTEGER PRIMARY KEY,
  title TEXT,
  embedding F32_BLOB(384)  -- 384-dimensional float32 vector
);
```

### Vector Types

| Type                       | Storage       | Description                           |
| -------------------------- | ------------- | ------------------------------------- |
| `FLOAT64` / `F64_BLOB`     | 8D + 1 bytes  | 64-bit double precision               |
| `FLOAT32` / `F32_BLOB`     | 4D bytes      | 32-bit single precision (recommended) |
| `FLOAT16` / `F16_BLOB`     | 2D + 1 bytes  | 16-bit half precision                 |
| `FLOAT8` / `F8_BLOB`       | D + 14 bytes  | 8-bit compressed                      |
| `FLOAT1BIT` / `F1BIT_BLOB` | D/8 + 3 bytes | 1-bit binary                          |

### Insert Vectors

```sql
INSERT INTO movies (title, embedding)
VALUES ('Inception', vector32('[0.1, 0.2, 0.3, ...]'));
```

### Similarity Search

```sql
SELECT title,
       vector_distance_cos(embedding, vector32('[0.1, 0.2, ...]')) AS distance
FROM movies
ORDER BY distance ASC
LIMIT 10;
```

### Vector Index (DiskANN)

```sql
-- Create index
CREATE INDEX movies_idx ON movies(libsql_vector_idx(embedding));

-- Query with index (much faster for large tables)
SELECT title
FROM vector_top_k('movies_idx', vector32('[0.1, 0.2, ...]'), 10)
JOIN movies ON movies.rowid = id;
```

### Index Settings

```sql
CREATE INDEX movies_idx ON movies(
  libsql_vector_idx(embedding, 'metric=cosine', 'compress_neighbors=float8')
);
```

| Setting              | Values         | Description               |
| -------------------- | -------------- | ------------------------- |
| `metric`             | `cosine`, `l2` | Distance function         |
| `max_neighbors`      | integer        | Graph connectivity        |
| `compress_neighbors` | vector type    | Compression for storage   |
| `search_l`           | integer        | Search precision vs speed |

## Drizzle ORM Integration

### Setup

```bash
npm install drizzle-orm @libsql/client
npm install -D drizzle-kit
```

### Configuration

```typescript
// drizzle.config.ts
import type { Config } from 'drizzle-kit'

export default {
  schema: './db/schema.ts',
  out: './migrations',
  dialect: 'turso',
  dbCredentials: {
    url: process.env.TURSO_DATABASE_URL!,
    authToken: process.env.TURSO_AUTH_TOKEN,
  },
} satisfies Config
```

### Schema Definition

```typescript
// db/schema.ts
import { text, integer, sqliteTable } from 'drizzle-orm/sqlite-core'

export const users = sqliteTable('users', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  name: text('name').notNull(),
  email: text('email').notNull().unique(),
})
```

### Client Setup

```typescript
import { drizzle } from 'drizzle-orm/libsql'
import { createClient } from '@libsql/client'

const turso = createClient({
  url: process.env.TURSO_DATABASE_URL!,
  authToken: process.env.TURSO_AUTH_TOKEN,
})

export const db = drizzle(turso)
```

### Migrations

```bash
# Generate migrations
npm run drizzle-kit generate

# Apply migrations
npm run drizzle-kit migrate
```

## Branching & Point-in-Time Recovery

### Create Branch

```bash
turso db create feature-branch --from-db production-db
```

### Point-in-Time Restore

```bash
turso db create restored-db --from-db production-db --timestamp 2024-01-15T10:00:00Z
```

### CI/CD Branching (GitHub Actions)

```yaml
name: Create Database Branch
on: create

jobs:
  create-branch:
    runs-on: ubuntu-latest
    steps:
      - name: Create Database
        run: |
          curl -X POST \
            -H "Authorization: Bearer ${{ secrets.TURSO_API_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d '{"name": "${{ github.ref_name }}", "group": "default", "seed": {"type": "database", "name": "production"}}' \
            "https://api.turso.tech/v1/organizations/${{ secrets.ORG }}/databases"
```

### Important Notes

- Branches are separate databases (no auto-merge)
- Need new token or group token for branch
- Count toward database quota
- Delete manually when done

## Encryption at Rest

### Generate Key

```bash
# 256-bit key for AEGIS-256/AES-256
openssl rand -base64 32

# 128-bit key for AEGIS-128/AES-128
openssl rand -base64 16
```

### Create Encrypted Database

```bash
turso db create secure-db \
  --remote-encryption-key "YOUR_KEY" \
  --remote-encryption-cipher aegis256
```

### Connect to Encrypted Database

```bash
turso db shell secure-db --remote-encryption-key "YOUR_KEY"
```

### Supported Ciphers

| Cipher             | Key Size | Recommendation           |
| ------------------ | -------- | ------------------------ |
| `aegis128l`        | 128-bit  | Recommended for speed    |
| `aegis256`         | 256-bit  | Recommended for security |
| `aes128gcm`        | 128-bit  | NIST compliance          |
| `aes256gcm`        | 256-bit  | NIST compliance          |
| `chacha20poly1305` | 256-bit  | AES alternative          |

## SQLite Extensions

### Preloaded (Always Available)

| Extension     | Description           |
| ------------- | --------------------- |
| JSON          | JSON functions        |
| FTS5          | Full-text search      |
| R*Tree        | Spatial indexing      |
| SQLean Crypto | Hashing, encoding     |
| SQLean Fuzzy  | Fuzzy string matching |
| SQLean Math   | Advanced math         |
| SQLean Stats  | Statistical functions |
| SQLean Text   | String manipulation   |
| SQLean UUID   | UUID generation       |

### Enable Additional Extensions

```bash
turso db create my-db --enable-extensions
```
