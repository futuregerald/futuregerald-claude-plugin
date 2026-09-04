# Team Pulse Report Format

## Structure

```
## Team Pulse: {scope} ({date range})

**Headline:** {one sentence — on track, at risk, or blocked + why}

### Active Work

{For each initiative/epic with activity in the window:}

**{Epic/Initiative name}** ({Jira key}) — {assessment badge}
- {1-2 bullet progress summary}
- {blocker or risk if any}

### PRs in Flight

| PR | Author | Status | Age | Review |
|----|--------|--------|-----|--------|
| #123 Title | @handle | Open/Merged | 2d | Approved/Pending/Changes |

{Flag stale PRs (>3 days without review) explicitly}

### People

**{Name}** — {assessment badge}
{1-3 sentences: what they're working on, PR activity, anything notable}

### Risks & Blockers

- {Numbered list of items that need Gerald's attention}
- {Include: stale PRs, stuck tickets, unassigned work, cross-team dependencies}

### Bottom Line

{2-3 sentences: overall team health, what to bring up in calls, decisions needed}
```

## Badge Format

Use inline text badges:
- **On Track** for green
- **Needs Attention** for yellow
- **At Risk** for red
- **Blocked** for stopped

## Length Guidelines

| Scope | Target Length |
|-------|-------------|
| Full team summary | 300-500 words |
| Single person (1:1 prep) | 150-250 words |
| Single epic/initiative | 200-350 words |
| "What did we ship" | 100-200 words |

## Example Snippet

```
## Team Pulse: Delivery Domain (May 26 - Jun 3, 2026)

**Headline:** On track overall, but Flywheel tester compensation has no code
pushed yet despite tickets showing "In Progress" — flag with Paul.

### Active Work

**Billing Overhaul** (ABC-100) — Needs Attention
- ADR (ABC-121) and schema migration (ABC-118) both in code review
- No feature branches exist in repo — "Code Review" status may be misleading
- 4 stories still To Do with no assignee (ABC-119, ABC-120, ABC-124, ABC-128)

### PRs in Flight

| PR | Author | Status | Age | Review |
|----|--------|--------|-----|--------|
| #412 Fix pagination | @dev-one | Merged | 1d | Approved |
| #409 Schema migration | @dev-two | Open | 3d | Pending |

### People

**Paul Ursache** — Needs Attention
ABC-100 owner. ADR and A1 schema story in code review but no branches
pushed to repo. Clarify if work is local or if status needs updating.

**Dev One** — On Track
Shipped #412 (pagination fix). Picked up ABC-104 slim serializer. Active reviewer.

### Risks & Blockers

1. ABC-100 has 4 unassigned stories — need sprint planning
2. No feature branches for compensation work — verify with Paul
3. Flywheel effort algorithm handoff from Data team has no tracking ticket

### Bottom Line

Team is productive on BAU work. Flywheel compensation is the concern —
stories exist but no code is flowing. Raise in next standup. Consider
assigning ABC-119 and ABC-120 this sprint.
```
