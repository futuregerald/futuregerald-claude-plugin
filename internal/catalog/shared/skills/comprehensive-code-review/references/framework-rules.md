# Framework-Specific Rules

Set `{FRAMEWORK_CONTEXT}` per the detected framework. Include in BOTH sub-agent prompts.
Key principle: **understand what the language runtime guarantees before claiming a vulnerability exists.**

## Ruby on Rails

```
## Framework: Ruby on Rails

Respect how Ruby and Rails actually work. False positives from ignoring
the framework erode trust and waste the author's time.

### Type safety in Ruby
- Ruby's `.to_i` ALWAYS returns an Integer. `"anything".to_i` → `0`.
  An Integer interpolated into a string (`"(#{int_val})"`) can ONLY
  produce a decimal number. This is NOT SQL injection — Ruby's type
  system guarantees safety. Do NOT flag integer interpolation as CWE-89.
- Ruby's `.to_f` ALWAYS returns a Float. Same type safety applies.
- `.to_s` on Integer/Float cannot produce SQL metacharacters.
- Only flag SQL injection when STRINGS from user/external input are
  interpolated into queries without `conn.quote()` or parameterization.

### ActiveRecord conventions
- `conn.quote(value)` is for string values in raw SQL. Using it on
  integers is unnecessary (it returns the decimal string anyway).
- `conn.quote_column_name` / `conn.quote_table_name` are for dynamic
  identifiers. Hardcoded string constants do NOT need quoting.
- `where(column: value)` is parameterized — not injection.
- `find_by(column: value)` is parameterized — not injection.
- `pluck(:column)` returns typed Ruby values from the DB adapter.

### Rails mass assignment
- Strong parameters (`params.require().permit()`) handle mass assignment
  in controllers. Internal service objects that don't accept user params
  do NOT need strong parameters.
- `create!` / `update!` with hardcoded hashes is safe.

### Sidekiq / ActiveJob
- Without `retry_on`, Sidekiq uses its OWN retry mechanism (25 retries,
  exponential backoff). `retry_on` is ActiveJob's DSL and is optional.
  Note the default behavior but don't flag absence as CRITICAL.
- `perform_now` runs synchronously (not queued). Used in rake tasks.

### Rails conventions to NOT flag
- `ENV.fetch('KEY')` raising KeyError is intentional fail-fast, not a bug.
- `Time.current` vs `Time.now` — `Time.current` is the Rails convention.
- `present?` / `blank?` are Rails core extensions, not custom code.
- `squish` on strings is a Rails method that collapses whitespace.
```

## Go

```
## Framework: Go

- Go is statically typed. Integer values CANNOT contain SQL injection
  when used with `fmt.Sprintf("%d", val)` or direct interpolation.
- String values from user input CAN be dangerous in `fmt.Sprintf`
  SQL construction — flag these.
- `database/sql` with `?` placeholders is parameterized.
```

## JavaScript/TypeScript

```
## Framework: JavaScript/TypeScript

- JS has NO integer type — all numbers are IEEE 754 doubles. However,
  `parseInt(val, 10)` returns NaN for non-numeric strings, not a
  dangerous value. `NaN` in SQL causes a query error, not injection.
- Template literals with user strings ARE dangerous in raw SQL.
- ORMs (Prisma, TypeORM, Knex) parameterize by default when using
  their query builder APIs. Raw SQL methods (`.raw()`, `.$queryRaw()`)
  need manual parameterization.
```

Add rules for other frameworks as encountered.
