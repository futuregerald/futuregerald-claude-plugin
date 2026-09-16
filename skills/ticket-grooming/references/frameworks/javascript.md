# JavaScript / TypeScript investigation rules

Loaded by the investigation sub-agent after Phase 0 detects the framework.
See [README.md](README.md) for the detection table.

- JS has NO integer type — all numbers are IEEE 754 doubles. However, `parseInt(val, 10)` returns NaN for non-numeric strings, not a dangerous value. `NaN` in SQL causes a query error, not injection.
- Template literals with user strings ARE dangerous in raw SQL.
- ORMs (Prisma, TypeORM, Knex) parameterize by default when using their query builder APIs. Raw SQL methods (`.raw()`, `.$queryRaw()`) need manual parameterization.

## This file is a stub

These notes are thin compared with `rails.md`. Treat that as "rules not yet written", not as
"nothing else matters" — follow the discipline in [README.md](README.md) for a framework without a
rules file: verify every framework-dependent claim against the framework's own source before
asserting it, and say in the notes that you did.

If you learn something durable about this framework, add it here.
