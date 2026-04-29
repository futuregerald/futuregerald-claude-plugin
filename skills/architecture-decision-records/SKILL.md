---
name: architecture-decision-records
description: Write and maintain Architecture Decision Records (ADRs) following best practices for technical decision documentation. Use when documenting significant technical decisions, reviewing past architectural choices, or establishing decision processes.
tags: [architecture]
---

# Architecture Decision Records

## Reference Files

| File | When to read |
|------|-------------|
| [references/templates-and-management.md](references/templates-and-management.md) | Need Y-Statement, Deprecation, or RFC-style ADR templates; ADR directory management; automation with adr-tools; review process checklists |

---

## When to Use This Skill

- Making significant architectural decisions
- Documenting technology choices
- Recording design trade-offs
- Onboarding new team members
- Reviewing historical decisions

## Core Concepts

### What is an ADR?

An Architecture Decision Record captures:

- **Context**: Why we needed to make a decision
- **Decision**: What we decided
- **Consequences**: What happens as a result

### When to Write an ADR

| Write ADR                  | Skip ADR               |
| -------------------------- | ---------------------- |
| New framework adoption     | Minor version upgrades |
| Database technology choice | Bug fixes              |
| API design patterns        | Implementation details |
| Security architecture      | Routine maintenance    |
| Integration patterns       | Configuration changes  |

### ADR Lifecycle

```
Proposed -> Accepted -> Deprecated -> Superseded
               |
            Rejected
```

## Templates

### Template 1: Standard ADR (MADR Format)

```markdown
# ADR-0001: Use PostgreSQL as Primary Database

## Status

Accepted

## Context

We need to select a primary database for our new e-commerce platform. The system
will handle:

- ~10,000 concurrent users
- Complex product catalog with hierarchical categories
- Transaction processing for orders and payments
- Full-text search for products
- Geospatial queries for store locator

The team has experience with MySQL, PostgreSQL, and MongoDB. We need ACID
compliance for financial transactions.

## Decision Drivers

- **Must have ACID compliance** for payment processing
- **Must support complex queries** for reporting
- **Should support full-text search** to reduce infrastructure complexity
- **Should have good JSON support** for flexible product attributes
- **Team familiarity** reduces onboarding time

## Considered Options

### Option 1: PostgreSQL

- **Pros**: ACID compliant, excellent JSON support (JSONB), built-in full-text
  search, PostGIS for geospatial, team has experience
- **Cons**: Slightly more complex replication setup than MySQL

### Option 2: MySQL

- **Pros**: Very familiar to team, simple replication, large community
- **Cons**: Weaker JSON support, no built-in full-text search

### Option 3: MongoDB

- **Pros**: Flexible schema, native JSON, horizontal scaling
- **Cons**: No ACID for multi-document transactions, limited team experience

## Decision

We will use **PostgreSQL 15** as our primary database.

## Rationale

PostgreSQL provides the best balance of ACID compliance, built-in capabilities
(full-text search, JSONB, PostGIS), and team familiarity.

## Consequences

### Positive

- Single database handles transactions, search, and geospatial queries
- Reduced operational complexity (fewer services to manage)
- Strong consistency guarantees for financial data

### Negative

- Need to learn PostgreSQL-specific features
- Vertical scaling limits may require read replicas sooner
- Some team members need PostgreSQL-specific training

## Implementation Notes

- Use JSONB for flexible product attributes
- Implement connection pooling with PgBouncer
- Set up streaming replication for read replicas
```

### Template 2: Lightweight ADR

```markdown
# ADR-0012: Adopt TypeScript for Frontend Development

**Status**: Accepted
**Date**: 2024-01-15
**Deciders**: @alice, @bob, @charlie

## Context

Our React codebase has grown to 50+ components with increasing bug reports
related to prop type mismatches and undefined errors.

## Decision

Adopt TypeScript for all new frontend code. Migrate existing code incrementally.

## Consequences

**Good**: Catch type errors at compile time, better IDE support, self-documenting
code.

**Bad**: Learning curve for team, initial slowdown, build complexity increase.

**Mitigations**: TypeScript training sessions, allow gradual adoption with
`allowJs: true`.
```

## Best Practices

### Do's

- **Write ADRs early** - Before implementation starts
- **Keep them short** - 1-2 pages maximum
- **Be honest about trade-offs** - Include real cons
- **Link related decisions** - Build decision graph
- **Update status** - Deprecate when superseded

### Don'ts

- **Don't change accepted ADRs** - Write new ones to supersede
- **Don't skip context** - Future readers need background
- **Don't hide failures** - Rejected decisions are valuable
- **Don't be vague** - Specific decisions, specific consequences
- **Don't forget implementation** - ADR without action is waste
