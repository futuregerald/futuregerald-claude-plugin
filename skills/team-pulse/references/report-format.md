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

**Tester Compensation Overhaul** (DL-2094) — Needs Attention
- ADR (DL-2220) and schema migration (DL-2190) both in code review
- No feature branches exist in repo — "Code Review" status may be misleading
- 4 stories still To Do with no assignee (DL-2192, DL-2193, DL-2196, DL-2200)

### PRs in Flight

| PR | Author | Status | Age | Review |
|----|--------|--------|-----|--------|
| #7823 Fix pagination | @mauricio-reis | Merged | 1d | Approved |
| #7819 Schema migration | @Lucianolo | Open | 3d | Pending |

### People

**Paul Ursache** — Needs Attention
DL-2094 owner. ADR and A1 schema story in code review but no branches
pushed to repo. Clarify if work is local or if status needs updating.

**Mauricio Reis** — On Track
Shipped #7823 (pagination fix). Picked up DL-2078 slim serializer. Active reviewer.

### Risks & Blockers

1. DL-2094 has 4 unassigned stories — need sprint planning
2. No feature branches for compensation work — verify with Paul
3. Flywheel effort algorithm handoff from Data team has no tracking ticket

### Bottom Line

Team is productive on BAU work. Flywheel compensation is the concern —
stories exist but no code is flowing. Raise in next standup. Consider
assigning DL-2192 and DL-2193 this sprint.
```
