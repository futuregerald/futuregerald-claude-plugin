# Team Pulse Report Format

## Structure

The pulse is delivered both as a concise summary in chat, and as a complete, publication-grade markdown document (`team-pulse-<END_DATE>.md`) ready to post into a GitHub Issue/Discussion, Jira ticket, or Confluence document:

```markdown
# 📊 Team Pulse — {scope}
> **Window:** {date range}  
> **Engineering Manager:** {Manager Name}  
> **Primary Repos:** {repo links}  
> **Tracking:** {tracker links}  
> **Interactive Dashboard:** [`team-pulse-<END_DATE>.html`](file:///path/to/team-pulse-<END_DATE>.html)

---

## 🧭 Executive Scorecard

| Metric | Status / Value | Operational Context |
| :--- | :---: | :--- |
| **Overall Health** | {Badge} | {1-sentence context} |
| **PR Velocity** | {N Merged} | {In flight breakdown} |
| **Net Code Impact** | {Lines +/-} | {Major technical debt or feature impact} |
| **Review Bottlenecks** | {N Stale PRs} | {Stale review flags} |

---

## 01. Executive Summary & Strategic Context

> **Headline:** {one sentence — on track, at risk, or blocked + why}

### 🏛️ Leadership & Strategic Shifts
- {Key leadership, org, or reorg context}

### 🎯 Scope & Milestone Timelines
- {Deadlines, GA dates, scope boundaries}

---

## 02. Active Initiatives & Epics Status

### 1. [{Epic Name}]({Jira URL}) ([`{Key}`]({Jira URL})) — {Badge}
*{Initiative summary}*  
**Lead:** {Lead Name} · **Progress:** `[█████░░░░░]` **{N}% Complete** ({Done}/{Total} issues done)

{If assessment is Needs Attention, At Risk, or Blocked — MANDATORY:}
> ⚠️ **Why It Needs Attention / At Risk:**  
> {Exact root cause, failure mode, idle duration, or blocking dependency. If tracker commentary is silent, cite empirical observation: e.g. "No commits/transitions for N days despite 'In Progress' status (no blocker recorded in tracker; direct check-in required)."}

* **Activity in Window:** {1-2 bullets with hyperlinked PRs}
* **What's Left (TL;DR):**
  - **{Area 1}:** {Deliverable with linked PR/issue}
  - **{Area 2}:** {Deliverable with linked PR/issue}

---

## 03. PRs in Flight & Review Backlog

> **Summary:** {N} PRs in flight across repositories. {N} stale PRs flagged.

| Repo / PR | Title | Author | Age | Status | Impact | Action Required |
| :--- | :--- | :--- | :---: | :---: | :---: | :--- |
| [`repo #123`]({PR URL}) | [`{Key}`]({Jira URL}) {Title} | [@handle]({GH URL}) | {Age} | {Status Badge} | `+{A} / -{D}` | {Specific action and reviewers} |

---

## 04. Team Roster & Individual Workloads

### {Person Name} — {Role} ([`@{handle}`]({GH URL})) — {Badge}
* **Summary:** {1-3 sentences: focus area, merged PRs, active work}
{If Needs Attention / At Risk: state exact reason why with alert}
* **Metrics:** {N} Merged PRs · {N} In Flight · **Focus:** {Domain}

---

### 👑 Dedicated Engineering Manager Section: {EM Name} ([`@{handle}`]({GH URL})) — 🟢 On Track
* **Technical Spikes & Backlog Architecture:** {Spikes, discovery, ADR reviews}
* **Domain Leadership & Governance:** {Leadership, cross-team alignment, hiring}

---

## 05. Master Risks & Blockers Register

### 🔴 P0 — {Critical Issue Name} ([`{Key}`]({Jira URL}))
* **Risk & Impact:** {Description of failure mode, legal/contractual/auth risks}
* **Immediate Action:** {Explicit owner and action item for today}

### 🟡 P1 — {High Priority Issue Name}
* **Risk & Impact:** {Description of blocker or stale review chain}
* **Immediate Action:** {Explicit action item}

---

## 06. Engineering Manager Bottom Line

> **Bottom Line:** {2-3 sentences: synthesis of team health, top 3 priorities for today's standup, and escalations needed}
```

## Badge Format

Use inline text badges:
- **On Track** for green
- **Needs Attention** for yellow
- **At Risk** for red
- **Blocked** for stopped

## Standalone HTML Dashboard Specification

Every team pulse run generates a standalone visual HTML dashboard (`team-pulse-<END_DATE>.html`) by populating the bundled template at `assets/template.html`, and opens it in the browser.

### Zero External Dependencies (Offline-Safe)
The dashboard and template must **never reference external files, CDN scripts, remote stylesheets, or external fonts (such as Google Fonts)**. All styling, SVG icons, and interaction logic must be completely self-contained. Typography relies on local fonts with robust system font fallbacks so the dashboard renders identically and instantly offline or behind corporate firewalls.

### Bundled Template: `assets/template.html`
Populate `assets/template.html` by replacing its semantic placeholders:
- `{{REPORT_TITLE}}` & `{{REPORT_HEADING}}`: Report title and header banner.
- `{{DATE_RANGE}}`, `{{MANAGER_NAME}}`, `{{PRIMARY_REPOS}}`, `{{TRACKER_INFO}}`: Metadata bar items.
- `{{METRIC_TILES}}`: 4 metric summary tiles (Overall Health, PR Velocity, Net Impact, Review Bottlenecks).
- `{{HEADLINE}}` & `{{STRATEGIC_CONTEXT}}`: Executive narrative and org context.
- `{{ACTIVE_EPICS_CARDS}}`: Grid of epic cards with progress bars, `%` complete, `.why-box` root cause callouts, and `.whats-left` lists.
- `{{PR_SUMMARY_LINE}}` & `{{PR_TABLE_ROWS}}`: PR status summary and table rows with hyperlinks.
- `{{PEOPLE_CARDS}}` & `{{EM_CARD}}`: Team roster workload cards and dedicated EM card.
- `{{RISK_ITEMS}}`: Prioritized P0/P1/P2 operational risk cards.
- `{{BOTTOM_LINE}}`: Bottom line synthesis callout.

### Visual Design System

- **Typography (with Offline System Font Fallbacks):**
  - Serif: `"Newsreader", Georgia, Cambria, serif` (weights 400, 500, 600; with `font-display: swap`)
  - Sans-serif: `"Bricolage Grotesque", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif` (weights 500, 600, 700, 800)
  - Monospace: `"IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace` (weights 400, 500, 600)
- **Color Palette & Dark Mode Support:**
  - CSS variables for light/dark mode (`--ground`, `--surface`, `--sunk`, `--ink`, `--ink2`, `--ink3`, `--line`, `--soft`, `--acc`, `--accbg`, `--good`, `--warn`, `--crit`) with `@media (prefers-color-scheme: dark)`.
- **Key Dashboard Components:**
  1. **Header & Metric Tiles:** Overall Health tile, PR Velocity tile (merged / in-flight), Net Code Impact tile, Review Bottlenecks tile.
  2. **Executive Summary & Strategic Context:** High-level narrative, leadership/reorg changes, scope ceilings.
  3. **Active Initiatives Grid (`.card-grid`):**
     - Epic title linked to tracker
     - Visual progress bar (`.prog-bar`, `.prog-fill`) with quantitative completion `%` (`Done / Total` issues)
     - Explicit `.why-box` warning/danger callout for any epic not On Track
     - Scannable `.whats-left` block detailing remaining deliverables
  4. **Interactive PR Table:** PR link, author, age, status badge, lines (+/-), and required action. Stale PRs styled in warm warning background.
  5. **People Grid:** Roster cards with role, summary, stats, and explicit "Why Needs Attention" micro-callouts when flagged. Dedicated EM card.
  6. **Risks & Blockers Register:** Prioritized P0/P1/P2 cards with direct issue links.
  7. **Bottom Line Callout:** Highlighted synthesis card for executive readouts.
  8. **100% Hyperlinked References:** Every mention of a Jira key (`ABC-123`, `OPS-45`) and GitHub PR (`#xxxx`) anywhere in prose, table titles, action items, why boxes, what's left lists, or card metadata MUST be rendered as clickable hyperlinks. Never leave a bare identifier that forces the reader to search manually.

## Length Guidelines

| Scope | Target Length |
|-------|-------------|
| Full team summary (Markdown) | 300-500 words |
| Full team HTML dashboard | Complete standalone dashboard |
| Single person (1:1 prep) | 150-250 words |
| Single epic/initiative | 200-350 words |
| "What did we ship" | 100-200 words |

## Example Markdown Snippet

```
## Team Pulse: Platform Team (May 26 - Jun 3, 2026)

**Headline:** On track overall, but the billing payout work has no code pushed
yet despite tickets showing "In Progress" — flag with Dev Two.

### Active Work

**Billing Overhaul** (ABC-100) — Needs Attention · **35% Complete** (7/20 issues)
- ADR (ABC-121) and schema migration (ABC-118) both in code review.
- Initial payout models merged to main.

> ⚠️ **Why It Needs Attention:** No feature branches exist in repo for active payout stories — "Code Review" status is misleading and 4 stories have no assignee.

**What's Left (TL;DR):**
- Unblock ABC-121 ADR signoff.
- Assign and groom remaining 4 stories (ABC-119, ABC-120, ABC-124, ABC-128).
- Data team estimation model handoff.

### PRs in Flight

| PR | Author | Status | Age | Review |
|----|--------|--------|-----|--------|
| #412 Fix pagination | @dev-one | Merged | 1d | Approved |
| #409 Schema migration | @dev-two | Open | 3d | Pending |

### People

**Dev Two** — Needs Attention
ABC-100 owner. ADR and schema story in review but no branches pushed. Clarify if work is local.

**Dev One** — On Track
Shipped #412. Picked up ABC-104. Active reviewer across the team.

### Risks & Blockers

1. ABC-100 has 4 unassigned stories — needs sprint planning.
2. Estimation-model handoff from data team lacks tracking ticket.

### Bottom Line

Team is productive on BAU work. The payout work is the concern — stories exist but no code is flowing. Raise in standup today.
```
