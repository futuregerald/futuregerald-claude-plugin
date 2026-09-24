# 📊 Team Pulse — {{SCOPE}}
> **Window:** {{DATE_RANGE}}  
> **Engineering Manager:** {{MANAGER_NAME}}  
> **Primary Repos:** {{PRIMARY_REPOS}}  
> **Tracking:** {{TRACKER_INFO}}  
> **Interactive Dashboard:** [`team-pulse-{{END_DATE}}.html`](file:///{{HTML_REPORT_PATH}})

---

## 🧭 Executive Scorecard

| Metric | Status / Value | Operational Context |
| :--- | :---: | :--- |
| **Overall Health** | {{OVERALL_HEALTH_BADGE}} | {{OVERALL_HEALTH_CONTEXT}} |
| **PR Velocity** | {{PR_VELOCITY_VALUE}} | {{PR_VELOCITY_CONTEXT}} |
| **Net Code Impact** | {{CODE_IMPACT_VALUE}} | {{CODE_IMPACT_CONTEXT}} |
| **Review Bottlenecks** | {{REVIEW_BOTTLENECKS_VALUE}} | {{REVIEW_BOTTLENECKS_CONTEXT}} |

<!--
Single-person (1:1) scope: replace the rows above with this week's numbers only, each value
linked to the query that produced it, and add below the table:
"Epic progress below covers the whole epic, not just this week."
| **Overall** | {Badge} | {1 sentence} |
| **PRs this week** | [N merged](search) · [N opened](search) | {window} |
| **Open PRs now** | [N open](search) | {oldest, stale, unreviewed, and any old drafts, each linked} |
| **Tickets done this week** | [N done](query) | {which epics} |
| **Reviews given this week** | [N reviews](search) | {how it compares with teammates} |
-->

---

## 01. Executive Summary & Strategic Context

> **Headline:** {{HEADLINE}}

### 🏛️ Leadership & Strategic Shifts
{{STRATEGIC_SHIFTS}}

### 🎯 Scope & Milestone Timelines
{{MILESTONE_TIMELINES}}

---

## 02. Active Initiatives & Epics Status

<!-- Repeat for each active initiative/epic -->
<!--
### {Index}. [{Epic Name}]({Jira URL}) ([`{Key}`]({Jira URL})) — {🟢 On Track | 🟡 Needs Attention | 🔴 At Risk}
*{Priority} · part of [{Parent Key} {Parent Name}]({Parent URL}).* {One sentence: what the epic delivers}  
**Lead:** {Lead Name} · **Progress:** `[{Unicode Progress Bar e.g. █████░░░░░}]` **[{N}% Complete]({query URL})** ({Done} of {Total} issues done)  
**Split:** Frontend {N} done / {N} left · Backend {N} done / {N} left

{If Needs Attention, At Risk, or Blocked — MANDATORY:}
> ⚠️ **Why It Needs Attention / At Risk:**  
> {Exact root cause, failure mode, idle duration, or blocking dependency. If tracker commentary is silent, cite empirical observation.}

* **Activity in Window:**
  - {Summary bullet 1 with hyperlinked PRs}
* **What's Left (TL;DR):**
  - **{Category}:** {Remaining task or PR}
-->
{{ACTIVE_EPICS_BLOCKS}}

---

{{WORK_BREAKDOWN_BLOCKS}}
<!-- Single-person and single-epic scopes only; delete for team scope. Format: section 02b in references/report-format.md -->

## 03. PRs in Flight & Review Backlog

> **Summary:** {{PR_SUMMARY_LINE}}

| Repo / PR | Title | Author | Opened | Age | Status | Impact | Action Required |
| :--- | :--- | :--- | :---: | :---: | :---: | :---: | :--- |
{{PR_TABLE_ROWS}}

<!--
Example row format:
| [`repo #123`](https://github.com/org/repo/pull/123) | [`KEY-456`](https://tracker/browse/KEY-456) Title | [@author](https://github.com/author) | Jun 1 | 2d | 🟡 **Stale Review** | `+120 / -10` | Needs review from @teammate |
-->

---

## 04. Team Roster & Individual Workloads

<!-- Repeat for each team member -->
<!--
### {Name} — {Role} ([`@{handle}`]({GH URL})) — {🟢 On Track | 🟡 Needs Attention | 🔴 At Risk}
* **Summary:** {1-3 sentences: what they worked on, PR activity, blockers}
{If Needs Attention or At Risk:}
* > ⚠️ **Why Needs Attention:** {Exact failure mode or stale PR delay}
* **Metrics:** {N} Merged PRs · {N} In Flight · **Focus:** {Focus Area}
-->
{{PEOPLE_BLOCKS}}

---

### 👑 Dedicated Engineering Manager Section: {{MANAGER_NAME}} ([`@{{MANAGER_GH_HANDLE}}`]({{MANAGER_GH_URL}})) — 🟢 On Track
* **Technical Spikes & Backlog Architecture:**
{{EM_TECHNICAL_WORK}}
* **Domain Leadership & Governance:**
{{EM_LEADERSHIP_WORK}}

---

## 05. Master Risks & Blockers Register

<!-- Repeat for each prioritized risk -->
<!--
### {🔴 P0 | 🟡 P1 | 🔵 P2} — {Risk Title} ([`{Key}`]({Jira URL}))
* **Risk & Impact:** {Description of failure mode, security, auth or billing implications}
* **Immediate Action:** {Explicit owner and action item for today's standup}
-->
{{RISK_ITEMS_BLOCKS}}

---

## 06. Engineering Manager Bottom Line

> **Bottom Line:** {{BOTTOM_LINE}}
