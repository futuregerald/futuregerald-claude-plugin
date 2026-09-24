# Team Pulse Report Format

## Detail Depends on Scope

The same skeleton serves every scope; how deep each section goes does not.

| Scope | Epic cards | Work breakdown | Person sections |
|-------|-----------|----------------|-----------------|
| **Team or multi-initiative** | Description, parent, priority, progress, why, what's left | **One line per epic:** `Frontend: N done / N left · Backend: N done / N left` | 1–3 sentences each |
| **Single person (1:1 prep)** | Same | **Full section 02b:** per epic, frontend and backend tables with one line per PR, remaining work described, and sibling work the epic depends on | Wins, 30-day trend, reviews given, talking points, questions to ask |
| **Single epic or initiative** | Same | Full section 02b for that epic only | 1–3 sentences each |

**In a 1:1, statistics cover the window only (default: the past week).** The scorecard, PR
counts, tickets done and reviews given are all for that week. Epic progress is the exception:
it covers the whole epic, because where the epic stands is worth discussing whatever the
window. The section 02b tables may go back further, since they describe the epic's work, not
the person's week.

A team report with per-PR tables is too long to read before a standup. A 1:1 report without them
leaves the manager unable to say what the person actually built.

## Rules for Every Scope

- **Link everything a reader might want to open.** Ticket keys, PRs, people (their profile),
  repos, and **every count**: "34 tickets resolved" links to the tracker query that returns
  those 34, "48 reviews" links to the GitHub search that returns them. A number without a link
  cannot be checked.
- **Every epic card says what the epic is.** One plain sentence from the epic's description,
  its parent initiative (linked) and its priority. A title alone does not tell the reader
  whether the work matters.
- **Every PR table has Opened and Merged date columns.** Merged shows the date, `open`, or
  `closed unmerged <date>`. Age alone hides when work landed.
- **Remaining work is described, not just listed.** Each open ticket gets a short line on what
  it is, taken from its description; if it has none, say so and describe it from its title.

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
*{Priority} · part of [{Parent Key} {Parent Name}]({Parent URL}).* {One sentence: what the epic delivers, from its description}  
**Lead:** {Lead Name} · **Progress:** `[█████░░░░░]` **[{N}% Complete]({tracker query for the epic's children})** ({Done}/{Total} issues done)
**Split:** Frontend {N} done / {N} left · Backend {N} done / {N} left

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

| Repo / PR | Title | Author | Opened | Age | Status | Impact | Action Required |
| :--- | :--- | :--- | :---: | :---: | :---: | :---: | :--- |
| [`repo #123`]({PR URL}) | [`{Key}`]({Jira URL}) {Title} | [@handle]({GH URL}) | {Mon D} | {Age} | {Status Badge} | `+{A} / -{D}` | {Specific action and reviewers} |

---

## 02b. Work Breakdown by Epic (single-person and single-epic scopes only)

### {Epic Name} ([`{Key}`]({Jira URL})): what was built

**Summary:** {2–4 sentences: which side of the stack the work is on, what is real vs stubbed, and what the epic still depends on}

**Backend ([{repo}]({repo URL})): {N} done**

| Ticket | Status | PR | Opened | Merged | What it does |
|---|---|---|---|---|---|
| [{Key}]({URL}) | Done | [#123]({PR URL}) | {Mon D} | {Mon D / open / closed unmerged Mon D} | {One line from the PR body: the user-visible change} |

**Frontend ([{repo}]({repo URL})): {N} merged, {N} in open PRs**

{Same table}

**Still to do:**

| Ticket | Status / owner | What it is |
|---|---|---|
| [{Key}]({URL}) | {Status}, {owner or unassigned} | {One line from the description} |

**For the 1:1:** {1–2 questions this breakdown raises}

---

## 04. Team Roster & Individual Workloads

### {Person Name} — {Role} ([`@{handle}`]({GH URL})) — {Badge}
* **Summary:** {1-3 sentences: focus area, merged PRs, active work}
{Single-person scope adds:}
* **Wins to recognise:** {specific merged work, with links}
* **This week:** {[N PRs merged](search) · [N opened](search) · [N tickets done](query) · [N reviews given](search)}. Statistics cover the window only; do not add a longer trend
* **Reviews given:** {[N reviews](search), and whose work they concentrate on}
* **Talking points:** {numbered, each tied to linked evidence}
* **Questions to ask:** {2–4 open questions}
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
- `{{ACTIVE_EPICS_CARDS}}`: Grid of epic cards. Each card carries `.card-meta` (priority and linked parent), `p.card-summary` (what the epic delivers), the progress bar with a linked `%`, a `.card-bullets` list (activity and the frontend/backend split), a `.why-box` when not On Track, and `.whats-left`.
- `{{WORK_BREAKDOWN}}`: Section 02b as HTML tables for single-person and single-epic scopes. **Replace it with an empty string for team scope**; the template hides the section when it is empty.
- `{{PR_SUMMARY_LINE}}` & `{{PR_TABLE_ROWS}}`: PR status summary and table rows with hyperlinks.
- `{{PEOPLE_CARDS}}` & `{{EM_CARD}}`: Team roster workload cards and dedicated EM card.
- `{{RISK_ITEMS}}`: Prioritized P0/P1/P2 operational risk cards.
- `{{BOTTOM_LINE}}`: Bottom line synthesis callout.

### Visual Design System

The look is a dark instrument panel: graphite ground, one amber accent, narrow bold uppercase
headings, monospace numbers, bordered cards with no shadows. Both `assets/template.html` (pulse) and
`assets/forecast.html` (forecast) carry the same tokens, so every report reads as one family.

- **Type (offline-safe, system fallbacks):** display `"Archivo Narrow", "Arial Narrow", "Roboto Condensed", system sans`;
  body `"Archivo", system sans`; numbers and labels `"JetBrains Mono", ui-monospace`. No web-font
  links: the named faces are used when installed and the fallbacks otherwise.
- **Tokens:** `--ground #0F1113`, `--surface #171A1E`, `--sunk #1E2227`, `--line #2B3139`,
  `--ink #ECEFF2`, `--ink2 #B3BCC6`, `--ink3 #8B96A2`, `--acc #FFB020` (links `#FFC857`).
- **Status colours differ in lightness, not only hue:** on track `--good #5AB8FF` (blue), needs
  attention `--warn #FFB020` (amber), at risk `--crit #FF7A59` (orange-red), each with a dark tint
  background (`--goodbg`, `--warnbg`, `--critbg`). Never red against green.
- **Light and print:** `<html data-theme="light">` switches to a light palette with darker accents,
  and `@media print` applies it automatically, so a printed or PDF'd report is legible.
- **Components:** section numbers print as `[02]` in the accent; headings are uppercase; cards are
  6px-radius with a 1px border; pills are fully rounded; the bottom line is an amber-bordered panel.
- **Every reference is a link** — tickets, PRs, people, repos, and every count.

## Length Guidelines

| Scope | Target Length |
|-------|-------------|
| Full team summary (Markdown) | 300-500 words |
| Full team HTML dashboard | Complete standalone dashboard |
| Single person (1:1 prep) | 250-400 words, plus the section 02b tables |
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
