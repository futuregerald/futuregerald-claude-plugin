# ADR Templates & Management (Reference)

Additional ADR templates, management patterns, and review processes. See [../SKILL.md](../SKILL.md) for core concepts and standard templates.

---

## Template: Y-Statement Format

```markdown
# ADR-0015: API Gateway Selection

In the context of **building a microservices architecture**,
facing **the need for centralized API management, authentication, and rate limiting**,
we decided for **Kong Gateway**
and against **AWS API Gateway and custom Nginx solution**,
to achieve **vendor independence, plugin extensibility, and team familiarity with Lua**,
accepting that **we need to manage Kong infrastructure ourselves**.
```

## Template: ADR for Deprecation

```markdown
# ADR-0020: Deprecate MongoDB in Favor of PostgreSQL

## Status

Accepted (Supersedes ADR-0003)

## Context

ADR-0003 (2021) chose MongoDB for user profile storage due to schema flexibility
needs. Since then:

- MongoDB's multi-document transactions remain problematic for our use case
- Our schema has stabilized and rarely changes
- We now have PostgreSQL expertise from other services
- Maintaining two databases increases operational burden

## Decision

Deprecate MongoDB and migrate user profiles to PostgreSQL.

## Migration Plan

1. **Phase 1** (Week 1-2): Create PostgreSQL schema, dual-write enabled
2. **Phase 2** (Week 3-4): Backfill historical data, validate consistency
3. **Phase 3** (Week 5): Switch reads to PostgreSQL, monitor
4. **Phase 4** (Week 6): Remove MongoDB writes, decommission

## Consequences

### Positive

- Single database technology reduces operational complexity
- ACID transactions for user data
- Team can focus PostgreSQL expertise

### Negative

- Migration effort (~4 weeks)
- Risk of data issues during migration
- Lose some schema flexibility

## Lessons Learned

Document from ADR-0003 experience:

- Schema flexibility benefits were overestimated
- Operational cost of multiple databases was underestimated
- Consider long-term maintenance in technology decisions
```

## Template: Request for Comments (RFC) Style

```markdown
# RFC-0025: Adopt Event Sourcing for Order Management

## Summary

Propose adopting event sourcing pattern for the order management domain to
improve auditability, enable temporal queries, and support business analytics.

## Motivation

Current challenges:

1. Audit requirements need complete order history
2. "What was the order state at time X?" queries are impossible
3. Analytics team needs event stream for real-time dashboards
4. Order state reconstruction for customer support is manual

## Detailed Design

### Event Store

OrderCreated { orderId, customerId, items[], timestamp }
OrderItemAdded { orderId, item, timestamp }
OrderItemRemoved { orderId, itemId, timestamp }
PaymentReceived { orderId, amount, paymentId, timestamp }
OrderShipped { orderId, trackingNumber, timestamp }

### Projections

- **CurrentOrderState**: Materialized view for queries
- **OrderHistory**: Complete timeline for audit
- **DailyOrderMetrics**: Analytics aggregation

### Technology

- Event Store: EventStoreDB (purpose-built, handles projections)
- Alternative considered: Kafka + custom projection service

## Drawbacks

- Learning curve for team
- Increased complexity vs. CRUD
- Need to design events carefully (immutable once stored)
- Storage growth (events never deleted)

## Alternatives

1. **Audit tables**: Simpler but doesn't enable temporal queries
2. **CDC from existing DB**: Complex, doesn't change data model
3. **Hybrid**: Event source only for order state changes

## Unresolved Questions

- [ ] Event schema versioning strategy
- [ ] Retention policy for events
- [ ] Snapshot frequency for performance

## Implementation Plan

1. Prototype with single order type (2 weeks)
2. Team training on event sourcing (1 week)
3. Full implementation and migration (4 weeks)
4. Monitoring and optimization (ongoing)

## References

- [Event Sourcing by Martin Fowler](https://martinfowler.com/eaaDev/EventSourcing.html)
- [EventStoreDB Documentation](https://www.eventstore.com/docs)
```

## ADR Management

### Directory Structure

```
docs/
  adr/
    README.md           # Index and guidelines
    template.md         # Team's ADR template
    0001-use-postgresql.md
    0002-caching-strategy.md
    0003-mongodb-user-profiles.md  # [DEPRECATED]
    0020-deprecate-mongodb.md      # Supersedes 0003
```

### ADR Index (README.md)

```markdown
# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for [Project Name].

## Index

| ADR                                   | Title                              | Status     | Date       |
| ------------------------------------- | ---------------------------------- | ---------- | ---------- |
| [0001](0001-use-postgresql.md)        | Use PostgreSQL as Primary Database | Accepted   | 2024-01-10 |
| [0002](0002-caching-strategy.md)      | Caching Strategy with Redis        | Accepted   | 2024-01-12 |
| [0003](0003-mongodb-user-profiles.md) | MongoDB for User Profiles          | Deprecated | 2023-06-15 |
| [0020](0020-deprecate-mongodb.md)     | Deprecate MongoDB                  | Accepted   | 2024-01-15 |

## Creating a New ADR

1. Copy `template.md` to `NNNN-title-with-dashes.md`
2. Fill in the template
3. Submit PR for review
4. Update this index after approval
```

### Automation (adr-tools)

```bash
# Install adr-tools
brew install adr-tools

# Initialize ADR directory
adr init docs/adr

# Create new ADR
adr new "Use PostgreSQL as Primary Database"

# Supersede an ADR
adr new -s 3 "Deprecate MongoDB in Favor of PostgreSQL"

# Generate table of contents
adr generate toc > docs/adr/README.md

# Link related ADRs
adr link 2 "Complements" 1 "Is complemented by"
```

## Review Process

```markdown
## ADR Review Checklist

### Before Submission

- [ ] Context clearly explains the problem
- [ ] All viable options considered
- [ ] Pros/cons balanced and honest
- [ ] Consequences (positive and negative) documented
- [ ] Related ADRs linked

### During Review

- [ ] At least 2 senior engineers reviewed
- [ ] Affected teams consulted
- [ ] Security implications considered
- [ ] Cost implications documented
- [ ] Reversibility assessed

### After Acceptance

- [ ] ADR index updated
- [ ] Team notified
- [ ] Implementation tickets created
- [ ] Related documentation updated
```

## Resources

- [Documenting Architecture Decisions (Michael Nygard)](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [MADR Template](https://adr.github.io/madr/)
- [ADR GitHub Organization](https://adr.github.io/)
- [adr-tools](https://github.com/npryce/adr-tools)
