# Estimation & Priority

## Estimation

Reference similar completed tickets if found during history research. Use surface area as a proxy: files affected, repos involved, schema changes. Always include a confidence qualifier.

| Size | Typical scope | Time (1 engineer) |
|------|--------------|-------------------|
| S | Single file, clear fix, no schema change | < 1 day |
| M | 2-5 files, straightforward logic, minor schema change possible | 1-3 days |
| L | 5-15 files, cross-cutting logic, schema migration, multi-repo possible | 3-7 days |
| XL | 15+ files, architectural change, multi-repo, data migration | 1-2 weeks |

## Priority (P1-P3)

Determine priority using two axes:

**Severity** — what is the impact?
- Security vulnerability or data loss
- Customer-facing workflow broken
- Customer-facing degraded (not broken)
- Internal workflow / developer experience
- Cosmetic / nice-to-have

**Urgency** — how pressing is it?
- No workaround / blocking someone
- Workaround exists / not blocking

**Matrix:**

| Severity | No workaround / Blocking | Workaround exists / Not blocking |
|----------|--------------------------|----------------------------------|
| Security vuln or data loss | **P1** | **P1** |
| Customer workflow broken | **P1** | **P2** |
| Customer-facing degraded | **P2** | **P3** |
| Internal workflow / DX | **P2** | **P3** |
| Cosmetic / nice-to-have | **P3** | **P3** |

- **P1** = Fix now, interrupt current work
- **P2** = Fix this sprint
- **P3** = Backlog

Include the matched severity row, urgency column, and resulting priority with a one-sentence justification.

## Story Points

Map T-shirt sizes to story points for sprint planning:

| Size | Story Points |
|------|-------------|
| S | 1-2 |
| M | 3-5 |
| L | 8 |
| XL | 13 |

Pick within the range based on complexity factors (schema changes, multi-repo, test coverage needed).
